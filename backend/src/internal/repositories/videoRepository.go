package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
)

type VideoRepository interface {
	CreateVideo(ctx context.Context, params db.CreateVideoParams) (*db.Video, error)
	UpdateVideoAfterProcessing(ctx context.Context, params db.UpdateVideoAfterProcessingParams) (*db.Video, error)
}

type videoRepository struct {
	queries *db.Queries
}

func NewVideoRepository(pool *pgxpool.Pool) VideoRepository {
	return &videoRepository{queries: db.New(pool)}
}

func (r *videoRepository) CreateVideo(ctx context.Context, params db.CreateVideoParams) (*db.Video, error) {
	video, err := r.queries.CreateVideo(ctx, params)
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) UpdateVideoAfterProcessing(ctx context.Context, params db.UpdateVideoAfterProcessingParams) (*db.Video, error) {
	_ = uuid.UUID{}
	video, err := r.queries.UpdateVideoAfterProcessing(ctx, params)
	if err != nil {
		return nil, err
	}
	return &video, nil
}
