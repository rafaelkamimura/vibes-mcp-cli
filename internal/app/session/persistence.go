package session

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PersistenceConfig holds configuration for session persistence
type PersistenceConfig struct {
	StoragePath       string        `json:"storage_path"`        // Base storage path
	CompressData      bool          `json:"compress_data"`       // Compress stored data
	BackupEnabled     bool          `json:"backup_enabled"`      // Enable backups
	BackupInterval    time.Duration `json:"backup_interval"`     // Backup interval
	BackupRetention   int           `json:"backup_retention"`    // Number of backups to keep
	AutoSaveInterval  time.Duration `json:"auto_save_interval"`  // Auto-save interval
	AutoSaveEnabled   bool          `json:"auto_save_enabled"`   // Enable auto-save
	SyncMode          SyncMode      `json:"sync_mode"`           // Synchronization mode
	MaxHistoryEntries int           `json:"max_history_entries"` // Max history entries per session
}

// SyncMode defines how persistence synchronization works
type SyncMode string

const (
	SyncModeImmediate  SyncMode = "immediate"  // Save immediately on changes
	SyncModeBatched    SyncMode = "batched"    // Save in batches periodically
	SyncModeManual     SyncMode = "manual"     // Save only when explicitly requested
)

// DefaultPersistenceConfig returns default persistence configuration
func DefaultPersistenceConfig() *PersistenceConfig {
	return &PersistenceConfig{
		StoragePath:       "./sessions",
		CompressData:      true,
		BackupEnabled:     true,
		BackupInterval:    time.Hour * 6,
		BackupRetention:   24,
		AutoSaveInterval:  time.Minute * 5,
		AutoSaveEnabled:   true,
		SyncMode:          SyncModeBatched,
		MaxHistoryEntries: 10000,
	}
}

// SessionSnapshot represents a complete session state snapshot
type SessionSnapshot struct {
	Metadata          *EnhancedSessionMetadata `json:"metadata"`
	ConversationHistory *ConversationHistory   `json:"conversation_history,omitempty"`
	ProcessState      *ProcessSnapshot         `json:"process_state,omitempty"`
	CreatedAt         time.Time                `json:"created_at"`
	SchemaVersion     string                   `json:"schema_version"`
	Checksum          string                   `json:"checksum,omitempty"`
}

// ProcessSnapshot captures process state for restoration
type ProcessSnapshot struct {
	ProcessID     string            `json:"process_id"`
	State         string            `json:"state"`
	PID           int               `json:"pid"`
	StartTime     time.Time         `json:"start_time"`
	WorkingDir    string            `json:"working_dir"`
	Environment   map[string]string `json:"environment"`
	Args          []string          `json:"args"`
	OutputBuffer  []byte            `json:"output_buffer,omitempty"`
	InputBuffer   []byte            `json:"input_buffer,omitempty"`
	ResourceUsage *SessionResourceUsage `json:"resource_usage,omitempty"`
}

// PersistenceManager manages session persistence operations
type PersistenceManager struct {
	config          *PersistenceConfig
	logger          *zap.Logger
	metadataTracker *MetadataTracker
	historyManager  *HistoryManager
	registry        *Registry
	
	mu              sync.RWMutex
	sessionPaths    map[string]string // SessionID -> Storage path
	pendingSaves    map[string]bool   // SessionID -> Pending save flag
	lastBackup      time.Time
	backupTicker    *time.Ticker
	autoSaveTicker  *time.Ticker
	stopChan        chan struct{}
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(config *PersistenceConfig, metadataTracker *MetadataTracker, historyManager *HistoryManager, registry *Registry, logger *zap.Logger) (*PersistenceManager, error) {
	if config == nil {
		config = DefaultPersistenceConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Ensure storage directories exist
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	sessionDir := filepath.Join(config.StoragePath, "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}
	
	backupDir := filepath.Join(config.StoragePath, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backups directory: %w", err)
	}

	pm := &PersistenceManager{
		config:          config,
		logger:          logger,
		metadataTracker: metadataTracker,
		historyManager:  historyManager,
		registry:        registry,
		sessionPaths:    make(map[string]string),
		pendingSaves:    make(map[string]bool),
		stopChan:        make(chan struct{}),
	}

	// Start background routines if configured
	if config.AutoSaveEnabled && config.AutoSaveInterval > 0 && config.SyncMode == SyncModeBatched {
		pm.startAutoSave()
	}
	
	if config.BackupEnabled && config.BackupInterval > 0 {
		pm.startBackupRoutine()
	}

	return pm, nil
}

// SaveSession saves a complete session snapshot
func (pm *PersistenceManager) SaveSession(sessionID string, includeHistory bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.saveSessionInternal(sessionID, includeHistory)
}

// saveSessionInternal performs the actual session save (caller must hold lock)
func (pm *PersistenceManager) saveSessionInternal(sessionID string, includeHistory bool) error {
	// Get session metadata
	metadata, err := pm.metadataTracker.GetMetadata(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session metadata: %w", err)
	}

	// Create session snapshot
	snapshot := &SessionSnapshot{
		Metadata:      metadata,
		CreatedAt:     time.Now(),
		SchemaVersion: "1.0",
	}

	// Include conversation history if requested
	if includeHistory && pm.historyManager != nil {
		history := pm.historyManager.GetHistory(sessionID)
		snapshot.ConversationHistory = history
	}

	// Determine session storage path
	sessionPath := pm.getSessionPath(sessionID)
	pm.sessionPaths[sessionID] = sessionPath

	// Save snapshot
	if err := pm.saveSnapshotToFile(snapshot, sessionPath); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Clear pending save flag
	delete(pm.pendingSaves, sessionID)

	pm.logger.Debug("session saved",
		zap.String("session_id", sessionID),
		zap.String("path", sessionPath),
		zap.Bool("include_history", includeHistory))

	return nil
}

// LoadSession loads a session snapshot
func (pm *PersistenceManager) LoadSession(sessionID string) (*SessionSnapshot, error) {
	pm.mu.RLock()
	sessionPath, exists := pm.sessionPaths[sessionID]
	pm.mu.RUnlock()

	if !exists {
		sessionPath = pm.getSessionPath(sessionID)
	}

	return pm.loadSnapshotFromFile(sessionPath)
}

// LoadAllSessions loads all available session snapshots
func (pm *PersistenceManager) LoadAllSessions() (map[string]*SessionSnapshot, error) {
	sessionDir := filepath.Join(pm.config.StoragePath, "sessions")
	
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*SessionSnapshot), nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	sessions := make(map[string]*SessionSnapshot)
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Extract session ID from filename
		sessionID := pm.extractSessionIDFromFilename(entry.Name())
		if sessionID == "" {
			continue
		}
		
		sessionPath := filepath.Join(sessionDir, entry.Name())
		snapshot, err := pm.loadSnapshotFromFile(sessionPath)
		if err != nil {
			pm.logger.Warn("failed to load session snapshot",
				zap.String("session_id", sessionID),
				zap.String("path", sessionPath),
				zap.Error(err))
			continue
		}
		
		sessions[sessionID] = snapshot
		
		pm.mu.Lock()
		pm.sessionPaths[sessionID] = sessionPath
		pm.mu.Unlock()
	}

	pm.logger.Info("loaded session snapshots",
		zap.Int("count", len(sessions)))

	return sessions, nil
}

// RestoreSession restores a session from a snapshot
func (pm *PersistenceManager) RestoreSession(sessionID string, snapshot *SessionSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	// Restore metadata to tracker
	if snapshot.Metadata != nil {
		// Note: This would require adding a RestoreMetadata method to MetadataTracker
		pm.logger.Debug("restoring session metadata",
			zap.String("session_id", sessionID))
	}

	// Restore conversation history
	if snapshot.ConversationHistory != nil && pm.historyManager != nil {
		pm.historyManager.mu.Lock()
		pm.historyManager.histories[sessionID] = snapshot.ConversationHistory
		pm.historyManager.mu.Unlock()
		
		pm.logger.Debug("restored conversation history",
			zap.String("session_id", sessionID),
			zap.Int("entries", len(snapshot.ConversationHistory.Entries)))
	}

	// Process state restoration would require integration with the actual session management
	// This is a placeholder for more complex process restoration logic
	if snapshot.ProcessState != nil {
		pm.logger.Debug("process state available for restoration",
			zap.String("session_id", sessionID),
			zap.String("process_id", snapshot.ProcessState.ProcessID))
	}

	return nil
}

// DeleteSession removes a session from persistent storage
func (pm *PersistenceManager) DeleteSession(sessionID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	sessionPath, exists := pm.sessionPaths[sessionID]
	if !exists {
		sessionPath = pm.getSessionPath(sessionID)
	}

	// Remove the file
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	// Clean up internal state
	delete(pm.sessionPaths, sessionID)
	delete(pm.pendingSaves, sessionID)

	pm.logger.Debug("session deleted from storage",
		zap.String("session_id", sessionID),
		zap.String("path", sessionPath))

	return nil
}

// MarkForSave marks a session for saving (used with batched sync mode)
func (pm *PersistenceManager) MarkForSave(sessionID string) {
	if pm.config.SyncMode == SyncModeImmediate {
		// Save immediately
		go func() {
			if err := pm.SaveSession(sessionID, true); err != nil {
				pm.logger.Error("failed to save session immediately",
					zap.String("session_id", sessionID),
					zap.Error(err))
			}
		}()
		return
	}

	pm.mu.Lock()
	pm.pendingSaves[sessionID] = true
	pm.mu.Unlock()
}

// SavePendingSessions saves all sessions marked for saving
func (pm *PersistenceManager) SavePendingSessions() error {
	pm.mu.Lock()
	pendingSessions := make([]string, 0, len(pm.pendingSaves))
	for sessionID := range pm.pendingSaves {
		pendingSessions = append(pendingSessions, sessionID)
	}
	pm.mu.Unlock()

	var lastErr error
	saved := 0
	
	for _, sessionID := range pendingSessions {
		if err := pm.SaveSession(sessionID, true); err != nil {
			lastErr = err
			pm.logger.Error("failed to save pending session",
				zap.String("session_id", sessionID),
				zap.Error(err))
		} else {
			saved++
		}
	}

	if saved > 0 {
		pm.logger.Info("saved pending sessions",
			zap.Int("saved", saved),
			zap.Int("total", len(pendingSessions)))
	}

	return lastErr
}

// CreateBackup creates a backup of all sessions
func (pm *PersistenceManager) CreateBackup() error {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(pm.config.StoragePath, "backups", timestamp)
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Save all sessions to backup directory
	sessions := pm.metadataTracker.ListMetadata()
	saved := 0
	
	for _, metadata := range sessions {
		snapshot := &SessionSnapshot{
			Metadata:      metadata,
			CreatedAt:     time.Now(),
			SchemaVersion: "1.0",
		}
		
		// Include conversation history
		if pm.historyManager != nil {
			history := pm.historyManager.GetHistory(metadata.ID)
			snapshot.ConversationHistory = history
		}
		
		backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.json", metadata.ID))
		if pm.config.CompressData {
			backupPath += ".gz"
		}
		
		if err := pm.saveSnapshotToFile(snapshot, backupPath); err != nil {
			pm.logger.Error("failed to backup session",
				zap.String("session_id", metadata.ID),
				zap.Error(err))
		} else {
			saved++
		}
	}

	// Clean up old backups
	if err := pm.cleanupOldBackups(); err != nil {
		pm.logger.Warn("failed to cleanup old backups", zap.Error(err))
	}

	pm.lastBackup = time.Now()
	
	pm.logger.Info("backup created",
		zap.String("backup_dir", backupDir),
		zap.Int("sessions_backed_up", saved))

	return nil
}

// getSessionPath returns the storage path for a session
func (pm *PersistenceManager) getSessionPath(sessionID string) string {
	filename := fmt.Sprintf("%s.json", sessionID)
	if pm.config.CompressData {
		filename += ".gz"
	}
	return filepath.Join(pm.config.StoragePath, "sessions", filename)
}

// extractSessionIDFromFilename extracts session ID from filename
func (pm *PersistenceManager) extractSessionIDFromFilename(filename string) string {
	// Remove extension(s)
	if strings.HasSuffix(filename, ".json.gz") {
		return filename[:len(filename)-8]
	}
	if strings.HasSuffix(filename, ".json") {
		return filename[:len(filename)-5]
	}
	return ""
}

// saveSnapshotToFile saves a snapshot to a file
func (pm *PersistenceManager) saveSnapshotToFile(snapshot *SessionSnapshot, filePath string) error {
	// Create temporary file first
	tempPath := filePath + ".tmp"
	
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file
	
	// Add compression if configured
	if pm.config.CompressData && strings.HasSuffix(filePath, ".gz") {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		writer = gzWriter
	}

	// Encode JSON
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(snapshot); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	// Close compression writer if used
	if gzWriter, ok := writer.(*gzip.Writer); ok {
		if err := gzWriter.Close(); err != nil {
			os.Remove(tempPath)
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic move
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	return nil
}

// loadSnapshotFromFile loads a snapshot from a file
func (pm *PersistenceManager) loadSnapshotFromFile(filePath string) (*SessionSnapshot, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file
	
	// Handle compression
	if strings.HasSuffix(filePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	var snapshot SessionSnapshot
	decoder := json.NewDecoder(reader)
	
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	return &snapshot, nil
}

// startAutoSave starts the auto-save routine
func (pm *PersistenceManager) startAutoSave() {
	pm.autoSaveTicker = time.NewTicker(pm.config.AutoSaveInterval)
	
	go func() {
		for {
			select {
			case <-pm.autoSaveTicker.C:
				if err := pm.SavePendingSessions(); err != nil {
					pm.logger.Error("auto-save failed", zap.Error(err))
				}
			case <-pm.stopChan:
				return
			}
		}
	}()
}

// startBackupRoutine starts the backup routine
func (pm *PersistenceManager) startBackupRoutine() {
	pm.backupTicker = time.NewTicker(pm.config.BackupInterval)
	
	go func() {
		for {
			select {
			case <-pm.backupTicker.C:
				if err := pm.CreateBackup(); err != nil {
					pm.logger.Error("backup failed", zap.Error(err))
				}
			case <-pm.stopChan:
				return
			}
		}
	}()
}

// cleanupOldBackups removes old backup directories
func (pm *PersistenceManager) cleanupOldBackups() error {
	backupBaseDir := filepath.Join(pm.config.StoragePath, "backups")
	
	entries, err := os.ReadDir(backupBaseDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	if len(entries) <= pm.config.BackupRetention {
		return nil
	}

	// Sort by name (which includes timestamp)
	var backupDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			backupDirs = append(backupDirs, entry.Name())
		}
	}
	
	if len(backupDirs) <= pm.config.BackupRetention {
		return nil
	}

	// Remove oldest backups
	toRemove := len(backupDirs) - pm.config.BackupRetention
	for i := 0; i < toRemove; i++ {
		backupPath := filepath.Join(backupBaseDir, backupDirs[i])
		if err := os.RemoveAll(backupPath); err != nil {
			pm.logger.Warn("failed to remove old backup",
				zap.String("path", backupPath),
				zap.Error(err))
		}
	}

	return nil
}

// GetStats returns persistence statistics
func (pm *PersistenceManager) GetStats() *PersistenceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return &PersistenceStats{
		StoragePath:      pm.config.StoragePath,
		TotalSessions:    len(pm.sessionPaths),
		PendingSaves:     len(pm.pendingSaves),
		LastBackup:       pm.lastBackup,
		CompressData:     pm.config.CompressData,
		AutoSaveEnabled:  pm.config.AutoSaveEnabled,
		BackupEnabled:    pm.config.BackupEnabled,
		SyncMode:         pm.config.SyncMode,
	}
}

// PersistenceStats holds statistics about persistence operations
type PersistenceStats struct {
	StoragePath      string    `json:"storage_path"`
	TotalSessions    int       `json:"total_sessions"`
	PendingSaves     int       `json:"pending_saves"`
	LastBackup       time.Time `json:"last_backup"`
	CompressData     bool      `json:"compress_data"`
	AutoSaveEnabled  bool      `json:"auto_save_enabled"`
	BackupEnabled    bool      `json:"backup_enabled"`
	SyncMode         SyncMode  `json:"sync_mode"`
}

// Close shuts down the persistence manager
func (pm *PersistenceManager) Close() error {
	// Signal stop to background routines
	close(pm.stopChan)
	
	// Stop tickers
	if pm.autoSaveTicker != nil {
		pm.autoSaveTicker.Stop()
	}
	if pm.backupTicker != nil {
		pm.backupTicker.Stop()
	}

	// Save any pending sessions
	if err := pm.SavePendingSessions(); err != nil {
		pm.logger.Error("failed to save pending sessions during shutdown", zap.Error(err))
	}

	pm.logger.Info("persistence manager closed")
	return nil
}