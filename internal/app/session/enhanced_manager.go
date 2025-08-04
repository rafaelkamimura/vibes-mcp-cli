package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
)

// EnhancedManagerConfig holds configuration for the enhanced session manager
type EnhancedManagerConfig struct {
	*ManagerConfig              // Embed base manager config
	*PersistenceConfig          // Persistence configuration
	MaxHistoryEntries    int    `json:"max_history_entries"`     // Max history entries per session
	EnableInteractive    bool   `json:"enable_interactive"`      // Enable interactive controller
	EnableMetadataTracking bool `json:"enable_metadata_tracking"` // Enable detailed metadata tracking
	EnableSearching      bool   `json:"enable_searching"`        // Enable search capabilities
	EnablePersistence    bool   `json:"enable_persistence"`      // Enable session persistence
	TelemetryEnabled     bool   `json:"telemetry_enabled"`       // Enable telemetry integration
}

// DefaultEnhancedManagerConfig returns default enhanced manager configuration
func DefaultEnhancedManagerConfig() *EnhancedManagerConfig {
	return &EnhancedManagerConfig{
		ManagerConfig:          DefaultManagerConfig(),
		PersistenceConfig:      DefaultPersistenceConfig(),
		MaxHistoryEntries:      10000,
		EnableInteractive:      true,
		EnableMetadataTracking: true,
		EnableSearching:        true,
		EnablePersistence:      true,
		TelemetryEnabled:       true,
	}
}

// EnhancedManager provides comprehensive session management with all enhanced features
type EnhancedManager struct {
	// Core components
	*Manager                    // Embed base manager
	config          *EnhancedManagerConfig
	logger          *zap.Logger
	
	// Enhanced components
	metadataTracker *MetadataTracker
	historyManager  *HistoryManager
	interactiveCtrl *InteractiveController
	searcher        *SessionSearcher
	persistence     *PersistenceManager
	
	// State management
	mu              sync.RWMutex
	isInitialized   bool
	shutdownChan    chan struct{}
}

// NewEnhancedManager creates a new enhanced session manager
func NewEnhancedManager(config *EnhancedManagerConfig, logger *zap.Logger) (*EnhancedManager, error) {
	if config == nil {
		config = DefaultEnhancedManagerConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create base manager
	baseManager, err := NewManager(config.ManagerConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base manager: %w", err)
	}

	enhanced := &EnhancedManager{
		Manager:      baseManager,
		config:       config,
		logger:       logger,
		shutdownChan: make(chan struct{}),
	}

	// Initialize enhanced components
	if err := enhanced.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize enhanced components: %w", err)
	}

	enhanced.isInitialized = true
	
	logger.Info("enhanced session manager created",
		zap.Bool("interactive_enabled", config.EnableInteractive),
		zap.Bool("metadata_tracking_enabled", config.EnableMetadataTracking),
		zap.Bool("searching_enabled", config.EnableSearching),
		zap.Bool("persistence_enabled", config.EnablePersistence))

	return enhanced, nil
}

// initializeComponents initializes all enhanced components
func (em *EnhancedManager) initializeComponents() error {
	var err error

	// Initialize metadata tracker
	if em.config.EnableMetadataTracking {
		em.metadataTracker = NewMetadataTracker()
		em.logger.Debug("metadata tracker initialized")
	}

	// Initialize history manager
	em.historyManager, err = NewHistoryManager(
		em.config.PersistenceConfig.StoragePath,
		em.config.MaxHistoryEntries,
		em.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create history manager: %w", err)
	}
	em.logger.Debug("history manager initialized")

	// Initialize interactive controller
	if em.config.EnableInteractive {
		em.interactiveCtrl = NewInteractiveController(
			em.Manager,
			em.historyManager,
			em.logger,
		)
		em.logger.Debug("interactive controller initialized")
	}

	// Initialize searcher
	if em.config.EnableSearching {
		em.searcher = NewSessionSearcher(
			em.metadataTracker,
			em.historyManager,
			em.registry,
		)
		em.logger.Debug("session searcher initialized")
	}

	// Initialize persistence manager
	if em.config.EnablePersistence {
		em.persistence, err = NewPersistenceManager(
			em.config.PersistenceConfig,
			em.metadataTracker,
			em.historyManager,
			em.registry,
			em.logger,
		)
		if err != nil {
			return fmt.Errorf("failed to create persistence manager: %w", err)
		}
		em.logger.Debug("persistence manager initialized")
	}

	return nil
}

// CreateEnhancedSession creates a new session with enhanced capabilities
func (em *EnhancedManager) CreateEnhancedSession(name, description string, config *claude.SessionConfig, tags []string, priority SessionPriority) (*claude.Session, error) {
	// Create base session
	session, err := em.Manager.CreateSession(name, config)
	if err != nil {
		return nil, err
	}

	sessionID := session.GetID()

	// Track metadata if enabled
	if em.metadataTracker != nil {
		metadata := em.metadataTracker.TrackSessionCreated(sessionID, name, config)
		metadata.Description = description
		metadata.Priority = priority
		if tags != nil {
			metadata.Tags = make([]string, len(tags))
			copy(metadata.Tags, tags)
		}
	}

	// Initialize conversation history
	if em.historyManager != nil {
		em.historyManager.GetHistory(sessionID) // This creates it if it doesn't exist
	}

	// Mark for persistence if enabled
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	em.logger.Info("enhanced session created",
		zap.String("session_id", sessionID),
		zap.String("name", name),
		zap.String("description", description),
		zap.Strings("tags", tags),
		zap.String("priority", string(priority)))

	return session, nil
}

// StartEnhancedSession starts a session with enhanced tracking
func (em *EnhancedManager) StartEnhancedSession(sessionID string) error {
	if err := em.Manager.StartSession(sessionID); err != nil {
		return err
	}

	// Track session start
	if em.metadataTracker != nil {
		if err := em.metadataTracker.TrackSessionStarted(sessionID); err != nil {
			em.logger.Error("failed to track session start",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Add system message to history
	if em.historyManager != nil {
		history := em.historyManager.GetHistory(sessionID)
		history.AddSystemEntry(
			"Session started",
			map[string]interface{}{
				"event": "session_started",
				"timestamp": time.Now(),
			},
		)
	}

	// Mark for persistence
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	return nil
}

// SendInteractiveInput sends input with interactive response handling
func (em *EnhancedManager) SendInteractiveInput(ctx context.Context, sessionID, input string, options *InteractionOptions) (*InteractionContext, error) {
	if em.interactiveCtrl == nil {
		return nil, fmt.Errorf("interactive controller not enabled")
	}

	// Track input in metadata
	if em.metadataTracker != nil {
		metadata := make(map[string]interface{})
		if options != nil && options.Metadata != nil {
			for k, v := range options.Metadata {
				metadata[k] = v
			}
		}
		
		if err := em.metadataTracker.TrackInput(sessionID, input, metadata); err != nil {
			em.logger.Error("failed to track input",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Send interactive input
	interactionCtx, err := em.interactiveCtrl.SendInteractiveInput(ctx, sessionID, input, options)
	if err != nil {
		// Track error
		if em.metadataTracker != nil {
			em.metadataTracker.TrackError(sessionID, err, map[string]interface{}{
				"operation": "send_interactive_input",
			})
		}
		return nil, err
	}

	// Mark for persistence
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	return interactionCtx, nil
}

// ExecuteCommand executes a command with full tracking
func (em *EnhancedManager) ExecuteCommand(ctx context.Context, sessionID, command string, options *InteractionOptions) (*InteractionResponse, error) {
	if em.interactiveCtrl == nil {
		return nil, fmt.Errorf("interactive controller not enabled")
	}

	startTime := time.Now()

	// Track command execution start
	if em.metadataTracker != nil {
		metadata := map[string]interface{}{
			"command": command,
			"start_time": startTime,
		}
		
		if err := em.metadataTracker.TrackInput(sessionID, command, metadata); err != nil {
			em.logger.Error("failed to track command",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Execute command
	response, err := em.interactiveCtrl.ExecuteCommand(ctx, sessionID, command, options)
	
	duration := time.Since(startTime)

	if err != nil {
		// Track error
		if em.metadataTracker != nil {
			em.metadataTracker.TrackError(sessionID, err, map[string]interface{}{
				"operation": "execute_command",
				"command": command,
				"duration": duration,
			})
		}
		return nil, err
	}

	// Track output
	if em.metadataTracker != nil && response != nil {
		metadata := map[string]interface{}{
			"command": command,
			"response_type": string(response.Type),
			"duration": duration,
		}
		
		if err := em.metadataTracker.TrackOutput(sessionID, response.Content, duration, metadata); err != nil {
			em.logger.Error("failed to track output",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Mark for persistence
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	return response, nil
}

// SearchSessions searches for sessions using enhanced search capabilities
func (em *EnhancedManager) SearchSessions(criteria *SearchCriteria, options *SearchOptions) (*SearchResult, error) {
	if em.searcher == nil {
		return nil, fmt.Errorf("session searcher not enabled")
	}

	return em.searcher.Search(criteria, options)
}

// QuickSearchSessions provides quick search functionality
func (em *EnhancedManager) QuickSearchSessions(query string, limit int) (*SearchResult, error) {
	if em.searcher == nil {
		return nil, fmt.Errorf("session searcher not enabled")
	}

	return em.searcher.QuickSearch(query, limit)
}

// GetSessionMetadata returns enhanced metadata for a session
func (em *EnhancedManager) GetSessionMetadata(sessionID string) (*EnhancedSessionMetadata, error) {
	if em.metadataTracker == nil {
		return nil, fmt.Errorf("metadata tracker not enabled")
	}

	return em.metadataTracker.GetMetadata(sessionID)
}

// GetSessionHistory returns conversation history for a session
func (em *EnhancedManager) GetSessionHistory(sessionID string, limit int) ([]*ConversationEntry, error) {
	if em.historyManager == nil {
		return nil, fmt.Errorf("history manager not available")
	}

	history := em.historyManager.GetHistory(sessionID)
	entries := history.GetEntries()

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// GetSessionStats returns comprehensive session statistics
func (em *EnhancedManager) GetSessionStats(sessionID string) (*ConversationStats, error) {
	if em.historyManager == nil {
		return nil, fmt.Errorf("history manager not available")
	}

	history := em.historyManager.GetHistory(sessionID)
	return history.GetStats(), nil
}

// UpdateSessionTags updates session tags
func (em *EnhancedManager) UpdateSessionTags(sessionID string, tags []string) error {
	if em.metadataTracker == nil {
		return fmt.Errorf("metadata tracker not enabled")
	}

	if err := em.metadataTracker.UpdateTags(sessionID, tags); err != nil {
		return err
	}

	// Mark for persistence
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	return nil
}

// UpdateSessionLabels updates session labels
func (em *EnhancedManager) UpdateSessionLabels(sessionID string, labels map[string]string) error {
	if em.metadataTracker == nil {
		return fmt.Errorf("metadata tracker not enabled")
	}

	if err := em.metadataTracker.UpdateLabels(sessionID, labels); err != nil {
		return err
	}

	// Mark for persistence
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	return nil
}

// UpdateSessionCustomData updates custom session metadata
func (em *EnhancedManager) UpdateSessionCustomData(sessionID string, customData map[string]interface{}) error {
	if em.metadataTracker == nil {
		return fmt.Errorf("metadata tracker not enabled")
	}

	if err := em.metadataTracker.UpdateCustomData(sessionID, customData); err != nil {
		return err
	}

	// Mark for persistence
	if em.persistence != nil {
		em.persistence.MarkForSave(sessionID)
	}

	return nil
}

// SaveSession manually saves a session
func (em *EnhancedManager) SaveSession(sessionID string) error {
	if em.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	return em.persistence.SaveSession(sessionID, true)
}

// SaveAllSessions saves all sessions
func (em *EnhancedManager) SaveAllSessions() error {
	if em.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	return em.persistence.SavePendingSessions()
}

// LoadSession loads a session from storage
func (em *EnhancedManager) LoadSession(sessionID string) (*SessionSnapshot, error) {
	if em.persistence == nil {
		return nil, fmt.Errorf("persistence not enabled")
	}

	return em.persistence.LoadSession(sessionID)
}

// RestoreSession restores a session from a snapshot
func (em *EnhancedManager) RestoreSession(sessionID string, snapshot *SessionSnapshot) error {
	if em.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	return em.persistence.RestoreSession(sessionID, snapshot)
}

// CreateBackup creates a backup of all sessions
func (em *EnhancedManager) CreateBackup() error {
	if em.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	return em.persistence.CreateBackup()
}

// TerminateEnhancedSession terminates a session with enhanced cleanup
func (em *EnhancedManager) TerminateEnhancedSession(sessionID string) error {
	// Terminate base session
	if err := em.Manager.TerminateSession(sessionID); err != nil {
		return err
	}

	// Track termination
	if em.metadataTracker != nil {
		if err := em.metadataTracker.TrackSessionTerminated(sessionID); err != nil {
			em.logger.Error("failed to track session termination",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Add system message to history
	if em.historyManager != nil {
		history := em.historyManager.GetHistory(sessionID)
		history.AddSystemEntry(
			"Session terminated",
			map[string]interface{}{
				"event": "session_terminated",
				"timestamp": time.Now(),
			},
		)
	}

	// Save final state
	if em.persistence != nil {
		if err := em.persistence.SaveSession(sessionID, true); err != nil {
			em.logger.Error("failed to save session on termination",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	return nil
}

// DeleteEnhancedSession deletes a session with complete cleanup
func (em *EnhancedManager) DeleteEnhancedSession(sessionID string, force bool) error {
	// Delete base session
	if err := em.Manager.DeleteSession(sessionID, force); err != nil {
		return err
	}

	// Clean up metadata
	if em.metadataTracker != nil {
		if err := em.metadataTracker.RemoveMetadata(sessionID); err != nil {
			em.logger.Error("failed to remove session metadata",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Clean up history
	if em.historyManager != nil {
		if err := em.historyManager.DeleteHistory(sessionID); err != nil {
			em.logger.Error("failed to delete session history",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	// Clean up persistence
	if em.persistence != nil {
		if err := em.persistence.DeleteSession(sessionID); err != nil {
			em.logger.Error("failed to delete session from storage",
				zap.String("session_id", sessionID),
				zap.Error(err))
		}
	}

	em.logger.Info("enhanced session deleted",
		zap.String("session_id", sessionID))

	return nil
}

// GetEnhancedStats returns comprehensive manager statistics
func (em *EnhancedManager) GetEnhancedStats() *EnhancedManagerStats {
	baseStats := em.Manager.GetStats()
	
	stats := &EnhancedManagerStats{
		ManagerStats: baseStats,
		Initialized:  em.isInitialized,
	}

	// Add component stats
	if em.historyManager != nil {
		stats.HistoryStats = em.historyManager.GetStats()
	}
	
	if em.persistence != nil {
		stats.PersistenceStats = em.persistence.GetStats()
	}

	// Add interactive stats
	if em.interactiveCtrl != nil {
		stats.ActiveRequests = len(em.interactiveCtrl.GetActiveRequests())
	}

	return stats
}

// EnhancedManagerStats holds comprehensive manager statistics
type EnhancedManagerStats struct {
	*ManagerStats                          // Base manager stats
	Initialized      bool                  `json:"initialized"`
	HistoryStats     *HistoryManagerStats  `json:"history_stats,omitempty"`
	PersistenceStats *PersistenceStats     `json:"persistence_stats,omitempty"`
	ActiveRequests   int                   `json:"active_requests"`
}

// CleanupExpiredRequests cleans up expired interaction requests
func (em *EnhancedManager) CleanupExpiredRequests() int {
	if em.interactiveCtrl == nil {
		return 0
	}

	return em.interactiveCtrl.CleanupExpiredRequests()
}

// CleanupOldSessions cleans up old session data
func (em *EnhancedManager) CleanupOldSessions(maxAge time.Duration) (int, error) {
	var totalCleaned int

	// Clean up old metadata
	if em.metadataTracker != nil {
		cleaned := em.metadataTracker.CleanupOldMetadata(maxAge)
		totalCleaned += cleaned
	}

	// Clean up old histories
	if em.historyManager != nil {
		cleaned, err := em.historyManager.CleanupOldHistories(maxAge)
		if err != nil {
			return totalCleaned, fmt.Errorf("failed to cleanup old histories: %w", err)
		}
		totalCleaned += cleaned
	}

	em.logger.Info("cleaned up old session data",
		zap.Int("total_cleaned", totalCleaned),
		zap.Duration("max_age", maxAge))

	return totalCleaned, nil
}

// Close shuts down the enhanced manager
func (em *EnhancedManager) Close() error {
	em.logger.Info("shutting down enhanced session manager")

	// Signal shutdown
	close(em.shutdownChan)

	// Close interactive controller
	if em.interactiveCtrl != nil {
		if err := em.interactiveCtrl.Close(); err != nil {
			em.logger.Error("failed to close interactive controller", zap.Error(err))
		}
	}

	// Close persistence manager
	if em.persistence != nil {
		if err := em.persistence.Close(); err != nil {
			em.logger.Error("failed to close persistence manager", zap.Error(err))
		}
	}

	// Close history manager
	if em.historyManager != nil {
		if err := em.historyManager.Close(); err != nil {
			em.logger.Error("failed to close history manager", zap.Error(err))
		}
	}

	// Close base manager
	if err := em.Manager.Close(); err != nil {
		em.logger.Error("failed to close base manager", zap.Error(err))
	}

	em.logger.Info("enhanced session manager shutdown complete")
	return nil
}