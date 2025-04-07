package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors" // Import the CORS package
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Swagger docs.
	"github.com/mirjalilova/voice_transcribe/config"
	_ "github.com/mirjalilova/voice_transcribe/docs"
	"github.com/mirjalilova/voice_transcribe/internal/controller/http/handler"
	"github.com/mirjalilova/voice_transcribe/internal/usecase"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
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
// @title       1009 API
// @description This is a sample server 1009 server.
// @version     1.0
// @BasePath    /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewRouter(engine *gin.Engine, l *logger.Logger, config *config.Config, useCase *usecase.UseCase) {
	// Options
	engine.Use(gin.Logger())
	//engine.Use(gin.Recovery())

	handlerV1 := handler.NewHandler(l, config, useCase)

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
	fmt.Println("router", 1)
	// Swagger
	url := ginSwagger.URL("swagger/doc.json") // The url pointing to API definition
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
	fmt.Println("Swagger", 2)
	// K8s probe
	engine.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	fmt.Println("Swagger", 3)
	// Prometheus metrics
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Routes
	router := engine.Group("/api/v1")
	{
		//auth
		router.POST("/auth/login", handlerV1.Login)
		router.GET("/region/list", handlerV1.GetRegions)
		router.GET("/region/:id", handlerV1.GetRegion)
		router.PUT("/region/:id", handlerV1.UpdateRegion)
		router.DELETE("/region/:id", handlerV1.DeleteRegion)

		//city
		router.POST("/city", handlerV1.CreateCity)
		router.GET("/city/list", handlerV1.GetCities)
		router.GET("/city/:id", handlerV1.GetCity)
		router.PUT("/city/:id", handlerV1.UpdateCity)
		router.DELETE("/city/:id", handlerV1.DeleteCity)

		router.POST("/data/import", handlerV1.UploadFile)
	}
}
