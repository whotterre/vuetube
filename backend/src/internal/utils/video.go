package utils

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type VideoMetadata struct {
	Duration   float64
	Resolution string
}

func ExtractMetadata(ctx context.Context, filePath string) (*VideoMetadata, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Get duration
		durCmd := exec.Command("ffprobe", "-v", "error", "-show_entries",
			"format=duration", "-of",
			"default=noprint_wrappers=1:nokey=1:sk=1", filePath)
		durOut, err := durCmd.Output()
		if err != nil {
			return nil, err
		}
		duration, _ := strconv.ParseFloat(strings.TrimSpace(string(durOut)), 64)

		// Get resolution
		resCmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
			"-show_entries", "stream=width,height", "-of",
			"csv=s=x:p=0", filePath)
		resOut, err := resCmd.Output()
		if err != nil {
			return nil, err
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

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
