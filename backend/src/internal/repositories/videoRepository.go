package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
)

type VideoRepository interface {
	CreateVideo (ctx context.Context, params db.CreateVideoParams)  (*db.Video, error)
}

type videoRepository struct {
	queries *db.Queries
}

func NewVideoRepository(pool *pgxpool.Pool) VideoRepository {
	return &videoRepository{queries: db.New(pool)}
}

func (r *videoRepository) CreateVideo (ctx context.Context, params db.CreateVideoParams)  (*db.Video, error) {
	createdVideo, err := r.queries.CreateVideo(ctx, params)
	if err != nil {
		return nil, err
	}

	return &createdVideo, nil
}

