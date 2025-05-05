package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
)

// GetAudioSegment godoc
// @Router /api/v1/audio_segment/{id} [get]
// @Summary Get a audio_segment by ID
// @Description Get a audio_segment by ID
// @Security BearerAuth
// @Tags audio_segment
// @Accept  json
// @Produce  json
// @Param id query int true "AudioSegment ID"
// @Success 200 {object} entity.AudioSegment
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetAudioSegment(ctx *gin.Context) {

	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("GetAudioSegment error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid audio_segment ID",
		})
	}

	audio_segment, err := h.UseCase.AudioSegmentRepo.GetById(ctx, intId)
	if h.HandleDbError(ctx, err, "Error getting audio_segment") {
		slog.Error("GetAudioSegment error", slog.String("error", err.Error()))
		return
	}

	baseURL := "http://192.168.31.50:8080"
	audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, audio_segment.FilePath)
	audio_segment.FilePath = audioSegmentURL

	slog.Info("AudioSegment retrieved successfully")
	ctx.JSON(200, audio_segment)
}

// GetAudioSegments godoc
// @Router /api/v1/audio_segment [get]
// @Summary Get a list of audio_segment
// @Description Get a list of audio_segment
// @Security BearerAuth
// @Tags audio_segment
// @Accept  json
// @Produce  json
// @Param audio_id query int false "Filter by audio id"
// @Param user_id query string false "user id"
// @Param status query string false "Filter by status"
// @Success 200 {object} entity.AudioSegmentList
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetAudioSegments(ctx *gin.Context) {
	var req entity.GetAudioSegmentReq

	var user_id string
	claims, exists := ctx.Get("claims")
	if !exists {
		slog.Error("error", "Unauthorized")
		ctx.JSON(401, entity.ErrorResponse{
			Code:    config.ErrorUnauthorized,
			Message: "Unauthorizedd",
		})
		return
	} else {
		user_id = claims.(jwt.MapClaims)["id"].(string)
	}

	// Assign other filters
	req.AudioId = ctx.Query("audio_id")
	req.Status = ctx.Query("status")
	req.UIserId = user_id
	req.UserID = ctx.Query("user_id")

	// Fetch audio_segment
	audio_segment, err := h.UseCase.AudioSegmentRepo.GetList(ctx, &req)
	if h.HandleDbError(ctx, err, "Error getting audio_segment") {
		slog.Error("GetAudioSegments error", slog.String("error", err.Error()))
		return
	}

	if len(audio_segment.AudioSegments) == 0 {
		slog.Info("No audio_segment found")
		audio_segment, err := h.UseCase.AudioSegmentRepo.GetList(ctx, &entity.GetAudioSegmentReq{UIserId: user_id})
		if h.HandleDbError(ctx, err, "Error getting audio_segment") {
			slog.Error("GetAudioSegments error", slog.String("error", err.Error()))
			return
		}

		for i := range audio_segment.AudioSegments {
			baseURL := "http://192.168.31.50:8080"
			audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, audio_segment.AudioSegments[i].FilePath)
			audio_segment.AudioSegments[i].FilePath = audioSegmentURL
		}

		slog.Info("AudioSegment retrieved successfully")
		ctx.JSON(http.StatusOK, audio_segment)
		return
	}

	for i := range audio_segment.AudioSegments {
		baseURL := "http://192.168.31.50:8080"
		audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, audio_segment.AudioSegments[i].FilePath)
		audio_segment.AudioSegments[i].FilePath = audioSegmentURL
	}

	// Return response
	slog.Info("AudioSegment retrieved successfully")
	ctx.JSON(http.StatusOK, audio_segment)
}

// DeleteAudioSegment godoc
// @Router /api/v1/audio_segment/delete [delete]
// @Summary Delete a audio_segment
// @Description Delete a audio_segment
// @Security BearerAuth
// @Tags audio_segment
// @Accept  json
// @Produce  json
// @Param id query int true "AudioSegment ID"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) DeleteAudioSegment(ctx *gin.Context) {
	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("DeleteAudioSegment error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid audio_segment ID",
		})
	}

	err = h.UseCase.AudioSegmentRepo.Delete(ctx, intId)
	if h.HandleDbError(ctx, err, "Error deleting audio_segment") {
		slog.Error("DeleteAudioSegment error", slog.String("error", err.Error()))
		return
	}
	slog.Info("AudioSegment deleted successfully", slog.String("audio_segment_id", id))
	ctx.JSON(200, entity.SuccessResponse{
		Message: "AudioSegment deleted successfully",
	})
}

// GetTranscriptPercent godoc
// @Router /api/v1/dashboard [get]
// @Summary Get a list of audio percent
// @Description Get a list of audio percent
// @Security BearerAuth
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Success 200 {object} []entity.TranscriptPersent
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetTranscriptPercent(ctx *gin.Context) {
	res, err := h.UseCase.AudioSegmentRepo.GetTranscriptPercent(ctx)
	if h.HandleDbError(ctx, err, "Error getting calculate_transcription_percentage") {
		slog.Error("GetTranscriptPercent error", slog.String("error", err.Error()))
		return
	}

	// Return response
	ctx.JSON(http.StatusOK, res)
}

// GetUserTranscriptStatictics godoc
// @Router /api/v1/dashboard/user/{user_id} [get]
// @Summary Get the user dashboard
// @Description Get the user dashboard
// @Security BearerAuth
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Param user_id path string true "user id"
// @Success 200 {object} entity.UserTranscriptStatictics
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetUserTranscriptStatictics(ctx *gin.Context) {
	userId := ctx.Param("user_id")
	res, err := h.UseCase.AudioSegmentRepo.GetUserTranscriptStatictics(ctx, userId)
	if h.HandleDbError(ctx, err, "Error getting number of transcripts of users") {
		slog.Error("GetUserTranscriptStatictics error", slog.String("error", err.Error()))
		return
	}

	// Return response
	ctx.JSON(http.StatusOK, res)
}

// DatasetViewer godoc
// @Router /api/v1/dataset_viewer [get]
// @Summary Get a list of dataset_viewer
// @Description Get a list of dataset_viewer
// @Security BearerAuth
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Param offset query number false "Offset for pagination"
// @Param limit query number false "Limit for pagination"
// @Success 200 {object} entity.AudioSegmentList
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) DatasetViewer(ctx *gin.Context) {
	var req entity.Filter

	// Parse optional pagination parameters
	pageStr := ctx.Query("offset")
	limitStr := ctx.Query("limit")

	// If page & limit are provided, validate them
	limitValue, offsetValue, err := parsePaginationParams(ctx, limitStr, pageStr)
	if err != nil {
		ctx.JSON(400, gin.H{"Error": err.Error()})
		slog.Error("Error parsing pagination parameters: ", err)
		return
	}
	req.Limit = limitValue
	req.Offset = offsetValue

	// Fetch audio_segment
	dataset_viewer, err := h.UseCase.AudioSegmentRepo.DatasetViewer(ctx, &req)
	if h.HandleDbError(ctx, err, "Error getting audio_segment") {
		slog.Error("GetAudioSegments error", slog.String("error", err.Error()))
		return
	}

	baseURL := "http://192.168.31.50:8080"
	for i := range *dataset_viewer {
		audioSegmentURL := fmt.Sprintf("%s/chunks/%s", baseURL, (*dataset_viewer)[i].ChunkUrl)
		(*dataset_viewer)[i].ChunkUrl = audioSegmentURL
	}
	for i := range *dataset_viewer {
		audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, (*dataset_viewer)[i].AudioUrl)
		(*dataset_viewer)[i].AudioUrl = audioSegmentURL
	}

	// Return response
	slog.Info("AudioSegment retrieved successfully")
	ctx.JSON(http.StatusOK, dataset_viewer)
}
