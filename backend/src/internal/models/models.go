package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Password  string     `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	Videos       []Video        `json:"videos,omitempty"`
	WatchHistory []WatchHistory `json:"watch_history,omitempty"`
}

type Video struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Duration      int        `json:"duration"`
	Resolution    string     `json:"resolution"`
	FileSize      int64      `json:"file_size"`
	RawS3URL      string     `json:"raw_s3_url"`
	ManifestS3URL string     `json:"manifest_s3_url"`
	Status        string     `json:"status"`
	ThumbnailURL  string     `json:"thumbnail_url"`
	ViewCount     int64      `json:"view_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`

	User         User           `json:"user"`
	Jobs         []Job          `json:"jobs,omitempty"`
	WatchHistory []WatchHistory `json:"watch_history,omitempty"`
	Tags         []VideoTag     `json:"tags,omitempty"`
}

type Job struct {
	ID         uuid.UUID `json:"id"`
	VideoID    uuid.UUID `json:"video_id"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	RetryCount int       `json:"retry_count"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Video Video `json:"video"`
}

type WatchHistory struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	VideoID   uuid.UUID `json:"video_id"`
	WatchedAt time.Time `json:"watched_at"`
	Progress  int       `json:"progress"`

	User  User  `json:"user"`
	Video Video `json:"video"`
}

type VideoTag struct {
	ID      uuid.UUID `json:"id"`
	VideoID uuid.UUID `json:"video_id"`
	Tag     string    `json:"tag"`

	Video Video `json:"video"`
}
