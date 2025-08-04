package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles authentication operations with the backend API
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new authentication client
func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// Login authenticates with the backend and returns a token
func (c *Client) Login(ctx context.Context, username, password string) (*TokenInfo, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	url := fmt.Sprintf("%s/auth/login", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid credentials")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	tokenInfo := &TokenInfo{
		AccessToken: loginResp.AccessToken,
		TokenType:   loginResp.TokenType,
		Username:    username,
		// Note: Backend doesn't provide expiry, so we don't set ExpiresAt
		// Token validation will be done through API calls
	}

	return tokenInfo, nil
}

// ValidateToken checks if the token is still valid by making a test API call
func (c *Client) ValidateToken(ctx context.Context, token string) error {
	// Try to access a protected endpoint to validate the token
	url := fmt.Sprintf("%s/auth/me", c.baseURL) // Assuming there's a user info endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("token is invalid or expired")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed with status %d", resp.StatusCode)
	}

	return nil
}