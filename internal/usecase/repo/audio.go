package repo

import (
	"context"

	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

type AudioFileRepo struct {
	pg     *postgres.Postgres
	config *config.Config
	logger *logger.Logger
}

// New -.
func NewAudioFileRepo(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *AudioFileRepo {
	return &AudioFileRepo{
		pg:     pg,
		config: config,
		logger: logger,
	}
}

func (r *AudioFileRepo) Create(ctx context.Context, req *entity.CreateAudioFile) (*int, error) {
	query := `
	INSERT INTO audio_files (filename, file_path) VALUES($1, $2) RETURNING id`

	var id int
	err := r.pg.Pool.QueryRow(ctx, query, req.Filename, req.FilePath).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &id, nil
}
