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
	initName := "init.mp4"
	initPath := filepath.Join(dashDir, initName)

	// Use relative names for init and media segments and run ffmpeg with its
	// working directory set to dashDir. This prevents ffmpeg from prepending
	// the output directory to absolute paths (which caused double-prefixing
	// like C:/dir/C:/dir/init.mp4 on Windows).
	manifestOutPath := filepath.Join(dashDir, manifestName)
	ffInitName := initName
	// Use DASH muxer placeholders so ffmpeg expands segment numbers
	ffMediaPatternName := "chunk-$Number%05d$.m4s"
	ffManifestPath := filepath.ToSlash(manifestOutPath)

	ffmpegArgs := []string{
		"-y",
		"-i", dashInputPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-profile:v", "main",
		"-sc_threshold", "0",
		"-g", "48",
		"-keyint_min", "48",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "dash",
		"-seg_duration", "6",
		"-use_timeline", "1",
		"-use_template", "1",
		"-init_seg_name", ffInitName,
		"-media_seg_name", ffMediaPatternName,
		manifestName,
	}
	log.Printf("running ffmpeg in dir=%s: ffmpeg %s", dashDir, strings.Join(ffmpegArgs, " "))
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
	// ensure ffmpeg writes relative outputs into our temp dir
	cmd.Dir = dashDir
	out, err := cmd.CombinedOutput()
	log.Printf("ffmpeg output (len=%d): %s", len(out), out)
	if err != nil {
		return fmt.Errorf("ffmpeg dash packaging failed: %w\noutput: %s", err, out)
	}
	log.Printf("dash packaging complete: video_id=%s dash_dir=%s", videoID, dashDir)
	log.Printf("dash output paths: video_id=%s init=%s media_pattern=%s manifest=%s", videoID, ffInitName, ffMediaPatternName, ffManifestPath)

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

	initFile, err := os.Open(initPath)
	if err != nil {
		return fmt.Errorf("failed to open dash init segment at %s: %w", initPath, err)
	}
	if _, err := p.s3Helper.UploadFile(ctx, p.cfg.BucketName, filepath.ToSlash(filepath.Join(dashObjectPrefix, initName)), initFile); err != nil {
		initFile.Close()
		return fmt.Errorf("failed to upload dash init segment: %w", err)
	}
	initFile.Close()
	log.Printf("dash init segment uploaded: video_id=%s s3_key=%s", videoID, filepath.ToSlash(filepath.Join(dashObjectPrefix, initName)))

	// DASH muxer will replace the template into concrete filenames like
	// "chunk-<rep>-00001.m4s". Match any chunk-*.m4s to find them.
	segmentFiles, err := filepath.Glob(filepath.Join(dashDir, "chunk-*.m4s"))
	if err != nil {
		return fmt.Errorf("failed to list dash segments: %w", err)
	}
	if len(segmentFiles) == 0 {
		return fmt.Errorf("ffmpeg did not generate any dash segment files in %s (expected pattern %s)", dashDir, filepath.Join(dashDir, "chunk_*.m4s"))
	}
	log.Printf("dash segments generated: video_id=%s count=%d", videoID, len(segmentFiles))

	for _, segmentPath := range segmentFiles {
		segmentFile, err := os.Open(segmentPath)
		if err != nil {
			return fmt.Errorf("failed to open dash segment %s: %w", segmentPath, err)
		}
		objectKey := filepath.ToSlash(filepath.Join(dashObjectPrefix, filepath.Base(segmentPath)))
		_, err = p.s3Helper.UploadFile(ctx, p.cfg.BucketName, objectKey, segmentFile)
		segmentFile.Close()
		if err != nil {
			return fmt.Errorf("failed to upload dash segment %s: %w", segmentPath, err)
		}
		log.Printf("dash segment uploaded: video_id=%s s3_key=%s", videoID, objectKey)
	}

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
