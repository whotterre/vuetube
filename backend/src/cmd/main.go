package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/whotterre/vuetube/src/internal/config"
	"github.com/whotterre/vuetube/src/internal/initializers"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/routes"
	"github.com/whotterre/vuetube/src/internal/tasks"
	"github.com/whotterre/vuetube/src/internal/utils"
	"github.com/whotterre/vuetube/src/internal/workers"
)

func main() {
	app := gin.Default()
	// Tentative: restrict later
	app.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		return
	}

	db, err := initializers.ConnectToDB(cfg.DatabaseURL, logger)
	if err != nil {
		return
	}
	defer db.Close()

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr}

	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	asynqServer := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 5,
		Queues:      map[string]int{"default": 1},
	})

	// Register task handlers
	mux := asynq.NewServeMux()
	s3Helper := utils.NewS3Helper(context.Background())
	videoRepo := repositories.NewVideoRepository(db)
	videoProcessor := workers.NewVideoProcessor(videoRepo, s3Helper, workers.VideoProcessorConfig{
		BucketName: cfg.BucketName,
		AWSRegion:  cfg.AWSRegion,
	})
	mux.HandleFunc(tasks.TypeVideoUpload, videoProcessor.HandleVideoUploadTask)

	go func() {
		if err := asynqServer.Run(mux); err != nil {
			logger.Error("asynq worker server failed", "error", err)
		}
	}()

	routes.SetupRoutes(app, db, cfg, logger, asynqClient)

	server := &http.Server{
		Addr:    cfg.Port,
		Handler: app,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		asynqServer.Shutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shut down server", "error", err)
		}
	}
}
