package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"openai-cli/internal/app/claude"
	"openai-cli/internal/app/testutil"
)

func TestNewManager(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	// Create mock claude executable
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	t.Run("valid configuration", func(t *testing.T) {
		config := &ManagerConfig{
			StoragePath:     tempDir,
			MaxSessions:     5,
			DefaultTimeout:  time.Hour,
			CleanupInterval: time.Minute * 30,
			AutoCleanup:     true,
			ClaudePath:      claudePath,
		}
		
		manager, err := NewManager(config, logger)
		require.NoError(t, err)
		assert.NotNil(t, manager)
		assert.Equal(t, config, manager.config)
		assert.NotNil(t, manager.executor)
		assert.NotNil(t, manager.registry)
		assert.NotNil(t, manager.sessions)
		
		// Clean up
		manager.Close()
	})
	
	t.Run("nil configuration uses defaults", func(t *testing.T) {
		// Override default claude path
		defaultConfig := DefaultManagerConfig()
		defaultConfig.ClaudePath = claudePath
		defaultConfig.StoragePath = tempDir
		
		manager, err := NewManager(defaultConfig, logger)
		require.NoError(t, err)
		assert.NotNil(t, manager)
		
		manager.Close()
	})
	
	t.Run("invalid claude path", func(t *testing.T) {
		config := &ManagerConfig{
			StoragePath: tempDir,
			ClaudePath:  "/nonexistent/claude",
		}
		
		_, err := NewManager(config, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executor")
	})
	
	t.Run("invalid storage path", func(t *testing.T) {
		config := &ManagerConfig{
			StoragePath: "/root/nonexistent", // Should fail on permission
			ClaudePath:  claudePath,
		}
		
		_, err := NewManager(config, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registry")
	})
}

func TestManager_CreateSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath:     tempDir,
		MaxSessions:     3,
		DefaultTimeout:  time.Hour,
		CleanupInterval: time.Hour,
		AutoCleanup:     false, // Don't auto-cleanup during tests
		ClaudePath:      claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	t.Run("create valid session", func(t *testing.T) {
		sessionConfig := &claude.SessionConfig{
			Name:       "test-session",
			WorkingDir: tempDir,
			AutoSave:   true,
		}
		
		session, err := manager.CreateSession("test-session", sessionConfig)
		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, "test-session", session.GetName())
		assert.Equal(t, claude.SessionStateCreated, session.GetState())
		
		// Check that session is tracked
		assert.Equal(t, 1, manager.GetSessionCount())
	})
	
	t.Run("create session with nil config", func(t *testing.T) {
		session, err := manager.CreateSession("default-session", nil)
		assert.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, "default-session", session.GetName())
	})
	
	t.Run("exceed maximum sessions", func(t *testing.T) {
		// Create sessions up to the limit
		for i := 0; i < 2; i++ { // We already have 2 from previous tests
			_, err := manager.CreateSession(testutil.GenerateID(), nil)
			require.NoError(t, err)
		}
		
		// This should fail
		_, err := manager.CreateSession("too-many", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum number")
	})
}

func TestManager_GetSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	// Create test sessions
	session1, err := manager.CreateSession("session1", nil)
	require.NoError(t, err)
	
	session2, err := manager.CreateSession("session2", nil)
	require.NoError(t, err)
	
	t.Run("get existing session by ID", func(t *testing.T) {
		retrieved, err := manager.GetSession(session1.GetID())
		assert.NoError(t, err)
		assert.Equal(t, session1, retrieved)
	})
	
	t.Run("get existing session by name", func(t *testing.T) {
		retrieved, err := manager.GetSessionByName("session2")
		assert.NoError(t, err)
		assert.Equal(t, session2, retrieved)
	})
	
	t.Run("get non-existent session by ID", func(t *testing.T) {
		_, err := manager.GetSession("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
	
	t.Run("get non-existent session by name", func(t *testing.T) {
		_, err := manager.GetSessionByName("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestManager_ListSessions(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	t.Run("list empty sessions", func(t *testing.T) {
		sessions := manager.ListSessions()
		assert.Empty(t, sessions)
		
		activeSessions := manager.ListActiveSessions()
		assert.Empty(t, activeSessions)
		
		assert.Equal(t, 0, manager.GetSessionCount())
		assert.Equal(t, 0, manager.GetActiveSessionCount())
	})
	
	t.Run("list with sessions", func(t *testing.T) {
		// Create test sessions
		session1, err := manager.CreateSession("session1", nil)
		require.NoError(t, err)
		
		session2, err := manager.CreateSession("session2", nil)
		require.NoError(t, err)
		
		// Start one session
		err = session1.Start()
		require.NoError(t, err)
		
		// List all sessions
		sessions := manager.ListSessions()
		assert.Len(t, sessions, 2)
		
		// List active sessions
		activeSessions := manager.ListActiveSessions()
		assert.Len(t, activeSessions, 1)
		assert.Equal(t, session1, activeSessions[0])
		
		// Check counts
		assert.Equal(t, 2, manager.GetSessionCount())
		assert.Equal(t, 1, manager.GetActiveSessionCount())
	})
}

func TestManager_SessionLifecycle(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	// Create test session
	session, err := manager.CreateSession("lifecycle-test", nil)
	require.NoError(t, err)
	
	sessionID := session.GetID()
	
	t.Run("start session", func(t *testing.T) {
		err := manager.StartSession(sessionID)
		assert.NoError(t, err)
		assert.True(t, session.IsActive())
	})
	
	t.Run("pause session", func(t *testing.T) {
		err := manager.PauseSession(sessionID)
		assert.NoError(t, err)
		assert.True(t, session.IsPaused())
	})
	
	t.Run("resume session", func(t *testing.T) {
		err := manager.ResumeSession(sessionID)
		assert.NoError(t, err)
		assert.True(t, session.IsActive())
	})
	
	t.Run("terminate session", func(t *testing.T) {
		err := manager.TerminateSession(sessionID)
		assert.NoError(t, err)
		assert.True(t, session.IsTerminated())
	})
	
	t.Run("save session", func(t *testing.T) {
		err := manager.SaveSession(sessionID)
		assert.NoError(t, err)
	})
	
	t.Run("operations on non-existent session", func(t *testing.T) {
		nonExistentID := "nonexistent"
		
		err := manager.StartSession(nonExistentID)
		assert.Error(t, err)
		
		err = manager.PauseSession(nonExistentID)
		assert.Error(t, err)
		
		err = manager.ResumeSession(nonExistentID)
		assert.Error(t, err)
		
		err = manager.TerminateSession(nonExistentID)
		assert.Error(t, err)
		
		err = manager.SaveSession(nonExistentID)
		assert.Error(t, err)
	})
}

func TestManager_DeleteSession(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	t.Run("delete inactive session", func(t *testing.T) {
		session, err := manager.CreateSession("delete-test-1", nil)
		require.NoError(t, err)
		
		sessionID := session.GetID()
		initialCount := manager.GetSessionCount()
		
		err = manager.DeleteSession(sessionID, false)
		assert.NoError(t, err)
		
		// Session should be removed
		_, err = manager.GetSession(sessionID)
		assert.Error(t, err)
		
		// Count should decrease
		assert.Equal(t, initialCount-1, manager.GetSessionCount())
	})
	
	t.Run("delete active session without force", func(t *testing.T) {
		session, err := manager.CreateSession("delete-test-2", nil)
		require.NoError(t, err)
		
		sessionID := session.GetID()
		
		// Start the session
		err = manager.StartSession(sessionID)
		require.NoError(t, err)
		
		// Try to delete without force
		err = manager.DeleteSession(sessionID, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "active")
		
		// Session should still exist
		_, err = manager.GetSession(sessionID)
		assert.NoError(t, err)
	})
	
	t.Run("delete active session with force", func(t *testing.T) {
		session, err := manager.CreateSession("delete-test-3", nil)
		require.NoError(t, err)
		
		sessionID := session.GetID()
		
		// Start the session
		err = manager.StartSession(sessionID)
		require.NoError(t, err)
		
		initialCount := manager.GetSessionCount()
		
		// Delete with force
		err = manager.DeleteSession(sessionID, true)
		assert.NoError(t, err)
		
		// Session should be removed
		_, err = manager.GetSession(sessionID)
		assert.Error(t, err)
		
		// Count should decrease
		assert.Equal(t, initialCount-1, manager.GetSessionCount())
	})
	
	t.Run("delete non-existent session", func(t *testing.T) {
		err := manager.DeleteSession("nonexistent", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestManager_SessionIO(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	// Create and start test session
	session, err := manager.CreateSession("io-test", nil)
	require.NoError(t, err)
	
	err = manager.StartSession(session.GetID())
	require.NoError(t, err)
	
	sessionID := session.GetID()
	
	t.Run("send input", func(t *testing.T) {
		err := manager.SendInput(sessionID, "test input")
		assert.NoError(t, err)
	})
	
	t.Run("get output", func(t *testing.T) {
		output, err := manager.GetOutput(sessionID)
		assert.NoError(t, err)
		assert.NotNil(t, output)
	})
	
	t.Run("subscribe to output", func(t *testing.T) {
		outputChan, err := manager.SubscribeToOutput(sessionID)
		assert.NoError(t, err)
		assert.NotNil(t, outputChan)
		
		// Channel should be closed for sessions without active processes
		_, ok := <-outputChan
		assert.False(t, ok)
	})
	
	t.Run("IO operations on non-existent session", func(t *testing.T) {
		nonExistentID := "nonexistent"
		
		err := manager.SendInput(nonExistentID, "input")
		assert.Error(t, err)
		
		_, err = manager.GetOutput(nonExistentID)
		assert.Error(t, err)
		
		_, err = manager.SubscribeToOutput(nonExistentID)
		assert.Error(t, err)
	})
}

func TestManager_SaveAllSessions(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	// Create test sessions
	_, err = manager.CreateSession("save-test-1", nil)
	require.NoError(t, err)
	
	_, err = manager.CreateSession("save-test-2", nil)
	require.NoError(t, err)
	
	// Save all sessions
	err = manager.SaveAllSessions()
	assert.NoError(t, err)
}

func TestManager_GetStats(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	t.Run("empty stats", func(t *testing.T) {
		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats.TotalSessions)
		assert.Equal(t, 0, stats.ActiveSessions)
		assert.Equal(t, 0, stats.PausedSessions)
		assert.Equal(t, 0, stats.TerminatedSessions)
		assert.Equal(t, 0, stats.ErrorSessions)
	})
	
	t.Run("stats with sessions", func(t *testing.T) {
		// Create sessions in different states
		session1, err := manager.CreateSession("stats1", nil)
		require.NoError(t, err)
		
		session2, err := manager.CreateSession("stats2", nil)
		require.NoError(t, err)
		
		session3, err := manager.CreateSession("stats3", nil)
		require.NoError(t, err)
		
		// Start and pause sessions
		err = manager.StartSession(session1.GetID())
		require.NoError(t, err)
		
		err = manager.StartSession(session2.GetID())
		require.NoError(t, err)
		err = manager.PauseSession(session2.GetID())
		require.NoError(t, err)
		
		err = manager.TerminateSession(session3.GetID())
		require.NoError(t, err)
		
		stats := manager.GetStats()
		assert.Equal(t, 3, stats.TotalSessions)
		assert.Equal(t, 1, stats.ActiveSessions)
		assert.Equal(t, 1, stats.PausedSessions)
		assert.Equal(t, 1, stats.TerminatedSessions)
		assert.Equal(t, 0, stats.ErrorSessions)
	})
}

func TestManager_Close(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 10,
		ClaudePath:  claudePath,
		AutoCleanup: false, // Disable for controlled testing
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	
	// Create test sessions
	session1, err := manager.CreateSession("close-test-1", nil)
	require.NoError(t, err)
	
	session2, err := manager.CreateSession("close-test-2", nil)
	require.NoError(t, err)
	
	// Start sessions
	err = manager.StartSession(session1.GetID())
	require.NoError(t, err)
	
	err = manager.StartSession(session2.GetID())
	require.NoError(t, err)
	
	// Close manager
	err = manager.Close()
	assert.NoError(t, err)
	
	// Sessions should be terminated
	assert.True(t, session1.IsTerminated())
	assert.True(t, session2.IsTerminated())
}

func TestManager_ConcurrentOperations(t *testing.T) {
	tempDir := testutil.TempDir(t)
	logger := testutil.TestLogger(t)
	
	claudePath := testutil.MockExecutable(t, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 20,
		ClaudePath:  claudePath,
	}
	
	manager, err := NewManager(config, logger)
	require.NoError(t, err)
	defer manager.Close()
	
	t.Run("concurrent session creation", func(t *testing.T) {
		const numGoroutines = 10
		sessionChan := make(chan *claude.Session, numGoroutines)
		errorChan := make(chan error, numGoroutines)
		
		// Create sessions concurrently
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				sessionName := fmt.Sprintf("concurrent-%d", id)
				session, err := manager.CreateSession(sessionName, nil)
				if err != nil {
					errorChan <- err
				} else {
					sessionChan <- session
				}
			}(i)
		}
		
		// Collect results
		var sessions []*claude.Session
		for i := 0; i < numGoroutines; i++ {
			select {
			case session := <-sessionChan:
				sessions = append(sessions, session)
			case err := <-errorChan:
				t.Errorf("Concurrent session creation failed: %v", err)
			case <-time.After(time.Second * 5):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}
		
		assert.Len(t, sessions, numGoroutines)
		assert.GreaterOrEqual(t, manager.GetSessionCount(), numGoroutines)
	})
	
	t.Run("concurrent session operations", func(t *testing.T) {
		// Create a session to operate on
		session, err := manager.CreateSession("operation-test", nil)
		require.NoError(t, err)
		
		sessionID := session.GetID()
		const numGoroutines = 5
		errorChan := make(chan error, numGoroutines*3) // Each goroutine does multiple operations
		
		// Perform concurrent operations
		for i := 0; i < numGoroutines; i++ {
			go func() {
				// Start session
				if err := manager.StartSession(sessionID); err != nil {
					errorChan <- err
				}
				
				// Send input
				if err := manager.SendInput(sessionID, "test"); err != nil {
					errorChan <- err
				}
				
				// Get output
				if _, err := manager.GetOutput(sessionID); err != nil {
					errorChan <- err
				}
			}()
		}
		
		// Wait for operations to complete
		time.Sleep(100 * time.Millisecond)
		
		// Check for errors (some operations might fail due to state conflicts, which is expected)
		select {
		case err := <-errorChan:
			// Some errors are expected in concurrent operations
			t.Logf("Expected concurrent operation error: %v", err)
		default:
			// No immediate errors
		}
		
		// Session should still be valid
		retrievedSession, err := manager.GetSession(sessionID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedSession)
	})
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()
	
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.StoragePath)
	assert.Greater(t, config.MaxSessions, 0)
	assert.Greater(t, config.DefaultTimeout, time.Duration(0))
	assert.Greater(t, config.CleanupInterval, time.Duration(0))
	assert.NotEmpty(t, config.ClaudePath)
}

// Helper function to generate unique IDs for testing
func (testutil *testutil) GenerateID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

// Add the GenerateID function to testutil package
func init() {
	// This is a placeholder - the actual implementation would be in testutil package
}

// Benchmark tests for manager performance
func BenchmarkManager_CreateSession(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	claudePath := testutil.MockExecutable(b, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 1000, // High limit for benchmarking
		ClaudePath:  claudePath,
		AutoCleanup: false,
	}
	
	manager, err := NewManager(config, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionName := fmt.Sprintf("bench-session-%d", i)
		_, err := manager.CreateSession(sessionName, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManager_GetSession(b *testing.B) {
	tempDir := testutil.TempDir(b)
	logger := testutil.SilentLogger()
	
	claudePath := testutil.MockExecutable(b, tempDir, "claude", 
		"#!/bin/bash\nif [ \"$1\" = \"--version\" ]; then echo \"claude 1.0\"; exit 0; fi\necho \"mock claude\"")
	
	config := &ManagerConfig{
		StoragePath: tempDir,
		MaxSessions: 1000,
		ClaudePath:  claudePath,
		AutoCleanup: false,
	}
	
	manager, err := NewManager(config, logger)
	if err != nil {
		b.Fatal(err)
	}
	defer manager.Close()
	
	// Create test sessions
	var sessionIDs []string
	for i := 0; i < 100; i++ {
		session, err := manager.CreateSession(fmt.Sprintf("bench-%d", i), nil)
		if err != nil {
			b.Fatal(err)
		}
		sessionIDs = append(sessionIDs, session.GetID())
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := sessionIDs[i%len(sessionIDs)]
		_, err := manager.GetSession(sessionID)
		if err != nil {
			b.Fatal(err)
		}
	}
}