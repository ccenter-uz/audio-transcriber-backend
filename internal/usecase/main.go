package usecase

import (
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/usecase/repo"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

type UseCase struct {
	RegionRepo           RegionRepoI
}

func New(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *UseCase {
	return &UseCase{
		RegionRepo:           repo.NewRegionRepo(pg, config, logger),
	}
}
