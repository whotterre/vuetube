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
	"golang.org/x/time/rate"
)

func SetupRoutes(app *gin.Engine, db *pgxpool.Pool, cfg *config.Config, logger *slog.Logger, asynqClient *asynq.Client) {
	userRepository := repositories.NewUserRepository(db)
	authService := services.NewUserService(userRepository, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)

	app.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Hello"})
	})

	authLimiter := rate.NewLimiter(rate.Limit(5), 1)

	authRoutes := app.Group("/auth", middleware.RateLimitMiddleware(authLimiter))
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
	likeLimiter := rate.NewLimiter(rate.Limit(5), 1)
	feedLimiter := rate.NewLimiter(rate.Limit(10), 1)


	app.GET("/feed", middleware.RateLimitMiddleware(feedLimiter), videoHandler.GetGenericFeed)

	videoRoutes.GET("/feed/:id", middleware.RateLimitMiddleware(feedLimiter), videoHandler.GetRecommendationFeed)
	videoRoutes.PATCH("/:id/like", middleware.RateLimitMiddleware(likeLimiter), videoHandler.ToggleVideoLike)

	
	publicVideoRoutes := app.Group("/videos")
	publicVideoRoutes.GET("/:id", videoHandler.GetVideo)
	publicVideoRoutes.GET("/:id/dash/manifest", videoHandler.ServeDashManifest)
	publicVideoRoutes.GET("/:id/dash/segment/:filename", videoHandler.ServeDashSegment)
	publicVideoRoutes.GET("/:id/thumbnail", middleware.RateLimitMiddleware(feedLimiter), videoHandler.ServeThumbnail)



}
