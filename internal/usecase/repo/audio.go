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

func (r *AudioFileRepo) GetById(ctx context.Context, id int) (*entity.AudioFile, error) {
	query := `
	SELECT id, filename, status, user_id FROM audio_files WHERE id = $1`
	audioFile := &entity.AudioFile{}
	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(&audioFile.ID, &audioFile.Filename, &audioFile.Status, &audioFile.UserID)
	if err != nil {
		return nil, err
	}

	return audioFile, nil
}