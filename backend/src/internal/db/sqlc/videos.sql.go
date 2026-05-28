package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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
