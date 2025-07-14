package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
)

// GetTranscript godoc
// @Router /api/v1/transcript/{id} [get]
// @Summary Get a transcript by ID
// @Description Get a transcript by ID
// @Security BearerAuth
// @Tags transcript
// @Accept  json
// @Produce  json
// @Param id path int true "Chunk ID"
// @Success 200 {object} entity.Transcript
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetTranscript(ctx *gin.Context) {
	id := ctx.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("GetTranscript error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid transcript ID",
		})
		return
	}

	transcript, err := h.UseCase.TranscriptRepo.GetById(ctx, intId)
	if h.HandleDbError(ctx, err, "Error getting transcript") {
		slog.Error("GetTranscript error", slog.String("error", err.Error()))
		return
	}

	slog.Info("Transcript retrieved successfully")
	ctx.JSON(200, transcript)
}

// GetTranscripts godoc
// @Router /api/v1/transcript/list [get]
// @Summary Get a list of transcripts
// @Description Get a list of transcripts
// @Security BearerAuth
// @Tags transcript
// @Accept  json
// @Produce  json
// @Param offset query number false "Offset for pagination"
// @Param limit query number false "Limit for pagination"
// @Param audio_id query int false "Filter by audio id"
// @Param user_id query int false "Filter by user id"
// @Param status query string false "Filter by status"
// @Success 200 {object} entity.TranscriptList
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetTranscripts(ctx *gin.Context) {
	var req entity.GetTranscriptReq

	// Parse optional pagination parameters
	pageStr := ctx.Query("offset")
	limitStr := ctx.Query("limit")

	// If page & limit are provided, validate them
	limitValue, offsetValue, err := parsePaginationParams(ctx, limitStr, pageStr)
	if err != nil {
		ctx.JSON(400, gin.H{"Error": err.Error()})
		slog.Error("Error parsing pagination parameters: ", "err", err)
		return
	}

	// Assign other filters
	req.AudioId = ctx.Query("audio_id")
	req.UserId = ctx.Query("user_id")
	req.Status = ctx.Query("status")
	req.Filter.Limit = limitValue
	req.Filter.Offset = offsetValue

	// Fetch transcripts
	transcripts, err := h.UseCase.TranscriptRepo.GetList(ctx, &req)
	if h.HandleDbError(ctx, err, "Error getting transcripts") {
		slog.Error("GetTranscripts error", slog.String("error", err.Error()))
		return
	}

	// Return response
	ctx.JSON(http.StatusOK, transcripts)
}

// UpdateTranscript godoc
// @Router /api/v1/transcript/update [put]
// @Summary Update a transcript
// @Description Update a transcript
// @Security BearerAuth
// @Tags transcript
// @Accept  json
// @Produce  json
// @Param id query int true "Chunk ID"
// @Param transcript body entity.UpdateTranscriptBody true "Transcript object"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) UpdateTranscript(ctx *gin.Context) {
	var (
		body entity.UpdateTranscriptBody
	)

	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("UpdateTranscript error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid ustranscripter ID",
		})
	}

	err = ctx.ShouldBindJSON(&body)
	if err != nil {
		slog.Error("UpdateTranscript error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid request body",
		})
		return
	}

	var user_id string
	claims, exists := ctx.Get("claims")
	if !exists {
		slog.Error("error", "", "Unauthorized")
		ctx.JSON(401, entity.ErrorResponse{
			Code:    config.ErrorUnauthorized,
			Message: "Unauthorizedd",
		})
		return
	} else {
		user_id = claims.(jwt.MapClaims)["id"].(string)
	}

	err = h.UseCase.TranscriptRepo.Update(ctx, &entity.UpdateTranscript{
		Id:                 intId,
		TranscriptText:     body.TranscriptText,
		ReportText:         body.ReportText,
		UserID:             &user_id,
		EntireAudioInvalid: body.EntireAudioInvalid,
		Emotion:            body.Emotion,
	})
	if err != nil {
		slog.Error("UpdateTranscript error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Error updating transcript",
		})
		return
	}

	slog.Info("Transcript updated successfully")
	ctx.JSON(200, gin.H{
		"message": "Transcript updated successfully",
	})
}

// // UpdateStatus godoc
// // @Router /api/v1/transcript/update/status [put]
// // @Summary Update a transcript
// // @Description Update a transcript
// // @Security BearerAuth
// // @Tags transcript
// // @Accept  json
// // @Produce  json
// // @Param id query int true "Chunk ID"
// // @Success 200 {object} entity.SuccessResponse
// // @Failure 400 {object} entity.ErrorResponse
// func (h *Handler) UpdateStatus(ctx *gin.Context) {

// 	id := ctx.Query("id")
// 	intId, err := strconv.Atoi(id)
// 	if err != nil {
// 		slog.Error("UpdateStatus error", slog.String("error", err.Error()))
// 		ctx.JSON(400, entity.ErrorResponse{
// 			Code:    config.ErrorBadRequest,
// 			Message: "Invalid ustranscripter ID",
// 		})
// 	}

// 	user_id

// 	err = h.UseCase.TranscriptRepo.UpdateStatus(ctx, &intId)
// 	if err != nil {
// 		slog.Error("UpdateStatus error", slog.String("error", err.Error()))
// 		h.ReturnError(ctx, config.ErrorBadRequest, "Error updating transcript status", 400)
// 		return
// 	}

// 	slog.Info("Transcript status updated successfully")
// 	ctx.JSON(200, gin.H{
// 		"message": "Transcript status updated successfully",
// 	})
// }

// DeleteTranscript godoc
// @Router /api/v1/transcript/delete [delete]
// @Summary Delete a transcript
// @Description Delete a transcript
// @Security BearerAuth
// @Tags transcript
// @Accept  json
// @Produce  json
// @Param id query int true "Chunk ID"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) DeleteTranscript(ctx *gin.Context) {
	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("DeleteTranscript error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid transcript ID",
		})
	}

	err = h.UseCase.TranscriptRepo.Delete(ctx, intId)
	if h.HandleDbError(ctx, err, "Error deleting transcript") {
		slog.Error("DeleteTranscript error", slog.String("error", err.Error()))
		return
	}
	slog.Info("Transcript deleted successfully", slog.String("transcript_id", id))
	ctx.JSON(200, entity.SuccessResponse{
		Message: "Transcript deleted successfully",
	})
}

// StartTranscripts godoc
// @Router /api/v1/transcript/start [put]
// @Summary Start a transcript
// @Description Start a transcript
// @Security BearerAuth
// @Tags transcript
// @Accept  json
// @Produce  json
// @Param id query int true "Chunk ID"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) StartTranscripts(ctx *gin.Context) {

	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("UpdateTranscript error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid ustranscripter ID",
		})
	}

	err = h.UseCase.TranscriptRepo.StartTranscripts(ctx, intId)
	if err != nil {
		slog.Error("StartTranscripts error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Error updating transcript viewed at",
		})
		return
	}

	slog.Info("Transcript started successfully")
	ctx.JSON(200, gin.H{
		"message": "Transcript started successfully",
	})
}
