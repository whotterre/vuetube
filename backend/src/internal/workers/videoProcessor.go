package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	log.Printf("video upload task started: video_id=%s temp_file=%s", videoID, payload.TempFilePath)

	f, err := os.Open(payload.TempFilePath)
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer f.Close()
	defer os.Remove(payload.TempFilePath)

	metadata, err := utils.ExtractMetadata(ctx, f)
	if err != nil {
		return fmt.Errorf("metadata extraction failed: %w", err)
	}
	log.Printf("video metadata extracted: video_id=%s duration=%.2f resolution=%s", videoID, metadata.Duration, metadata.Resolution)
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file for thumbnail: %w", err)
	}

	thumbS3URL := ""
	thumbKey := fmt.Sprintf("thumb-%s", videoID)
	thumbExists, err := p.s3Helper.ObjectExists(ctx, p.cfg.BucketName, thumbKey)
	if err != nil {
		return fmt.Errorf("failed to check thumbnail existence: %w", err)
	}
	if thumbExists {
		log.Printf("thumbnail already exists, skipping generation: video_id=%s s3_key=%s", videoID, thumbKey)
		thumbS3URL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
			p.cfg.BucketName, p.cfg.AWSRegion, thumbKey)
	} else {
		thumbPath, err := utils.ExtractThumbnail(ctx, f)
		if err != nil {
			fmt.Printf("warn: thumbnail extraction failed for video %s: %v\n", videoID, err)
		} else {
			log.Printf("thumbnail extracted: video_id=%s path=%s", videoID, thumbPath)
			thumbFile, err := os.Open(thumbPath)
			if err != nil {
				fmt.Printf("warn: failed to open thumbnail: %v\n", err)
			} else {
				_, err = p.s3Helper.UploadFile(ctx, p.cfg.BucketName, thumbKey, thumbFile)
				thumbFile.Close()
				os.Remove(thumbPath)
				if err != nil {
					fmt.Printf("warn: thumbnail S3 upload failed: %v\n", err)
				} else {
					log.Printf("thumbnail uploaded: video_id=%s s3_key=%s", videoID, thumbKey)
					thumbS3URL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
						p.cfg.BucketName, p.cfg.AWSRegion, thumbKey)
				}
			}
		}
	}

	dashDir, err := os.MkdirTemp("", "vuetube-dash-*")
	if err != nil {
		return fmt.Errorf("failed to create dash temp dir: %w", err)
	}
	defer os.RemoveAll(dashDir)

	dashInputPath := payload.TempFilePath
	manifestName := "index.mpd"
	manifestOutPath := filepath.Join(dashDir, manifestName)
	ffManifestPath := filepath.ToSlash(manifestOutPath)

	hasAudio, err := hasAudioStream(ctx, dashInputPath)
	if err != nil {
		return fmt.Errorf("failed to inspect audio streams: %w", err)
	}

	// -adaptation_sets separates streams into distinct segment files. Only add
	// the audio adaptation set when the upload actually contains audio; ffmpeg
	// fails before the DB update if asked to package a missing audio stream.
	ffmpegArgs := []string{
		"-y",
		"-i", dashInputPath,
		"-map", "0:v:0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-profile:v", "main",
		"-sc_threshold", "0",
		"-g", "48",
		"-keyint_min", "48",
	}
	if hasAudio {
		ffmpegArgs = append(ffmpegArgs,
			"-map", "0:a:0",
			"-c:a", "aac",
			"-profile:a", "aac_low",
			"-b:a", "128k",
		)
	}
	ffmpegArgs = append(ffmpegArgs,
		"-f", "dash",
		"-seg_duration", "6",
		"-use_timeline", "1",
		"-use_template", "1",
		"-init_seg_name", "init-stream$RepresentationID$.mp4",
		"-media_seg_name", "chunk-stream$RepresentationID$-$Number%05d$.m4s",
		"-adaptation_sets", dashAdaptationSets(hasAudio),
		manifestName,
	)
	log.Printf("running ffmpeg in dir=%s: ffmpeg %s", dashDir, strings.Join(ffmpegArgs, " "))
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	cmd.Dir = dashDir
	out, err := cmd.CombinedOutput()
	log.Printf("ffmpeg output (len=%d): %s", len(out), out)
	if err != nil {
		return fmt.Errorf("ffmpeg dash packaging failed: %w\noutput: %s", err, out)
	}
	log.Printf("dash packaging complete: video_id=%s dash_dir=%s manifest=%s", videoID, dashDir, ffManifestPath)

	entries, err := os.ReadDir(dashDir)
	if err != nil {
		return fmt.Errorf("failed to read dash temp dir: %w", err)
	}
	for _, entry := range entries {
		log.Printf("dash temp file: video_id=%s name=%s is_dir=%t", videoID, entry.Name(), entry.IsDir())
	}

	dashObjectPrefix := fmt.Sprintf("videos/%s/dash", videoID)
	manifestPath := filepath.Join(dashDir, manifestName)
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to open dash manifest: %w", err)
	}
	if _, err := p.s3Helper.UploadFile(ctx, p.cfg.BucketName, filepath.ToSlash(filepath.Join(dashObjectPrefix, manifestName)), manifestFile); err != nil {
		manifestFile.Close()
		return fmt.Errorf("failed to upload dash manifest: %w", err)
	}
	manifestFile.Close()
	log.Printf("dash manifest uploaded: video_id=%s s3_key=%s", videoID, filepath.ToSlash(filepath.Join(dashObjectPrefix, manifestName)))

	assetFiles, err := filepath.Glob(filepath.Join(dashDir, "*"))
	if err != nil {
		return fmt.Errorf("failed to list dash assets: %w", err)
	}
	initCount := 0
	segmentCount := 0
	for _, assetPath := range assetFiles {
		name := filepath.Base(assetPath)
		if name == manifestName {
			continue
		}
		switch {
		case strings.HasPrefix(name, "init-"):
			initCount++
		case strings.HasPrefix(name, "chunk-"):
			segmentCount++
		default:
			log.Printf("skipping unexpected dash asset: video_id=%s name=%s", videoID, name)
			continue
		}

		assetFile, err := os.Open(assetPath)
		if err != nil {
			return fmt.Errorf("failed to open dash asset %s: %w", assetPath, err)
		}
		objectKey := filepath.ToSlash(filepath.Join(dashObjectPrefix, name))
		_, err = p.s3Helper.UploadFile(ctx, p.cfg.BucketName, objectKey, assetFile)
		assetFile.Close()
		if err != nil {
			return fmt.Errorf("failed to upload dash asset %s: %w", assetPath, err)
		}
		log.Printf("dash asset uploaded: video_id=%s s3_key=%s", videoID, objectKey)
	}
	if initCount == 0 {
		return fmt.Errorf("ffmpeg did not generate any dash init files in %s", dashDir)
	}
	if segmentCount == 0 {
		return fmt.Errorf("ffmpeg did not generate any dash media segments in %s", dashDir)
	}
	log.Printf("dash assets uploaded: video_id=%s init_count=%d segment_count=%d", videoID, initCount, segmentCount)

	videoS3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s/%s",
		p.cfg.BucketName, p.cfg.AWSRegion, dashObjectPrefix, manifestName)

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

	log.Printf("video processing completed: video_id=%s dash_manifest=%s", videoID, videoS3URL)

	return nil
}

func hasAudioStream(ctx context.Context, inputPath string) (bool, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=index",
		"-of", "csv=p=0",
		inputPath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ffprobe audio stream check failed: %w\noutput: %s", err, out)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func dashAdaptationSets(hasAudio bool) string {
	if hasAudio {
		return "id=0,streams=v id=1,streams=a"
	}
	return "id=0,streams=v"
}
