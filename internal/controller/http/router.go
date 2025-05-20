package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/mirjalilova/voice_transcribe/config"
	_ "github.com/mirjalilova/voice_transcribe/docs"
	"github.com/mirjalilova/voice_transcribe/internal/controller/http/handler"
	middleware "github.com/mirjalilova/voice_transcribe/internal/controller/http/middlerware"
	"github.com/mirjalilova/voice_transcribe/internal/usecase"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/minio"
)

func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// NewRouter -.
// Swagger spec:
// @title       Voice Transcribe API
// @description This is a sample server Voice Transcribe server.
// @version     1.0
// @BasePath    /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewRouter(engine *gin.Engine, l *logger.Logger, config *config.Config, useCase *usecase.UseCase, minioClient *minio.MinIO) {
	// Options
	engine.Use(gin.Logger())
	//engine.Use(gin.Recovery())

	handlerV1 := handler.NewHandler(l, config, useCase, *minioClient)

	// Initialize Casbin enforcer

	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Frontend domenini yozish
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Authentication"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// e := casbin.NewEnforcer("config/rbac.conf", "config/policy.csv")
	// engine.Use(handlerV1.AuthMiddleware(e))
	engine.Use(TimeoutMiddleware(5 * time.Second))
	// engine.Use(TimeoutMiddleware(5 * time.Second))
	url := ginSwagger.URL("swagger/doc.json") // The url pointing to API definition
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
	// K8s probe
	engine.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	engine.Use(cors.Default())
	// Prometheus metrics
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	enforcer, err := casbin.NewEnforcer("./internal/controller/http/casbin/model.conf", "./internal/controller/http/casbin/policy.csv")
	if err != nil {
		slog.Error("Error while creating enforcer: ", err)
	}

	if enforcer == nil {
		slog.Error("Enforcer is nil after initialization!")
	} else {
		slog.Info("Enforcer initialized successfully.")
	}

	// engine.Static("/audios", "./internal/media/audio")
	// engine.Static("/chunks", "./internal/media/segments")

	// Routes
	router := engine.Group("/api/v1")
	{
		// auth
		router.POST("/auth/login", handlerV1.Login)
		router.GET("/auth/one", handlerV1.GetUser)

		// // user
		// router.POST("/user/create", handlerV1.CreateUser)
		// router.GET("/user/list", handlerV1.GetUsers)
		// router.GET("/user/:id", handlerV1.GetUser)
		// router.PUT("/user/update", handlerV1.UpdateUser)
		// router.DELETE("/user/delete", handlerV1.DeleteUser)

		// transcript
		router.GET("/transcript/list", handlerV1.GetTranscripts)
		router.GET("/transcript/:id", handlerV1.GetTranscript)
		router.PUT("/transcript/update", middleware.NewAuth(enforcer), handlerV1.UpdateTranscript)
		// router.PUT("/transcript/update/status", handlerV1.UpdateStatus)
		router.DELETE("/transcript/delete", handlerV1.DeleteTranscript)

		// audio_segment
		router.GET("/audio_segment", middleware.NewAuth(enforcer), handlerV1.GetAudioSegments)
		router.GET("/audio_segment/:id", middleware.NewAuth(enforcer), handlerV1.GetAudioSegment)
		router.DELETE("/audio_segment/delete", handlerV1.DeleteAudioSegment)

		// dashboard
		router.GET("/dashboard", handlerV1.GetTranscriptPercent)
		router.GET("/dashboard/user/:user_id", handlerV1.GetUserTranscriptStatictics)
		router.GET("/dataset_viewer", handlerV1.DatasetViewer)
		router.GET("/statistic", handlerV1.GetStatistic)

		// audio
		router.POST("/upload-zip-audio", handlerV1.UploadZipAndExtractAudio)
		router.GET("/audio_file/:id", handlerV1.GetAudioFile)
	}
}
