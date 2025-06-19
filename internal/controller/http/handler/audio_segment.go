package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

	// var user_id string
	// claims, exists := ctx.Get("claims")
	// if !exists {
	// 	slog.Error("error", "Unauthorized")
	// 	ctx.JSON(401, entity.ErrorResponse{
	// 		Code:    config.ErrorUnauthorized,
	// 		Message: "Unauthorizedd",
	// 	})
	// 	return
	// } else {
	// 	user_id = claims.(jwt.MapClaims)["id"].(string)
	// }

	// allowed, err := redis.IsRequestAllowed(ctx, h.Redis, user_id, 5, 10, 60)

	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
	// 	slog.Error("Error checking rate limit", slog.String("error", err.Error()))
	// 	return
	// }

	// if !allowed {
	// 	ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Try again later."})
	// 	slog.Warn("Rate limit exceeded for user", slog.String("user_id", user_id))
	// 	return
	// }

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

	// baseURL := "http://192.168.31.50:8080"
	// audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, audio_segment.FilePath)
	// audio_segment.FilePath = audioSegmentURL

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

	// allowed, err := redis.IsRequestAllowed(ctx, h.Redis, user_id, 5, 10, 60)

	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
	// 	slog.Error("Error checking rate limit", slog.String("error", err.Error()))
	// 	return
	// }

	// if !allowed {
	// 	ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Try again later."})
	// 	slog.Warn("Rate limit exceeded for user", slog.String("user_id", user_id))
	// 	return
	// }

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

		// for i := range audio_segment.AudioSegments {
		// 	baseURL := "http://192.168.31.50:8080"
		// 	audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, audio_segment.AudioSegments[i].FilePath)
		// 	audio_segment.AudioSegments[i].FilePath = audioSegmentURL
		// }

		slog.Info("AudioSegment retrieved successfully")
		ctx.JSON(http.StatusOK, audio_segment)
		return
	}

	// for i := range audio_segment.AudioSegments {
	// 	baseURL := "http://192.168.31.50:8080"
	// 	audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, audio_segment.AudioSegments[i].FilePath)
	// 	audio_segment.AudioSegments[i].FilePath = audioSegmentURL
	// }

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
// @Param user_id query string false "User ID"
// @Param report query bool false "Report"
// @Param offset query number false "Offset for pagination"
// @Param limit query number false "Limit for pagination"
// @Success 200 {object} entity.DatasetViewerListResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) DatasetViewer(ctx *gin.Context) {
	var req entity.Filter

	// var userid string
	// claims, exists := ctx.Get("claims")
	// if !exists {
	// 	slog.Error("error", "Unauthorized")
	// 	ctx.JSON(401, entity.ErrorResponse{
	// 		Code:    config.ErrorUnauthorized,
	// 		Message: "Unauthorizedd",
	// 	})
	// 	return
	// } else {
	// 	userid = claims.(jwt.MapClaims)["id"].(string)
	// }

	// allowed, err := redis.IsRequestAllowed(ctx, h.Redis, userid, 5, 10, 60)

	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
	// 	slog.Error("Error checking rate limit", slog.String("error", err.Error()))
	// 	return
	// }

	// if !allowed {
	// 	ctx.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests. Try again later."})
	// 	slog.Warn("Rate limit exceeded for user", slog.String("user_id", userid))
	// 	return
	// }

	// Parse optional pagination parameters
	pageStr := ctx.Query("offset")
	limitStr := ctx.Query("limit")
	user_id := ctx.Query("user_id")
	report := ctx.Query("report")
	reportBool, err := strconv.ParseBool(report)
	if err != nil {
		slog.Error("Error parsing report parameter: ", err)
		ctx.JSON(400, gin.H{"Error": "Invalid report parameter"})
		return
	}

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
	dataset_viewer, err := h.UseCase.AudioSegmentRepo.DatasetViewer(ctx, &req, user_id, reportBool)
	if h.HandleDbError(ctx, err, "Error getting audio_segment") {
		slog.Error("DatasetViewer error", slog.String("error", err.Error()))
		return
	}

	// baseURL := "http://192.168.31.50:8080"
	// for i := range *dataset_viewer {
	// 	audioSegmentURL := fmt.Sprintf("%s/chunks/%s", baseURL, (*dataset_viewer)[i].ChunkUrl)
	// 	(*dataset_viewer)[i].ChunkUrl = audioSegmentURL
	// }
	// for i := range *dataset_viewer {
	// 	audioSegmentURL := fmt.Sprintf("%s/audios/%s", baseURL, (*dataset_viewer)[i].AudioUrl)
	// 	(*dataset_viewer)[i].AudioUrl = audioSegmentURL
	// }

	// Return response
	slog.Info("DatasetViewer retrieved successfully")
	ctx.JSON(http.StatusOK, dataset_viewer)
}

// GetStatistic godoc
// @Router /api/v1/statistic [get]
// @Summary Get statistic
// @Description Get statistic
// @Security BearerAuth
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Success 200 {object} entity.Statistics
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetStatistic(ctx *gin.Context) {

	statistic, err := h.UseCase.AudioSegmentRepo.GetStatistics(ctx)
	if h.HandleDbError(ctx, err, "Error getting statistic") {
		slog.Error("GetStatistic error", slog.String("error", err.Error()))
		return
	}

	// Return response
	slog.Info("Statistic retrieved successfully")
	ctx.JSON(http.StatusOK, statistic)
}

// GetAudioTranscriptStats godoc
// @Router /api/v1/dashboard/stats [get]
// @Summary Get  AudioT ranscript Stats
// @Description Get the Get  AudioT ranscript Stats
// @Security BearerAuth
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Param fromDate query string false "From Date"
// @Param toDate query string false "To Date"
// @Success 200 {object} []entity.TranscriptStatictics
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetAudioTranscriptStats(ctx *gin.Context) {

	fromDate, err := time.Parse("2006-01-02", ctx.Query("fromDate"))
	if err != nil {
		slog.Error("GetAudioTranscriptStats error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid fromDate format, expected YYYY-MM-DD",
		})
		return
	}
	toDate, err := time.Parse("2006-01-02", ctx.Query("toDate"))
	if err != nil {
		slog.Error("GetAudioTranscriptStats error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid toDate format, expected YYYY-MM-DD",
		})
		return
	}

	res, err := h.UseCase.AudioSegmentRepo.GetAudioTranscriptStats(ctx, fromDate, toDate)
	if h.HandleDbError(ctx, err, "Error getting audio transcript stats") {
		slog.Error("GetAudioTranscriptStats error", slog.String("error", err.Error()))
		return
	}

	// Return response
	ctx.JSON(http.StatusOK, res)
}
