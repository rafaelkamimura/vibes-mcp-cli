package claude

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionState represents the current state of a session
type SessionState int

const (
	SessionStateCreated SessionState = iota
	SessionStateActive
	SessionStatePaused
	SessionStateTerminated
	SessionStateError
)

// String returns the string representation of the session state
func (ss SessionState) String() string {
	switch ss {
	case SessionStateCreated:
		return "created"
	case SessionStateActive:
		return "active"
	case SessionStatePaused:
		return "paused"
	case SessionStateTerminated:
		return "terminated"
	case SessionStateError:
		return "error"
	default:
		return "unknown"
	}
}

// SessionConfig holds configuration for a session
type SessionConfig struct {
	Name        string            `json:"name"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
	Args        []string          `json:"args"`
	AutoSave    bool              `json:"auto_save"`
	MaxHistory  int               `json:"max_history"`
}

// DefaultSessionConfig returns default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		Environment: make(map[string]string),
		Args:        []string{},
		AutoSave:    true,
		MaxHistory:  1000,
	}
}

// SessionMetadata holds metadata about a session
type SessionMetadata struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	State     SessionState   `json:"state"`
	Config    *SessionConfig `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Tags      []string       `json:"tags"`
	Stats     *SessionStats  `json:"stats,omitempty"`
}

// SessionStats holds statistics about a session
type SessionStats struct {
	InputCount   int           `json:"input_count"`
	OutputBytes  int64         `json:"output_bytes"`
	Duration     time.Duration `json:"duration"`
	LastActive   time.Time     `json:"last_active"`
	ProcessCount int           `json:"process_count"`
	ErrorCount   int           `json:"error_count"`
}

// Session represents a Claude Code session
type Session struct {
	id          string
	name        string
	config      *SessionConfig
	executor    *Executor
	logger      *zap.Logger
	storagePath string

	mu       sync.RWMutex
	state    SessionState
	process  *Process
	metadata *SessionMetadata
	stats    *SessionStats

	createdAt time.Time
	updatedAt time.Time
}

// NewSession creates a new session
func NewSession(id, name string, config *SessionConfig, executor *Executor, logger *zap.Logger, storagePath string) *Session {
	if config == nil {
		config = DefaultSessionConfig()
	}
	if config.Name == "" {
		config.Name = name
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	now := time.Now()

	session := &Session{
		id:          id,
		name:        name,
		config:      config,
		executor:    executor,
		logger:      logger,
		storagePath: storagePath,
		state:       SessionStateCreated,
		createdAt:   now,
		updatedAt:   now,
		stats: &SessionStats{
			LastActive: now,
		},
	}

	session.metadata = &SessionMetadata{
		ID:        id,
		Name:      name,
		State:     SessionStateCreated,
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
		Tags:      []string{},
		Stats:     session.stats,
	}

	return session
}

// GetID returns the session ID
func (s *Session) GetID() string {
	return s.id
}

// GetName returns the session name
func (s *Session) GetName() string {
	return s.name
}

// GetState returns the current session state
func (s *Session) GetState() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetMetadata returns the session metadata
func (s *Session) GetMetadata() *SessionMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy
	metadata := *s.metadata
	metadata.State = s.state
	metadata.UpdatedAt = s.updatedAt

	// Copy stats if present
	if s.stats != nil {
		stats := *s.stats
		metadata.Stats = &stats
	}

	return &metadata
}

// IsActive returns true if the session is active
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == SessionStateActive
}

// IsPaused returns true if the session is paused
func (s *Session) IsPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == SessionStatePaused
}

// IsTerminated returns true if the session is terminated
func (s *Session) IsTerminated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == SessionStateTerminated
}

// Start starts the session
func (s *Session) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateCreated && s.state != SessionStatePaused {
		return fmt.Errorf("cannot start session in state: %s", s.state.String())
	}

	// Create command options for Claude CLI
	opts := &CommandOptions{
		WorkingDir:  s.config.WorkingDir,
		Environment: s.config.Environment,
		Args:        s.config.Args,
		Timeout:     time.Hour * 2, // Long timeout for interactive sessions
	}

	// Start Claude CLI process asynchronously
	ctx := context.Background()
	process, err := s.executor.ExecuteAsync(ctx, opts)
	if err != nil {
		s.state = SessionStateError
		return fmt.Errorf("failed to start Claude CLI process: %w", err)
	}

	// Store the process reference
	s.process = process
	s.state = SessionStateActive
	s.updatedAt = time.Now()
	s.stats.LastActive = s.updatedAt
	s.stats.ProcessCount++

	// Wait a moment for the process to fully start before logging PID
	pid := process.GetPID()
	if pid == -1 {
		// Process hasn't started yet, give it a moment
		time.Sleep(100 * time.Millisecond)
		pid = process.GetPID()
	}

	s.logger.Info("session started with Claude CLI process",
		zap.String("session_id", s.id),
		zap.String("process_id", process.ID),
		zap.Int("pid", pid))

	return nil
}

// Pause pauses the session
func (s *Session) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateActive {
		return fmt.Errorf("cannot pause session in state: %s", s.state.String())
	}

	s.state = SessionStatePaused
	s.updatedAt = time.Now()

	s.logger.Info("session paused", zap.String("session_id", s.id))
	return nil
}

// Resume resumes a paused session
func (s *Session) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStatePaused {
		return fmt.Errorf("cannot resume session in state: %s", s.state.String())
	}

	s.state = SessionStateActive
	s.updatedAt = time.Now()
	s.stats.LastActive = s.updatedAt

	s.logger.Info("session resumed", zap.String("session_id", s.id))
	return nil
}

// Terminate terminates the session
func (s *Session) Terminate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == SessionStateTerminated {
		return nil // Already terminated
	}

	// Kill any running process
	if s.process != nil && s.process.IsRunning() {
		s.logger.Info("terminating Claude CLI process",
			zap.String("session_id", s.id),
			zap.String("process_id", s.process.ID),
			zap.Int("pid", s.process.GetPID()))

		if err := s.process.Kill(); err != nil {
			s.logger.Error("failed to kill Claude CLI process during termination",
				zap.String("session_id", s.id),
				zap.String("process_id", s.process.ID),
				zap.Error(err))
			s.stats.ErrorCount++
		} else {
			s.logger.Info("Claude CLI process terminated successfully",
				zap.String("session_id", s.id),
				zap.String("process_id", s.process.ID))
		}
	}

	// Clean up process reference
	if s.process != nil {
		s.process.Close()
		s.process = nil
	}

	s.state = SessionStateTerminated
	s.updatedAt = time.Now()

	s.logger.Info("session terminated", zap.String("session_id", s.id))
	return nil
}

// SendInput sends input to the session
func (s *Session) SendInput(input string) error {
	// First check session state without holding write lock
	s.mu.RLock()
	if s.state != SessionStateActive {
		s.mu.RUnlock()
		return fmt.Errorf("cannot send input to session in state: %s", s.state.String())
	}
	s.mu.RUnlock()

	// Try to send input to existing process
	s.mu.RLock()
	process := s.process
	needsNewProcess := process == nil || process.IsFinished()
	s.mu.RUnlock()

	// If we need a new process, start one first
	if needsNewProcess {
		s.logger.Info("no active Claude CLI process, starting new one",
			zap.String("session_id", s.id))

		if err := s.Start(); err != nil {
			s.mu.Lock()
			s.stats.ErrorCount++
			s.mu.Unlock()
			return fmt.Errorf("failed to start Claude CLI process for input: %w", err)
		}

		// Update process reference
		s.mu.RLock()
		process = s.process
		s.mu.RUnlock()
	}

	// Now send input to the process
	if process != nil && process.IsRunning() {
		err := process.SendInput(input)

		// Update stats regardless of success/failure
		s.mu.Lock()
		s.stats.InputCount++
		s.stats.LastActive = time.Now()
		s.updatedAt = s.stats.LastActive

		if err != nil {
			s.logger.Error("failed to send input to Claude CLI process",
				zap.String("session_id", s.id),
				zap.String("process_id", process.ID),
				zap.Error(err))
			s.stats.ErrorCount++
			s.mu.Unlock()
			return fmt.Errorf("failed to send input to Claude CLI: %w", err)
		}

		s.logger.Debug("input sent to Claude CLI process",
			zap.String("session_id", s.id),
			zap.String("process_id", process.ID),
			zap.Int("input_length", len(input)))
		s.mu.Unlock()

		return nil
	}

	// No active process available
	s.mu.Lock()
	s.stats.ErrorCount++
	s.mu.Unlock()
	return fmt.Errorf("no active Claude CLI process available")
}

// GetOutput returns the accumulated output from the session
func (s *Session) GetOutput() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.process != nil {
		output := s.process.GetOutput()
		s.stats.OutputBytes = int64(len(output))
		s.logger.Debug("retrieved output from Claude CLI process",
			zap.String("session_id", s.id),
			zap.String("process_id", s.process.ID),
			zap.Int("output_bytes", len(output)))
		return output, nil
	}

	s.logger.Debug("no Claude CLI process available for output",
		zap.String("session_id", s.id))
	return []byte{}, nil
}

// SubscribeToOutput subscribes to real-time output from the session
func (s *Session) SubscribeToOutput() (<-chan []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.process != nil {
		subscription := s.process.SubscribeToOutput()
		s.logger.Debug("subscribed to Claude CLI process output",
			zap.String("session_id", s.id),
			zap.String("process_id", s.process.ID))
		return subscription, nil
	}

	// Return a closed channel if no process
	s.logger.Debug("no Claude CLI process available for output subscription",
		zap.String("session_id", s.id))
	ch := make(chan []byte)
	close(ch)
	return ch, nil
}

// Save saves the session state
func (s *Session) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// This would save session state to storage
	s.logger.Debug("session saved", zap.String("session_id", s.id))
	return nil
}

// Load loads the session state
func (s *Session) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// This would load session state from storage
	s.logger.Debug("session loaded", zap.String("session_id", s.id))
	return nil
}

// Delete deletes the session and its data
func (s *Session) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Terminate if not already terminated (do it without recursive locking)
	if s.state != SessionStateTerminated {
		s.terminateUnsafe()
	}

	// This would delete session files from storage
	s.logger.Info("session deleted", zap.String("session_id", s.id))
	return nil
}

// terminateUnsafe terminates the session without acquiring mutex (internal use only)
func (s *Session) terminateUnsafe() {
	if s.state == SessionStateTerminated {
		return // Already terminated
	}

	// Kill any running process
	if s.process != nil && s.process.IsRunning() {
		s.logger.Info("terminating Claude CLI process",
			zap.String("session_id", s.id),
			zap.String("process_id", s.process.ID),
			zap.Int("pid", s.process.GetPID()))

		if err := s.process.Kill(); err != nil {
			s.logger.Error("failed to kill Claude CLI process during termination",
				zap.String("session_id", s.id),
				zap.String("process_id", s.process.ID),
				zap.Error(err))
			s.stats.ErrorCount++
		} else {
			s.logger.Info("Claude CLI process terminated successfully",
				zap.String("session_id", s.id),
				zap.String("process_id", s.process.ID))
		}
	}

	// Clean up process reference
	if s.process != nil {
		s.process.Close()
		s.process = nil
	}

	s.state = SessionStateTerminated
	s.updatedAt = time.Now()

	s.logger.Info("session terminated", zap.String("session_id", s.id))
}
