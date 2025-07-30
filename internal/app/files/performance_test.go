package files

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/testutil"
)

// Performance tests for large directory handling
func TestNavigator_LargeDirectoryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tempDir := testutil.TempDir(t)
	
	// Create a large directory structure
	const (
		numDirectories = 100
		filesPerDir    = 500
		maxDepth       = 3
	)
	
	t.Logf("Creating test directory structure with %d dirs, %d files per dir, max depth %d", 
		numDirectories, filesPerDir, maxDepth)
	
	createLargeDirectoryStructure(t, tempDir, numDirectories, filesPerDir, maxDepth)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	
	t.Run("set root performance", func(t *testing.T) {
		start := time.Now()
		err := navigator.SetRoot(tempDir)
		duration := time.Since(start)
		
		require.NoError(t, err)
		t.Logf("SetRoot took %v", duration)
		
		// Should complete within reasonable time
		assert.Less(t, duration, time.Second*5, "SetRoot took too long")
		
		root := navigator.GetRoot()
		assert.NotNil(t, root)
		assert.True(t, len(root.Children) > 0)
	})
	
	t.Run("expand large directories", func(t *testing.T) {
		root := navigator.GetRoot()
		
		// Find a directory with many children
		var largeDir *FileNode
		for _, child := range root.Children {
			if child.IsDir {
				largeDir = child
				break
			}
		}
		require.NotNil(t, largeDir)
		
		start := time.Now()
		err := navigator.ToggleExpanded(largeDir)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		t.Logf("Expanding directory with %d children took %v", len(largeDir.Children), duration)
		
		// Should complete within reasonable time
		assert.Less(t, duration, time.Second*2, "Directory expansion took too long")
		assert.True(t, largeDir.IsExpanded)
		assert.True(t, largeDir.IsLoaded)
	})
	
	t.Run("flatten large tree performance", func(t *testing.T) {
		// Expand several directories to create a large flattened view
		root := navigator.GetRoot()
		expandedCount := 0
		for _, child := range root.Children {
			if child.IsDir && expandedCount < 5 {
				navigator.ToggleExpanded(child)
				expandedCount++
			}
		}
		
		start := time.Now()
		nodes := navigator.GetFlattenedNodes()
		duration := time.Since(start)
		
		t.Logf("Flattening tree with %d nodes took %v", len(nodes), duration)
		
		// Should complete quickly
		assert.Less(t, duration, time.Millisecond*100, "Tree flattening took too long")
		assert.Greater(t, len(nodes), 100, "Should have many nodes in flattened view")
	})
	
	t.Run("memory usage monitoring", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		// Perform memory-intensive operations
		root := navigator.GetRoot()
		for i, child := range root.Children {
			if child.IsDir && i < 10 { // Expand 10 directories
				navigator.ToggleExpanded(child)
			}
		}
		
		// Get flattened view multiple times
		for i := 0; i < 10; i++ {
			navigator.GetFlattenedNodes()
		}
		
		runtime.GC()
		runtime.ReadMemStats(&m2)
		
		memoryUsed := m2.Alloc - m1.Alloc
		t.Logf("Memory used: %d bytes (%.2f MB)", memoryUsed, float64(memoryUsed)/(1024*1024))
		
		// Memory usage should be reasonable (less than 100MB for this test)
		assert.Less(t, memoryUsed, uint64(100*1024*1024), "Memory usage too high")
	})
}

func TestFileSearcher_LargeDirectoryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tempDir := testutil.TempDir(t)
	
	// Create a large directory with many files
	const (
		numFiles = 10000
		numDirs  = 100
	)
	
	t.Logf("Creating search test structure with %d files and %d directories", numFiles, numDirs)
	createLargeSearchStructure(t, tempDir, numFiles, numDirs)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()
	
	t.Run("pattern search performance", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "*.go",
			MaxResults: 1000,
		}
		
		start := time.Now()
		results, err := searcher.Search(ctx, tempDir, options)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		t.Logf("Pattern search found %d results in %v", len(results), duration)
		
		// Should complete within reasonable time
		assert.Less(t, duration, time.Second*5, "Pattern search took too long")
		assert.Greater(t, len(results), 0, "Should find some results")
	})
	
	t.Run("regex search performance", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:    "file_[0-9]{3,4}\\.(go|txt)",
			IsRegex:    true,
			MaxResults: 1000,
		}
		
		start := time.Now()
		results, err := searcher.Search(ctx, tempDir, options)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		t.Logf("Regex search found %d results in %v", len(results), duration)
		
		// Regex should be slower but still reasonable
		assert.Less(t, duration, time.Second*10, "Regex search took too long")
	})
	
	t.Run("content search performance", func(t *testing.T) {
		options := &SearchOptions{
			Pattern:        "*.txt",
			SearchContent:  true,
			ContentPattern: "search content",
			MaxResults:     100, // Limit for content search
		}
		
		start := time.Now()
		results, err := searcher.Search(ctx, tempDir, options)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		t.Logf("Content search found %d results in %v", len(results), duration)
		
		// Content search is slower but should complete
		assert.Less(t, duration, time.Second*30, "Content search took too long")
	})
	
	t.Run("search with cancellation", func(t *testing.T) {
		// Test that search can be cancelled
		ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Millisecond*100)
		defer cancel()
		
		options := &SearchOptions{
			Pattern:        "*",
			SearchContent:  true,
			ContentPattern: "something",
			MaxResults:     10000,
		}
		
		start := time.Now()
		_, err := searcher.Search(ctxWithTimeout, tempDir, options)
		duration := time.Since(start)
		
		// Should be cancelled quickly
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Less(t, duration, time.Millisecond*200, "Cancellation should be quick")
	})
}

func TestNavigator_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	tempDir := testutil.TempDir(t)
	
	// Create extreme directory structure
	const (
		numOperations = 1000
		maxDepth      = 10
	)
	
	createDeepDirectoryStructure(t, tempDir, maxDepth, 50) // 50 files per level
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       15,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	t.Run("rapid navigation stress test", func(t *testing.T) {
		start := time.Now()
		
		// Perform many rapid operations
		for i := 0; i < numOperations; i++ {
			// Navigate randomly
			if i%10 == 0 {
				navigator.NavigateUp()
			}
			
			// Expand/collapse directories
			if i%20 == 0 {
				nodes := navigator.GetFlattenedNodes()
				for _, node := range nodes {
					if node.IsDir && len(nodes) < 100 { // Prevent excessive expansion
						navigator.ToggleExpanded(node)
						break
					}
				}
			}
			
			// Get flattened view
			navigator.GetFlattenedNodes()
			
			// Navigation history operations
			if navigator.CanNavigateBack() && i%30 == 0 {
				navigator.NavigateBack()
			}
			if navigator.CanNavigateForward() && i%35 == 0 {
				navigator.NavigateForward()
			}
		}
		
		duration := time.Since(start)
		t.Logf("Completed %d operations in %v (%.2f ops/sec)", 
			numOperations, duration, float64(numOperations)/duration.Seconds())
		
		// Should maintain good performance
		opsPerSecond := float64(numOperations) / duration.Seconds()
		assert.Greater(t, opsPerSecond, 50.0, "Operations per second too low")
	})
}

func TestFileSearcher_ConcurrentSearches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}
	
	tempDir := testutil.TempDir(t)
	createLargeSearchStructure(t, tempDir, 5000, 50)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	
	t.Run("concurrent pattern searches", func(t *testing.T) {
		const numGoroutines = 10
		const numSearches = 20
		
		patterns := []string{"*.go", "*.txt", "*.json", "*.md", "file_*"}
		
		start := time.Now()
		
		// Run concurrent searches
		doneChan := make(chan bool, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer func() { doneChan <- true }()
				
				for j := 0; j < numSearches; j++ {
					pattern := patterns[j%len(patterns)]
					options := &SearchOptions{
						Pattern:    pattern,
						MaxResults: 100,
					}
					
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					_, err := searcher.Search(ctx, tempDir, options)
					cancel()
					
					if err != nil && err != context.DeadlineExceeded {
						t.Errorf("Search failed: %v", err)
					}
				}
			}(i)
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-doneChan
		}
		
		duration := time.Since(start)
		totalSearches := numGoroutines * numSearches
		t.Logf("Completed %d concurrent searches in %v (%.2f searches/sec)", 
			totalSearches, duration, float64(totalSearches)/duration.Seconds())
		
		// Should handle concurrent searches reasonably well
		searchesPerSecond := float64(totalSearches) / duration.Seconds()
		assert.Greater(t, searchesPerSecond, 5.0, "Concurrent search performance too low")
	})
}

// Benchmark tests for various operations
func BenchmarkNavigator_LargeDirectory(b *testing.B) {
	tempDir := testutil.TempDir(b)
	createLargeDirectoryStructure(b, tempDir, 20, 100, 2) // Smaller for benchmarking
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	if err != nil {
		b.Fatal(err)
	}
	
	b.Run("GetFlattenedNodes", func(b *testing.B) {
		// Expand some directories first
		root := navigator.GetRoot()
		for i, child := range root.Children {
			if child.IsDir && i < 5 {
				navigator.ToggleExpanded(child)
			}
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			navigator.GetFlattenedNodes()
		}
	})
	
	b.Run("ToggleExpanded", func(b *testing.B) {
		root := navigator.GetRoot()
		var dirs []*FileNode
		for _, child := range root.Children {
			if child.IsDir {
				dirs = append(dirs, child)
			}
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dir := dirs[i%len(dirs)]
			navigator.ToggleExpanded(dir)
		}
	})
	
	b.Run("LoadChildren", func(b *testing.B) {
		root := navigator.GetRoot()
		var dirs []*FileNode
		for _, child := range root.Children {
			if child.IsDir {
				dirs = append(dirs, child)
			}
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dir := dirs[i%len(dirs)]
			dir.IsLoaded = false // Reset for testing
			navigator.LoadChildren(dir)
		}
	})
}

func BenchmarkFileSearcher_LargeDirectory(b *testing.B) {
	tempDir := testutil.TempDir(b)
	createLargeSearchStructure(b, tempDir, 1000, 20) // Smaller for benchmarking
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	validator := NewSecurityValidator(config)
	searcher := NewFileSearcher(validator)
	ctx := context.Background()
	
	b.Run("PatternSearch", func(b *testing.B) {
		options := &SearchOptions{
			Pattern:    "*.go",
			MaxResults: 100,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := searcher.Search(ctx, tempDir, options)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("RegexSearch", func(b *testing.B) {
		options := &SearchOptions{
			Pattern:    "file_[0-9]+\\.(go|txt)",
			IsRegex:    true,
			MaxResults: 100,
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := searcher.Search(ctx, tempDir, options)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper functions for creating test structures

func createLargeDirectoryStructure(t testing.TB, baseDir string, numDirs, filesPerDir, maxDepth int) {
	for d := 0; d < numDirs; d++ {
		dirName := fmt.Sprintf("dir_%03d", d)
		dirPath := testutil.CreateTestDir(t, baseDir, dirName)
		
		// Create files in this directory
		for f := 0; f < filesPerDir; f++ {
			fileName := fmt.Sprintf("file_%03d.go", f)
			content := fmt.Sprintf("package main\n\n// File %d in directory %s\nfunc main() {\n\tfmt.Println(\"Hello %d\")\n}", f, dirName, f)
			testutil.CreateTestFile(t, dirPath, fileName, content)
			
			// Also create some other file types
			if f%5 == 0 {
				txtName := fmt.Sprintf("file_%03d.txt", f)
				txtContent := fmt.Sprintf("Text file %d\nSome content here\nLine 3", f)
				testutil.CreateTestFile(t, dirPath, txtName, txtContent)
			}
		}
		
		// Create subdirectories recursively
		if maxDepth > 1 && d%10 == 0 { // Only some directories have subdirs
			createLargeDirectoryStructure(t, dirPath, numDirs/5, filesPerDir/2, maxDepth-1)
		}
	}
}

func createLargeSearchStructure(t testing.TB, baseDir string, numFiles, numDirs int) {
	fileTypes := []string{".go", ".txt", ".json", ".md", ".yaml"}
	
	// Create directories
	for d := 0; d < numDirs; d++ {
		dirName := fmt.Sprintf("search_dir_%03d", d)
		dirPath := testutil.CreateTestDir(t, baseDir, dirName)
		
		// Create files in each directory
		filesInDir := numFiles / numDirs
		for f := 0; f < filesInDir; f++ {
			fileType := fileTypes[f%len(fileTypes)]
			fileName := fmt.Sprintf("file_%04d%s", f, fileType)
			
			var content string
			switch fileType {
			case ".go":
				content = fmt.Sprintf("package main\n\n// File %d\nfunc search() {\n\t// search content here\n}", f)
			case ".txt":
				content = fmt.Sprintf("Text file %d\nThis contains search content\nEnd of file", f)
			case ".json":
				content = fmt.Sprintf(`{"id": %d, "name": "file_%d", "content": "search data"}`, f, f)
			case ".md":
				content = fmt.Sprintf("# File %d\n\nThis is a markdown file with search content.\n", f)
			case ".yaml":
				content = fmt.Sprintf("file_id: %d\nname: file_%d\ncontent: search content\n", f, f)
			}
			
			testutil.CreateTestFile(t, dirPath, fileName, content)
		}
	}
}

func createDeepDirectoryStructure(t testing.TB, baseDir string, depth, filesPerLevel int) {
	if depth <= 0 {
		return
	}
	
	// Create files at this level
	for f := 0; f < filesPerLevel; f++ {
		fileName := fmt.Sprintf("deep_file_%d_%d.txt", depth, f)
		content := fmt.Sprintf("File at depth %d, number %d", depth, f)
		testutil.CreateTestFile(t, baseDir, fileName, content)
	}
	
	// Create subdirectory and recurse
	subDir := testutil.CreateTestDir(t, baseDir, fmt.Sprintf("level_%d", depth))
	createDeepDirectoryStructure(t, subDir, depth-1, filesPerLevel)
}