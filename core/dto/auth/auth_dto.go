package auth

import "time"

type RegisterRequest struct {
	Name     string `json:"name" binding:"required" example:"Demo Client"`
	Email    string `json:"email" binding:"required,email" example:"client@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"client@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"ah_refresh_2P8mYwD9kYf3nQxV7bA1cE4rT6uI0oPz"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"ah_refresh_2P8mYwD9kYf3nQxV7bA1cE4rT6uI0oPz"`
}

type AuthData struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"ah_refresh_2P8mYwD9kYf3nQxV7bA1cE4rT6uI0oPz"`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

type AuthResponse struct {
	Status  bool     `json:"status" example:"true"`
	Message string   `json:"message" example:"Login successful"`
	Data    AuthData `json:"data"`
}

type SessionResponse struct {
	ID           string     `json:"id" example:"9fe9e122-bfb1-4f3b-a2d0-f4acdd4cbd2d"`
	TokenFamily  string     `json:"token_family" example:"0f1f1c17-f9e6-42f3-9341-1555a07ddf4e"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokeReason *string    `json:"revoke_reason,omitempty" example:"logout"`
	UserAgent    *string    `json:"user_agent,omitempty" example:"Mozilla/5.0"`
	IPAddress    *string    `json:"ip_address,omitempty" example:"127.0.0.1"`
}

type SessionsResponse struct {
	Status  bool              `json:"status" example:"true"`
	Message string            `json:"message" example:"Sessions retrieved successfully"`
	Data    []SessionResponse `json:"data"`
}
