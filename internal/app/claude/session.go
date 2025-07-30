package claude

import (
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
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	State     SessionState      `json:"state"`
	Config    *SessionConfig    `json:"config"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Tags      []string          `json:"tags"`
	Stats     *SessionStats     `json:"stats,omitempty"`
}

// SessionStats holds statistics about a session
type SessionStats struct {
	InputCount    int           `json:"input_count"`
	OutputBytes   int64         `json:"output_bytes"`
	Duration      time.Duration `json:"duration"`
	LastActive    time.Time     `json:"last_active"`
	ProcessCount  int           `json:"process_count"`
	ErrorCount    int           `json:"error_count"`
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
	
	s.state = SessionStateActive
	s.updatedAt = time.Now()
	s.stats.LastActive = s.updatedAt
	
	s.logger.Info("session started", zap.String("session_id", s.id))
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
		if err := s.process.Kill(); err != nil {
			s.logger.Error("failed to kill process during termination",
				zap.String("session_id", s.id),
				zap.Error(err))
		}
	}
	
	s.state = SessionStateTerminated
	s.updatedAt = time.Now()
	
	s.logger.Info("session terminated", zap.String("session_id", s.id))
	return nil
}

// SendInput sends input to the session
func (s *Session) SendInput(input string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.state != SessionStateActive {
		return fmt.Errorf("cannot send input to session in state: %s", s.state.String())
	}
	
	s.stats.InputCount++
	s.stats.LastActive = time.Now()
	s.updatedAt = s.stats.LastActive
	
	// If there's an active process, send input to it
	if s.process != nil && s.process.IsRunning() {
		return s.process.SendInput(input)
	}
	
	// Otherwise, this would start a new process or handle the input
	// For testing purposes, we'll just record the input
	s.logger.Debug("input received", 
		zap.String("session_id", s.id),
		zap.Int("input_length", len(input)))
	
	return nil
}

// GetOutput returns the accumulated output from the session
func (s *Session) GetOutput() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.process != nil {
		output := s.process.GetOutput()
		s.stats.OutputBytes = int64(len(output))
		return output, nil
	}
	
	return []byte{}, nil
}

// SubscribeToOutput subscribes to real-time output from the session
func (s *Session) SubscribeToOutput() (<-chan []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.process != nil {
		return s.process.SubscribeToOutput(), nil
	}
	
	// Return a closed channel if no process
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
	
	// Terminate if not already terminated
	if s.state != SessionStateTerminated {
		s.Terminate()
	}
	
	// This would delete session files from storage
	s.logger.Info("session deleted", zap.String("session_id", s.id))
	return nil
}