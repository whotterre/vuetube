package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/whotterre/vuetube/src/internal/config"
	"github.com/whotterre/vuetube/src/internal/services"
)

const VIDEO_UPLOAD_LIMIT = 500 * (1 << 20) // 500MB for now

type VideoHandler interface {
	UploadVideo(ctx *gin.Context)
}

type videoHandler struct {
	videoService services.VideoService
	cfg          *config.Config
}

func NewVideoHandler(videoService services.VideoService, cfg *config.Config) VideoHandler {
	return &videoHandler{
		videoService: videoService,
		cfg:          cfg,
	}
}

func (h *videoHandler) UploadVideo(ctx *gin.Context) {
	videoFile, err := ctx.FormFile("video_file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "video_file is required"})
		return
	}

	if videoFile.Size > VIDEO_UPLOAD_LIMIT {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds limit"})
		return
	}

	title := ctx.PostForm("title")
	if title == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	fileStream, err := videoFile.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer fileStream.Close()

	result, err := h.videoService.UploadVideo(ctx, h.cfg, fileStream, videoFile.Filename, videoFile.Size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, result)
}
