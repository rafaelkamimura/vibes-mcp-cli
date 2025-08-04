package auth

import (
	"context"
	"fmt"
)

// Manager handles authentication operations including login, token storage, and validation
type Manager struct {
	client  *Client
	storage *Storage
}

// NewManager creates a new authentication manager
func NewManager(baseURL string) (*Manager, error) {
	storage, err := NewStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	client := NewClient(baseURL)

	return &Manager{
		client:  client,
		storage: storage,
	}, nil
}

// Login authenticates with the backend and stores the token
func (m *Manager) Login(ctx context.Context, username, password string) error {
	tokenInfo, err := m.client.Login(ctx, username, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err := m.storage.SaveToken(*tokenInfo); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// GetValidToken returns a valid token, refreshing if necessary
func (m *Manager) GetValidToken(ctx context.Context) (string, error) {
	token, err := m.storage.LoadToken()
	if err != nil {
		return "", fmt.Errorf("failed to load token: %w", err)
	}

	if token == nil {
		return "", fmt.Errorf("no authentication token found, please login first")
	}

	// Validate the token
	if err := m.client.ValidateToken(ctx, token.AccessToken); err != nil {
		// Token is invalid, remove it
		_ = m.storage.DeleteToken()
		return "", fmt.Errorf("authentication token is invalid or expired, please login again: %w", err)
	}

	return token.AccessToken, nil
}

// IsLoggedIn checks if there's a valid token stored
func (m *Manager) IsLoggedIn(ctx context.Context) bool {
	_, err := m.GetValidToken(ctx)
	return err == nil
}

// Logout removes the stored authentication token
func (m *Manager) Logout() error {
	return m.storage.DeleteToken()
}

// GetStoredToken returns the stored token without validation (for inspection)
func (m *Manager) GetStoredToken() (*TokenInfo, error) {
	return m.storage.LoadToken()
}