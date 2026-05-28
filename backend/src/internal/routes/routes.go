package routes

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/whotterre/vuetube/src/internal/config"
	"github.com/whotterre/vuetube/src/internal/handlers"
	"github.com/whotterre/vuetube/src/internal/middleware"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/services"
	"github.com/whotterre/vuetube/src/internal/utils"
	"github.com/whotterre/vuetube/src/internal/workers"
)

func SetupRoutes(app *gin.Engine, db *pgxpool.Pool, cfg *config.Config, logger *slog.Logger, asynqClient *asynq.Client) {
	userRepository := repositories.NewUserRepository(db)
	authService := services.NewUserService(userRepository, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)

	app.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Hello"})
	})

	authRoutes := app.Group("/auth")
	authRoutes.POST("/login", authHandler.Login)
	authRoutes.POST("/signup", authHandler.Signup)

	videoRepository := repositories.NewVideoRepository(db)
	videoService := services.NewVideoService(videoRepository, asynqClient)
	videoHandler := handlers.NewVideoHandler(videoService, cfg)

	s3Helper := utils.NewS3Helper(context.Background())
	_ = workers.NewVideoProcessor(videoRepository, s3Helper, workers.VideoProcessorConfig{
		BucketName: cfg.BucketName,
		AWSRegion:  cfg.AWSRegion,
	})

	videoRoutes := app.Group("/videos", middleware.RequireAuth(cfg.JWTSecret))
	videoRoutes.POST("/upload", videoHandler.UploadVideo)
}
