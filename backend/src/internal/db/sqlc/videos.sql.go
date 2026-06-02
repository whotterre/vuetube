package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const checkUserLikedVideo = `-- name: CheckUserLikedVideo :one
SELECT video_id, user_id, created_at FROM video_likes WHERE video_id = $1 AND user_id = $2
`

type CheckUserLikedVideoParams struct {
	VideoID uuid.UUID
	UserID  uuid.UUID
}

func (q *Queries) CheckUserLikedVideo(ctx context.Context, arg CheckUserLikedVideoParams) (VideoLike, error) {
	row := q.db.QueryRow(ctx, checkUserLikedVideo, arg.VideoID, arg.UserID)
	var i VideoLike
	err := row.Scan(&i.VideoID, &i.UserID, &i.CreatedAt)
	return i, err
}

const countVideoLikes = `-- name: CountVideoLikes :one
SELECT COUNT(*) FROM video_likes WHERE video_id = $1
`

func (q *Queries) CountVideoLikes(ctx context.Context, videoID uuid.UUID) (int64, error) {
	row := q.db.QueryRow(ctx, countVideoLikes, videoID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createVideo = `-- name: CreateVideo :one
INSERT INTO "videos" (name, size, owner)
VALUES ($1, $2, $3)
RETURNING id, name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner, uploaded_at, updated_at
`

type CreateVideoParams struct {
	Name  string
	Size  int32
	Owner pgtype.UUID
}

func (q *Queries) CreateVideo(ctx context.Context, arg CreateVideoParams) (Video, error) {
	row := q.db.QueryRow(ctx, createVideo, arg.Name, arg.Size, arg.Owner)
	var i Video
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.S3Url,
		&i.ThumbnailUrl,
		&i.Duration,
		&i.Resolution,
		&i.Size,
		&i.Progress,
		&i.ViewCount,
		&i.Owner,
		&i.UploadedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteVideoLike = `-- name: DeleteVideoLike :exec
DELETE FROM video_likes
WHERE video_id = $1 AND user_id = $2
`

type DeleteVideoLikeParams struct {
	VideoID uuid.UUID
	UserID  uuid.UUID
}

func (q *Queries) DeleteVideoLike(ctx context.Context, arg DeleteVideoLikeParams) error {
	_, err := q.db.Exec(ctx, deleteVideoLike, arg.VideoID, arg.UserID)
	return err
}

const findVideoByVideoId = `-- name: FindVideoByVideoId :one
SELECT id, name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner, uploaded_at, updated_at FROM videos WHERE id = $1
`

func (q *Queries) FindVideoByVideoId(ctx context.Context, id uuid.UUID) (Video, error) {
	row := q.db.QueryRow(ctx, findVideoByVideoId, id)
	var i Video
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.S3Url,
		&i.ThumbnailUrl,
		&i.Duration,
		&i.Resolution,
		&i.Size,
		&i.Progress,
		&i.ViewCount,
		&i.Owner,
		&i.UploadedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const insertVideoLike = `-- name: InsertVideoLike :exec
INSERT INTO video_likes (video_id, user_id)
VALUES ($1, $2)
`

type InsertVideoLikeParams struct {
	VideoID uuid.UUID
	UserID  uuid.UUID
}

func (q *Queries) InsertVideoLike(ctx context.Context, arg InsertVideoLikeParams) error {
	_, err := q.db.Exec(ctx, insertVideoLike, arg.VideoID, arg.UserID)
	return err
}

const updateVideoAfterProcessing = `-- name: UpdateVideoAfterProcessing :one
UPDATE "videos"
SET
  s3_url       = $2,
  thumbnail_url = $3,
  duration     = $4,
  resolution   = $5,
  progress     = 100,
  updated_at   = now()
WHERE id = $1
RETURNING id, name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner, uploaded_at, updated_at
`

type UpdateVideoAfterProcessingParams struct {
	ID           uuid.UUID
	S3Url        string
	ThumbnailUrl string
	Duration     int32
	Resolution   string
}

func (q *Queries) UpdateVideoAfterProcessing(ctx context.Context, arg UpdateVideoAfterProcessingParams) (Video, error) {
	row := q.db.QueryRow(ctx, updateVideoAfterProcessing,
		arg.ID,
		arg.S3Url,
		arg.ThumbnailUrl,
		arg.Duration,
		arg.Resolution,
	)
	var i Video
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.S3Url,
		&i.ThumbnailUrl,
		&i.Duration,
		&i.Resolution,
		&i.Size,
		&i.Progress,
		&i.ViewCount,
		&i.Owner,
		&i.UploadedAt,
		&i.UpdatedAt,
	)
	return i, err
}
