package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/testutil"
)

func TestSecurityValidator_ValidatePath(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	forbiddenDir := testutil.CreateTestDir(t, tempDir, "forbidden")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{forbiddenDir},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024, // 1MB
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{
			name:    "valid path within allowed directory",
			path:    filepath.Join(allowedDir, "test.txt"),
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errType: ErrInvalidPath,
		},
		{
			name:    "path traversal with ..",
			path:    filepath.Join(allowedDir, "..", "forbidden", "test.txt"),
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "path traversal with URL encoding",
			path:    filepath.Join(allowedDir, "%2e%2e", "forbidden"),
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "path with tilde",
			path:    "~/secret.txt",
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "forbidden path access",
			path:    filepath.Join(forbiddenDir, "secret.txt"),
			wantErr: true,
			errType: ErrAccessDenied,
		},
		{
			name:    "path outside allowed directories",
			path:    "/etc/passwd",
			wantErr: true,
			errType: ErrAccessDenied,
		},
		{
			name:    "hidden file when not allowed",
			path:    filepath.Join(allowedDir, ".hidden"),
			wantErr: true,
			errType: ErrAccessDenied,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_PathTraversalAttacks(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	// Various path traversal attack vectors
	attacks := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"%2e%2e\\%2e%2e\\%2e%2e\\windows\\system32",
		"....//....//....//etc/passwd",
		"..%252f..%252f..%252fetc%252fpasswd",
		"..%c0%af..%c0%af..%c0%afetc%c0%afpasswd",
		"~/../../../etc/passwd",
		"~/../../etc/passwd",
		"./../../../etc/passwd",
		"./../../etc/passwd",
		"..//..//..//etc//passwd",
		".../.../...//etc/passwd",
	}
	
	for _, attack := range attacks {
		t.Run("attack_"+attack, func(t *testing.T) {
			// Test with attack vector directly
			err := validator.ValidatePath(attack)
			assert.Error(t, err, "Attack vector should be blocked: %s", attack)
			
			// Test with attack vector appended to allowed path
			attackPath := filepath.Join(allowedDir, attack)
			err = validator.ValidatePath(attackPath)
			assert.Error(t, err, "Attack vector in allowed path should be blocked: %s", attackPath)
		})
	}
}

func TestSecurityValidator_ValidateRead(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	
	// Create test files
	smallFile := testutil.CreateTestFile(t, allowedDir, "small.txt", "small content")
	largeFile := testutil.CreateLargeFile(t, allowedDir, "large.txt", 2) // 2MB file
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024, // 1MB limit
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid small file",
			path:    smallFile,
			wantErr: false,
		},
		{
			name:    "file exceeds size limit",
			path:    largeFile,
			wantErr: true,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(allowedDir, "nonexistent.txt"),
			wantErr: true,
		},
		{
			name:    "directory instead of file",
			path:    allowedDir,
			wantErr: false, // ValidateRead allows directories
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRead(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_ValidateDirectory(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	testFile := testutil.CreateTestFile(t, allowedDir, "test.txt", "content")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid directory",
			path:    allowedDir,
			wantErr: false,
		},
		{
			name:    "file instead of directory",
			path:    testFile,
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			path:    filepath.Join(allowedDir, "nonexistent"),
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDirectory(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_GetDepth(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       3,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name        string
		path        string
		wantDepth   int
		wantErr     bool
	}{
		{
			name:      "root level",
			path:      allowedDir,
			wantDepth: 0,
			wantErr:   false,
		},
		{
			name:      "one level deep",
			path:      filepath.Join(allowedDir, "sub"),
			wantDepth: 1,
			wantErr:   false,
		},
		{
			name:      "two levels deep",
			path:      filepath.Join(allowedDir, "sub", "deep"),
			wantDepth: 2,
			wantErr:   false,
		},
		{
			name:      "three levels deep",
			path:      filepath.Join(allowedDir, "sub", "deep", "deeper"),
			wantDepth: 3,
			wantErr:   false,
		},
		{
			name:      "exceeds max depth",
			path:      filepath.Join(allowedDir, "a", "b", "c", "d"),
			wantDepth: 0,
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			depth, err := validator.GetDepth(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantDepth, depth)
			}
		})
	}
}

func TestSecurityValidator_HiddenFiles(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	
	// Create hidden files and directories
	testutil.CreateTestFile(t, allowedDir, ".hidden_file", "hidden content")
	testutil.CreateTestDir(t, allowedDir, ".hidden_dir")
	testutil.CreateTestFile(t, allowedDir, "normal_file", "normal content")
	
	tests := []struct {
		name        string
		allowHidden bool
		path        string
		wantErr     bool
	}{
		{
			name:        "hidden file allowed",
			allowHidden: true,
			path:        filepath.Join(allowedDir, ".hidden_file"),
			wantErr:     false,
		},
		{
			name:        "hidden file not allowed",
			allowHidden: false,
			path:        filepath.Join(allowedDir, ".hidden_file"),
			wantErr:     true,
		},
		{
			name:        "hidden directory allowed",
			allowHidden: true,
			path:        filepath.Join(allowedDir, ".hidden_dir"),
			wantErr:     false,
		},
		{
			name:        "hidden directory not allowed",
			allowHidden: false,
			path:        filepath.Join(allowedDir, ".hidden_dir"),
			wantErr:     true,
		},
		{
			name:        "normal file always allowed",
			allowHidden: false,
			path:        filepath.Join(allowedDir, "normal_file"),
			wantErr:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SecurityConfig{
				AllowedPaths:   []string{allowedDir},
				ForbiddenPaths: []string{},
				MaxDepth:       5,
				AllowHidden:    tt.allowHidden,
				MaxFileSize:    1024 * 1024,
			}
			
			validator := NewSecurityValidator(config)
			err := validator.ValidatePath(tt.path)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrAccessDenied)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_SymlinkSecurity(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	forbiddenDir := testutil.CreateTestDir(t, tempDir, "forbidden")
	
	// Create test files
	allowedFile := testutil.CreateTestFile(t, allowedDir, "allowed.txt", "allowed content")
	forbiddenFile := testutil.CreateTestFile(t, forbiddenDir, "forbidden.txt", "forbidden content")
	
	// Create symlinks
	symlinkToAllowed := filepath.Join(allowedDir, "link_to_allowed")
	symlinkToForbidden := filepath.Join(allowedDir, "link_to_forbidden")
	symlinkToOutside := filepath.Join(allowedDir, "link_to_outside")
	
	testutil.CreateSymlink(t, allowedFile, symlinkToAllowed)
	testutil.CreateSymlink(t, forbiddenFile, symlinkToForbidden)
	testutil.CreateSymlink(t, "/etc/passwd", symlinkToOutside)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{forbiddenDir},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "symlink to allowed file",
			path:    symlinkToAllowed,
			wantErr: false,
		},
		{
			name:    "symlink to forbidden file",
			path:    symlinkToForbidden,
			wantErr: true,
		},
		{
			name:    "symlink to outside allowed paths",
			path:    symlinkToOutside,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecurityValidator_SanitizePath(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name     string
		path     string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "clean path",
			path:     filepath.Join(allowedDir, "test.txt"),
			wantPath: filepath.Join(allowedDir, "test.txt"),
			wantErr:  false,
		},
		{
			name:    "path with traversal",
			path:    filepath.Join(allowedDir, "..", "forbidden"),
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, err := validator.SanitizePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				abs, _ := filepath.Abs(tt.wantPath)
				assert.Equal(t, abs, sanitized)
			}
		})
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()
	
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.AllowedPaths)
	assert.NotEmpty(t, config.ForbiddenPaths)
	assert.Greater(t, config.MaxDepth, 0)
	assert.Greater(t, config.MaxFileSize, int64(0))
	
	// Check that forbidden paths include sensitive directories
	forbiddenMap := make(map[string]bool)
	for _, path := range config.ForbiddenPaths {
		forbiddenMap[path] = true
	}
	
	sensitiveDirectories := []string{"/etc", "/var/log", "/proc", "/sys", "/dev", "/root"}
	for _, sensitive := range sensitiveDirectories {
		assert.True(t, forbiddenMap[sensitive], "Should forbid access to %s", sensitive)
	}
}

func TestSecurityValidator_IsWithinAllowedPath(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir1 := testutil.CreateTestDir(t, tempDir, "allowed1")
	allowedDir2 := testutil.CreateTestDir(t, tempDir, "allowed2")
	forbiddenDir := testutil.CreateTestDir(t, tempDir, "forbidden")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir1, allowedDir2},
		ForbiddenPaths: []string{forbiddenDir},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{
			name:   "path in first allowed directory",
			path:   filepath.Join(allowedDir1, "test.txt"),
			expect: true,
		},
		{
			name:   "path in second allowed directory",
			path:   filepath.Join(allowedDir2, "test.txt"),
			expect: true,
		},
		{
			name:   "path in forbidden directory",
			path:   filepath.Join(forbiddenDir, "test.txt"),
			expect: false,
		},
		{
			name:   "path outside all directories",
			path:   "/tmp/outside.txt",
			expect: false,
		},
		{
			name:   "allowed directory itself",
			path:   allowedDir1,
			expect: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsWithinAllowedPath(tt.path)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestSecurityValidator_ConcurrentAccess(t *testing.T) {
	tempDir := testutil.TempDir(t)
	allowedDir := testutil.CreateTestDir(t, tempDir, "allowed")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       5,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	
	// Test concurrent access to the validator
	const numGoroutines = 100
	errChan := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			testPath := filepath.Join(allowedDir, "test", "file.txt")
			err := validator.ValidatePath(testPath)
			errChan <- err
		}(i)
	}
	
	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

// Benchmark tests for security validation performance
func BenchmarkSecurityValidator_ValidatePath(b *testing.B) {
	tempDir := testutil.TempDir(b)
	allowedDir := testutil.CreateTestDir(b, tempDir, "allowed")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{"/etc", "/var", "/tmp"},
		MaxDepth:       10,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	testPath := filepath.Join(allowedDir, "deep", "nested", "path", "test.txt")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidatePath(testPath)
	}
}

func BenchmarkSecurityValidator_PathTraversalCheck(b *testing.B) {
	tempDir := testutil.TempDir(b)
	allowedDir := testutil.CreateTestDir(b, tempDir, "allowed")
	
	config := &SecurityConfig{
		AllowedPaths:   []string{allowedDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    false,
		MaxFileSize:    1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	maliciousPath := filepath.Join(allowedDir, "../../../etc/passwd")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidatePath(maliciousPath)
	}
}