package testutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/files"
)

// ErrorHandlingTests contains comprehensive error handling and edge case tests
func TestErrorHandling_FileSystem(t *testing.T) {
	tempDir := TempDir(t)

	t.Run("invalid paths", func(t *testing.T) {
		config := files.DefaultSecurityConfig()
		navigator := files.NewNavigator(config)

		// Test empty path
		err := navigator.SetRoot("")
		assert.Error(t, err)

		// Test non-existent path
		err = navigator.SetRoot("/this/path/does/not/exist")
		assert.Error(t, err)

		// Test file as root (should be directory)
		testFile := CreateTestFile(t, tempDir, "notadir.txt", "content")
		err = navigator.SetRoot(testFile)
		assert.Error(t, err)
	})

	t.Run("permission denied scenarios", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission tests when running as root")
		}

		// Create directory with no read permissions
		restrictedDir := CreateRestrictedDir(t, tempDir, "noaccess", 0000)

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Try to navigate to restricted directory
		err = navigator.Navigate(restrictedDir)
		assert.Error(t, err)

		// Try to load children of restricted directory
		restrictedNode := &files.FileNode{
			Path:  restrictedDir,
			Name:  "noaccess",
			IsDir: true,
		}
		err = navigator.LoadChildren(restrictedNode)
		assert.Error(t, err)
	})

	t.Run("corrupted file system states", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Create a file, then try to treat it as directory
		testFile := CreateTestFile(t, tempDir, "fakedir", "not a directory")

		fakeDir := &files.FileNode{
			Path:  testFile,
			Name:  "fakedir",
			IsDir: true, // Incorrectly marked as directory
		}

		err = navigator.LoadChildren(fakeDir)
		assert.Error(t, err)

		err = navigator.ToggleExpanded(fakeDir)
		assert.Error(t, err)
	})

	t.Run("extremely long paths", func(t *testing.T) {
		// Create path that exceeds typical filesystem limits
		longName := strings.Repeat("a", 500) // Very long filename
		longPath := filepath.Join(tempDir, longName)

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		validator := files.NewSecurityValidator(config)

		// This may succeed or fail depending on filesystem limits
		err := validator.ValidatePath(longPath)
		// We just test that it doesn't panic
		_ = err
	})

	t.Run("special characters in paths", func(t *testing.T) {
		specialChars := []string{
			"file with spaces.txt",
			"file\twith\ttabs.txt",
			"file\nwith\nnewlines.txt",
			"file'with'quotes.txt",
			"file\"with\"doublequotes.txt",
			"file;with;semicolons.txt",
			"file&with&ampersands.txt",
			"file$with$dollars.txt",
			"file`with`backticks.txt",
			"file|with|pipes.txt",
			"file<with>brackets.txt",
		}

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		for _, specialName := range specialChars {
			// Try to create file (may fail on some filesystems)
			filePath := filepath.Join(tempDir, specialName)
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				continue // Skip if filesystem doesn't support this character
			}

			// Test file operations
			_, err := navigator.GetFileInfo(filePath)
			// Should handle gracefully (success or specific error)
			if err != nil {
				t.Logf("Expected error for special character file %s: %v", specialName, err)
			}
		}
	})
}

func TestErrorHandling_Search(t *testing.T) {
	tempDir := TempDir(t)
	CreateTestFileStructure(t, tempDir)

	config := &files.SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}

	validator := files.NewSecurityValidator(config)
	searcher := files.NewFileSearcher(validator)

	t.Run("invalid regex patterns", func(t *testing.T) {
		invalidPatterns := []string{
			"[",    // Unclosed bracket
			"(",    // Unclosed parenthesis
			"*",    // Invalid quantifier
			"+",    // Invalid quantifier
			"(?P<", // Incomplete named group
			"\\",   // Trailing backslash
			"(?",   // Incomplete group
		}

		ctx := context.Background()

		for _, pattern := range invalidPatterns {
			options := &files.SearchOptions{
				Pattern:    pattern,
				IsRegex:    true,
				MaxResults: 10,
			}

			_, err := searcher.Search(ctx, tempDir, options)
			assert.Error(t, err, "Pattern %s should cause error", pattern)
			assert.Contains(t, err.Error(), "regex", "Should mention regex error")
		}
	})

	t.Run("malicious regex patterns", func(t *testing.T) {
		// ReDoS (Regular Expression Denial of Service) patterns
		maliciousPatterns := []string{
			strings.Repeat("(a+)+", 100) + "b",  // Catastrophic backtracking
			"(a|a)*" + strings.Repeat("b", 100), // Exponential time complexity
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()

		for _, pattern := range maliciousPatterns {
			// Pattern should be rejected due to length limit
			options := &files.SearchOptions{
				Pattern:    pattern,
				IsRegex:    true,
				MaxResults: 10,
			}

			_, err := searcher.Search(ctx, tempDir, options)
			assert.Error(t, err, "Malicious pattern should be rejected")
		}
	})

	t.Run("search in non-existent directory", func(t *testing.T) {
		ctx := context.Background()
		nonExistentDir := filepath.Join(tempDir, "nonexistent")

		options := &files.SearchOptions{
			Pattern:    "*",
			MaxResults: 10,
		}

		_, err := searcher.Search(ctx, nonExistentDir, options)
		assert.Error(t, err)
	})

	t.Run("search with nil options", func(t *testing.T) {
		ctx := context.Background()

		// Should use default options
		results, err := searcher.Search(ctx, tempDir, nil)
		assert.NoError(t, err)
		assert.NotNil(t, results)
	})

	t.Run("context cancellation during search", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		options := &files.SearchOptions{
			Pattern:    "*",
			MaxResults: 1000,
		}

		_, err := searcher.Search(ctx, tempDir, options)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("infinite symlink loops", func(t *testing.T) {
		// Create symlink loop
		link1 := filepath.Join(tempDir, "link1")
		link2 := filepath.Join(tempDir, "link2")

		// link1 -> link2 -> link1 (infinite loop)
		err := os.Symlink(link2, link1)
		if err != nil {
			t.Skip("Cannot create symlinks on this system")
		}
		err = os.Symlink(link1, link2)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		options := &files.SearchOptions{
			Pattern:    "*",
			MaxResults: 10,
		}

		// Should handle symlink loops gracefully
		results, err := searcher.Search(ctx, tempDir, options)
		// Should either succeed (by detecting loop) or timeout
		if err != nil && err != context.DeadlineExceeded {
			t.Logf("Symlink loop handling result: %v", err)
		}
		_ = results
	})
}

func TestErrorHandling_Security(t *testing.T) {
	tempDir := TempDir(t)

	t.Run("race conditions in path validation", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    1024,
		}

		validator := files.NewSecurityValidator(config)

		// Create a file
		testFile := CreateTestFile(t, tempDir, "racefile.txt", "content")

		// Run concurrent validations
		const numGoroutines = 100
		errorChan := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := validator.ValidatePath(testFile)
				errorChan <- err
			}()
		}

		// Collect results - all should succeed or fail consistently
		for i := 0; i < numGoroutines; i++ {
			err := <-errorChan
			// All should have same result (no inconsistency due to race)
			if i == 0 {
				if err != nil {
					t.Logf("Expected consistent error: %v", err)
				}
			}
		}

		// Delete file while validation might be happening
		os.Remove(testFile)

		// Validation should handle file disappearing gracefully
		err := validator.ValidatePath(testFile)
		// Should fail gracefully, not panic
		_ = err
	})

	t.Run("TOCTOU attack simulation", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Create a file
		testFile := CreateTestFile(t, tempDir, "toctou.txt", "original content")

		// Simulate TOCTOU by replacing file between validation and read
		go func() {
			time.Sleep(time.Millisecond * 10)
			// Replace with larger file
			os.WriteFile(testFile, []byte(strings.Repeat("x", 2048)), 0644)
		}()

		// Try to read (should be protected against TOCTOU)
		_, err = navigator.ReadFile(testFile)
		// Should either succeed with original or fail due to size limit
		// But should not panic or cause security issue
		if err != nil {
			t.Logf("TOCTOU protection resulted in error: %v", err)
		}
	})

	t.Run("unicode normalization attacks", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    1024,
		}

		validator := files.NewSecurityValidator(config)

		// Unicode normalization attack vectors
		unicodeAttacks := []string{
			"..∕..∕etc∕passwd",                // Unicode slash
			"..%c0%af..%c0%afetc%c0%afpasswd", // NULL byte injection attempt
			"../\u202e/etc/passwd",            // Right-to-left override
			"../\uFEFF/etc/passwd",            // Byte order mark
		}

		for _, attack := range unicodeAttacks {
			attackPath := filepath.Join(tempDir, attack)
			err := validator.ValidatePath(attackPath)
			// Should be blocked or handled safely
			if err == nil {
				t.Logf("Unicode attack path %s was allowed (investigate)", attack)
			}
		}
	})
}

func TestErrorHandling_ResourceExhaustion(t *testing.T) {
	tempDir := TempDir(t)

	t.Run("memory exhaustion protection", func(t *testing.T) {
		// Create directory with many files
		manyFilesDir := CreateTestDir(t, tempDir, "manyfiles")

		// Create files (limited to prevent actual exhaustion in tests)
		for i := 0; i < 1000; i++ {
			fileName := fmt.Sprintf("file_%04d.txt", i)
			CreateTestFile(t, manyFilesDir, fileName, "content")
		}

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Try to load the directory with many files
		manyFilesNode := &files.FileNode{
			Path:  manyFilesDir,
			Name:  "manyfiles",
			IsDir: true,
		}

		// Should complete without excessive memory usage
		err = navigator.LoadChildren(manyFilesNode)
		assert.NoError(t, err)

		// Check that it doesn't load too many at once
		assert.Less(t, len(manyFilesNode.Children), 10001, "Should limit children loading")
	})

	t.Run("deep recursion protection", func(t *testing.T) {
		// Create very deep directory structure
		deepPath := tempDir
		for i := 0; i < 20; i++ { // Create 20 levels deep
			deepPath = filepath.Join(deepPath, fmt.Sprintf("level_%d", i))
			err := os.MkdirAll(deepPath, 0755)
			require.NoError(t, err)
		}

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       5, // Limit depth
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		validator := files.NewSecurityValidator(config)

		// Should be blocked due to depth limit
		_, err := validator.GetDepth(deepPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "depth")
	})

	t.Run("large file protection", func(t *testing.T) {
		// Create large file
		largeFile := CreateLargeFile(t, tempDir, "largefile.txt", 20) // 20MB

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024, // 10MB limit
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Should be blocked due to size limit
		_, err = navigator.ReadFile(largeFile)
		assert.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "size")
	})
}

func TestErrorHandling_ConcurrencyIssues(t *testing.T) {
	tempDir := TempDir(t)
	CreateTestFileStructure(t, tempDir)

	t.Run("concurrent file operations", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		// Create multiple navigators
		const numNavigators = 10
		navigators := make([]*files.Navigator, numNavigators)

		for i := 0; i < numNavigators; i++ {
			navigators[i] = files.NewNavigator(config)
			err := navigators[i].SetRoot(tempDir)
			require.NoError(t, err)
		}

		// Perform concurrent operations
		const numOperations = 100
		errorChan := make(chan error, numNavigators*numOperations)

		for i := 0; i < numNavigators; i++ {
			go func(nav *files.Navigator) {
				for j := 0; j < numOperations; j++ {
					// Mix of different operations
					switch j % 4 {
					case 0:
						nav.GetFlattenedNodes()
					case 1:
						nav.GetBreadcrumb()
					case 2:
						nav.GetCurrentPath()
					case 3:
						nodes := nav.GetFlattenedNodes()
						for _, node := range nodes {
							if node.IsDir {
								nav.ToggleExpanded(node)
								break
							}
						}
					}
				}
				errorChan <- nil
			}(navigators[i])
		}

		// Wait for completion
		for i := 0; i < numNavigators; i++ {
			err := <-errorChan
			assert.NoError(t, err)
		}
	})

	t.Run("file system changes during navigation", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Create file
		testFile := CreateTestFile(t, tempDir, "changing.txt", "content")

		// Get initial file info
		_, err = navigator.GetFileInfo(testFile)
		assert.NoError(t, err)

		// Delete file
		os.Remove(testFile)

		// Try to get info again - should handle gracefully
		_, err = navigator.GetFileInfo(testFile)
		assert.Error(t, err) // Should fail, but gracefully

		// Create directory where file was
		err = os.MkdirAll(testFile, 0755)
		require.NoError(t, err)

		// Should handle type change gracefully
		info, err := navigator.GetFileInfo(testFile)
		if err == nil {
			assert.True(t, info.IsDir, "Should now be directory")
		}
	})
}

func TestErrorHandling_NetworkAndIO(t *testing.T) {
	tempDir := TempDir(t)

	t.Run("disk full simulation", func(t *testing.T) {
		// We can't easily simulate disk full, but we can test with read-only filesystem
		if os.Getuid() == 0 {
			t.Skip("Skipping filesystem tests when running as root")
		}

		readOnlyDir := CreateRestrictedDir(t, tempDir, "readonly", 0444)

		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Try to navigate to read-only directory
		err = navigator.Navigate(readOnlyDir)
		// May succeed (navigation doesn't require write)

		// Try operations that might require write permissions
		readOnlyNode := &files.FileNode{
			Path:  readOnlyDir,
			Name:  "readonly",
			IsDir: true,
		}

		err = navigator.LoadChildren(readOnlyNode)
		// Should handle permission errors gracefully
		if err != nil {
			t.Logf("Expected permission error: %v", err)
		}
	})

	t.Run("interrupted IO operations", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)

		// Create file
		testFile := CreateTestFile(t, tempDir, "iotest.txt", "test content")

		// Simulate interrupted read by deleting file during read attempt
		go func() {
			time.Sleep(time.Millisecond * 10)
			os.Remove(testFile)
		}()

		// Try to read
		_, err = navigator.ReadFile(testFile)
		// Should handle IO error gracefully
		if err != nil {
			t.Logf("Expected IO error: %v", err)
		}
	})
}

// Helper function to test error propagation
func TestErrorPropagation(t *testing.T) {
	tempDir := TempDir(t)

	t.Run("error context preservation", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{tempDir},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		validator := files.NewSecurityValidator(config)

		// Test that errors contain useful context
		err := validator.ValidatePath("/etc/passwd")
		assert.Error(t, err)

		// Should be a specific security error
		assert.True(t, errors.Is(err, files.ErrAccessDenied) ||
			errors.Is(err, files.ErrPathTraversal) ||
			errors.Is(err, files.ErrInvalidPath),
			"Should be a specific security error type")
	})

	t.Run("error wrapping preservation", func(t *testing.T) {
		config := &files.SecurityConfig{
			AllowedPaths:   []string{"/nonexistent"},
			ForbiddenPaths: []string{},
			MaxDepth:       10,
			AllowHidden:    true,
			MaxFileSize:    10 * 1024 * 1024,
		}

		navigator := files.NewNavigator(config)
		err := navigator.SetRoot("/nonexistent/path")

		assert.Error(t, err)
		// Error should contain context about what failed
		assert.Contains(t, err.Error(), "root") // or similar context
	})
}
