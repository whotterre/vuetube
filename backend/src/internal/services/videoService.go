package services

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/whotterre/vuetube/src/internal/config"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
	"github.com/whotterre/vuetube/src/internal/middleware"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/utils"
)

const MAX_FILE_SIZE = 500 * (1 << 10) // 500MB limit for now (if you like upload Snyder's Cut)

type VideoService interface {
	UploadVideo(ctx *gin.Context,
		cfg *config.Config,
		videoFile multipart.File,
		videoFileName string,
		videoSize int64,
	) (*db.Video, error)
}

type videoService struct {
	videoRepository repositories.VideoRepository
	s3Helper        *utils.S3Helper
}

func NewVideoService(videoRepository repositories.VideoRepository, s3Helper *utils.S3Helper) VideoService {
	return &videoService{
		videoRepository: videoRepository,
		s3Helper:        s3Helper,
	}
}

func (s *videoService) UploadVideo(ctx *gin.Context,
	cfg *config.Config,
	videoFile multipart.File,
	videoFileName string,
	videoSize int64,
) (*db.Video, error) {
	metadata, err := utils.ExtractMetadata(ctx, videoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	if seeker, ok := videoFile.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to reset file position: %w", err)
		}
	}

	objectKey := fmt.Sprintf("%x", md5.Sum([]byte(videoFileName)))
	_, err = s.s3Helper.UploadFile(ctx, cfg.BucketName, objectKey, videoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}
	videoS3Url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.BucketName, cfg.AWSRegion, objectKey)

	// Generate a thumbnail from the second frame.
	// Seek back to start first — S3 upload consumed the reader.
	if seeker, ok := videoFile.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to reset file position for thumbnail: %w", err)
		}
	}
	thumbPath, err := utils.ExtractThumbnail(ctx, videoFile)
	if err != nil {
		fmt.Printf("warn: thumbnail extraction failed: %v\n", err)
	}

	thumbS3Url := ""
	if thumbPath != "" {
		thumbFile, err := os.Open(thumbPath)
		if err != nil {
			fmt.Printf("warn: failed to open thumbnail for upload: %v\n", err)
		} else {
			thumbKey := fmt.Sprintf("thumb-%x", md5.Sum([]byte(videoFileName)))
			_, err = s.s3Helper.UploadFile(ctx, cfg.BucketName, thumbKey, thumbFile)
			thumbFile.Close()
			os.Remove(thumbPath)
			if err != nil {
				fmt.Printf("warn: thumbnail S3 upload failed: %v\n", err)
			} else {
				thumbS3Url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
					cfg.BucketName, cfg.AWSRegion, thumbKey)
			}
		}
	}

	claimsVal, _ := ctx.Get(middleware.ClaimsKey)

	claims := claimsVal.(*utils.Claims)
	ownerUUID := pgtype.UUID{Bytes: claims.UserID, Valid: true}

	videoData := db.CreateVideoParams{
		Name:         videoFileName,
		S3Url:        videoS3Url,
		ThumbnailUrl: thumbS3Url,
		Duration:     int32(metadata.Duration),
		Resolution:   metadata.Resolution,
		Size:         int32(videoSize),
		Progress:     pgtype.Int4{Int32: 0, Valid: true},
		ViewCount:    0,
		Owner:        ownerUUID,
	}
	createdVideoDeets, err := s.videoRepository.CreateVideo(ctx, videoData)
	if err != nil {
		return nil, fmt.Errorf("failed to save video record: %w", err)
	}
	return createdVideoDeets, nil
}
