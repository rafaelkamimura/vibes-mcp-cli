package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConversationEntry represents a single interaction in a session
type ConversationEntry struct {
	ID          string              `json:"id"`           // Unique entry ID
	SessionID   string              `json:"session_id"`   // Associated session ID
	Type        ConversationEntryType `json:"type"`       // Entry type (input/output/system)
	Content     string              `json:"content"`      // Entry content
	Timestamp   time.Time           `json:"timestamp"`    // Entry timestamp
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
	TokenCount  int                 `json:"token_count,omitempty"` // Token count if applicable
	Duration    time.Duration       `json:"duration,omitempty"`    // Processing duration
	Error       string              `json:"error,omitempty"`       // Error message if any
}

// ConversationEntryType represents the type of conversation entry
type ConversationEntryType string

const (
	ConversationEntryTypeInput      ConversationEntryType = "input"
	ConversationEntryTypeOutput     ConversationEntryType = "output"
	ConversationEntryTypeSystem     ConversationEntryType = "system"
	ConversationEntryTypeError      ConversationEntryType = "error"
	ConversationEntryTypeCommand    ConversationEntryType = "command"
	ConversationEntryTypeResponse   ConversationEntryType = "response"
)

// ConversationHistory manages the conversation history for a session
type ConversationHistory struct {
	SessionID   string              `json:"session_id"`
	Entries     []*ConversationEntry `json:"entries"`
	TotalTokens int                 `json:"total_tokens"`
	StartTime   time.Time           `json:"start_time"`
	LastUpdate  time.Time           `json:"last_update"`
	MaxEntries  int                 `json:"max_entries"` // Maximum entries to keep
}

// NewConversationHistory creates a new conversation history
func NewConversationHistory(sessionID string, maxEntries int) *ConversationHistory {
	if maxEntries <= 0 {
		maxEntries = 1000 // Default maximum
	}

	return &ConversationHistory{
		SessionID:  sessionID,
		Entries:    make([]*ConversationEntry, 0),
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		MaxEntries: maxEntries,
	}
}

// AddEntry adds a new conversation entry
func (h *ConversationHistory) AddEntry(entryType ConversationEntryType, content string, metadata map[string]interface{}) *ConversationEntry {
	entry := &ConversationEntry{
		ID:        fmt.Sprintf("%s-%d", h.SessionID, time.Now().UnixNano()),
		SessionID: h.SessionID,
		Type:      entryType,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	h.Entries = append(h.Entries, entry)
	h.LastUpdate = time.Now()

	// Enforce maximum entries limit
	if len(h.Entries) > h.MaxEntries {
		// Remove oldest entries, keeping the most recent ones
		keep := h.MaxEntries - 100 // Remove 100 entries at a time for efficiency
		if keep < h.MaxEntries/2 {
			keep = h.MaxEntries / 2
		}
		h.Entries = h.Entries[len(h.Entries)-keep:]
	}

	return entry
}

// AddInputEntry adds an input entry
func (h *ConversationHistory) AddInputEntry(content string, metadata map[string]interface{}) *ConversationEntry {
	return h.AddEntry(ConversationEntryTypeInput, content, metadata)
}

// AddOutputEntry adds an output entry
func (h *ConversationHistory) AddOutputEntry(content string, metadata map[string]interface{}) *ConversationEntry {
	return h.AddEntry(ConversationEntryTypeOutput, content, metadata)
}

// AddSystemEntry adds a system entry
func (h *ConversationHistory) AddSystemEntry(content string, metadata map[string]interface{}) *ConversationEntry {
	return h.AddEntry(ConversationEntryTypeSystem, content, metadata)
}

// AddErrorEntry adds an error entry
func (h *ConversationHistory) AddErrorEntry(content string, err error, metadata map[string]interface{}) *ConversationEntry {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if err != nil {
		metadata["error_type"] = fmt.Sprintf("%T", err)
	}

	entry := h.AddEntry(ConversationEntryTypeError, content, metadata)
	if err != nil {
		entry.Error = err.Error()
	}
	return entry
}

// GetEntries returns all conversation entries
func (h *ConversationHistory) GetEntries() []*ConversationEntry {
	entries := make([]*ConversationEntry, len(h.Entries))
	copy(entries, h.Entries)
	return entries
}

// GetEntriesAfter returns entries after a specific timestamp
func (h *ConversationHistory) GetEntriesAfter(timestamp time.Time) []*ConversationEntry {
	var result []*ConversationEntry
	for _, entry := range h.Entries {
		if entry.Timestamp.After(timestamp) {
			result = append(result, entry)
		}
	}
	return result
}

// GetEntriesByType returns entries of a specific type
func (h *ConversationHistory) GetEntriesByType(entryType ConversationEntryType) []*ConversationEntry {
	var result []*ConversationEntry
	for _, entry := range h.Entries {
		if entry.Type == entryType {
			result = append(result, entry)
		}
	}
	return result
}

// GetLastEntry returns the most recent entry
func (h *ConversationHistory) GetLastEntry() *ConversationEntry {
	if len(h.Entries) == 0 {
		return nil
	}
	return h.Entries[len(h.Entries)-1]
}

// GetLastEntryOfType returns the most recent entry of a specific type
func (h *ConversationHistory) GetLastEntryOfType(entryType ConversationEntryType) *ConversationEntry {
	for i := len(h.Entries) - 1; i >= 0; i-- {
		if h.Entries[i].Type == entryType {
			return h.Entries[i]
		}
	}
	return nil
}

// GetStats returns conversation statistics
func (h *ConversationHistory) GetStats() *ConversationStats {
	stats := &ConversationStats{
		TotalEntries:  len(h.Entries),
		TotalTokens:   h.TotalTokens,
		StartTime:     h.StartTime,
		LastUpdate:    h.LastUpdate,
		EntryTypeCount: make(map[ConversationEntryType]int),
	}

	if len(h.Entries) > 0 {
		stats.Duration = h.LastUpdate.Sub(h.StartTime)
	}

	// Count entries by type
	for _, entry := range h.Entries {
		stats.EntryTypeCount[entry.Type]++
		if entry.TokenCount > 0 {
			stats.TotalTokens += entry.TokenCount
		}
	}

	return stats
}

// ConversationStats holds statistics about conversation history
type ConversationStats struct {
	TotalEntries   int                              `json:"total_entries"`
	TotalTokens    int                              `json:"total_tokens"`
	StartTime      time.Time                        `json:"start_time"`
	LastUpdate     time.Time                        `json:"last_update"`
	Duration       time.Duration                    `json:"duration"`
	EntryTypeCount map[ConversationEntryType]int    `json:"entry_type_count"`
}

// Clear removes all entries from the history
func (h *ConversationHistory) Clear() {
	h.Entries = make([]*ConversationEntry, 0)
	h.TotalTokens = 0
	h.LastUpdate = time.Now()
}

// Truncate removes entries older than the specified duration
func (h *ConversationHistory) Truncate(maxAge time.Duration) int {
	cutoff := time.Now().Add(-maxAge)
	var kept []*ConversationEntry
	
	for _, entry := range h.Entries {
		if entry.Timestamp.After(cutoff) {
			kept = append(kept, entry)
		}
	}
	
	removed := len(h.Entries) - len(kept)
	h.Entries = kept
	h.LastUpdate = time.Now()
	
	return removed
}

// HistoryManager manages conversation histories for all sessions
type HistoryManager struct {
	storagePath string
	logger      *zap.Logger
	mu          sync.RWMutex
	histories   map[string]*ConversationHistory // SessionID -> History
	maxEntries  int
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(storagePath string, maxEntries int, logger *zap.Logger) (*HistoryManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if maxEntries <= 0 {
		maxEntries = 1000
	}

	// Ensure storage directory exists
	historyDir := filepath.Join(storagePath, "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history storage directory: %w", err)
	}

	manager := &HistoryManager{
		storagePath: historyDir,
		logger:      logger,
		histories:   make(map[string]*ConversationHistory),
		maxEntries:  maxEntries,
	}

	return manager, nil
}

// GetHistory returns the conversation history for a session
func (hm *HistoryManager) GetHistory(sessionID string) *ConversationHistory {
	hm.mu.RLock()
	history, exists := hm.histories[sessionID]
	hm.mu.RUnlock()

	if !exists {
		hm.mu.Lock()
		// Double-check pattern
		if history, exists = hm.histories[sessionID]; !exists {
			// Try to load from disk first
			if loadedHistory, err := hm.loadHistoryFromDisk(sessionID); err == nil {
				history = loadedHistory
				hm.histories[sessionID] = history
			} else {
				// Create new history
				history = NewConversationHistory(sessionID, hm.maxEntries)
				hm.histories[sessionID] = history
			}
		}
		hm.mu.Unlock()
	}

	return history
}

// SaveHistory saves a conversation history to disk
func (hm *HistoryManager) SaveHistory(sessionID string) error {
	hm.mu.RLock()
	history, exists := hm.histories[sessionID]
	hm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("history not found for session: %s", sessionID)
	}

	return hm.saveHistoryToDisk(history)
}

// SaveAllHistories saves all conversation histories to disk
func (hm *HistoryManager) SaveAllHistories() error {
	hm.mu.RLock()
	histories := make([]*ConversationHistory, 0, len(hm.histories))
	for _, history := range hm.histories {
		histories = append(histories, history)
	}
	hm.mu.RUnlock()

	var lastErr error
	for _, history := range histories {
		if err := hm.saveHistoryToDisk(history); err != nil {
			lastErr = err
			hm.logger.Error("failed to save history",
				zap.String("session_id", history.SessionID),
				zap.Error(err))
		}
	}

	return lastErr
}

// DeleteHistory removes a conversation history
func (hm *HistoryManager) DeleteHistory(sessionID string) error {
	hm.mu.Lock()
	delete(hm.histories, sessionID)
	hm.mu.Unlock()

	// Remove from disk
	historyFile := filepath.Join(hm.storagePath, fmt.Sprintf("%s.json", sessionID))
	if err := os.Remove(historyFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete history file: %w", err)
	}

	return nil
}

// ListHistories returns a list of all session IDs with histories
func (hm *HistoryManager) ListHistories() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	sessionIDs := make([]string, 0, len(hm.histories))
	for sessionID := range hm.histories {
		sessionIDs = append(sessionIDs, sessionID)
	}

	sort.Strings(sessionIDs)
	return sessionIDs
}

// SearchEntries searches for entries across all histories
func (hm *HistoryManager) SearchEntries(query string, entryType ConversationEntryType, maxResults int) ([]*ConversationEntry, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var results []*ConversationEntry
	
	for _, history := range hm.histories {
		for _, entry := range history.Entries {
			// Type filter
			if entryType != "" && entry.Type != entryType {
				continue
			}
			
			// Content search (case-insensitive)
			if query != "" {
				// Simple substring search - could be enhanced with regex or fuzzy matching
				if !containsIgnoreCase(entry.Content, query) {
					continue
				}
			}
			
			results = append(results, entry)
			
			// Limit results
			if maxResults > 0 && len(results) >= maxResults {
				break
			}
		}
		
		if maxResults > 0 && len(results) >= maxResults {
			break
		}
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	
	return results, nil
}

// containsIgnoreCase performs case-insensitive substring search
func containsIgnoreCase(s, substr string) bool {
	// Simple implementation - could use strings.ToLower but this is more efficient
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	
	// Convert to lowercase for comparison
	s = fmt.Sprintf("%s", s) // This is inefficient but works for now
	substr = fmt.Sprintf("%s", substr)
	
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if (s[i+j] | 32) != (substr[j] | 32) { // Simple ASCII lowercase
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// CleanupOldHistories removes histories older than the specified duration
func (hm *HistoryManager) CleanupOldHistories(maxAge time.Duration) (int, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var toDelete []string

	for sessionID, history := range hm.histories {
		if history.LastUpdate.Before(cutoff) {
			toDelete = append(toDelete, sessionID)
		}
	}

	// Remove from memory and disk
	for _, sessionID := range toDelete {
		delete(hm.histories, sessionID)
		
		historyFile := filepath.Join(hm.storagePath, fmt.Sprintf("%s.json", sessionID))
		if err := os.Remove(historyFile); err != nil && !os.IsNotExist(err) {
			hm.logger.Warn("failed to delete history file during cleanup",
				zap.String("session_id", sessionID),
				zap.String("file", historyFile),
				zap.Error(err))
		}
	}

	hm.logger.Info("cleaned up old histories",
		zap.Int("count", len(toDelete)),
		zap.Duration("max_age", maxAge))

	return len(toDelete), nil
}

// GetStats returns overall history manager statistics
func (hm *HistoryManager) GetStats() *HistoryManagerStats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	stats := &HistoryManagerStats{
		TotalHistories: len(hm.histories),
		TotalEntries:   0,
		TotalTokens:    0,
	}

	var oldestUpdate, newestUpdate time.Time
	for _, history := range hm.histories {
		historyStats := history.GetStats()
		stats.TotalEntries += historyStats.TotalEntries
		stats.TotalTokens += historyStats.TotalTokens

		if oldestUpdate.IsZero() || history.LastUpdate.Before(oldestUpdate) {
			oldestUpdate = history.LastUpdate
		}
		if history.LastUpdate.After(newestUpdate) {
			newestUpdate = history.LastUpdate
		}
	}

	stats.OldestHistory = oldestUpdate
	stats.NewestHistory = newestUpdate

	return stats
}

// HistoryManagerStats holds statistics about the history manager
type HistoryManagerStats struct {
	TotalHistories int       `json:"total_histories"`
	TotalEntries   int       `json:"total_entries"`
	TotalTokens    int       `json:"total_tokens"`
	OldestHistory  time.Time `json:"oldest_history"`
	NewestHistory  time.Time `json:"newest_history"`
}

// saveHistoryToDisk saves a conversation history to disk
func (hm *HistoryManager) saveHistoryToDisk(history *ConversationHistory) error {
	historyFile := filepath.Join(hm.storagePath, fmt.Sprintf("%s.json", history.SessionID))
	
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(historyFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	hm.logger.Debug("history saved to disk",
		zap.String("session_id", history.SessionID),
		zap.String("file", historyFile),
		zap.Int("entries", len(history.Entries)))

	return nil
}

// loadHistoryFromDisk loads a conversation history from disk
func (hm *HistoryManager) loadHistoryFromDisk(sessionID string) (*ConversationHistory, error) {
	historyFile := filepath.Join(hm.storagePath, fmt.Sprintf("%s.json", sessionID))
	
	data, err := os.ReadFile(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("history file does not exist")
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var history ConversationHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}

	hm.logger.Debug("history loaded from disk",
		zap.String("session_id", sessionID),
		zap.String("file", historyFile),
		zap.Int("entries", len(history.Entries)))

	return &history, nil
}

// Close shuts down the history manager and saves all histories
func (hm *HistoryManager) Close() error {
	return hm.SaveAllHistories()
}