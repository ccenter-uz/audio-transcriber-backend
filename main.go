package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Chunk struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	ChunkID string  `json:"chunk_id"`
}

type Response struct {
	JobID  string  `json:"job_id"`
	Chunks []Chunk `json:"chunks"`
}

func main() {
	url := "http://192.168.31.24:8000/vad-chunk"
	audioDir := "./internal/media/audio"

	allFiles, err := filepath.Glob(filepath.Join(audioDir, "*"))
	if err != nil {
		panic(err)
	}

	var audioFiles []string
	for _, file := range allFiles {
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" {
			audioFiles = append(audioFiles, file)
		}
	}

	for _, audioPath := range audioFiles {
		fmt.Println("Processing audio file:", audioPath)

		file, err := os.Open(audioPath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		var requestBody bytes.Buffer
		writer := multipart.NewWriter(&requestBody)

		part, err := writer.CreateFormFile("audio_file", filepath.Base(audioPath))
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(part, file)
		if err != nil {
			panic(err)
		}

		writer.WriteField("min_duration", "1")
		writer.WriteField("max_duration", "20")

		err = writer.Close()
		if err != nil {
			panic(err)
		}

		req, err := http.NewRequest("POST", url, &requestBody)
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Accept", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		var result Response
		err = json.Unmarshal(body, &result)
		if err != nil {
			panic(err)
		}

		outputDir := "./internal/media/segments"
		os.MkdirAll(outputDir, os.ModePerm)

		fmt.Println("Downloading chunks...")
		for _, chunk := range result.Chunks {
			downloadURL := fmt.Sprintf("http://192.168.31.24:8000/download/%s/%s", result.JobID, chunk.ChunkID)

			resp, err := http.Get(downloadURL)
			if err != nil {
				fmt.Println("Failed to download:", downloadURL)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				fmt.Println("Error response for", chunk.ChunkID, ":", resp.Status)
				continue
			}

			filename := filepath.Join(outputDir, chunk.ChunkID)
			outFile, err := os.Create(filename)
			if err != nil {
				fmt.Println("Error creating file:", filename)
				continue
			}

			_, err = io.Copy(outFile, resp.Body)
			if err != nil {
				fmt.Println("Error saving file:", filename)
			}
			outFile.Close()

			fmt.Println(result.JobID)
			fmt.Printf("Saved: %s (start=%.2f, end=%.2f)\n", chunk.ChunkID, chunk.Start, chunk.End)
		}
	}
}
