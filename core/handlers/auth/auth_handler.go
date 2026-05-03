package auth

import (
	"errors"
	"net"
	"net/http"

	authDto "alerthub/core/dto/auth"
	"alerthub/core/middleware"
	authService "alerthub/core/services/auth"
	"alerthub/core/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler interface {
	Register(*gin.Context)
	Login(*gin.Context)
	Refresh(*gin.Context)
	Logout(*gin.Context)
	LogoutAll(*gin.Context)
	Sessions(*gin.Context)
	RevokeSession(*gin.Context)
}

type authHandler struct{ service authService.AuthService }

func NewAuthHandler(service authService.AuthService) AuthHandler {
	return &authHandler{service: service}
}

// Register godoc
// @Summary Register client
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Register request"
// @Success 201 {object} auth.AuthResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 409 {object} common.ErrorResponse
// @Router /auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	var req authDto.RegisterRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		handleAuthError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Client registered successfully", data)
}

// Login godoc
// @Summary Login client
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Login request"
// @Success 200 {object} auth.AuthResponse
// @Failure 400 {object} common.ErrorResponse
// @Failure 401 {object} common.ErrorResponse
// @Router /auth/login [post]
func (h *authHandler) Login(c *gin.Context) {
	var req authDto.LoginRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.Login(c.Request.Context(), req, c.GetHeader("User-Agent"), net.ParseIP(c.ClientIP()))
	if err != nil {
		handleAuthError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Login successful", data)
}

// Refresh godoc
// @Summary Refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshTokenRequest true "Refresh request"
// @Success 200 {object} auth.AuthResponse
// @Failure 401 {object} common.ErrorResponse
// @Router /auth/refresh [post]
func (h *authHandler) Refresh(c *gin.Context) {
	var req authDto.RefreshTokenRequest
	if !bind(c, &req) {
		return
	}
	data, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.GetHeader("User-Agent"), net.ParseIP(c.ClientIP()))
	if err != nil {
		handleAuthError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Token refreshed successfully", data)
}

// Logout godoc
// @Summary Logout current session
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body auth.LogoutRequest true "Logout request"
// @Success 200 {object} common.APIResponse
// @Router /auth/logout [post]
func (h *authHandler) Logout(c *gin.Context) {
	var req authDto.LogoutRequest
	if !bind(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		handleAuthError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Logout successful", nil)
}

// LogoutAll godoc
// @Summary Logout all sessions
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.APIResponse
// @Router /auth/logout-all [post]
func (h *authHandler) LogoutAll(c *gin.Context) {
	clientID := c.MustGet(middleware.ClientIDKey).(uuid.UUID)
	if err := h.service.LogoutAll(c.Request.Context(), clientID); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "All sessions logged out successfully", nil)
}

// Sessions godoc
// @Summary List sessions
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} auth.SessionsResponse
// @Router /auth/sessions [get]
func (h *authHandler) Sessions(c *gin.Context) {
	clientID := c.MustGet(middleware.ClientIDKey).(uuid.UUID)
	sessions, err := h.service.ListSessions(c.Request.Context(), clientID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	response.Success(c, http.StatusOK, "Sessions retrieved successfully", sessions)
}

// RevokeSession godoc
// @Summary Revoke session
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} common.APIResponse
// @Router /auth/sessions/{id} [delete]
func (h *authHandler) RevokeSession(c *gin.Context) {
	clientID := c.MustGet(middleware.ClientIDKey).(uuid.UUID)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid session id", nil)
		return
	}
	if err := h.service.RevokeSession(c.Request.Context(), clientID, sessionID); err != nil {
		handleAuthError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Session revoked successfully", nil)
}

func bind(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", err.Error())
		return false
	}
	return true
}
func handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, authService.ErrEmailAlreadyExists):
		response.Error(c, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "Email already exists", nil)
	case errors.Is(err, authService.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid credentials", nil)
	case errors.Is(err, authService.ErrInvalidRefreshToken):
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token", nil)
	default:
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
