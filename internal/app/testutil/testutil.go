package testutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TempDir creates a temporary directory for testing
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "claude-test-*")
	require.NoError(t, err)
	
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to clean up temp dir %s: %v", dir, err)
		}
	})
	
	return dir
}

// CreateTestFile creates a test file with specified content
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("Failed to create parent directory: %v", err)
	}
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	
	return filePath
}

// CreateTestDir creates a test directory
func CreateTestDir(t *testing.T, dir, dirname string) string {
	t.Helper()
	dirPath := filepath.Join(dir, dirname)
	
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory %s: %v", dirPath, err)
	}
	
	return dirPath
}

// CreateTestFileStructure creates a complex file structure for testing
func CreateTestFileStructure(t *testing.T, baseDir string) {
	t.Helper()
	
	// Create directories
	dirs := []string{
		"src/main",
		"src/test",
		"docs",
		"config",
		".git",
		".hidden",
		"large_dir",
	}
	
	for _, dir := range dirs {
		CreateTestDir(t, baseDir, dir)
	}
	
	// Create files
	files := map[string]string{
		"README.md":                    "# Test Project\nThis is a test project.",
		"go.mod":                       "module test\n\ngo 1.21",
		"src/main/main.go":             "package main\n\nfunc main() {\n\tfmt.Println(\"Hello World\")\n}",
		"src/main/util.go":             "package main\n\nfunc util() string {\n\treturn \"utility\"\n}",
		"src/test/main_test.go":        "package main\n\nfunc TestMain(t *testing.T) {\n\t// test\n}",
		"docs/architecture.md":         "# Architecture\nSystem design documentation.",
		"config/config.yaml":           "app:\n  name: test\n  port: 8080",
		".gitignore":                   "*.log\n*.tmp\n.env",
		".hidden/secret.txt":           "secret content",
		"large_file.txt":               strings.Repeat("test data\n", 10000),
		"binary_file.bin":              "\x00\x01\x02\x03\x04",
		"Dockerfile":                   "FROM alpine\nRUN echo hello",
		"Makefile":                     "build:\n\tgo build .",
	}
	
	for path, content := range files {
		CreateTestFile(t, baseDir, path, content)
	}
	
	// Create many files in large_dir for performance testing
	for i := 0; i < 100; i++ {
		CreateTestFile(t, baseDir, fmt.Sprintf("large_dir/file_%03d.txt", i), fmt.Sprintf("Content of file %d", i))
	}
}

// CreateMaliciousFileStructure creates files for security testing
func CreateMaliciousFileStructure(t *testing.T, baseDir string) {
	t.Helper()
	
	// Create various malicious path scenarios
	files := map[string]string{
		"normal.txt":                   "normal content",
		"..%2f..%2fetc%2fpasswd":       "fake passwd",
		"..%5c..%5cwindows%5csystem32": "fake system32",
		"%2e%2e%2fconfig":              "fake config",
		"~/.ssh/id_rsa":                "fake ssh key",
	}
	
	for path, content := range files {
		// Use direct file creation to avoid path validation
		safePath := filepath.Join(baseDir, strings.ReplaceAll(path, "/", "_"))
		if err := os.WriteFile(safePath, []byte(content), 0644); err != nil {
			t.Logf("Failed to create malicious test file %s: %v", safePath, err)
		}
	}
}

// TestLogger creates a test logger
func TestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
}

// SilentLogger creates a silent logger for tests that don't need logging output
func SilentLogger() *zap.Logger {
	return zap.NewNop()
}

// CreateLargeFile creates a large file for testing file size limits
func CreateLargeFile(t *testing.T, dir string, filename string, sizeMB int) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	
	file, err := os.Create(filePath)
	require.NoError(t, err)
	defer file.Close()
	
	// Write data in chunks to avoid memory issues
	chunk := make([]byte, 1024*1024) // 1MB chunk
	for i := 0; i < len(chunk); i++ {
		chunk[i] = byte(i % 256)
	}
	
	for i := 0; i < sizeMB; i++ {
		_, err := file.Write(chunk)
		require.NoError(t, err)
	}
	
	return filePath
}

// WithTimeout wraps a test function with a timeout
func WithTimeout(t *testing.T, timeout time.Duration, fn func(context.Context)) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(ctx)
	}()
	
	select {
	case <-done:
		// Test completed successfully
	case <-ctx.Done():
		t.Fatalf("Test timed out after %v", timeout)
	}
}

// AssertFileExists checks if a file exists
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.NoError(t, err, "File should exist: %s", path)
}

// AssertFileNotExists checks if a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "File should not exist: %s", path)
}

// AssertDirExists checks if a directory exists
func AssertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err, "Directory should exist: %s", path)
	require.True(t, info.IsDir(), "Path should be a directory: %s", path)
}

// CreateSymlink creates a symbolic link for testing
func CreateSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	err := os.Symlink(oldname, newname)
	require.NoError(t, err, "Failed to create symlink from %s to %s", oldname, newname)
}

// MockExecutable creates a mock executable for testing
func MockExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, name)
	
	// Create a simple shell script that exits with the desired code
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock executable: %v", err)
	}
	
	return scriptPath
}

// CaptureOutput captures stdout/stderr for testing
func CaptureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	
	// Save original stdout/stderr
	origStdout := os.Stdout
	origStderr := os.Stderr
	
	// Create pipes
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)
	
	// Replace stdout/stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW
	
	// Channels to capture output
	stdoutCh := make(chan string, 1)
	stderrCh := make(chan string, 1)
	
	// Read from pipes
	go func() {
		defer close(stdoutCh)
		data, _ := io.ReadAll(stdoutR)
		stdoutCh <- string(data)
	}()
	
	go func() {
		defer close(stderrCh)
		data, _ := io.ReadAll(stderrR)
		stderrCh <- string(data)
	}()
	
	// Run the function
	fn()
	
	// Close writers and restore original stdout/stderr
	stdoutW.Close()
	stderrW.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr
	
	// Get captured output
	stdout := <-stdoutCh
	stderr := <-stderrCh
	
	// Close readers
	stdoutR.Close()
	stderrR.Close()
	
	return stdout, stderr
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Condition not met within timeout %v", timeout)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// CreateRestrictedDir creates a directory with restricted permissions
func CreateRestrictedDir(t *testing.T, baseDir, dirname string, perm os.FileMode) string {
	t.Helper()
	dirPath := filepath.Join(baseDir, dirname)
	
	err := os.MkdirAll(dirPath, perm)
	require.NoError(t, err)
	
	// Change permissions after creation to ensure restriction
	err = os.Chmod(dirPath, perm)
	require.NoError(t, err)
	
	t.Cleanup(func() {
		// Restore permissions for cleanup
		os.Chmod(dirPath, 0755)
	})
	
	return dirPath
}

// AssertEventuallyTrue asserts that a condition becomes true within a timeout
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()
	WaitForCondition(t, timeout, condition)
}

// MockProcess represents a mock process for testing
type MockProcess struct {
	PID       int
	ExitCode  int
	Output    []byte
	Error     error
	Started   bool
	Killed    bool
	WaitCalls int
}

// NewMockProcess creates a new mock process
func NewMockProcess(pid int) *MockProcess {
	return &MockProcess{
		PID:      pid,
		ExitCode: 0,
		Output:   []byte("mock output"),
	}
}

// Start simulates starting the process
func (m *MockProcess) Start() error {
	if m.Error != nil {
		return m.Error
	}
	m.Started = true
	return nil
}

// Wait simulates waiting for the process
func (m *MockProcess) Wait() error {
	m.WaitCalls++
	return m.Error
}

// Kill simulates killing the process
func (m *MockProcess) Kill() error {
	m.Killed = true
	return nil
}

// GetOutput returns the mock output
func (m *MockProcess) GetOutput() []byte {
	return m.Output
}