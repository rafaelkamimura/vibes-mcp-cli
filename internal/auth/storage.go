package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	authDirName  = ".vibes-mcp-cli"
	authFileName = "auth.json"
)

// Storage handles secure token storage and retrieval
type Storage struct {
	authDir  string
	authFile string
}

// NewStorage creates a new storage instance
func NewStorage() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	authDir := filepath.Join(homeDir, authDirName)
	authFile := filepath.Join(authDir, authFileName)

	// Ensure auth directory exists with proper permissions
	if err := os.MkdirAll(authDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create auth directory: %w", err)
	}

	return &Storage{
		authDir:  authDir,
		authFile: authFile,
	}, nil
}

// SaveToken securely stores the authentication token
func (s *Storage) SaveToken(token TokenInfo) error {
	authStore := AuthStore{
		Token: token,
	}

	data, err := json.MarshalIndent(authStore, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(s.authFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken loads the stored authentication token
func (s *Storage) LoadToken() (*TokenInfo, error) {
	data, err := os.ReadFile(s.authFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No token stored
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var authStore AuthStore
	if err := json.Unmarshal(data, &authStore); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return &authStore.Token, nil
}

// DeleteToken removes the stored authentication token
func (s *Storage) DeleteToken() error {
	if err := os.Remove(s.authFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}
	return nil
}

// HasToken checks if a token file exists
func (s *Storage) HasToken() bool {
	_, err := os.Stat(s.authFile)
	return err == nil
}