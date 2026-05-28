package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createVideo = `-- name: CreateVideo :one
INSERT INTO "videos" 
(name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner, uploaded_at, updated_at
`

type CreateVideoParams struct {
	Name         string
	S3Url        string
	ThumbnailUrl string
	Duration     int32
	Resolution   string
	Size         int32
	Progress     pgtype.Int4
	ViewCount    int32
	Owner        pgtype.UUID
}

func (q *Queries) CreateVideo(ctx context.Context, arg CreateVideoParams) (Video, error) {
	row := q.db.QueryRow(ctx, createVideo,
		arg.Name,
		arg.S3Url,
		arg.ThumbnailUrl,
		arg.Duration,
		arg.Resolution,
		arg.Size,
		arg.Progress,
		arg.ViewCount,
		arg.Owner,
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
