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
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthClientResponse struct {
	ID        string    `json:"id" example:"8c5c8a5e-6b74-48b4-9e0b-85a0c9ef1234"`
	Email     string    `json:"email" example:"client@example.com"`
	Name      string    `json:"name" example:"Acme Inc"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthData struct {
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	TokenType    string             `json:"token_type" example:"Bearer"`
	ExpiresIn    int64              `json:"expires_in" example:"900"`
	Client       AuthClientResponse `json:"client"`
}

type AuthResponse struct {
	Status  bool     `json:"status" example:"true"`
	Message string   `json:"message" example:"Login successful"`
	Data    AuthData `json:"data"`
}

type SessionResponse struct {
	ID           string     `json:"id"`
	TokenFamily  string     `json:"token_family"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	RevokeReason *string    `json:"revoke_reason,omitempty"`
	UserAgent    *string    `json:"user_agent,omitempty"`
	IPAddress    *string    `json:"ip_address,omitempty"`
}

type SessionsResponse struct {
	Status  bool              `json:"status" example:"true"`
	Message string            `json:"message" example:"Sessions retrieved successfully"`
	Data    []SessionResponse `json:"data"`
}
