package handler

import (
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/usecase"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
)

type Handler struct {
	Logger  *logger.Logger
	Config  *config.Config
	UseCase *usecase.UseCase
}

func NewHandler(l *logger.Logger, c *config.Config, useCase *usecase.UseCase) *Handler {
	return &Handler{
		Logger:  l,
		Config:  c,
		UseCase: useCase,
	}
}
