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

const getCrossPoolRecommendations = `-- name: GetCrossPoolRecommendations :many
WITH cross_pool AS (
  SELECT v.id, v.name, v.s3_url, v.thumbnail_url, v.duration, v.resolution, v.size, v.progress, v.view_count, v.owner, v.uploaded_at, v.updated_at FROM videos v
  WHERE v.id != $1
    AND v.progress > 0
    AND v.id NOT IN (
      SELECT video_id FROM video_category
      WHERE category_tag = ANY(SELECT category_tag FROM video_category WHERE video_id = $1)
    )
  ORDER BY v.view_count DESC, v.uploaded_at DESC
  LIMIT $2
)
SELECT id, name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner, uploaded_at, updated_at FROM cross_pool
UNION ALL
(
  SELECT v.id, v.name, v.s3_url, v.thumbnail_url, v.duration, v.resolution, v.size, v.progress, v.view_count, v.owner, v.uploaded_at, v.updated_at FROM videos v
  WHERE v.id != $1
    AND NOT EXISTS (SELECT 1 FROM cross_pool)
  ORDER BY RANDOM()
  LIMIT $2
)
LIMIT $2
`

type GetCrossPoolRecommendationsParams struct {
	ID    uuid.UUID
	Limit int32
}

type GetCrossPoolRecommendationsRow struct {
	ID           uuid.UUID
	Name         string
	S3Url        string
	ThumbnailUrl string
	Duration     int32
	Resolution   string
	Size         int32
	Progress     int32
	ViewCount    int32
	Owner        pgtype.UUID
	UploadedAt   pgtype.Timestamp
	UpdatedAt    pgtype.Timestamp
}

func (q *Queries) GetCrossPoolRecommendations(ctx context.Context, arg GetCrossPoolRecommendationsParams) ([]GetCrossPoolRecommendationsRow, error) {
	rows, err := q.db.Query(ctx, getCrossPoolRecommendations, arg.ID, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCrossPoolRecommendationsRow
	for rows.Next() {
		var i GetCrossPoolRecommendationsRow
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getGenericFeed = `-- name: GetGenericFeed :many
SELECT id, name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner, uploaded_at, updated_at FROM videos
WHERE progress > 0
ORDER BY view_count DESC, uploaded_at DESC
LIMIT $1
`

func (q *Queries) GetGenericFeed(ctx context.Context, limit int32) ([]Video, error) {
	rows, err := q.db.Query(ctx, getGenericFeed, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Video
	for rows.Next() {
		var i Video
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
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
