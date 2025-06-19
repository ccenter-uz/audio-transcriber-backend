package handler

import (
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/usecase"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/minio"
)

type Handler struct {
	Logger  *logger.Logger
	Config  *config.Config
	UseCase *usecase.UseCase
	MinIO   *minio.MinIO
	// Redis   *redis.Client
}

func NewHandler(l *logger.Logger, c *config.Config, useCase *usecase.UseCase, mn minio.MinIO) *Handler {
	return &Handler{
		Logger:  l,
		Config:  c,
		UseCase: useCase,
		MinIO:   &mn,
		// Redis:   rdb,
	}
}
