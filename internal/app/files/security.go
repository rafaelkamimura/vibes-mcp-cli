package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrPathTraversal indicates a potential path traversal attack
	ErrPathTraversal = errors.New("path traversal detected")
	// ErrAccessDenied indicates access to a restricted path
	ErrAccessDenied = errors.New("access denied to path")
	// ErrInvalidPath indicates an invalid file path
	ErrInvalidPath = errors.New("invalid file path")
)

// SecurityConfig holds security settings for file access
type SecurityConfig struct {
	// AllowedPaths contains the list of allowed base paths
	AllowedPaths []string
	// ForbiddenPaths contains paths that are explicitly forbidden
	ForbiddenPaths []string
	// MaxDepth limits the maximum directory depth for traversal
	MaxDepth int
	// AllowHidden determines if hidden files/directories are accessible
	AllowHidden bool
	// MaxFileSize limits the maximum file size in bytes for reading
	MaxFileSize int64
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	cwd, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()
	
	return &SecurityConfig{
		AllowedPaths: []string{cwd, homeDir},
		ForbiddenPaths: []string{
			"/etc",
			"/var/log",
			"/proc",
			"/sys",
			"/dev",
			"/root",
			filepath.Join(homeDir, ".ssh"),
			filepath.Join(homeDir, ".gnupg"),
		},
		MaxDepth:    10,
		AllowHidden: false,
		MaxFileSize: 10 * 1024 * 1024, // 10MB
	}
}

// SecurityValidator provides methods for validating file access
type SecurityValidator struct {
	config *SecurityConfig
}

// NewSecurityValidator creates a new security validator with the given configuration
func NewSecurityValidator(config *SecurityConfig) *SecurityValidator {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	return &SecurityValidator{config: config}
}

// ValidatePath validates if the given path is safe to access
func (sv *SecurityValidator) ValidatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	// Clean and resolve the path to prevent path traversal
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check for path traversal attempts - more robust validation
	cleanedParts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range cleanedParts {
		if part == ".." || part == "~" {
			return ErrPathTraversal
		}
		// Check for URL-encoded traversal attempts
		if decoded := strings.ReplaceAll(part, "%2e%2e", ".."); strings.Contains(decoded, "..") {
			return ErrPathTraversal
		}
		if decoded := strings.ReplaceAll(part, "%7e", "~"); strings.Contains(decoded, "~") {
			return ErrPathTraversal
		}
	}

	// Validate against forbidden paths
	for _, forbidden := range sv.config.ForbiddenPaths {
		forbiddenAbs, err := filepath.Abs(forbidden)
		if err != nil {
			continue
		}
		if strings.HasPrefix(cleanPath, forbiddenAbs) {
			return ErrAccessDenied // Don't leak path information in error
		}
	}

	// Check if path is within allowed paths
	allowed := false
	for _, allowedPath := range sv.config.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		if strings.HasPrefix(cleanPath, allowedAbs) {
			allowed = true
			break
		}
	}

	if !allowed {
		return ErrAccessDenied // Don't leak path information
	}

	// Check hidden file access
	if !sv.config.AllowHidden {
		parts := strings.Split(cleanPath, string(filepath.Separator))
		for _, part := range parts {
			if strings.HasPrefix(part, ".") && part != "." && part != ".." {
				return ErrAccessDenied // Hidden files not allowed
			}
		}
	}

	return nil
}

// ValidateRead validates if a file can be safely read
func (sv *SecurityValidator) ValidateRead(path string) error {
	if err := sv.ValidatePath(path); err != nil {
		return err
	}

	// Check if file exists and get its info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check file size limits
	if !info.IsDir() && info.Size() > sv.config.MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), sv.config.MaxFileSize)
	}

	return nil
}

// ValidateDirectory validates if a directory can be safely accessed
func (sv *SecurityValidator) ValidateDirectory(path string) error {
	if err := sv.ValidatePath(path); err != nil {
		return err
	}

	// Check if it's actually a directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// GetDepth calculates the depth of a path relative to the allowed base paths
func (sv *SecurityValidator) GetDepth(path string) (int, error) {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return 0, err
	}

	minDepth := sv.config.MaxDepth + 1
	for _, allowedPath := range sv.config.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		if strings.HasPrefix(cleanPath, allowedAbs) {
			rel, err := filepath.Rel(allowedAbs, cleanPath)
			if err != nil {
				continue
			}
			depth := len(strings.Split(rel, string(filepath.Separator)))
			if rel == "." {
				depth = 0
			}
			if depth < minDepth {
				minDepth = depth
			}
		}
	}

	if minDepth > sv.config.MaxDepth {
		return 0, fmt.Errorf("path depth exceeds maximum allowed depth")
	}

	return minDepth, nil
}

// IsWithinAllowedPath checks if the path is within any of the allowed base paths
func (sv *SecurityValidator) IsWithinAllowedPath(path string) bool {
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return false
	}

	for _, allowedPath := range sv.config.AllowedPaths {
		allowedAbs, err := filepath.Abs(allowedPath)
		if err != nil {
			continue
		}
		if strings.HasPrefix(cleanPath, allowedAbs) {
			return true
		}
	}
	return false
}

// SanitizePath cleans and validates a path, returning the safe absolute path
func (sv *SecurityValidator) SanitizePath(path string) (string, error) {
	if err := sv.ValidatePath(path); err != nil {
		return "", err
	}

	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("failed to sanitize path: %w", err)
	}

	return cleanPath, nil
}