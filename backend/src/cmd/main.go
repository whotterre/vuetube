package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/whotterre/vuetube/src/internal/config"
	"github.com/whotterre/vuetube/src/internal/initializers"
	"github.com/whotterre/vuetube/src/internal/models"
	"github.com/whotterre/vuetube/src/internal/routes"
)

func main() {
	app := gin.Default()
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

	err = db.AutoMigrate(&models.User{}, &models.Job{}, &models.Video{}, &models.VideoTag{}, &models.WatchHistory{})
	if err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		return
	}

	routes.SetupRoutes(app, logger)

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

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Failed to shut down server", "error", err)
		}
	}
}
