package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mirjalilova/voice_transcribe/config"
	v1 "github.com/mirjalilova/voice_transcribe/internal/controller/http"
	"github.com/mirjalilova/voice_transcribe/internal/usecase"

	"github.com/mirjalilova/voice_transcribe/pkg/httpserver"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/minio"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

func Run(cfg *config.Config) {

	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		panic(err)
	}

	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// "time" atributini Asia/Tashkentga o'zgartirish
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					return slog.Attr{
						Key:   slog.TimeKey,
						Value: slog.AnyValue(t.In(loc)),
					}
				}
			}
			return a
		},
	}))

	slog.SetDefault(slogger)

	l := logger.New(cfg.Log.Level)

	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal(fmt.Errorf("app - Run - postgres.New: %w", err))
	}
	defer pg.Close()

	// Use case
	useCase := usecase.New(pg, cfg, l)

	//MinIO
	minioClient, err := minio.MinIOConnect(cfg)
	if err != nil {
		slog.Error("Failed to connect to MinIO", err)
		return
	}

	// // Redis
	// var rdb = redis.NewClient(&redis.Options{
	// 	Addr: "redis:6379",
	// })

	// HTTP Server
	handler := gin.New()
	v1.NewRouter(handler, l, cfg, useCase, minioClient)

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
