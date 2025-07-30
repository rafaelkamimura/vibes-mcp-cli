package files

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/testutil"
)

func TestNewNavigator(t *testing.T) {
	config := DefaultSecurityConfig()
	navigator := NewNavigator(config)
	
	assert.NotNil(t, navigator)
	assert.NotNil(t, navigator.validator)
	assert.NotNil(t, navigator.detector)
	assert.NotNil(t, navigator.searcher)
	assert.NotNil(t, navigator.history)
}

func TestNavigator_SetRoot(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	
	t.Run("valid root directory", func(t *testing.T) {
		err := navigator.SetRoot(tempDir)
		require.NoError(t, err)
		
		root := navigator.GetRoot()
		assert.NotNil(t, root)
		assert.True(t, root.IsDir)
		assert.Equal(t, tempDir, root.Path)
		assert.Equal(t, filepath.Base(tempDir), root.Name)
		assert.True(t, root.IsExpanded)
		assert.True(t, root.IsLoaded)
		assert.Greater(t, len(root.Children), 0)
	})
	
	t.Run("invalid root path", func(t *testing.T) {
		err := navigator.SetRoot("/nonexistent/path")
		assert.Error(t, err)
	})
	
	t.Run("file instead of directory", func(t *testing.T) {
		testFile := testutil.CreateTestFile(t, tempDir, "test.txt", "content")
		err := navigator.SetRoot(testFile)
		assert.Error(t, err)
	})
	
	t.Run("restricted root path", func(t *testing.T) {
		navigator := NewNavigator(DefaultSecurityConfig())
		err := navigator.SetRoot("/etc")
		assert.Error(t, err)
	})
}

func TestNavigator_Navigate(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	srcDir := filepath.Join(tempDir, "src")
	
	t.Run("navigate to valid directory", func(t *testing.T) {
		err := navigator.Navigate(srcDir)
		assert.NoError(t, err)
		assert.Equal(t, srcDir, navigator.GetCurrentPath())
	})
	
	t.Run("navigate to invalid path", func(t *testing.T) {
		err := navigator.Navigate("/invalid/path")
		assert.Error(t, err)
	})
	
	t.Run("navigate to file instead of directory", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "README.md")
		err := navigator.Navigate(testFile)
		assert.Error(t, err)
	})
}

func TestNavigator_NavigationHistory(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	srcDir := filepath.Join(tempDir, "src")
	docsDir := filepath.Join(tempDir, "docs")
	
	// Navigate to multiple directories
	err = navigator.Navigate(srcDir)
	require.NoError(t, err)
	
	err = navigator.Navigate(docsDir)
	require.NoError(t, err)
	
	t.Run("can navigate back", func(t *testing.T) {
		assert.True(t, navigator.CanNavigateBack())
		
		err := navigator.NavigateBack()
		assert.NoError(t, err)
		assert.Equal(t, srcDir, navigator.GetCurrentPath())
		
		err = navigator.NavigateBack()
		assert.NoError(t, err)
		assert.Equal(t, tempDir, navigator.GetCurrentPath())
	})
	
	t.Run("can navigate forward", func(t *testing.T) {
		assert.True(t, navigator.CanNavigateForward())
		
		err := navigator.NavigateForward()
		assert.NoError(t, err)
		assert.Equal(t, srcDir, navigator.GetCurrentPath())
	})
	
	t.Run("cannot navigate back at beginning", func(t *testing.T) {
		// Go back to the beginning
		for navigator.CanNavigateBack() {
			navigator.NavigateBack()
		}
		
		assert.False(t, navigator.CanNavigateBack())
		err := navigator.NavigateBack()
		assert.Error(t, err)
	})
	
	t.Run("cannot navigate forward at end", func(t *testing.T) {
		// Go forward to the end
		for navigator.CanNavigateForward() {
			navigator.NavigateForward()
		}
		
		assert.False(t, navigator.CanNavigateForward())
		err := navigator.NavigateForward()
		assert.Error(t, err)
	})
}

func TestNavigator_NavigateUp(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	srcDir := filepath.Join(tempDir, "src")
	mainDir := filepath.Join(srcDir, "main")
	
	err = navigator.Navigate(mainDir)
	require.NoError(t, err)
	
	t.Run("navigate up from subdirectory", func(t *testing.T) {
		err := navigator.NavigateUp()
		assert.NoError(t, err)
		assert.Equal(t, srcDir, navigator.GetCurrentPath())
		
		err = navigator.NavigateUp()
		assert.NoError(t, err)
		assert.Equal(t, tempDir, navigator.GetCurrentPath())
	})
	
	t.Run("cannot navigate up from root", func(t *testing.T) {
		err := navigator.NavigateUp()
		assert.Error(t, err)
	})
}

func TestNavigator_LoadChildren(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	root := navigator.GetRoot()
	
	t.Run("children loaded correctly", func(t *testing.T) {
		assert.True(t, root.IsLoaded)
		assert.Greater(t, len(root.Children), 0)
		
		// Check that directories are sorted before files
		var foundFirstFile bool
		for _, child := range root.Children {
			if !child.IsDir {
				foundFirstFile = true
			} else if foundFirstFile {
				t.Error("Directories should come before files in sorted order")
			}
		}
	})
	
	t.Run("load children of non-directory fails", func(t *testing.T) {
		// Find a file child
		var fileChild *FileNode
		for _, child := range root.Children {
			if !child.IsDir {
				fileChild = child
				break
			}
		}
		require.NotNil(t, fileChild)
		
		err := navigator.LoadChildren(fileChild)
		assert.Error(t, err)
	})
	
	t.Run("child nodes have correct metadata", func(t *testing.T) {
		for _, child := range root.Children {
			assert.NotEmpty(t, child.Name)
			assert.NotEmpty(t, child.Path)
			assert.Equal(t, root, child.Parent)
			assert.Equal(t, root.Depth+1, child.Depth)
			assert.NotEqual(t, FileTypeUnknown, child.FileType)
		}
	})
}

func TestNavigator_ToggleExpanded(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	root := navigator.GetRoot()
	
	// Find a directory child
	var dirChild *FileNode
	for _, child := range root.Children {
		if child.IsDir {
			dirChild = child
			break
		}
	}
	require.NotNil(t, dirChild)
	
	t.Run("expand directory", func(t *testing.T) {
		assert.False(t, dirChild.IsExpanded)
		assert.False(t, dirChild.IsLoaded)
		
		err := navigator.ToggleExpanded(dirChild)
		assert.NoError(t, err)
		assert.True(t, dirChild.IsExpanded)
		assert.True(t, dirChild.IsLoaded)
	})
	
	t.Run("collapse directory", func(t *testing.T) {
		err := navigator.ToggleExpanded(dirChild)
		assert.NoError(t, err)
		assert.False(t, dirChild.IsExpanded)
		assert.True(t, dirChild.IsLoaded) // Should remain loaded
	})
	
	t.Run("cannot expand file", func(t *testing.T) {
		var fileChild *FileNode
		for _, child := range root.Children {
			if !child.IsDir {
				fileChild = child
				break
			}
		}
		require.NotNil(t, fileChild)
		
		err := navigator.ToggleExpanded(fileChild)
		assert.Error(t, err)
	})
}

func TestNavigator_RefreshNode(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	root := navigator.GetRoot()
	initialChildCount := len(root.Children)
	
	t.Run("refresh after adding file", func(t *testing.T) {
		// Add a new file
		newFile := filepath.Join(tempDir, "new_file.txt")
		err := os.WriteFile(newFile, []byte("content"), 0644)
		require.NoError(t, err)
		
		// Refresh the node
		err = navigator.RefreshNode(root)
		assert.NoError(t, err)
		
		// Should have one more child
		assert.Equal(t, initialChildCount+1, len(root.Children))
		
		// Find the new file
		found := false
		for _, child := range root.Children {
			if child.Name == "new_file.txt" {
				found = true
				break
			}
		}
		assert.True(t, found, "New file should be found after refresh")
	})
	
	t.Run("refresh non-directory fails", func(t *testing.T) {
		var fileChild *FileNode
		for _, child := range root.Children {
			if !child.IsDir {
				fileChild = child
				break
			}
		}
		require.NotNil(t, fileChild)
		
		err := navigator.RefreshNode(fileChild)
		assert.Error(t, err)
	})
}

func TestNavigator_FindNode(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	t.Run("find root node", func(t *testing.T) {
		node := navigator.FindNode(tempDir)
		assert.NotNil(t, node)
		assert.Equal(t, tempDir, node.Path)
		assert.True(t, node.IsRoot())
	})
	
	t.Run("find child node", func(t *testing.T) {
		readmePath := filepath.Join(tempDir, "README.md")
		node := navigator.FindNode(readmePath)
		assert.NotNil(t, node)
		assert.Equal(t, readmePath, node.Path)
		assert.Equal(t, "README.md", node.Name)
	})
	
	t.Run("find non-existent node", func(t *testing.T) {
		node := navigator.FindNode(filepath.Join(tempDir, "nonexistent.txt"))
		assert.Nil(t, node)
	})
}

func TestNavigator_GetFlattenedNodes(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	t.Run("flattened nodes with collapsed directories", func(t *testing.T) {
		nodes := navigator.GetFlattenedNodes()
		
		// Should only include root and its direct children
		expectedCount := 1 + len(navigator.GetRoot().Children)
		assert.Equal(t, expectedCount, len(nodes))
		
		// First node should be root
		assert.Equal(t, navigator.GetRoot(), nodes[0])
		
		// All nodes should have correct depth
		for i, node := range nodes {
			if i == 0 {
				assert.Equal(t, 0, node.Depth)
			} else {
				assert.Equal(t, 1, node.Depth)
			}
		}
	})
	
	t.Run("flattened nodes with expanded directory", func(t *testing.T) {
		root := navigator.GetRoot()
		
		// Find and expand a directory
		var dirChild *FileNode
		for _, child := range root.Children {
			if child.IsDir {
				dirChild = child
				break
			}
		}
		require.NotNil(t, dirChild)
		
		err := navigator.ToggleExpanded(dirChild)
		require.NoError(t, err)
		
		nodes := navigator.GetFlattenedNodes()
		
		// Should include root, its children, and the expanded directory's children
		expectedMinCount := 1 + len(root.Children) + len(dirChild.Children)
		assert.GreaterOrEqual(t, len(nodes), expectedMinCount)
		
		// Find the expanded directory in the flattened list
		var foundExpanded bool
		var expandedIndex int
		for i, node := range nodes {
			if node == dirChild {
				foundExpanded = true
				expandedIndex = i
				break
			}
		}
		assert.True(t, foundExpanded)
		
		// Check that children of expanded directory follow it
		childrenFound := 0
		for i := expandedIndex + 1; i < len(nodes) && childrenFound < len(dirChild.Children); i++ {
			if nodes[i].Parent == dirChild {
				childrenFound++
				assert.Equal(t, dirChild.Depth+1, nodes[i].Depth)
			} else {
				break
			}
		}
		assert.Equal(t, len(dirChild.Children), childrenFound)
	})
}

func TestNavigator_ReadFile(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	t.Run("read text file", func(t *testing.T) {
		readmePath := filepath.Join(tempDir, "README.md")
		content, err := navigator.ReadFile(readmePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "# Test Project")
	})
	
	t.Run("read binary file fails", func(t *testing.T) {
		binaryPath := filepath.Join(tempDir, "binary_file.bin")
		_, err := navigator.ReadFile(binaryPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "binary file")
	})
	
	t.Run("read non-existent file fails", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")
		_, err := navigator.ReadFile(nonExistentPath)
		assert.Error(t, err)
	})
	
	t.Run("read file outside allowed paths fails", func(t *testing.T) {
		_, err := navigator.ReadFile("/etc/passwd")
		assert.Error(t, err)
	})
}

func TestNavigator_GetFileInfo(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	t.Run("get file info for regular file", func(t *testing.T) {
		readmePath := filepath.Join(tempDir, "README.md")
		fileInfo, err := navigator.GetFileInfo(readmePath)
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.Equal(t, "README.md", fileInfo.Name)
		assert.Equal(t, readmePath, fileInfo.Path)
		assert.False(t, fileInfo.IsDir)
		assert.Equal(t, FileTypeMarkdown, fileInfo.FileType)
		assert.Greater(t, fileInfo.Size, int64(0))
	})
	
	t.Run("get file info for directory", func(t *testing.T) {
		srcPath := filepath.Join(tempDir, "src")
		fileInfo, err := navigator.GetFileInfo(srcPath)
		assert.NoError(t, err)
		assert.NotNil(t, fileInfo)
		assert.Equal(t, "src", fileInfo.Name)
		assert.True(t, fileInfo.IsDir)
	})
	
	t.Run("get file info for non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")
		_, err := navigator.GetFileInfo(nonExistentPath)
		assert.Error(t, err)
	})
}

func TestNavigator_Search(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	ctx := context.Background()
	
	t.Run("search for files by name", func(t *testing.T) {
		testutil.WithTimeout(t, 5*time.Second, func(ctx context.Context) {
			options := &SearchOptions{
				Pattern:    "*.go",
				MaxResults: 10,
			}
			
			results, err := navigator.Search(ctx, options)
			assert.NoError(t, err)
			assert.Greater(t, len(results), 0)
			
			// All results should be Go files
			for _, result := range results {
				assert.True(t, strings.HasSuffix(result.Name, ".go"))
			}
		})
	})
	
	t.Run("search with regex pattern", func(t *testing.T) {
		testutil.WithTimeout(t, 5*time.Second, func(ctx context.Context) {
			options := &SearchOptions{
				Pattern:    "main\\.(go|txt)",
				IsRegex:    true,
				MaxResults: 10,
			}
			
			results, err := navigator.Search(ctx, options)
			assert.NoError(t, err)
			assert.Greater(t, len(results), 0)
		})
	})
	
	t.Run("search with file type filter", func(t *testing.T) {
		testutil.WithTimeout(t, 5*time.Second, func(ctx context.Context) {
			options := &SearchOptions{
				Pattern:   "*",
				FileTypes: []FileType{FileTypeMarkdown},
				MaxResults: 10,
			}
			
			results, err := navigator.Search(ctx, options)
			assert.NoError(t, err)
			
			// All results should be markdown files
			for _, result := range results {
				assert.Equal(t, FileTypeMarkdown, result.FileType)
			}
		})
	})
}

func TestNavigator_QuickSearch(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	ctx := context.Background()
	
	testutil.WithTimeout(t, 5*time.Second, func(ctx context.Context) {
		results, err := navigator.QuickSearch(ctx, "main")
		assert.NoError(t, err)
		assert.Greater(t, len(results), 0)
		
		// All results should contain "main" in the name or path
		for _, result := range results {
			nameMatch := strings.Contains(strings.ToLower(result.Name), "main")
			pathMatch := strings.Contains(strings.ToLower(result.RelativePath), "main")
			assert.True(t, nameMatch || pathMatch, "Result should contain 'main': %s", result.Name)
		}
	})
}

func TestNavigator_GetBreadcrumb(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	t.Run("breadcrumb for root", func(t *testing.T) {
		breadcrumb := navigator.GetBreadcrumb()
		assert.NotEmpty(t, breadcrumb)
		assert.Equal(t, filepath.Base(tempDir), breadcrumb[len(breadcrumb)-1])
	})
	
	t.Run("breadcrumb for nested path", func(t *testing.T) {
		nestedPath := filepath.Join(tempDir, "src", "main")
		err := navigator.Navigate(nestedPath)
		require.NoError(t, err)
		
		breadcrumb := navigator.GetBreadcrumb()
		assert.Contains(t, breadcrumb, "main")
		assert.Contains(t, breadcrumb, "src")
	})
}

func TestNavigator_FileTypeDetection(t *testing.T) {
	tempDir := testutil.TempDir(t)
	testutil.CreateTestFileStructure(t, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	err := navigator.SetRoot(tempDir)
	require.NoError(t, err)
	
	tests := []struct {
		filename string
		expected FileType
		isText   bool
		language string
	}{
		{"README.md", FileTypeMarkdown, true, "markdown"},
		{"go.mod", FileTypeGo, true, "go"},
		{"main.go", FileTypeGo, true, "go"},
		{"config.yaml", FileTypeYAML, true, "yaml"},
		{"Dockerfile", FileTypeDockerfile, true, "dockerfile"},
		{"Makefile", FileTypeMakefile, true, "makefile"},
		{"binary_file.bin", FileTypeBinary, false, "text"},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)
			
			fileType := navigator.GetFileType(filePath)
			assert.Equal(t, tt.expected, fileType, "File type mismatch for %s", tt.filename)
			
			isText := navigator.IsTextFile(filePath)
			assert.Equal(t, tt.isText, isText, "Text file detection mismatch for %s", tt.filename)
			
			language := navigator.GetLanguage(filePath)
			assert.Equal(t, tt.language, language, "Language detection mismatch for %s", tt.filename)
		})
	}
}

func TestNavigationHistory(t *testing.T) {
	history := NewNavigationHistory(5)
	
	t.Run("empty history", func(t *testing.T) {
		assert.False(t, history.CanGoBack())
		assert.False(t, history.CanGoForward())
		
		_, ok := history.Back()
		assert.False(t, ok)
		
		_, ok = history.Forward()
		assert.False(t, ok)
	})
	
	t.Run("single item", func(t *testing.T) {
		history.Push("/path1")
		
		assert.False(t, history.CanGoBack())
		assert.False(t, history.CanGoForward())
	})
	
	t.Run("multiple items", func(t *testing.T) {
		history.Push("/path2")
		history.Push("/path3")
		
		assert.True(t, history.CanGoBack())
		assert.False(t, history.CanGoForward())
		
		// Go back
		path, ok := history.Back()
		assert.True(t, ok)
		assert.Equal(t, "/path2", path)
		
		// Now we can go forward
		assert.True(t, history.CanGoForward())
		
		path, ok = history.Forward()
		assert.True(t, ok)
		assert.Equal(t, "/path3", path)
	})
	
	t.Run("duplicate paths not added", func(t *testing.T) {
		initialSize := len(history.history)
		history.Push("/path3") // Same as current
		assert.Equal(t, initialSize, len(history.history))
	})
	
	t.Run("history size limit", func(t *testing.T) {
		// Fill beyond capacity
		for i := 0; i < 10; i++ {
			history.Push(filepath.Join("/path", string(rune('a'+i))))
		}
		
		assert.LessOrEqual(t, len(history.history), 5)
	})
}

func TestFileNode_Methods(t *testing.T) {
	parent := &FileNode{
		Path:  "/parent",
		Name:  "parent",
		IsDir: true,
		Depth: 0,
	}
	
	child := &FileNode{
		Path:       "/parent/child.go",
		Name:       "child.go",
		IsDir:      false,
		FileType:   FileTypeGo,
		Parent:     parent,
		Depth:      1,
		IsExpanded: false,
		Children:   []*FileNode{},
	}
	
	t.Run("IsRoot", func(t *testing.T) {
		assert.True(t, parent.IsRoot())
		assert.False(t, child.IsRoot())
	})
	
	t.Run("GetIcon", func(t *testing.T) {
		assert.Equal(t, "📁", parent.GetIcon()) // Collapsed directory
		parent.IsExpanded = true
		assert.Equal(t, "📂", parent.GetIcon()) // Expanded directory
		
		assert.Equal(t, "🐹", child.GetIcon()) // Go file icon
	})
	
	t.Run("GetDisplayName", func(t *testing.T) {
		// Directory with children shows count
		parent.Children = []*FileNode{child}
		assert.Equal(t, "parent (1)", parent.GetDisplayName())
		
		// File shows just name
		assert.Equal(t, "child.go", child.GetDisplayName())
		
		// Empty directory shows just name
		emptyDir := &FileNode{Name: "empty", IsDir: true, Children: []*FileNode{}}
		assert.Equal(t, "empty", emptyDir.GetDisplayName())
	})
}

// Benchmark tests for navigation performance
func BenchmarkNavigator_SetRoot(b *testing.B) {
	tempDir := testutil.TempDir(b)
	testutil.CreateTestFileStructure(b, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		navigator := NewNavigator(config)
		navigator.SetRoot(tempDir)
	}
}

func BenchmarkNavigator_LoadChildren(b *testing.B) {
	tempDir := testutil.TempDir(b)
	
	// Create many files for benchmarking
	for i := 0; i < 1000; i++ {
		testutil.CreateTestFile(b, tempDir, filepath.Join("files", "file_"+string(rune(i))+".txt"), "content")
	}
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	navigator.SetRoot(tempDir)
	
	filesDir := filepath.Join(tempDir, "files")
	dirNode := &FileNode{
		Path:  filesDir,
		Name:  "files",
		IsDir: true,
		Depth: 1,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dirNode.IsLoaded = false
		navigator.LoadChildren(dirNode)
	}
}

func BenchmarkNavigator_GetFlattenedNodes(b *testing.B) {
	tempDir := testutil.TempDir(b)
	testutil.CreateTestFileStructure(b, tempDir)
	
	config := &SecurityConfig{
		AllowedPaths:   []string{tempDir},
		ForbiddenPaths: []string{},
		MaxDepth:       10,
		AllowHidden:    true,
		MaxFileSize:    10 * 1024 * 1024,
	}
	
	navigator := NewNavigator(config)
	navigator.SetRoot(tempDir)
	
	// Expand all directories
	root := navigator.GetRoot()
	for _, child := range root.Children {
		if child.IsDir {
			navigator.ToggleExpanded(child)
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		navigator.GetFlattenedNodes()
	}
}