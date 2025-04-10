package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
	"github.com/mirjalilova/voice_transcribe/pkg/hash"
	"github.com/mirjalilova/voice_transcribe/pkg/jwt"
)

// Login godoc
// @Router /api/v1/auth/login [post]
// @Summary Login
// @Description Login
// @Tags auth
// @Accept  json
// @Produce  json
// @Param body body entity.LoginReq true "User"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) Login(ctx *gin.Context) {
	var (
		body entity.LoginReq
	)

	err := ctx.ShouldBindJSON(&body)
	if err != nil {
		h.ReturnError(ctx, config.ErrorBadRequest, "Invalid request body", 400)
		return
	}

	user, err := h.UseCase.AuthRepo.Login(ctx, &body)
	if err != nil {
		slog.Error("Login error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid username or password",
		})
	}

	// generate jwt token
	jwtFields := map[string]interface{}{
		"id":   user.Id,
		"role": user.Role,
	}

	token, err := jwt.GenerateJWT(jwtFields, h.Config.JWT.Secret)
	if err != nil {
		h.ReturnError(ctx, config.ErrorInternalServer, "Oops, something went wrong!!!", http.StatusInternalServerError)
		return
	}

	slog.Info("Login successful", slog.String("username", user.Username))
	ctx.JSON(200, gin.H{
		"message":       "Login successful",
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
	})
}

// CreateUser godoc
// @Router /api/v1/user/create [post]
// @Summary Create a new user
// @Description Create a new user
// @Security BearerAuth
// @Tags user
// @Accept  json
// @Produce  json
// @Param user body entity.CreateUser true "User object"
// @Success 201 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) CreateUser(ctx *gin.Context) {
	var (
		body entity.CreateUser
	)

	err := ctx.ShouldBindJSON(&body)
	if err != nil {
		slog.Error("CreateUser error", slog.String("error", err.Error()))
		h.ReturnError(ctx, config.ErrorBadRequest, "Invalid request body", 400)
		return
	}

	body.Password, err = hash.HashPassword(body.Password)
	if err != nil {
		slog.Error("CreateUser error", slog.String("error", err.Error()))
		h.ReturnError(ctx, config.ErrorBadRequest, "Error hashing password", 400)
		return
	}

	err = h.UseCase.AuthRepo.Create(ctx, &body)
	if h.HandleDbError(ctx, err, "Error creating user") {
		slog.Error("CreateUser error", slog.String("error", err.Error()))
		return
	}

	slog.Info("User created successfully", slog.String("username", body.Username))
	ctx.JSON(201, gin.H{"message": "User created successfully"})
}

// GetUser godoc
// @Router /api/v1/user/{id} [get]
// @Summary Get a user by ID
// @Description Get a user by ID
// @Security BearerAuth
// @Tags user
// @Accept  json
// @Produce  json
// @Param id query int true "User ID"
// @Success 200 {object} entity.User
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetUser(ctx *gin.Context) {

	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("GetUser error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid user ID",
		})
	}

	user, err := h.UseCase.AuthRepo.GetById(ctx, intId)
	if h.HandleDbError(ctx, err, "Error getting user") {
		slog.Error("GetUser error", slog.String("error", err.Error()))
		return
	}

	slog.Info("User retrieved successfully", slog.String("username", user.Username))
	ctx.JSON(200, user)
}

// GetUsers godoc
// @Router /api/v1/user/list [get]
// @Summary Get a list of users
// @Description Get a list of users
// @Security BearerAuth
// @Tags user
// @Accept  json
// @Produce  json
// @Param offset query number false "Offset for pagination"
// @Param limit query number false "Limit for pagination"
// @Param username query string false "Search by username"
// @Param role query string false "Filter by user role"
// @Success 200 {object} entity.UserList
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) GetUsers(ctx *gin.Context) {
	var req entity.GetUserReq

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

	// Assign other filters
	req.Username = ctx.Query("username")
	req.Role = ctx.Query("role")
	req.Filter.Limit = limitValue
	req.Filter.Offset = offsetValue

	// Fetch users
	users, err := h.UseCase.AuthRepo.GetList(ctx, &req)
	if h.HandleDbError(ctx, err, "Error getting users") {
		slog.Error("GetUsers error", slog.String("error", err.Error()))
		return
	}

	// Return response
	ctx.JSON(http.StatusOK, users)
}

// UpdateUser godoc
// @Router /api/v1/user/update [put]
// @Summary Update a user
// @Description Update a user
// @Security BearerAuth
// @Tags user
// @Accept  json
// @Produce  json
// @Param id query int true "User ID"
// @Param user body entity.UpdateUserBody true "User object"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) UpdateUser(ctx *gin.Context) {
	var (
		body entity.UpdateUserBody
	)

	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("GetUser error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid user ID",
		})
	}

	err = ctx.ShouldBindJSON(&body)
	if err != nil {
		slog.Error("UpdateUser error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid request body",
		})
		return
	}

	err = h.UseCase.AuthRepo.Update(ctx, &entity.UpdateUser{
		Id:       intId,
		Username: body.Username,
		// Role:     body.Role,
	})
	if err != nil {
		slog.Error("UpdateUser error", slog.String("error", err.Error()))
		h.ReturnError(ctx, config.ErrorBadRequest, "Error updating user", 400)
		return
	}

	slog.Info("User updated successfully", slog.String("username", body.Username))
	ctx.JSON(200, gin.H{
		"message": "User updated successfully",
	})
}

// DeleteUser godoc
// @Router /api/v1/user/delete [delete]
// @Summary Delete a user
// @Description Delete a user
// @Security BearerAuth
// @Tags user
// @Accept  json
// @Produce  json
// @Param id query int true "User ID"
// @Success 200 {object} entity.SuccessResponse
// @Failure 400 {object} entity.ErrorResponse
func (h *Handler) DeleteUser(ctx *gin.Context) {
	id := ctx.Query("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("GetUser error", slog.String("error", err.Error()))
		ctx.JSON(400, entity.ErrorResponse{
			Code:    config.ErrorBadRequest,
			Message: "Invalid user ID",
		})
	}

	err = h.UseCase.AuthRepo.Delete(ctx, intId)
	if h.HandleDbError(ctx, err, "Error deleting user") {
		slog.Error("DeleteUser error", slog.String("error", err.Error()))
		return
	}
	slog.Info("User deleted successfully", slog.String("user_id", id))
	ctx.JSON(200, entity.SuccessResponse{
		Message: "User deleted successfully",
	})
}

func parsePaginationParams(c *gin.Context, limit, offset string) (int, int, error) {
	limitValue := 10
	offsetValue := 0

	if limit != "" {
		parsedLimit, err := strconv.Atoi(limit)
		if err != nil {
			slog.Error("Invalid limit value", err)
			c.JSON(400, gin.H{"error": "Invalid limit value"})
			return 0, 0, err
		}
		limitValue = parsedLimit
	}

	if offset != "" {
		parsedOffset, err := strconv.Atoi(offset)
		if err != nil {
			slog.Error("Invalid offset value", err)
			c.JSON(400, gin.H{"error": "Invalid offset value"})
			return 0, 0, err
		}
		offsetValue = parsedOffset
	}

	return limitValue, offsetValue, nil
}
