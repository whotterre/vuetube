package initializers

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectToDB(dbURL string, logger *slog.Logger) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("Failed to connect to db", "error", err)
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		logger.Error("Failed to ping db", "error", err)
		return nil, err
	}
	logger.Info("Connected to db successfully")

	return pool, nil
}
