package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"time"
)

const TypeVideoUpload = "video:upload"

type VideoUploadPayload struct {
	VideoID      string `json:"video_id"`
	TempFilePath string `json:"temp_file_path"`
	FileName     string `json:"file_name"`
}

func NewVideoUploadTask(videoID, tempFilePath, fileName string) (*asynq.Task, error) {
	payload, err := json.Marshal(VideoUploadPayload{
		VideoID:      videoID,
		TempFilePath: tempFilePath,
		FileName:     fileName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task payload: %w", err)
	}
	return asynq.NewTask(
		TypeVideoUpload,
		payload,
		asynq.MaxRetry(3),
		asynq.Timeout(10 * time.Minute),
		asynq.TaskID("video-upload:"+videoID),
	), nil
}
