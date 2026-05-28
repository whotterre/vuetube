package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/tasks"
	"github.com/whotterre/vuetube/src/internal/utils"
)

type VideoProcessor struct {
	videoRepository repositories.VideoRepository
	s3Helper        *utils.S3Helper
	cfg             struct {
		BucketName string
		AWSRegion  string
	}
}

type VideoProcessorConfig struct {
	BucketName string
	AWSRegion  string
}

func NewVideoProcessor(repo repositories.VideoRepository, s3Helper *utils.S3Helper, cfg VideoProcessorConfig) *VideoProcessor {
	p := &VideoProcessor{
		videoRepository: repo,
		s3Helper:        s3Helper,
	}
	p.cfg.BucketName = cfg.BucketName
	p.cfg.AWSRegion = cfg.AWSRegion
	return p
}

func (p *VideoProcessor) HandleVideoUploadTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.VideoUploadPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	videoID, err := uuid.Parse(payload.VideoID)
	if err != nil {
		return fmt.Errorf("invalid video ID: %w", err)
	}

	// Open the temp file written by the handler
	f, err := os.Open(payload.TempFilePath)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer f.Close()
	defer os.Remove(payload.TempFilePath)

	// Extract metadata (duration + resolution) via ffprobe
	metadata, err := utils.ExtractMetadata(ctx, f)
	if err != nil {
		return fmt.Errorf("metadata extraction failed: %w", err)
	}

	// Seek back to start for the S3 upload
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Upload video to S3
	objectKey := fmt.Sprintf("%s", videoID)
	_, err = p.s3Helper.UploadFile(ctx, p.cfg.BucketName, objectKey, f)
	if err != nil {
		return fmt.Errorf("S3 video upload failed: %w", err)
	}
	videoS3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		p.cfg.BucketName, p.cfg.AWSRegion, objectKey)

	// Seek back again for thumbnail extraction
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file for thumbnail: %w", err)
	}

	// Generate and upload thumbnail (non-fatal)
	thumbS3URL := ""
	thumbPath, err := utils.ExtractThumbnail(ctx, f)
	if err != nil {
		fmt.Printf("warn: thumbnail extraction failed for video %s: %v\n", videoID, err)
	} else {
		thumbFile, err := os.Open(thumbPath)
		if err != nil {
			fmt.Printf("warn: failed to open thumbnail: %v\n", err)
		} else {
			thumbKey := fmt.Sprintf("thumb-%s", videoID)
			_, err = p.s3Helper.UploadFile(ctx, p.cfg.BucketName, thumbKey, thumbFile)
			thumbFile.Close()
			os.Remove(thumbPath)
			if err != nil {
				fmt.Printf("warn: thumbnail S3 upload failed: %v\n", err)
			} else {
				thumbS3URL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
					p.cfg.BucketName, p.cfg.AWSRegion, thumbKey)
			}
		}
	}

	// Update the DB record with real data
	_, err = p.videoRepository.UpdateVideoAfterProcessing(ctx, db.UpdateVideoAfterProcessingParams{
		ID:           videoID,
		S3Url:        videoS3URL,
		ThumbnailUrl: thumbS3URL,
		Duration:     int32(metadata.Duration),
		Resolution:   metadata.Resolution,
	})
	if err != nil {
		return fmt.Errorf("failed to update video record: %w", err)
	}

	return nil
}
