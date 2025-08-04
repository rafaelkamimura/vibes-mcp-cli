package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
)

// ManagerConfig holds configuration for the session manager
type ManagerConfig struct {
	StoragePath     string        `json:"storage_path"`     // Path to session storage directory
	MaxSessions     int           `json:"max_sessions"`     // Maximum concurrent sessions
	DefaultTimeout  time.Duration `json:"default_timeout"`  // Default session timeout
	CleanupInterval time.Duration `json:"cleanup_interval"` // Cleanup interval for terminated sessions
	AutoCleanup     bool          `json:"auto_cleanup"`     // Enable automatic cleanup
	ClaudePath      string        `json:"claude_path"`      // Path to Claude executable
}

// DefaultManagerConfig returns default manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		StoragePath:     "./sessions",
		MaxSessions:     10,
		DefaultTimeout:  time.Hour * 2,
		CleanupInterval: time.Hour,
		AutoCleanup:     true,
		ClaudePath:      "claude",
	}
}

// Manager manages multiple Claude Code sessions
type Manager struct {
	config   *ManagerConfig
	executor *claude.Executor
	registry *Registry
	logger   *zap.Logger
	mu       sync.RWMutex
	sessions map[string]*claude.Session // Session ID -> Session
	ctx      context.Context
	cancel   context.CancelFunc
	cleanup  *time.Ticker
}

// NewManager creates a new session manager
func NewManager(config *ManagerConfig, logger *zap.Logger) (*Manager, error) {
	if config == nil {
		config = DefaultManagerConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create executor
	executor, err := claude.NewExecutor(logger, config.ClaudePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	// Create registry
	registry, err := NewRegistry(config.StoragePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config:   config,
		executor: executor,
		registry: registry,
		logger:   logger,
		sessions: make(map[string]*claude.Session),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start automatic cleanup if enabled
	if config.AutoCleanup && config.CleanupInterval > 0 {
		manager.startCleanup()
	}

	// Load existing sessions
	if err := manager.loadExistingSessions(); err != nil {
		logger.Warn("failed to load existing sessions", zap.Error(err))
	}

	return manager, nil
}

// CreateSession creates a new Claude Code session
func (m *Manager) CreateSession(name string, config *claude.SessionConfig) (*claude.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check session limit
	if len(m.sessions) >= m.config.MaxSessions {
		return nil, fmt.Errorf("maximum number of sessions reached (%d)", m.config.MaxSessions)
	}

	// Generate unique session ID
	sessionID := m.generateSessionID()

	// Create session configuration if not provided
	if config == nil {
		config = claude.DefaultSessionConfig()
	}
	if config.Name == "" {
		config.Name = name
	}

	// Create session
	session := claude.NewSession(sessionID, name, config, m.executor, m.logger, m.config.StoragePath)

	// Register session
	if err := m.registry.RegisterSession(session.GetMetadata()); err != nil {
		return nil, fmt.Errorf("failed to register session: %w", err)
	}

	// Store session
	m.sessions[sessionID] = session

	m.logger.Info("session created",
		zap.String("session_id", sessionID),
		zap.String("name", name))

	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*claude.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// GetSessionByName retrieves a session by name (first match)
func (m *Manager) GetSessionByName(name string) (*claude.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, session := range m.sessions {
		if session.GetName() == name {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found with name: %s", name)
}

// ListSessions returns a list of all sessions
func (m *Manager) ListSessions() []*claude.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*claude.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// ListActiveSessions returns a list of active sessions
func (m *Manager) ListActiveSessions() []*claude.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*claude.Session, 0)
	for _, session := range m.sessions {
		if session.IsActive() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetSessionCount returns the current number of sessions
func (m *Manager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// GetActiveSessionCount returns the number of active sessions
func (m *Manager) GetActiveSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, session := range m.sessions {
		if session.IsActive() {
			count++
		}
	}
	return count
}

// StartSession starts a session
func (m *Manager) StartSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if err := session.Start(); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Update registry
	if err := m.registry.UpdateSession(session.GetMetadata()); err != nil {
		m.logger.Error("failed to update session in registry",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}

	return nil
}

// TerminateSession terminates a session
func (m *Manager) TerminateSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if err := session.Terminate(); err != nil {
		return fmt.Errorf("failed to terminate session: %w", err)
	}

	// Update registry
	if err := m.registry.UpdateSession(session.GetMetadata()); err != nil {
		m.logger.Error("failed to update session in registry",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}

	return nil
}

// DeleteSession deletes a session (terminates if active)
func (m *Manager) DeleteSession(sessionID string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Terminate if active and force is true, with timeout
	if session.IsActive() || session.IsPaused() {
		if !force {
			return fmt.Errorf("session is active, use force=true to terminate and delete")
		}
		
		// Use goroutine with timeout to prevent hanging
		terminateDone := make(chan error, 1)
		go func() {
			terminateDone <- session.Terminate()
		}()
		
		select {
		case err := <-terminateDone:
			if err != nil {
				m.logger.Error("failed to terminate session during deletion",
					zap.String("session_id", sessionID),
					zap.Error(err))
			}
		case <-time.After(10 * time.Second):
			m.logger.Error("session termination timed out during deletion",
				zap.String("session_id", sessionID))
		}
	}

	// Delete from registry (non-blocking)
	go func() {
		if err := m.registry.UnregisterSession(sessionID); err != nil {
			m.logger.Error("failed to unregister session",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}()

	// Delete session files (non-blocking)
	go func() {
		if err := session.Delete(); err != nil {
			m.logger.Error("failed to delete session files",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}()

	// Remove from memory immediately
	delete(m.sessions, sessionID)

	m.logger.Info("session deleted",
		zap.String("session_id", sessionID))

	return nil
}

// SendInput sends input to a session
func (m *Manager) SendInput(sessionID, input string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if err := session.SendInput(input); err != nil {
		return fmt.Errorf("failed to send input to session: %w", err)
	}

	// Update session metadata in registry
	if err := m.registry.UpdateSession(session.GetMetadata()); err != nil {
		m.logger.Error("failed to update session metadata",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}

	return nil
}

// GetOutput retrieves output from a session
func (m *Manager) GetOutput(sessionID string) ([]byte, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	return session.GetOutput()
}

// SubscribeToOutput subscribes to real-time output from a session
func (m *Manager) SubscribeToOutput(sessionID string) (<-chan []byte, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	return session.SubscribeToOutput()
}

// PauseSession pauses a session
func (m *Manager) PauseSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if err := session.Pause(); err != nil {
		return fmt.Errorf("failed to pause session: %w", err)
	}

	// Update registry
	if err := m.registry.UpdateSession(session.GetMetadata()); err != nil {
		m.logger.Error("failed to update session in registry",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}

	return nil
}

// ResumeSession resumes a paused session
func (m *Manager) ResumeSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if err := session.Resume(); err != nil {
		return fmt.Errorf("failed to resume session: %w", err)
	}

	// Update registry
	if err := m.registry.UpdateSession(session.GetMetadata()); err != nil {
		m.logger.Error("failed to update session in registry",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}

	return nil
}

// SaveSession manually saves a session
func (m *Manager) SaveSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if err := session.Save(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Update registry
	if err := m.registry.UpdateSession(session.GetMetadata()); err != nil {
		m.logger.Error("failed to update session in registry",
			zap.String("session_id", sessionID),
			zap.Error(err))
	}

	return nil
}

// SaveAllSessions saves all sessions
func (m *Manager) SaveAllSessions() error {
	m.mu.RLock()
	sessions := make([]*claude.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	var lastErr error
	for _, session := range sessions {
		if err := session.Save(); err != nil {
			lastErr = err
			m.logger.Error("failed to save session",
				zap.String("session_id", session.GetID()),
				zap.Error(err))
		}
	}

	return lastErr
}

// loadExistingSessions loads sessions from the registry
func (m *Manager) loadExistingSessions() error {
	sessionMetadataList, err := m.registry.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions from registry: %w", err)
	}

	for _, metadata := range sessionMetadataList {
		// Create session object
		session := claude.NewSession(
			metadata.ID,
			metadata.Name,
			metadata.Config,
			m.executor,
			m.logger,
			m.config.StoragePath,
		)

		// Load session state
		if err := session.Load(); err != nil {
			m.logger.Error("failed to load session",
				zap.String("session_id", metadata.ID),
				zap.Error(err))
			continue
		}

		// Add to sessions map
		m.sessions[metadata.ID] = session

		m.logger.Debug("loaded session",
			zap.String("session_id", metadata.ID),
			zap.String("name", metadata.Name))
	}

	m.logger.Info("loaded existing sessions",
		zap.Int("count", len(m.sessions)))

	return nil
}

// generateSessionID generates a unique session ID
func (m *Manager) generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

// startCleanup starts the automatic cleanup routine
func (m *Manager) startCleanup() {
	m.cleanup = time.NewTicker(m.config.CleanupInterval)

	go func() {
		for {
			select {
			case <-m.cleanup.C:
				m.performCleanup()
			case <-m.ctx.Done():
				return
			}
		}
	}()
}

// performCleanup performs cleanup of terminated sessions
func (m *Manager) performCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var toDelete []string
	for sessionID, session := range m.sessions {
		if session.IsTerminated() {
			// Check if session is old enough to clean up
			metadata := session.GetMetadata()
			if time.Since(metadata.UpdatedAt) > time.Hour {
				toDelete = append(toDelete, sessionID)
			}
		}
	}

	// Delete old terminated sessions
	for _, sessionID := range toDelete {
		if err := m.DeleteSession(sessionID, true); err != nil {
			m.logger.Error("failed to cleanup session",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	if len(toDelete) > 0 {
		m.logger.Info("cleaned up terminated sessions",
			zap.Int("count", len(toDelete)))
	}
}

// GetStats returns manager statistics
func (m *Manager) GetStats() *ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &ManagerStats{
		TotalSessions:      len(m.sessions),
		ActiveSessions:     0,
		PausedSessions:     0,
		TerminatedSessions: 0,
		ErrorSessions:      0,
	}

	for _, session := range m.sessions {
		switch session.GetState() {
		case claude.SessionStateActive:
			stats.ActiveSessions++
		case claude.SessionStatePaused:
			stats.PausedSessions++
		case claude.SessionStateTerminated:
			stats.TerminatedSessions++
		case claude.SessionStateError:
			stats.ErrorSessions++
		}
	}

	return stats
}

// ManagerStats holds statistics about the session manager
type ManagerStats struct {
	TotalSessions      int `json:"total_sessions"`
	ActiveSessions     int `json:"active_sessions"`
	PausedSessions     int `json:"paused_sessions"`
	TerminatedSessions int `json:"terminated_sessions"`
	ErrorSessions      int `json:"error_sessions"`
}

// Close shuts down the session manager
func (m *Manager) Close() error {
	m.logger.Info("shutting down session manager")

	// Stop cleanup routine
	if m.cleanup != nil {
		m.cleanup.Stop()
	}

	// Cancel context
	m.cancel()

	// Save all sessions
	if err := m.SaveAllSessions(); err != nil {
		m.logger.Error("failed to save sessions during shutdown", zap.Error(err))
	}

	// Terminate all active sessions with timeout
	m.mu.RLock()
	sessions := make([]*claude.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	// Use channel to coordinate termination
	terminationDone := make(chan struct{})
	go func() {
		defer close(terminationDone)
		var wg sync.WaitGroup
		
		for _, session := range sessions {
			if session.IsActive() || session.IsPaused() {
				wg.Add(1)
				go func(s *claude.Session) {
					defer wg.Done()
					if err := s.Terminate(); err != nil {
						m.logger.Error("failed to terminate session during shutdown",
							zap.String("session_id", s.GetID()),
							zap.Error(err))
					}
				}(session)
			}
		}
		wg.Wait()
	}()

	// Wait for termination with timeout
	select {
	case <-terminationDone:
		m.logger.Info("all sessions terminated successfully")
	case <-time.After(15 * time.Second):
		m.logger.Warn("session termination timed out during shutdown")
	}

	// Close executor
	if err := m.executor.Close(); err != nil {
		m.logger.Error("failed to close executor", zap.Error(err))
	}

	// Close registry
	if err := m.registry.Close(); err != nil {
		m.logger.Error("failed to close registry", zap.Error(err))
	}

	m.logger.Info("session manager shutdown complete")
	return nil
}
