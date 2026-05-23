package initializers

import (
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectToDB(dbURL string, logger *slog.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to db", "error", err.Error())
		return nil, err
	}
	logger.Info("Connected to db successfully")

	return db, nil
}