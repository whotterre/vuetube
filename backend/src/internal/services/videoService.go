package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/whotterre/vuetube/src/internal/config"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
	"github.com/whotterre/vuetube/src/internal/middleware"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/tasks"
	"github.com/whotterre/vuetube/src/internal/utils"
)

const MAX_FILE_SIZE = 500 * (1 << 20) // 500MB for now (if you like upload Snyder's Cut)

type VideoService interface {
	UploadVideo(ctx *gin.Context,
		cfg *config.Config,
		videoFile multipart.File,
		videoFileName string,
		videoSize int64,
	) (*db.Video, error)
	ToggleLikeVideo(ctx context.Context, videoID, userID uuid.UUID) (bool, int64, error)
	GetRecommendationFeed(ctx context.Context, limit int, videoId uuid.UUID) ([]db.GetCrossPoolRecommendationsRow, error)
	GetGenericFeed(ctx context.Context, limit int) ([]db.Video, error)
}

type videoService struct {
	videoRepository repositories.VideoRepository
	asynqClient     *asynq.Client
}

func NewVideoService(videoRepository repositories.VideoRepository, asynqClient *asynq.Client) VideoService {
	return &videoService{
		videoRepository: videoRepository,
		asynqClient:     asynqClient,
	}
}

func (s *videoService) UploadVideo(ctx *gin.Context,
	cfg *config.Config,
	videoFile multipart.File,
	videoFileName string,
	videoSize int64,
) (*db.Video, error) {
	claimsVal, _ := ctx.Get(middleware.ClaimsKey)
	claims := claimsVal.(*utils.Claims)
	ownerUUID := pgtype.UUID{Bytes: claims.UserID, Valid: true}

	tmpFile, err := os.CreateTemp("", "vuetube-upload-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := copyFile(videoFile, tmpFile); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to save upload: %w", err)
	}
	tmpFile.Close()

	video, err := s.videoRepository.CreateVideo(ctx, db.CreateVideoParams{
		Name:  videoFileName,
		Size:  int32(videoSize),
		Owner: ownerUUID,
	})
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to create video record: %w", err)
	}

	task, err := tasks.NewVideoUploadTask(video.ID.String(), tmpFile.Name(), videoFileName)
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	if _, err := s.asynqClient.EnqueueContext(ctx, task); err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to enqueue upload task: %w", err)
	}

	return video, nil
}

func copyFile(src multipart.File, dst *os.File) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
		}
		if er != nil {
			if er.Error() == "EOF" {
				break
			}
			return written, er
		}
	}
	return written, nil
}

func (s *videoService) ToggleLikeVideo(ctx context.Context, videoID, userID uuid.UUID) (bool, int64, error) {
	if _, err := s.videoRepository.FindVideoByVideoID(ctx, videoID); err != nil {
		return false, 0, fmt.Errorf("video not found: %w", err)
	}

	return s.videoRepository.ToggleLikeTx(ctx, userID, videoID)
}

func (s *videoService) GetRecommendationFeed(ctx context.Context, limit int, videoId uuid.UUID) ([]db.GetCrossPoolRecommendationsRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	if _, err := s.videoRepository.FindVideoByVideoID(ctx, videoId); err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	feed, err := s.videoRepository.GetRecommendationFeed(ctx, limit, videoId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recommendation feed: %w", err)
	}

	return feed, nil
}

func (s *videoService) GetGenericFeed(ctx context.Context, limit int) ([]db.Video, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	return s.videoRepository.GetGenericFeed(ctx, limit)
}