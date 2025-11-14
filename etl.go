package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/pkg/minio"
)

type Handler struct {
  Config *config.Config
  DB     *sql.DB
  MinIO  *minio.MinIO
}

type TranscriptionItem struct {
  Audio         string `json:"audio"`
  Transcription string `json:"transcription"`
}

var (
  ChunksDir = "/home/feruza/Documents/chunks"
  ResultsDir = "/home/feruza/Documents/results"
)
func main() {
db, err := sql.Open("postgres", "postgres://postgres:mirxonjon@10.145.20.8:5432/voice_transcribe")

  if err != nil {
    log.Fatal("DB connect error:", err)
  }

  defer db.Close()

  cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

  minioClient, err := minio.MinIOConnect(cfg)
    if err != nil {
        log.Fatal("Failed to connect MinIO:", err)
    }


  h := &Handler{
    Config: cfg,
    DB:     db,
	MinIO:  minioClient,
  }

  err = h.ProcessAll()
  if err != nil {
    log.Fatal("Error:", err)
  }

  log.Println("All done!")
}

func (h *Handler) ProcessAll() error {
  entries, err := os.ReadDir(ChunksDir)
  if err != nil {
    return fmt.Errorf("read chunks folder: %w", err)
  }

  for _, entry := range entries {
    if !entry.IsDir() {
      continue
    }
    uuid := entry.Name()
    log.Println("Processing:", uuid)

    if err := h.ProcessOne(uuid); err != nil {
      log.Println("Error processing", uuid, err)
      continue
    }
  }

  return nil
}

func (h *Handler) ProcessOne(uuid string) error {
  ctx := context.Background()

  tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{})
  if err != nil {
    return err
  }

  // 1️⃣ audio_files ga yozish
  audioID, err := h.insertAudioFile(ctx, tx, uuid)
  if err != nil {
    tx.Rollback()
    return err
  }

  // 2️⃣ audit uchun segment nomlari → segment_id larini saqlaymiz
  segmentMap := make(map[string]int)

chunkDir := filepath.Join(ChunksDir, uuid)

err = filepath.WalkDir(chunkDir, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    if d.IsDir() {
        return nil
    }

    filename := d.Name()

    // Upload to MinIO → URL qaytadi
    filePath, err := h.MinIO.Upload(*h.Config, filename, path)
    if err != nil {
        return fmt.Errorf("minio upload error: %w", err)
    }

    // Insert segment — URLni filename ustuniga yozamiz
    segID, err := h.insertSegment(ctx, tx, audioID, filePath)
    if err != nil {
        return err
    }

    // Transcripts bilan match qilish uchun mapping filename → segmentId
    segmentMap[filename] = segID

    return nil
})


  if err != nil {
    tx.Rollback()
    return err
  }

  // 3️⃣ transcripts JSON o‘qish
  jsonPath := filepath.Join(ResultsDir, uuid+".json")
  jsonBytes, err := os.ReadFile(jsonPath)
  if err != nil {
    // JSON bo‘lmasa — faqat audio + segmentlar commit qilinadi
    return tx.Commit()
  }

  var items []TranscriptionItem
  if err := json.Unmarshal(jsonBytes, &items); err != nil {
    tx.Rollback()
    return err
  }

  // 4️⃣ transcripts ga INSERT
  for _, item := range items {
    segID := segmentMap[item.Audio]
    if segID == 0 {
      log.Println("WARNING:", item.Audio, "segment topilmadi")
      continue
    }

    if err := h.insertTranscript(ctx, tx, segID, item.Transcription); err != nil {
      tx.Rollback()
      return err
    }
  }

  return tx.Commit()
}

func (h *Handler) insertAudioFile(ctx context.Context, tx *sql.Tx, uuid string) (int, error) {
  query := `
    INSERT INTO audio_files (filename, file_path)
    VALUES ($1, $2)
    RETURNING id
  `
  // file_path misol sifatida
  filePath := fmt.Sprintf("minio://audio/%s/", uuid)

  var id int
  err := tx.QueryRowContext(ctx, query, uuid, filePath).Scan(&id)
  return id, err
}

func (h *Handler) insertSegment(ctx context.Context, tx *sql.Tx, audioID int, filePath string) (int, error) {
    query := `
        INSERT INTO audio_file_segments (audio_id, filename)
        VALUES ($1, $2)
        RETURNING id
    `
    var id int
    err := tx.QueryRowContext(ctx, query, audioID, filePath).Scan(&id)
    return id, err
}

func (h *Handler) insertTranscript(ctx context.Context, tx *sql.Tx, segmentID int, text string) error {
  query := `
    INSERT INTO transcripts (segment_id, transcribe_text, transcribe_option)
    VALUES ($1, $2, $3)
    ON CONFLICT (segment_id, deleted_at) DO NOTHING;
  `
  _, err := tx.ExecContext(ctx, query, segmentID, text, text)
  return err
}