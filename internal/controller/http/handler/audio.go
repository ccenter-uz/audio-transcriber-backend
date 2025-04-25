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
	"strings"

	"github.com/gin-gonic/gin"
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

		audio_id, err := h.UseCase.AudioFileRepo.Create(c, &entity.CreateAudioFile{
			Filename: f.Name,
			FilePath: dstPath,
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
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Audio files saved successfully",
	})
}

func isAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".m4a"
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
	url := "http://192.168.31.24:8000/vad-chunk"

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
		downloadURL := fmt.Sprintf("http://192.168.31.24:8000/download/%s/%s", result.JobID, chunk.ChunkID)

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

		err = h.UseCase.AudioSegmentRepo.Create(c, &entity.CreateAudioSegment{
			AudioId:  audio_id,
			FileName: chunk.ChunkID,
		})
		if err != nil {
			return fmt.Errorf("failed to create audio segment: %w", err)
		}
	}
	return nil
}
