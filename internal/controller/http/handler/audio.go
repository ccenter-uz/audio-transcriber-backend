package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	tempPath := filepath.Join(os.TempDir(), file.Filename)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving zip file"})
		return
	}
	defer os.Remove(tempPath)

	r, err := zip.OpenReader(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to open zip file"})
		return
	}
	defer r.Close()

	outputDir := "./internal/media/audio"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create output folder"})
		return
	}

	for _, f := range r.File {
		if !isAudioFile(f.Name) {
			continue
		}

		dstPath := filepath.Join(outputDir, filepath.Base(f.Name))
		dstFile, err := os.Create(dstPath)
		if err != nil {
			fmt.Println("Error creating file:", err)
			continue
		}
		fmt.Println(dstPath, f.Name)
		rc, err := f.Open()
		if err != nil {
			dstFile.Close()
			fmt.Println("Error opening file:", err)
			continue
		}
		_, err = io.Copy(dstFile, rc)
		dstFile.Close()
		rc.Close()
		if err != nil {
			fmt.Println("Error writing file:", err)
			continue
		}

		err = h.UseCase.AudioFileRepo.Create(c, &entity.CreateAudioFile{
			Filename: f.Name,
			FilePath: dstPath,
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err})
			slog.Error("error:", err)
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
