package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(app *gin.Engine, logger *slog.Logger) {
	app.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "Hello"})
	})
}
