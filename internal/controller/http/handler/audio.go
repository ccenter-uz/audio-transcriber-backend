package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
)

// UploadZipAndExtractAudio godoc
// @Summary Upload Zip file
// @Description Upload Zip file
// @Tags audio
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Zip file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/upload-zip-audio [post]
func (h *Handler) UploadZipAndExtractAudio(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		slog.Error("Error getting file from form", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	tempPath := filepath.Join(os.TempDir(), file.Filename)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		slog.Error("Error saving zip file", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving zip file"})
		return
	}
	defer os.Remove(tempPath)

	r, err := zip.OpenReader(tempPath)
	if err != nil {
		slog.Error("Error opening zip file", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to open zip file"})
		return
	}
	defer r.Close()

	outputDir := "./internal/media/audio"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		slog.Error("Error creating output folder", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create output folder"})
		return
	}

	for _, f := range r.File {
		if !isAudioFile(f.Name) {
			slog.Info("Skipping non-audio file", f.Name)
			continue
		}

		dstPath := filepath.Join(outputDir, filepath.Base(f.Name))
		dstFile, err := os.Create(dstPath)
		if err != nil {
			slog.Error("Error creating file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create file"})
			return
		}
		rc, err := f.Open()
		if err != nil {
			dstFile.Close()
			slog.Error("Error opening file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to open file"})
			return
		}
		_, err = io.Copy(dstFile, rc)
		dstFile.Close()
		rc.Close()
		if err != nil {
			slog.Error("Error writing file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to write file"})
			return
		}

		minioURL, err := h.MinIO.Upload(*h.Config, filepath.Base(dstPath), dstPath)
		if err != nil {
			slog.Error("Failed to upload file to MinIO", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to storage"})
			return
		}

		audio_id, err := h.UseCase.AudioFileRepo.Create(c, &entity.CreateAudioFile{
			Filename: f.Name,
			FilePath: minioURL,
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err})
			slog.Error("error:", err)
			return
		}

		err = h.Chunking(c, *audio_id, dstPath)
		if err != nil {
			slog.Error("Error chunking audio file", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to chunk audio file"})
			return
		}

		err = os.Remove(dstPath)
		if err != nil {
			slog.Error("Failed to remove local file after upload", "file", dstPath, "err", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Audio files saved successfully",
	})
}

func isAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".m4a" || ext == ".spx"
}

type Chunk struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	ChunkID string  `json:"chunk_id"`
}

type Response struct {
	JobID  string  `json:"job_id"`
	Chunks []Chunk `json:"chunks"`
}

func (h *Handler) Chunking(c *gin.Context, audio_id int, audioPath string) error {
	url := "http://192.168.31.24:8080/vad-chunk"

	file, err := os.Open(audioPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("audio_file", filepath.Base(audioPath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	writer.WriteField("min_duration", "1")
	writer.WriteField("max_duration", "20")

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to vad-chunk: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	outputDir := "./internal/media/segments"
	os.MkdirAll(outputDir, os.ModePerm)

	for _, chunk := range result.Chunks {
		downloadURL := fmt.Sprintf("http://192.168.31.24:8080/download/%s/%s", result.JobID, chunk.ChunkID)

		resp, err := http.Get(downloadURL)
		if err != nil {
			return fmt.Errorf("failed to download chunk: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download chunk: %s", resp.Status)
		}

		filename := filepath.Join(outputDir, chunk.ChunkID)
		outFile, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("error creating file: %w", err)
		}

		_, err = io.Copy(outFile, resp.Body)
		if err != nil {
			return fmt.Errorf("error saving file: %w", err)
		}
		outFile.Close()

		minioURL, err := h.MinIO.Upload(*h.Config, filepath.Base(filename), filename)
		if err != nil {
			slog.Error("Failed to upload file to MinIO", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to storage"})
			return err
		}

		err = h.UseCase.AudioSegmentRepo.Create(c, &entity.CreateAudioSegment{
			AudioId:  audio_id,
			FileName: minioURL,
			Duration: float32(chunk.End - chunk.Start),
		})
		if err != nil {
			return fmt.Errorf("failed to create audio segment: %w", err)
		}

		err = os.Remove(filename)
		if err != nil {
			slog.Error("Failed to remove local file after upload", "file", chunk.ChunkID, "err", err)
		}
	}
	return nil
}

// GetAudioFile godoc
// @Router /api/v1/audio_file/{id} [get]
// @Summary Get audio file
// @Description Get audio file
// @Security BearerAuth
// @Tags audio
// @Accept  json
// @Produce  json
// @Param id path int true "Audio ID"
// @Success 200 {object} entity.AudioFile
// @Failure 400 {object} entity.ErrorResponse
// @Failure 500 {object} entity.ErrorResponse
func (h *Handler) GetAudioFile(ctx *gin.Context) {

	// var user_id string
	// claims, exists := ctx.Get("claims")
	// if !exists {
	// 	slog.Error("error", "Unauthorized")
	// 	ctx.JSON(401, entity.ErrorResponse{
	// 		Code:    config.ErrorUnauthorized,
	// 		Message: "Unauthorizedd",
	// 	})
	// 	return
	// } else {
	// 	user_id = claims.(jwt.MapClaims)["id"].(string)
	// }

	// allowed, err := redis.IsRequestAllowed(ctx, h.Redis, user_id, 5, 10, 60)

	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
	// 	slog.Error("Error checking rate limit", slog.String("error", err.Error()))
	// 	return
	// }

	// if !allowed {
	// 	ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Try again later."})
	// 	slog.Warn("Rate limit exceeded for user", slog.String("user_id", user_id))
	// 	return
	// }

	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("GetAudioFile error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid audio ID",
		})
		return
	}

	audioFile, err := h.UseCase.AudioFileRepo.GetById(ctx, intId)
	if h.HandleDbError(ctx, err, "Error getting audio file") {
		slog.Error("GetAudioFile error", slog.String("error", err.Error()))
		return
	}

	slog.Info("AudioFile retrieved successfully")
	ctx.JSON(200, audioFile)
}
