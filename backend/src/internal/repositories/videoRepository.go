package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
)

type VideoRepository interface {
	CreateVideo(ctx context.Context, params db.CreateVideoParams) (*db.Video, error)
	UpdateVideoAfterProcessing(ctx context.Context, params db.UpdateVideoAfterProcessingParams) (*db.Video, error)
	FindVideoByVideoID(ctx context.Context, videoID uuid.UUID) (*db.Video, error)
	CheckUserLikedVideo(ctx context.Context, userID, videoID uuid.UUID) (*db.VideoLike, error)
	DeleteLike(ctx context.Context, userID, videoID uuid.UUID) error
	CreateLike(ctx context.Context, userID, videoID uuid.UUID) error
	CountVideoLikes(ctx context.Context, videoID uuid.UUID) (int64, error)
	ToggleLikeTx(ctx context.Context, userID, videoID uuid.UUID) (bool, int64, error)
	GetRecommendationFeed(ctx context.Context, limit int, videoID uuid.UUID) ([]db.GetCrossPoolRecommendationsRow, error)
	GetGenericFeed(ctx context.Context, limit int) ([]db.Video, error)
}

type videoRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewVideoRepository(pool *pgxpool.Pool) VideoRepository {
	return &videoRepository{queries: db.New(pool), pool: pool}
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

func (r *videoRepository) FindVideoByVideoID(ctx context.Context, videoID uuid.UUID) (*db.Video, error) {
	video, err := r.queries.FindVideoByVideoId(ctx, videoID)
	if err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) CheckUserLikedVideo(ctx context.Context, userID, videoID uuid.UUID) (*db.VideoLike, error) {
	params := db.CheckUserLikedVideoParams{
		VideoID: videoID,
		UserID:  userID,
	}
	liked, err := r.queries.CheckUserLikedVideo(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &liked, nil
}

func (r *videoRepository) DeleteLike(ctx context.Context, userID, videoID uuid.UUID) error {
	params := db.DeleteVideoLikeParams{
		VideoID: videoID,
		UserID:  userID,
	}

	err := r.queries.DeleteVideoLike(ctx, params)
	if err != nil {
		return err
	}

	return nil
}

func (r *videoRepository) CreateLike(ctx context.Context, userID, videoID uuid.UUID) error {
	params := db.InsertVideoLikeParams{
		VideoID: videoID,
		UserID:  userID,
	}
	err := r.queries.InsertVideoLike(ctx, params)
	if err != nil {
		return err
	}

	return nil
}

func (r *videoRepository) CountVideoLikes(ctx context.Context, videoID uuid.UUID) (int64, error) {
	count, err := r.queries.CountVideoLikes(ctx, videoID)

	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *videoRepository) ToggleLikeTx(ctx context.Context, userID, videoID uuid.UUID) (bool, int64, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, 0, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.queries.WithTx(tx)

	_, err = qtx.CheckUserLikedVideo(ctx, db.CheckUserLikedVideoParams{
		VideoID: videoID,
		UserID:  userID,
	})
	var liked bool
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := qtx.InsertVideoLike(ctx, db.InsertVideoLikeParams{
				VideoID: videoID,
				UserID:  userID,
			}); err != nil {
				return false, 0, err
			}
			liked = true
		} else {
			return false, 0, err
		}
	} else {
		if err := qtx.DeleteVideoLike(ctx, db.DeleteVideoLikeParams{
			VideoID: videoID,
			UserID:  userID,
		}); err != nil {
			return false, 0, err
		}
		liked = false
	}

	count, err := qtx.CountVideoLikes(ctx, videoID)
	if err != nil {
		return false, 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, err
	}

	return liked, count, nil
}


func (r *videoRepository) GetRecommendationFeed(ctx context.Context, limit int, videoID uuid.UUID) ([]db.GetCrossPoolRecommendationsRow, error){
	params := db.GetCrossPoolRecommendationsParams{
		ID: videoID,
		Limit: int32(limit),
	}
	recs, err := r.queries.GetCrossPoolRecommendations(ctx, params)
	if err != nil {
		return nil, err
	}

	return recs, nil
}

func (r *videoRepository) GetGenericFeed(ctx context.Context, limit int) ([]db.Video, error) {
	videos, err := r.queries.GetGenericFeed(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	return videos, nil
}