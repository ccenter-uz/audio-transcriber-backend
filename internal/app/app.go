package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/mirjalilova/voice_transcribe/config"
	v1 "github.com/mirjalilova/voice_transcribe/internal/controller/http"
	"github.com/mirjalilova/voice_transcribe/internal/usecase"

	"github.com/mirjalilova/voice_transcribe/pkg/httpserver"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

func Run(cfg *config.Config) {
	l := logger.New(cfg.Log.Level)

	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal(fmt.Errorf("app - Run - postgres.New: %w", err))
	}
	defer pg.Close()

	// Use case
	useCase := usecase.New(pg, cfg, l)

	// HTTP Server
	handler := gin.New()
	v1.NewRouter(handler, l, cfg, useCase)

	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Port))

	l.Info("app - Run - httpServer: %s", cfg.HTTP.Port)

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: %s", s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}

}
