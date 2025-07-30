package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/claude"
	"openai-cli/internal/app/testutil"
)

func TestNewRegistry(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	t.Run("valid registry creation", func(t *testing.T) {
		registry, err := NewRegistry(tempDir, logger)
		require.NoError(t, err)
		assert.NotNil(t, registry)
		assert.NotNil(t, registry.sessions)
		assert.NotNil(t, registry.indexByName)
		assert.NotNil(t, registry.indexByTags)
		assert.NotNil(t, registry.indexByState)
		
		// Registry file should be created
		registryFile := filepath.Join(tempDir, "registry.json")
		testutil.AssertFileExists(t, registryFile)
		
		registry.Close()
	})
	
	t.Run("invalid storage path", func(t *testing.T) {
		// Try to create registry in a non-existent parent directory
		invalidPath := "/root/nonexistent/registry"
		_, err := NewRegistry(invalidPath, logger)
		assert.Error(t, err)
	})
	
	t.Run("registry with existing data", func(t *testing.T) {
		registryDir := testutil.CreateTestDir(t, tempDir, "existing")
		
		// Create registry first time
		registry1, err := NewRegistry(registryDir, logger)
		require.NoError(t, err)
		
		// Add some data
		metadata := &claude.SessionMetadata{
			ID:        "test-session",
			Name:      "Test Session",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{"test"},
		}
		
		err = registry1.RegisterSession(metadata)
		require.NoError(t, err)
		registry1.Close()
		
		// Create registry second time (should load existing data)
		registry2, err := NewRegistry(registryDir, logger)
		require.NoError(t, err)
		defer registry2.Close()
		
		// Should have loaded the session
		loadedMetadata, err := registry2.GetSession("test-session")
		assert.NoError(t, err)
		assert.Equal(t, "test-session", loadedMetadata.ID)
		assert.Equal(t, "Test Session", loadedMetadata.Name)
	})
}

func TestRegistry_RegisterSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	t.Run("register valid session", func(t *testing.T) {
		metadata := &claude.SessionMetadata{
			ID:        "register-test-1",
			Name:      "Register Test 1",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{"test", "register"},
		}
		
		err := registry.RegisterSession(metadata)
		assert.NoError(t, err)
		
		// Verify registration
		retrieved, err := registry.GetSession("register-test-1")
		assert.NoError(t, err)
		assert.Equal(t, metadata.ID, retrieved.ID)
		assert.Equal(t, metadata.Name, retrieved.Name)
	})
	
	t.Run("register nil metadata", func(t *testing.T) {
		err := registry.RegisterSession(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
	
	t.Run("register duplicate session", func(t *testing.T) {
		metadata := &claude.SessionMetadata{
			ID:        "duplicate-test",
			Name:      "Duplicate Test",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		// Register first time
		err := registry.RegisterSession(metadata)
		require.NoError(t, err)
		
		// Register again (should fail)
		err = registry.RegisterSession(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})
}

func TestRegistry_UpdateSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	// Register initial session
	originalMetadata := &claude.SessionMetadata{
		ID:        "update-test",
		Name:      "Original Name",
		State:     claude.SessionStateCreated,
		Config:    claude.DefaultSessionConfig(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tags:      []string{"original"},
	}
	
	err = registry.RegisterSession(originalMetadata)
	require.NoError(t, err)
	
	t.Run("update existing session", func(t *testing.T) {
		updatedMetadata := &claude.SessionMetadata{
			ID:        "update-test",
			Name:      "Updated Name",
			State:     claude.SessionStateActive,
			Config:    originalMetadata.Config,
			CreatedAt: originalMetadata.CreatedAt,
			UpdatedAt: time.Now(),
			Tags:      []string{"updated", "modified"},
		}
		
		err := registry.UpdateSession(updatedMetadata)
		assert.NoError(t, err)
		
		// Verify update
		retrieved, err := registry.GetSession("update-test")
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, claude.SessionStateActive, retrieved.State)
		assert.Contains(t, retrieved.Tags, "updated")
		assert.Contains(t, retrieved.Tags, "modified")
		assert.NotContains(t, retrieved.Tags, "original")
	})
	
	t.Run("update nil metadata", func(t *testing.T) {
		err := registry.UpdateSession(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
	
	t.Run("update non-existent session", func(t *testing.T) {
		nonExistentMetadata := &claude.SessionMetadata{
			ID:        "nonexistent",
			Name:      "Non-existent",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err := registry.UpdateSession(nonExistentMetadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRegistry_UnregisterSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	// Register test session
	metadata := &claude.SessionMetadata{
		ID:        "unregister-test",
		Name:      "Unregister Test",
		State:     claude.SessionStateCreated,
		Config:    claude.DefaultSessionConfig(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Tags:      []string{"test"},
	}
	
	err = registry.RegisterSession(metadata)
	require.NoError(t, err)
	
	t.Run("unregister existing session", func(t *testing.T) {
		err := registry.UnregisterSession("unregister-test")
		assert.NoError(t, err)
		
		// Verify removal
		_, err = registry.GetSession("unregister-test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("unregister non-existent session", func(t *testing.T) {
		err := registry.UnregisterSession("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRegistry_GetSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	// Register test sessions
	metadata1 := &claude.SessionMetadata{
		ID:        "get-test-1",
		Name:      "Get Test 1",
		State:     claude.SessionStateCreated,
		Config:    claude.DefaultSessionConfig(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	metadata2 := &claude.SessionMetadata{
		ID:        "get-test-2",
		Name:      "Get Test 2",
		State:     claude.SessionStateActive,
		Config:    claude.DefaultSessionConfig(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = registry.RegisterSession(metadata1)
	require.NoError(t, err)
	err = registry.RegisterSession(metadata2)
	require.NoError(t, err)
	
	t.Run("get existing session", func(t *testing.T) {
		retrieved, err := registry.GetSession("get-test-1")
		assert.NoError(t, err)
		assert.Equal(t, metadata1.ID, retrieved.ID)
		assert.Equal(t, metadata1.Name, retrieved.Name)
		assert.Equal(t, metadata1.State, retrieved.State)
	})
	
	t.Run("get non-existent session", func(t *testing.T) {
		_, err := registry.GetSession("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("returned metadata is copy", func(t *testing.T) {
		retrieved, err := registry.GetSession("get-test-1")
		require.NoError(t, err)
		
		// Modify retrieved metadata
		retrieved.Name = "Modified Name"
		
		// Original should be unchanged
		retrievedAgain, err := registry.GetSession("get-test-1")
		require.NoError(t, err)
		assert.Equal(t, "Get Test 1", retrievedAgain.Name)
	})
}

func TestRegistry_ListSessions(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	t.Run("list empty registry", func(t *testing.T) {
		sessions, err := registry.ListSessions()
		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})
	
	t.Run("list with sessions", func(t *testing.T) {
		// Create sessions with different creation times
		baseTime := time.Now()
		
		metadata1 := &claude.SessionMetadata{
			ID:        "list-test-1",
			Name:      "List Test 1",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: baseTime.Add(-time.Hour),
			UpdatedAt: baseTime.Add(-time.Hour),
		}
		
		metadata2 := &claude.SessionMetadata{
			ID:        "list-test-2",
			Name:      "List Test 2",
			State:     claude.SessionStateActive,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: baseTime.Add(-time.Minute),
			UpdatedAt: baseTime.Add(-time.Minute),
		}
		
		metadata3 := &claude.SessionMetadata{
			ID:        "list-test-3",
			Name:      "List Test 3",
			State:     claude.SessionStatePaused,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: baseTime,
			UpdatedAt: baseTime,
		}
		
		err = registry.RegisterSession(metadata1)
		require.NoError(t, err)
		err = registry.RegisterSession(metadata2)
		require.NoError(t, err)
		err = registry.RegisterSession(metadata3)
		require.NoError(t, err)
		
		sessions, err := registry.ListSessions()
		assert.NoError(t, err)
		assert.Len(t, sessions, 3)
		
		// Should be sorted by creation time (newest first)
		assert.Equal(t, "list-test-3", sessions[0].ID)
		assert.Equal(t, "list-test-2", sessions[1].ID)
		assert.Equal(t, "list-test-1", sessions[2].ID)
	})
}

func TestRegistry_FindSessions(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	// Create test sessions
	sessions := []*claude.SessionMetadata{
		{
			ID:        "find-test-1",
			Name:      "Development Session",
			State:     claude.SessionStateActive,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Hour),
			Tags:      []string{"development", "coding"},
		},
		{
			ID:        "find-test-2",
			Name:      "Testing Session",
			State:     claude.SessionStatePaused,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now().Add(-time.Minute * 30),
			UpdatedAt: time.Now().Add(-time.Minute * 30),
			Tags:      []string{"testing", "qa"},
		},
		{
			ID:        "find-test-3",
			Name:      "Production Deploy",
			State:     claude.SessionStateTerminated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{"production", "deployment"},
		},
	}
	
	for _, session := range sessions {
		err = registry.RegisterSession(session)
		require.NoError(t, err)
	}
	
	t.Run("find by name pattern", func(t *testing.T) {
		// Find sessions with "Session" in name
		results, err := registry.FindSessionsByName("Session")
		assert.NoError(t, err)
		assert.Len(t, results, 2) // Development Session, Testing Session
		
		// Find sessions with "Deploy" in name
		results, err = registry.FindSessionsByName("Deploy")
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Production Deploy", results[0].Name)
		
		// Case insensitive search
		results, err = registry.FindSessionsByName("session")
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		
		// Non-matching pattern
		results, err = registry.FindSessionsByName("NonExistent")
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
	
	t.Run("find by tag", func(t *testing.T) {
		// Find by specific tag
		results, err := registry.FindSessionsByTag("development")
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "find-test-1", results[0].ID)
		
		// Find by another tag
		results, err = registry.FindSessionsByTag("production")
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "find-test-3", results[0].ID)
		
		// Non-existent tag
		results, err = registry.FindSessionsByTag("nonexistent")
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
	
	t.Run("find by state", func(t *testing.T) {
		// Find active sessions
		results, err := registry.FindSessionsByState(claude.SessionStateActive)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "find-test-1", results[0].ID)
		
		// Find paused sessions
		results, err = registry.FindSessionsByState(claude.SessionStatePaused)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "find-test-2", results[0].ID)
		
		// Find terminated sessions
		results, err = registry.FindSessionsByState(claude.SessionStateTerminated)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "find-test-3", results[0].ID)
		
		// Find non-existent state
		results, err = registry.FindSessionsByState(claude.SessionStateError)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestRegistry_GetStats(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	t.Run("empty registry stats", func(t *testing.T) {
		stats := registry.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats.TotalSessions)
		assert.Empty(t, stats.StateCount)
		assert.Empty(t, stats.TagCount)
		assert.True(t, stats.OldestSession.IsZero())
		assert.True(t, stats.NewestSession.IsZero())
	})
	
	t.Run("registry with sessions stats", func(t *testing.T) {
		baseTime := time.Now()
		
		sessions := []*claude.SessionMetadata{
			{
				ID:        "stats-test-1",
				Name:      "Stats Test 1",
				State:     claude.SessionStateActive,
				Config:    claude.DefaultSessionConfig(),
				CreatedAt: baseTime.Add(-time.Hour),
				UpdatedAt: baseTime.Add(-time.Hour),
				Tags:      []string{"tag1", "common"},
			},
			{
				ID:        "stats-test-2",
				Name:      "Stats Test 2",
				State:     claude.SessionStateActive,
				Config:    claude.DefaultSessionConfig(),
				CreatedAt: baseTime.Add(-time.Minute * 30),
				UpdatedAt: baseTime.Add(-time.Minute * 30),
				Tags:      []string{"tag2", "common"},
			},
			{
				ID:        "stats-test-3",
				Name:      "Stats Test 3",
				State:     claude.SessionStatePaused,
				Config:    claude.DefaultSessionConfig(),
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
				Tags:      []string{"tag3"},
			},
		}
		
		for _, session := range sessions {
			err = registry.RegisterSession(session)
			require.NoError(t, err)
		}
		
		stats := registry.GetStats()
		assert.Equal(t, 3, stats.TotalSessions)
		
		// State counts
		assert.Equal(t, 2, stats.StateCount["active"])
		assert.Equal(t, 1, stats.StateCount["paused"])
		
		// Tag counts
		assert.Equal(t, 2, stats.TagCount["common"])
		assert.Equal(t, 1, stats.TagCount["tag1"])
		assert.Equal(t, 1, stats.TagCount["tag2"])
		assert.Equal(t, 1, stats.TagCount["tag3"])
		
		// Time ranges
		assert.Equal(t, baseTime.Add(-time.Hour).Truncate(time.Second), stats.OldestSession.Truncate(time.Second))
		assert.Equal(t, baseTime.Truncate(time.Second), stats.NewestSession.Truncate(time.Second))
	})
}

func TestRegistry_BackupAndRestore(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	t.Run("backup creation", func(t *testing.T) {
		registry, err := NewRegistry(tempDir, logger)
		require.NoError(t, err)
		defer registry.Close()
		
		// Add some data
		metadata := &claude.SessionMetadata{
			ID:        "backup-test",
			Name:      "Backup Test",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err = registry.RegisterSession(metadata)
		require.NoError(t, err)
		
		// Force a sync to create backup
		err = registry.Sync()
		assert.NoError(t, err)
		
		// Check for backup files
		files, err := filepath.Glob(filepath.Join(tempDir, "registry-backup-*.json"))
		assert.NoError(t, err)
		assert.Greater(t, len(files), 0)
	})
	
	t.Run("registry reload", func(t *testing.T) {
		registryDir := testutil.CreateTestDir(t, tempDir, "reload")
		
		// Create first registry and add data
		registry1, err := NewRegistry(registryDir, logger)
		require.NoError(t, err)
		
		metadata := &claude.SessionMetadata{
			ID:        "reload-test",
			Name:      "Reload Test",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err = registry1.RegisterSession(metadata)
		require.NoError(t, err)
		registry1.Close()
		
		// Create second registry and reload
		registry2, err := NewRegistry(registryDir, logger)
		require.NoError(t, err)
		defer registry2.Close()
		
		// Verify data was loaded
		retrieved, err := registry2.GetSession("reload-test")
		assert.NoError(t, err)
		assert.Equal(t, "reload-test", retrieved.ID)
		assert.Equal(t, "Reload Test", retrieved.Name)
		
		// Test explicit reload
		err = registry2.Reload()
		assert.NoError(t, err)
	})
}

func TestRegistry_IndexManagement(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	t.Run("index updates on registration", func(t *testing.T) {
		metadata := &claude.SessionMetadata{
			ID:        "index-test",
			Name:      "Index Test",
			State:     claude.SessionStateActive,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{"index", "test"},
		}
		
		err := registry.RegisterSession(metadata)
		require.NoError(t, err)
		
		// Verify indices are updated
		nameResults, err := registry.FindSessionsByName("Index Test")
		assert.NoError(t, err)
		assert.Len(t, nameResults, 1)
		
		tagResults, err := registry.FindSessionsByTag("index")
		assert.NoError(t, err)
		assert.Len(t, tagResults, 1)
		
		stateResults, err := registry.FindSessionsByState(claude.SessionStateActive)
		assert.NoError(t, err)
		assert.Len(t, stateResults, 1)
	})
	
	t.Run("index updates on modification", func(t *testing.T) {
		// Update the session
		updatedMetadata := &claude.SessionMetadata{
			ID:        "index-test",
			Name:      "Updated Index Test",
			State:     claude.SessionStatePaused,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{"updated", "modified"},
		}
		
		err := registry.UpdateSession(updatedMetadata)
		require.NoError(t, err)
		
		// Old indices should be cleared
		oldNameResults, err := registry.FindSessionsByName("Index Test")
		assert.NoError(t, err)
		assert.Empty(t, oldNameResults)
		
		oldTagResults, err := registry.FindSessionsByTag("index")
		assert.NoError(t, err)
		assert.Empty(t, oldTagResults)
		
		oldStateResults, err := registry.FindSessionsByState(claude.SessionStateActive)
		assert.NoError(t, err)
		assert.Empty(t, oldStateResults)
		
		// New indices should be populated
		newNameResults, err := registry.FindSessionsByName("Updated Index Test")
		assert.NoError(t, err)
		assert.Len(t, newNameResults, 1)
		
		newTagResults, err := registry.FindSessionsByTag("updated")
		assert.NoError(t, err)
		assert.Len(t, newTagResults, 1)
		
		newStateResults, err := registry.FindSessionsByState(claude.SessionStatePaused)
		assert.NoError(t, err)
		assert.Len(t, newStateResults, 1)
	})
	
	t.Run("index cleanup on unregistration", func(t *testing.T) {
		err := registry.UnregisterSession("index-test")
		require.NoError(t, err)
		
		// All indices should be cleaned up
		nameResults, err := registry.FindSessionsByName("Updated Index Test")
		assert.NoError(t, err)
		assert.Empty(t, nameResults)
		
		tagResults, err := registry.FindSessionsByTag("updated")
		assert.NoError(t, err)
		assert.Empty(t, tagResults)
		
		stateResults, err := registry.FindSessionsByState(claude.SessionStatePaused)
		assert.NoError(t, err)
		assert.Empty(t, stateResults)
	})
}

func TestRegistry_ConcurrentOperations(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	registry, err := NewRegistry(tempDir, logger)
	require.NoError(t, err)
	defer registry.Close()
	
	t.Run("concurrent registrations", func(t *testing.T) {
		const numGoroutines = 10
		errorChan := make(chan error, numGoroutines)
		
		// Register sessions concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				metadata := &claude.SessionMetadata{
					ID:        fmt.Sprintf("concurrent-%d", id),
					Name:      fmt.Sprintf("Concurrent %d", id),
					State:     claude.SessionStateCreated,
					Config:    claude.DefaultSessionConfig(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				
				err := registry.RegisterSession(metadata)
				errorChan <- err
			}(i)
		}
		
		// Collect results
		for i := 0; i < numGoroutines; i++ {
			err := <-errorChan
			assert.NoError(t, err)
		}
		
		// Verify all sessions were registered
		sessions, err := registry.ListSessions()
		assert.NoError(t, err)
		assert.Len(t, sessions, numGoroutines)
	})
	
	t.Run("concurrent reads and writes", func(t *testing.T) {
		// Add a session to read
		testMetadata := &claude.SessionMetadata{
			ID:        "read-write-test",
			Name:      "Read Write Test",
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err := registry.RegisterSession(testMetadata)
		require.NoError(t, err)
		
		const numGoroutines = 5
		errorChan := make(chan error, numGoroutines*2)
		
		// Concurrent reads
		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := registry.GetSession("read-write-test")
				errorChan <- err
			}()
		}
		
		// Concurrent updates
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				updatedMetadata := &claude.SessionMetadata{
					ID:        "read-write-test",
					Name:      fmt.Sprintf("Updated %d", id),
					State:     claude.SessionStateActive,
					Config:    testMetadata.Config,
					CreatedAt: testMetadata.CreatedAt,
					UpdatedAt: time.Now(),
				}
				
				err := registry.UpdateSession(updatedMetadata)
				errorChan <- err
			}(i)
		}
		
		// Collect results
		for i := 0; i < numGoroutines*2; i++ {
			err := <-errorChan
			assert.NoError(t, err)
		}
		
		// Verify session still exists and is consistent
		final, err := registry.GetSession("read-write-test")
		assert.NoError(t, err)
		assert.NotNil(t, final)
		assert.Equal(t, "read-write-test", final.ID)
	})
}

func TestDefaultRegistryConfig(t *testing.T) {
	config := DefaultRegistryConfig()
	
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.StoragePath)
	assert.True(t, config.BackupEnabled)
	assert.Greater(t, config.BackupCount, 0)
	assert.Greater(t, config.SyncInterval, time.Duration(0))
}

// Benchmark tests for registry performance
func BenchmarkRegistry_RegisterSession(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	registry, err := NewRegistry(tempDir, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer registry.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metadata := &claude.SessionMetadata{
			ID:        fmt.Sprintf("bench-%d", i),
			Name:      fmt.Sprintf("Bench %d", i),
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{"benchmark"},
		}
		
		err := registry.RegisterSession(metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRegistry_GetSession(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	registry, err := NewRegistry(tempDir, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer registry.Close()
	
	// Pre-populate registry
	var sessionIDs []string
	for i := 0; i < 100; i++ {
		metadata := &claude.SessionMetadata{
			ID:        fmt.Sprintf("bench-%d", i),
			Name:      fmt.Sprintf("Bench %d", i),
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err := registry.RegisterSession(metadata)
		if err != nil {
			b.Fatal(err)
		}
		sessionIDs = append(sessionIDs, metadata.ID)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := sessionIDs[i%len(sessionIDs)]
		_, err := registry.GetSession(sessionID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRegistry_FindSessionsByTag(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	registry, err := NewRegistry(tempDir, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer registry.Close()
	
	// Pre-populate registry with tagged sessions
	tags := []string{"development", "testing", "production", "staging", "research"}
	for i := 0; i < 100; i++ {
		metadata := &claude.SessionMetadata{
			ID:        fmt.Sprintf("bench-%d", i),
			Name:      fmt.Sprintf("Bench %d", i),
			State:     claude.SessionStateCreated,
			Config:    claude.DefaultSessionConfig(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Tags:      []string{tags[i%len(tags)]},
		}
		
		err := registry.RegisterSession(metadata)
		if err != nil {
			b.Fatal(err)
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tag := tags[i%len(tags)]
		_, err := registry.FindSessionsByTag(tag)
		if err != nil {
			b.Fatal(err)
		}
	}
}