package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type VideoMetadata struct {
	Duration   float64
	Resolution string
}

func ExtractMetadata(ctx context.Context, reader io.Reader) (*VideoMetadata, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		tmpFile, err := os.CreateTemp("", "video-*.mp4")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := io.Copy(tmpFile, reader); err != nil {
			tmpFile.Close()
			return nil, fmt.Errorf("failed to write video to temp file: %w", err)
		}
		tmpFile.Close()

		// Get duration
		durCmd := exec.Command("ffprobe", "-v", "error", "-show_entries",
			"format=duration", "-of",
			"default=noprint_wrappers=1:nokey=1:sk=1", tmpFile.Name())
		durOut, err := durCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("ffprobe duration failed: %w", err)
		}
		duration, _ := strconv.ParseFloat(strings.TrimSpace(string(durOut)), 64)

		// Get resolution
		resCmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
			"-show_entries", "stream=width,height", "-of",
			"csv=s=x:p=0", tmpFile.Name())
		resOut, err := resCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("ffprobe resolution failed: %w", err)
		}

		parts := strings.Split(strings.TrimSpace(string(resOut)), "x")
		width, _ := strconv.Atoi(parts[0])

		return &VideoMetadata{
			Duration:   duration,
			Resolution: getResolution(width),
		}, nil
	}
}

func getResolution(height int) string {
	// the res is the closest value in the range of (widths)
	// [144p, 240p, 360p, 480p, 720p, 1080p] (let's pretend 4k doesn't exist)
	usefulResolutions := []int{144, 240, 360, 480, 720, 1080}
	// loop through this and take the difference between them
	minIdx := 0
	minDiff := abs(usefulResolutions[0] - height)
	for i, res := range usefulResolutions {
		diff := abs(res - height)

		if diff < minDiff {
			minDiff = diff
			minIdx = i
		}
	}
	chosenOne := usefulResolutions[minIdx]
	return fmt.Sprintf("%dp", chosenOne)
}

// ExtractThumbnail extracts a thumbnail from the second frame of the video
// stream. It returns the path to a temp JPEG file that the caller is
// responsible for deleting when done.
func ExtractThumbnail(ctx context.Context, reader io.Reader) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		tmpIn, err := os.CreateTemp("", "video-thumb-in-*.mp4")
		if err != nil {
			return "", fmt.Errorf("failed to create temp input file for thumbnail: %w", err)
		}
		defer os.Remove(tmpIn.Name())

		if _, err := io.Copy(tmpIn, reader); err != nil {
			tmpIn.Close()
			return "", fmt.Errorf("failed to write video to temp file: %w", err)
		}
		tmpIn.Close()

		// Create the output JPEG temp file
		tmpOut, err := os.CreateTemp("", "thumb-*.jpg")
		if err != nil {
			return "", fmt.Errorf("failed to create thumbnail temp file: %w", err)
		}
		tmpOut.Close()

		// Seek to ~second frame (frame 2 at 30fps ≈ 66ms) and extract one JPEG.
		// -q:v 2 → high quality JPEG (scale 1–31, lower is better).
		cmd := exec.CommandContext(ctx,
			"ffmpeg",
			"-ss", "00:00:00.066", // seek to second frame
			"-i", tmpIn.Name(),
			"-frames:v", "1", // extract exactly one frame
			"-q:v", "2", // high quality JPEG
			"-y", // overwrite without prompting
			tmpOut.Name(),
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			os.Remove(tmpOut.Name())
			return "", fmt.Errorf("ffmpeg thumbnail extraction failed: %w\noutput: %s", err, out)
		}

		return tmpOut.Name(), nil
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
