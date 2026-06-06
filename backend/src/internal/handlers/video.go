package handlers

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/whotterre/vuetube/src/dto"
	"github.com/whotterre/vuetube/src/internal/config"
	"github.com/whotterre/vuetube/src/internal/middleware"
	"github.com/whotterre/vuetube/src/internal/services"
	"github.com/whotterre/vuetube/src/internal/utils"
)

const VIDEO_UPLOAD_LIMIT = 500 * (1 << 20) // 500MB for now

type VideoHandler interface {
	UploadVideo(ctx *gin.Context)
	ToggleVideoLike(ctx *gin.Context)
	GetRecommendationFeed(ctx *gin.Context)
	ServeDashManifest(ctx *gin.Context)
	ServeDashSegment(ctx *gin.Context)
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

func (h *videoHandler) ToggleVideoLike(ctx *gin.Context) {
	var req dto.LikeVideoDto
	videoId := ctx.Param("id")
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	claimsVal, _ := ctx.Get(middleware.ClaimsKey)
	claims := claimsVal.(*utils.Claims)
	userUUID := claims.UserID

	videoUUID, err := uuid.Parse(videoId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	liked, count, err := h.videoService.ToggleLikeVideo(ctx.Request.Context(), videoUUID, userUUID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":    "Successfully toggled like for video",
		"liked":      liked,
		"like_count": count,
	})
}

func (h *videoHandler) GetRecommendationFeed(ctx *gin.Context) {
	vId, exists := ctx.Params.Get("id")
	if !exists || vId == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing video id"})
		return
	}

	videoId, err := uuid.Parse(vId)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	limit := 20
	if limStr := ctx.Query("limit"); limStr != "" {
		parsed, err := strconv.Atoi(limStr)
		if err != nil || parsed <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	recommendations, err := h.videoService.GetRecommendationFeed(ctx.Request.Context(), limit, videoId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":         "recommendation feed fetched successfully",
		"recommendations": recommendations,
	})
}

// ServeDashManifest fetches the MPD from S3, rewrites segment URLs to point
// at this backend's segment proxy, and returns it as application/dash+xml.
func (h *videoHandler) ServeDashManifest(ctx *gin.Context) {
	videoID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	manifest, err := h.videoService.GetDashManifest(ctx.Request.Context(), h.cfg, videoID)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case strings.Contains(err.Error(), "still processing"):
			ctx.JSON(http.StatusAccepted, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	ctx.Header("Content-Type", "application/dash+xml")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Data(http.StatusOK, "application/dash+xml", manifest)
}

// ServeDashSegment fetches the requested segment (init-streamN.* or chunk-NNNNN.m4s)
// directly from S3 and streams the bytes to the client. Proxying avoids the
// 307→S3 redirect which would forward the Authorization header and break
// S3's pre-signed URL auth.
func (h *videoHandler) ServeDashSegment(ctx *gin.Context) {
	videoID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}

	filename := ctx.Param("filename")
	if strings.Contains(filename, "/") || strings.HasPrefix(filename, ".") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	body, err := h.videoService.GetDashSegment(ctx.Request.Context(), h.cfg, videoID, filename)
	if err != nil {
		if strings.Contains(err.Error(), "invalid segment filename") {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer body.Close()

	contentType := "video/mp4"
	if strings.HasSuffix(filename, ".m4s") {
		contentType = "video/iso.segment"
	}

	ctx.Header("Cache-Control", "public, max-age=3600")
	ctx.Header("Accept-Ranges", "bytes")
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", contentType)
	io.Copy(ctx.Writer, body)
}
