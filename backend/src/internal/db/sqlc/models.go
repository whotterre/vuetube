package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID        uuid.UUID
	Email     string
	FirstName string
	LastName  string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt pgtype.Timestamptz
}

type Video struct {
	ID           uuid.UUID
	Name         string
	S3Url        string
	ThumbnailUrl string
	Duration     int32
	Resolution   string
	Size         int32
	Progress     pgtype.Int4
	ViewCount    int32
	Owner        pgtype.UUID
	UploadedAt   pgtype.Timestamp
	UpdatedAt    pgtype.Timestamp
}
