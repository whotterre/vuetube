package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"regexp"
	"strings"

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
	GetDashManifest(ctx context.Context, cfg *config.Config, videoID uuid.UUID) ([]byte, error)
	GetDashSegment(ctx context.Context, cfg *config.Config, videoID uuid.UUID, filename string) (io.ReadCloser, error)
	GetGenericFeed(ctx context.Context, limit int) ([]db.Video, error)
	GetThumbnail(ctx context.Context, cfg *config.Config, videoID uuid.UUID) (io.ReadCloser, error)
	GetVideo(ctx context.Context, videoID uuid.UUID) (*db.Video, error)
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
	buf := make([]byte, (1 << 15))
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

// GetDashManifest fetches the MPD from S3, rewrites the relative segment
// references (init.mp4, chunk-*.m4s) to absolute proxy URLs served by this
// backend, and returns the modified XML. The DASH player will then request
// each segment through the /dash/segment/* endpoint, which presigns and
// redirects to S3 — keeping the bucket private at all times.
func (s *videoService) GetDashManifest(ctx context.Context, cfg *config.Config, videoID uuid.UUID) ([]byte, error) {
	video, err := s.videoRepository.FindVideoByVideoID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}
	if video.S3Url == "" {
		return nil, fmt.Errorf("video has no stream URL")
	}
	if !strings.HasSuffix(video.S3Url, ".mpd") {
		return nil, fmt.Errorf("video stream URL is not a DASH manifest: %s", video.S3Url)
	}

	// s3_url stores the full manifest URL, e.g.
	// https://<bucket>.s3.<region>.amazonaws.com/videos/<id>/dash/index.mpd
	manifestKey := utils.ExtractS3Key(video.S3Url, cfg.BucketName, cfg.AWSRegion)

	s3Helper := utils.NewS3Helper(ctx)
	body, err := s3Helper.GetObject(ctx, cfg.BucketName, manifestKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest from S3: %w", err)
	}
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest body: %w", err)
	}
	if err := validateDashManifestBytes(raw); err != nil {
		return nil, err
	}

	// Rewrite every initialization="..." and media="..." attribute to an
	// absolute backend proxy URL, regardless of filename pattern.
	// With -adaptation_sets ffmpeg emits names like init-stream0.m4s and
	// chunk-stream0-$Number%05d$.m4s — the regex handles both old and new forms.
	base := fmt.Sprintf("/videos/%s/dash/segment", videoID)
	reInit := regexp.MustCompile(`initialization="([^"]+)"`)
	reMedia := regexp.MustCompile(`media="([^"]+)"`)
	manifest := string(raw)
	manifest = reInit.ReplaceAllString(manifest, fmt.Sprintf(`initialization="%s/$1"`, base))
	manifest = reMedia.ReplaceAllString(manifest, fmt.Sprintf(`media="%s/$1"`, base))

	go func() {
		_ = s.videoRepository.IncrementViewCount(context.Background(), videoID)
		// Add to watch history 
	}()

	return []byte(manifest), nil
}

func validateDashManifestBytes(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("dash manifest is empty")
	}
	if trimmed[0] != '<' {
		return fmt.Errorf("dash manifest object is not XML; stored s3_url may point to a media segment or original video")
	}

	window := string(trimmed)
	if len(window) > 512 {
		window = window[:512]
	}
	if !strings.Contains(window, "<MPD") {
		return fmt.Errorf("dash manifest XML is missing MPD root")
	}
	return nil
}

// GetDashSegment validates the filename and fetches the segment bytes directly
// from S3, returning a ReadCloser for the handler to proxy to the client.
// The caller must close the returned ReadCloser.
func (s *videoService) GetDashSegment(ctx context.Context, cfg *config.Config, videoID uuid.UUID, filename string) (io.ReadCloser, error) {
	if !isValidSegmentFilename(filename) {
		return nil, fmt.Errorf("invalid segment filename")
	}

	objectKey := fmt.Sprintf("videos/%s/dash/%s", videoID, filename)
	s3Helper := utils.NewS3Helper(ctx)
	body, err := s3Helper.GetObject(ctx, cfg.BucketName, objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch segment: %w", err)
	}
	return body, nil
}

// isValidSegmentFilename accepts init and chunk filenames from both the old
// muxed output (init.mp4, chunk-NNNNN.m4s) and the new adaptation_sets output
// (init-streamN.mp4/init-streamN.m4s, chunk-streamN-NNNNN.m4s).
func isValidSegmentFilename(name string) bool {
	return isValidInitName(name) || isValidChunkName(name)
}

func isValidInitName(name string) bool {
	if name == "init.mp4" {
		return true
	}
	if !strings.HasPrefix(name, "init-stream") {
		return false
	}
	stem, ok := strings.CutSuffix(name, ".mp4")
	if !ok {
		stem, ok = strings.CutSuffix(name, ".m4s")
	}
	if !ok {
		return false
	}
	return isAllDigits(strings.TrimPrefix(stem, "init-stream"))
}

func isValidChunkName(name string) bool {
	if !strings.HasPrefix(name, "chunk-") || !strings.HasSuffix(name, ".m4s") {
		return false
	}
	middle := strings.TrimPrefix(strings.TrimSuffix(name, ".m4s"), "chunk-")
	// chunk-streamN-NNNNN form
	if strings.HasPrefix(middle, "stream") {
		parts := strings.SplitN(middle, "-", 2)
		if len(parts) != 2 {
			return false
		}
		return isAllDigits(strings.TrimPrefix(parts[0], "stream")) && isAllDigits(parts[1])
	}
	// chunk-NNNNN form
	return isAllDigits(middle)
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *videoService) GetGenericFeed(ctx context.Context, limit int) ([]db.Video, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	return s.videoRepository.GetGenericFeed(ctx, limit)
}

// GetThumbnail fetches the thumbnail for videoID directly from S3 and returns
// the raw byte stream. The key format mirrors what the worker writes:
// "thumb-<videoID>". The caller must close the returned ReadCloser.
func (s *videoService) GetThumbnail(ctx context.Context, cfg *config.Config, videoID uuid.UUID) (io.ReadCloser, error) {
	video, err := s.videoRepository.FindVideoByVideoID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}
	if video.ThumbnailUrl == "" {
		return nil, fmt.Errorf("thumbnail not ready")
	}

	thumbKey := fmt.Sprintf("thumb-%s", videoID)
	s3Helper := utils.NewS3Helper(ctx)
	body, err := s3Helper.GetObject(ctx, cfg.BucketName, thumbKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thumbnail: %w", err)
	}
	return body, nil
}

func (s *videoService) GetVideo(ctx context.Context, videoID uuid.UUID) (*db.Video, error) {
	video, err := s.videoRepository.FindVideoByVideoID(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}
	return video, nil
}
