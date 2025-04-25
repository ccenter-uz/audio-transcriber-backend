package usecase

import (
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/usecase/repo"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

type UseCase struct {
	// AuthRepo         AuthRepoI
	TranscriptRepo   TranscriptRepoI
	AudioSegmentRepo AudioSegmentRepoI
	AudioFileRepo    AudioFileRepoI
}

func New(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *UseCase {
	return &UseCase{
		// AuthRepo:         repo.NewAuthRepo(pg, config, logger),
		TranscriptRepo:   repo.NewTranscriptRepo(pg, config, logger),
		AudioSegmentRepo: repo.NewAudioSegmentRepo(pg, config, logger),
		AudioFileRepo:    repo.NewAudioFileRepo(pg, config, logger),
	}
}
