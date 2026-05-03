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
// @Summary Register a new client and issue tokens
// @Description Creates a client account with name, unique email, and password. The response intentionally returns only access_token, refresh_token, token_type, and expires_in; it does not return client profile data. Use the access token as `Authorization: Bearer <access_token>` for protected APIs.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RegisterRequest true "Client registration payload. Password must be at least 8 characters."
// @Success 201 {object} auth.AuthResponse "Client registered successfully; refresh_token is shown only in this response and is stored server-side only as a hash in client_tokens."
// @Failure 400 {object} common.ErrorResponse "Validation error, such as invalid email or short password."
// @Failure 409 {object} common.ErrorResponse "Email already exists."
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
// @Summary Login an existing client and issue tokens
// @Description Authenticates by email and password. The response returns a short-lived JWT access token plus an opaque refresh token. Invalid email and invalid password both return the same unauthorized response so credential existence is not leaked.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "Client login payload. Development demo credentials: client@example.com / password123."
// @Success 200 {object} auth.AuthResponse "Login successful; use data.access_token in the BearerAuth authorize button."
// @Failure 400 {object} common.ErrorResponse "Validation error."
// @Failure 401 {object} common.ErrorResponse "Invalid credentials."
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
// @Summary Rotate a refresh token and issue a new access token
// @Description Exchanges a valid refresh_token for a new access_token and refresh_token. The submitted refresh token is marked used/replaced; reusing an old replaced refresh token revokes its token family and returns 401.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body auth.RefreshTokenRequest true "Refresh payload containing the raw refresh_token from register/login/previous refresh."
// @Success 200 {object} auth.AuthResponse "Token refreshed successfully; replace the old refresh_token with the new one."
// @Failure 400 {object} common.ErrorResponse "Validation error, such as missing refresh_token."
// @Failure 401 {object} common.ErrorResponse "Invalid, expired, revoked, or reused refresh token."
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
// @Summary Logout the session for one refresh token
// @Description Revokes the client-token session identified by the submitted refresh_token. This prevents that refresh token from being used again. Requires BearerAuth so Swagger users should authorize with the current access_token first.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body auth.LogoutRequest true "Logout payload containing the refresh_token for the session to revoke."
// @Success 200 {object} common.APIResponse "Logout successful."
// @Failure 400 {object} common.ErrorResponse "Validation error, such as missing refresh_token."
// @Failure 401 {object} common.ErrorResponse "Missing/invalid access token or invalid refresh token."
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
// @Summary Logout all sessions for the current client
// @Description Revokes every active client-token session owned by the authenticated client. Use this when the client wants to sign out from all devices/browsers.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.APIResponse "All sessions logged out successfully."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
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
// @Summary List current client's token sessions
// @Description Returns active and historical client-token sessions for the authenticated client. Raw refresh tokens and token hashes are never returned; only safe session metadata such as id, token_family, timestamps, user_agent, and ip_address are exposed.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} auth.SessionsResponse "Sessions retrieved successfully. Use a session id from this response with DELETE /auth/sessions/{id}."
// @Failure 401 {object} common.ErrorResponse "Missing or invalid access token."
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
// @Summary Revoke one client-token session
// @Description Revokes a single session owned by the authenticated client. The id must come from GET /auth/sessions. Revocation is scoped by client ownership, so a client cannot revoke another client's session.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID from GET /auth/sessions" example(9fe9e122-bfb1-4f3b-a2d0-f4acdd4cbd2d)
// @Success 200 {object} common.APIResponse "Session revoked successfully."
// @Failure 400 {object} common.ErrorResponse "Invalid session id."
// @Failure 401 {object} common.ErrorResponse "Missing/invalid access token or session does not belong to the client."
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
