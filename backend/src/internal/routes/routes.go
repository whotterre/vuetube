package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/whotterre/vuetube/src/handlers"
	"github.com/whotterre/vuetube/src/internal/config"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/services"
)

func SetupRoutes(app *gin.Engine, db *pgxpool.Pool, cfg *config.Config, logger *slog.Logger) {
	userRepository := repositories.NewUserRepository(db)
	authService := services.NewUserService(userRepository, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)
	app.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Hello"})
	})
	
	authRoutes := app.Group("/auth")
	authRoutes.POST("/login", authHandler.Login)
	authRoutes.POST("/signup", authHandler.Signup)

}
