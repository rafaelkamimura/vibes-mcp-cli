package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
)

// RegistryConfig holds configuration for the session registry
type RegistryConfig struct {
	StoragePath   string        `json:"storage_path"`   // Path to registry storage
	BackupEnabled bool          `json:"backup_enabled"` // Enable registry backups
	BackupCount   int           `json:"backup_count"`   // Number of backups to keep
	SyncInterval  time.Duration `json:"sync_interval"`  // Registry sync interval
}

// DefaultRegistryConfig returns default registry configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		StoragePath:   "./sessions",
		BackupEnabled: true,
		BackupCount:   5,
		SyncInterval:  time.Minute * 5,
	}
}

// Registry manages session metadata and provides persistent storage
type Registry struct {
	config       *RegistryConfig
	logger       *zap.Logger
	mu           sync.RWMutex
	sessions     map[string]*claude.SessionMetadata // Session ID -> Metadata
	indexByName  map[string][]string                // Name -> Session IDs
	indexByTags  map[string][]string                // Tag -> Session IDs
	indexByState map[claude.SessionState][]string   // State -> Session IDs
	registryFile string                             // Registry file path
	lastSync     time.Time                          // Last sync time
	modified     bool                               // Registry modified flag
}

// NewRegistry creates a new session registry
func NewRegistry(storagePath string, logger *zap.Logger) (*Registry, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	config := DefaultRegistryConfig()
	config.StoragePath = storagePath

	registry := &Registry{
		config:       config,
		logger:       logger,
		sessions:     make(map[string]*claude.SessionMetadata),
		indexByName:  make(map[string][]string),
		indexByTags:  make(map[string][]string),
		indexByState: make(map[claude.SessionState][]string),
		registryFile: filepath.Join(storagePath, "registry.json"),
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Load existing registry
	if err := registry.load(); err != nil {
		logger.Info("creating new registry (load failed)", zap.Error(err))
		// Create empty registry if load fails
		if err := registry.save(); err != nil {
			return nil, fmt.Errorf("failed to create new registry: %w", err)
		}
	}

	return registry, nil
}

// RegisterSession registers a new session in the registry
func (r *Registry) RegisterSession(metadata *claude.SessionMetadata) error {
	if metadata == nil {
		return fmt.Errorf("session metadata is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if session already exists
	if _, exists := r.sessions[metadata.ID]; exists {
		return fmt.Errorf("session already registered: %s", metadata.ID)
	}

	// Create a copy of metadata
	sessionMeta := *metadata
	r.sessions[metadata.ID] = &sessionMeta

	// Update indices
	r.updateIndices(metadata.ID, &sessionMeta)

	r.modified = true

	r.logger.Debug("session registered",
		zap.String("session_id", metadata.ID),
		zap.String("name", metadata.Name))

	return r.save()
}

// UpdateSession updates session metadata in the registry
func (r *Registry) UpdateSession(metadata *claude.SessionMetadata) error {
	if metadata == nil {
		return fmt.Errorf("session metadata is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if session exists
	existing, exists := r.sessions[metadata.ID]
	if !exists {
		return fmt.Errorf("session not found: %s", metadata.ID)
	}

	// Remove from old indices
	r.removeFromIndices(metadata.ID, existing)

	// Update metadata
	sessionMeta := *metadata
	sessionMeta.UpdatedAt = time.Now()
	r.sessions[metadata.ID] = &sessionMeta

	// Update indices with new metadata
	r.updateIndices(metadata.ID, &sessionMeta)

	r.modified = true

	r.logger.Debug("session updated",
		zap.String("session_id", metadata.ID),
		zap.String("name", metadata.Name))

	return r.save()
}

// UnregisterSession removes a session from the registry
func (r *Registry) UnregisterSession(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata, exists := r.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Remove from indices
	r.removeFromIndices(sessionID, metadata)

	// Remove from sessions map
	delete(r.sessions, sessionID)

	r.modified = true

	r.logger.Debug("session unregistered",
		zap.String("session_id", sessionID))

	return r.save()
}

// GetSession retrieves session metadata by ID
func (r *Registry) GetSession(sessionID string) (*claude.SessionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Return a copy
	sessionMeta := *metadata
	return &sessionMeta, nil
}

// ListSessions returns all session metadata
func (r *Registry) ListSessions() ([]*claude.SessionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*claude.SessionMetadata, 0, len(r.sessions))
	for _, metadata := range r.sessions {
		// Return copies
		sessionMeta := *metadata
		sessions = append(sessions, &sessionMeta)
	}

	// Sort by creation time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// FindSessionsByName finds sessions by name (partial match)
func (r *Registry) FindSessionsByName(namePattern string) ([]*claude.SessionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*claude.SessionMetadata
	pattern := strings.ToLower(namePattern)

	for _, metadata := range r.sessions {
		if strings.Contains(strings.ToLower(metadata.Name), pattern) {
			sessionMeta := *metadata
			matches = append(matches, &sessionMeta)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})

	return matches, nil
}

// FindSessionsByTag finds sessions by tag
func (r *Registry) FindSessionsByTag(tag string) ([]*claude.SessionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionIDs, exists := r.indexByTags[tag]
	if !exists {
		return []*claude.SessionMetadata{}, nil
	}

	sessions := make([]*claude.SessionMetadata, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if metadata, exists := r.sessions[sessionID]; exists {
			sessionMeta := *metadata
			sessions = append(sessions, &sessionMeta)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// FindSessionsByState finds sessions by state
func (r *Registry) FindSessionsByState(state claude.SessionState) ([]*claude.SessionMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionIDs, exists := r.indexByState[state]
	if !exists {
		return []*claude.SessionMetadata{}, nil
	}

	sessions := make([]*claude.SessionMetadata, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if metadata, exists := r.sessions[sessionID]; exists {
			sessionMeta := *metadata
			sessions = append(sessions, &sessionMeta)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// GetStats returns registry statistics
func (r *Registry) GetStats() *RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &RegistryStats{
		TotalSessions: len(r.sessions),
		StateCount:    make(map[string]int),
		TagCount:      make(map[string]int),
		LastSync:      r.lastSync,
	}

	// Count by state
	for state, sessionIDs := range r.indexByState {
		stats.StateCount[state.String()] = len(sessionIDs)
	}

	// Count by tag
	for tag, sessionIDs := range r.indexByTags {
		stats.TagCount[tag] = len(sessionIDs)
	}

	// Find oldest and newest
	for _, metadata := range r.sessions {
		if stats.OldestSession.IsZero() || metadata.CreatedAt.Before(stats.OldestSession) {
			stats.OldestSession = metadata.CreatedAt
		}
		if metadata.CreatedAt.After(stats.NewestSession) {
			stats.NewestSession = metadata.CreatedAt
		}
	}

	return stats
}

// RegistryStats holds statistics about the registry
type RegistryStats struct {
	TotalSessions int            `json:"total_sessions"`
	StateCount    map[string]int `json:"state_count"`
	TagCount      map[string]int `json:"tag_count"`
	OldestSession time.Time      `json:"oldest_session"`
	NewestSession time.Time      `json:"newest_session"`
	LastSync      time.Time      `json:"last_sync"`
}

// updateIndices updates all indices for a session
func (r *Registry) updateIndices(sessionID string, metadata *claude.SessionMetadata) {
	// Update name index
	if metadata.Name != "" {
		r.indexByName[metadata.Name] = append(r.indexByName[metadata.Name], sessionID)
	}

	// Update tag index
	for _, tag := range metadata.Tags {
		if tag != "" {
			r.indexByTags[tag] = append(r.indexByTags[tag], sessionID)
		}
	}

	// Update state index
	r.indexByState[metadata.State] = append(r.indexByState[metadata.State], sessionID)
}

// removeFromIndices removes a session from all indices
func (r *Registry) removeFromIndices(sessionID string, metadata *claude.SessionMetadata) {
	// Remove from name index
	if metadata.Name != "" {
		r.indexByName[metadata.Name] = r.removeFromSlice(r.indexByName[metadata.Name], sessionID)
		if len(r.indexByName[metadata.Name]) == 0 {
			delete(r.indexByName, metadata.Name)
		}
	}

	// Remove from tag index
	for _, tag := range metadata.Tags {
		if tag != "" {
			r.indexByTags[tag] = r.removeFromSlice(r.indexByTags[tag], sessionID)
			if len(r.indexByTags[tag]) == 0 {
				delete(r.indexByTags, tag)
			}
		}
	}

	// Remove from state index
	r.indexByState[metadata.State] = r.removeFromSlice(r.indexByState[metadata.State], sessionID)
	if len(r.indexByState[metadata.State]) == 0 {
		delete(r.indexByState, metadata.State)
	}
}

// removeFromSlice removes an item from a string slice
func (r *Registry) removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// save saves the registry to disk
func (r *Registry) save() error {
	if !r.modified {
		return nil
	}

	// Create registry data structure
	registryData := struct {
		Version   string                             `json:"version"`
		Timestamp time.Time                          `json:"timestamp"`
		Sessions  map[string]*claude.SessionMetadata `json:"sessions"`
	}{
		Version:   "1.0",
		Timestamp: time.Now(),
		Sessions:  r.sessions,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(registryData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry data: %w", err)
	}

	// Create backup if enabled
	if r.config.BackupEnabled {
		if err := r.createBackup(); err != nil {
			r.logger.Warn("failed to create registry backup", zap.Error(err))
		}
	}

	// Write to file
	if err := os.WriteFile(r.registryFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	r.modified = false
	r.lastSync = time.Now()

	r.logger.Debug("registry saved",
		zap.String("file", r.registryFile),
		zap.Int("sessions", len(r.sessions)))

	return nil
}

// load loads the registry from disk
func (r *Registry) load() error {
	data, err := os.ReadFile(r.registryFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("registry file does not exist")
		}
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	// Parse registry data
	var registryData struct {
		Version   string                             `json:"version"`
		Timestamp time.Time                          `json:"timestamp"`
		Sessions  map[string]*claude.SessionMetadata `json:"sessions"`
	}

	if err := json.Unmarshal(data, &registryData); err != nil {
		return fmt.Errorf("failed to unmarshal registry data: %w", err)
	}

	// Clear existing data
	r.sessions = make(map[string]*claude.SessionMetadata)
	r.indexByName = make(map[string][]string)
	r.indexByTags = make(map[string][]string)
	r.indexByState = make(map[claude.SessionState][]string)

	// Load sessions and rebuild indices
	for sessionID, metadata := range registryData.Sessions {
		if metadata != nil {
			r.sessions[sessionID] = metadata
			r.updateIndices(sessionID, metadata)
		}
	}

	r.lastSync = registryData.Timestamp
	r.modified = false

	r.logger.Debug("registry loaded",
		zap.String("file", r.registryFile),
		zap.Int("sessions", len(r.sessions)))

	return nil
}

// createBackup creates a backup of the registry file
func (r *Registry) createBackup() error {
	if _, err := os.Stat(r.registryFile); os.IsNotExist(err) {
		return nil // No registry file to backup
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(r.config.StoragePath, fmt.Sprintf("registry-backup-%s.json", timestamp))

	// Copy current registry to backup
	data, err := os.ReadFile(r.registryFile)
	if err != nil {
		return fmt.Errorf("failed to read registry for backup: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	// Clean up old backups
	if err := r.cleanupBackups(); err != nil {
		r.logger.Warn("failed to cleanup old backups", zap.Error(err))
	}

	r.logger.Debug("registry backup created", zap.String("file", backupFile))
	return nil
}

// cleanupBackups removes old backup files
func (r *Registry) cleanupBackups() error {
	pattern := filepath.Join(r.config.StoragePath, "registry-backup-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob backup files: %w", err)
	}

	if len(matches) <= r.config.BackupCount {
		return nil // No cleanup needed
	}

	// Sort by name (which includes timestamp)
	sort.Strings(matches)

	// Remove oldest backups
	toRemove := len(matches) - r.config.BackupCount
	for i := 0; i < toRemove; i++ {
		if err := os.Remove(matches[i]); err != nil {
			r.logger.Warn("failed to remove old backup",
				zap.String("file", matches[i]),
				zap.Error(err))
		}
	}

	return nil
}

// Sync forces a sync of the registry to disk
func (r *Registry) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.save()
}

// Reload reloads the registry from disk
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.load()
}

// Close closes the registry and saves any pending changes
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.save(); err != nil {
		return fmt.Errorf("failed to save registry during close: %w", err)
	}

	r.logger.Debug("registry closed")
	return nil
}
