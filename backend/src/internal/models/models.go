package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Videos       []Video        `json:"videos,omitempty"`
	WatchHistory []WatchHistory `json:"watch_history,omitempty"`
}

type Video struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Title         string         `json:"title" gorm:"not null;index"`
	Description   string         `json:"description"`
	Duration      int            `json:"duration"`
	Resolution    string         `json:"resolution"`
	FileSize      int64          `json:"file_size"`
	RawS3URL      string         `json:"raw_s3_url"`
	ManifestS3URL string         `json:"manifest_s3_url"`
	Status        string         `json:"status" gorm:"default:uploading;index"`
	ThumbnailURL  string         `json:"thumbnail_url"`
	ViewCount     int64          `json:"view_count" gorm:"default:0"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	User         User           `json:"user" gorm:"foreignKey:UserID"`
	Jobs         []Job          `json:"jobs,omitempty"`
	WatchHistory []WatchHistory `json:"watch_history,omitempty"`
	Tags         []VideoTag     `json:"tags,omitempty" gorm:"foreignKey:VideoID"`
}

type Job struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VideoID    uuid.UUID `json:"video_id" gorm:"type:uuid;not null;index"`
	Type       string    `json:"type"`
	Status     string    `json:"status" gorm:"default:pending;index"`
	RetryCount int       `json:"retry_count" gorm:"default:0"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Video Video `json:"video" gorm:"foreignKey:VideoID"`
}

type WatchHistory struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	VideoID   uuid.UUID `json:"video_id" gorm:"type:uuid;not null;index"`
	WatchedAt time.Time `json:"watched_at" gorm:"default:now()"`
	Progress  int       `json:"progress"`

	User  User  `json:"user" gorm:"foreignKey:UserID"`
	Video Video `json:"video" gorm:"foreignKey:VideoID"`
}

type VideoTag struct {
	ID      uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	VideoID uuid.UUID `json:"video_id" gorm:"type:uuid;not null;index:idx_video_tag,unique"`
	Tag     string    `json:"tag" gorm:"not null;index;index:idx_video_tag,unique"`

	Video Video `json:"video" gorm:"foreignKey:VideoID"`
}
