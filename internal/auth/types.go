package auth

import "time"

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the response from the auth login endpoint
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// TokenInfo holds token data with metadata
type TokenInfo struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	Username    string    `json:"username,omitempty"`
}

// AuthStore represents the structure of the auth storage file
type AuthStore struct {
	Token TokenInfo `json:"token"`
}