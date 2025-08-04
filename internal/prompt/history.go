package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

// HistoryTrackerImpl implements the HistoryTracker interface
type HistoryTrackerImpl struct {
	logger      *zap.Logger
	historyFile string
	entries     []HistoryEntry
	loaded      bool
}

// NewHistoryTracker creates a new history tracker
func NewHistoryTracker(dataDir string, logger *zap.Logger) HistoryTracker {
	historyFile := filepath.Join(dataDir, "prompt-history.json")
	return &HistoryTrackerImpl{
		logger:      logger,
		historyFile: historyFile,
		entries:     make([]HistoryEntry, 0),
		loaded:      false,
	}
}

// Record records a new prompt generation entry
func (h *HistoryTrackerImpl) Record(entry *HistoryEntry) error {
	h.logger.Debug("Recording history entry",
		zap.String("id", entry.ID),
		zap.String("template", entry.Template))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return err
	}

	// Validate entry
	if err := h.validateEntry(entry); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "invalid history entry",
			Cause:   err,
		}
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = h.generateID()
	}

	// Add to entries
	h.entries = append(h.entries, *entry)

	// Sort entries by timestamp (newest first)
	sort.Slice(h.entries, func(i, j int) bool {
		return h.entries[i].Timestamp.After(h.entries[j].Timestamp)
	})

	// Maintain size limit (keep only most recent 1000 entries)
	if len(h.entries) > 1000 {
		h.entries = h.entries[:1000]
	}

	// Save to file
	if err := h.saveToFile(); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to save history",
			Cause:   err,
		}
	}

	h.logger.Info("History entry recorded successfully",
		zap.String("id", entry.ID),
		zap.String("template", entry.Template))

	return nil
}

// GetHistory returns history entries with optional filtering
func (h *HistoryTrackerImpl) GetHistory(limit int, filter string) ([]HistoryEntry, error) {
	h.logger.Debug("Getting history",
		zap.Int("limit", limit),
		zap.String("filter", filter))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return nil, err
	}

	// Apply filter if specified
	var filteredEntries []HistoryEntry
	if filter == "" {
		filteredEntries = h.entries
	} else {
		filteredEntries = h.filterEntries(filter)
	}

	// Apply limit
	if limit > 0 && len(filteredEntries) > limit {
		filteredEntries = filteredEntries[:limit]
	}

	h.logger.Info("History retrieved",
		zap.Int("total_entries", len(h.entries)),
		zap.Int("filtered_entries", len(filteredEntries)),
		zap.String("filter", filter))

	return filteredEntries, nil
}

// GetStats returns usage statistics
func (h *HistoryTrackerImpl) GetStats() (*HistoryStats, error) {
	h.logger.Debug("Getting history statistics")

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return nil, err
	}

	stats := &HistoryStats{
		TotalGenerations: len(h.entries),
		PeriodStats:      make(map[string]int),
	}

	// Calculate statistics
	templateCounts := make(map[string]int)
	languageCounts := make(map[string]int)
	repositoryCounts := make(map[string]int)
	successCount := 0
	totalWordCount := 0
	
	// Period statistics
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	thisWeek := today.AddDate(0, 0, -int(today.Weekday()))
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	dailyCount := 0
	weeklyCount := 0
	monthlyCount := 0

	for _, entry := range h.entries {
		// Template statistics
		templateCounts[entry.Template]++

		// Language statistics
		if entry.Language != "" {
			languageCounts[entry.Language]++
		}

		// Repository statistics
		if entry.Repository != "" {
			repositoryCounts[entry.Repository]++
		}

		// Success rate
		if entry.Success {
			successCount++
		}

		// Word count
		totalWordCount += entry.WordCount

		// Period statistics
		if entry.Timestamp.After(today) {
			dailyCount++
		}
		if entry.Timestamp.After(thisWeek) {
			weeklyCount++
		}
		if entry.Timestamp.After(thisMonth) {
			monthlyCount++
		}
	}

	// Calculate success rate
	if len(h.entries) > 0 {
		stats.SuccessRate = float64(successCount) / float64(len(h.entries))
		stats.AverageWordCount = totalWordCount / len(h.entries)
	}

	// Sort and get top items
	stats.TopTemplates = h.getTopItems(templateCounts, 10)
	stats.TopLanguages = h.convertToLanguageUsage(languageCounts, 5)
	stats.TopRepositories = h.convertToRepositoryUsage(repositoryCounts, 5)

	// Period statistics
	stats.PeriodStats["daily"] = dailyCount
	stats.PeriodStats["weekly"] = weeklyCount
	stats.PeriodStats["monthly"] = monthlyCount

	h.logger.Info("History statistics calculated",
		zap.Int("total", stats.TotalGenerations),
		zap.Float64("success_rate", stats.SuccessRate),
		zap.Int("avg_word_count", stats.AverageWordCount))

	return stats, nil
}

// Cleanup removes old history entries
func (h *HistoryTrackerImpl) Cleanup(olderThan time.Duration) error {
	h.logger.Info("Cleaning up history", zap.Duration("older_than", olderThan))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return err
	}

	cutoff := time.Now().Add(-olderThan)
	originalCount := len(h.entries)

	// Filter out old entries
	var keptEntries []HistoryEntry
	for _, entry := range h.entries {
		if entry.Timestamp.After(cutoff) {
			keptEntries = append(keptEntries, entry)
		}
	}

	h.entries = keptEntries
	removedCount := originalCount - len(h.entries)

	// Save updated entries
	if removedCount > 0 {
		if err := h.saveToFile(); err != nil {
			return &PromptError{
				Type:    ErrorTypeHistory,
				Message: "failed to save history after cleanup",
				Cause:   err,
			}
		}
	}

	h.logger.Info("History cleanup completed",
		zap.Int("removed", removedCount),
		zap.Int("remaining", len(h.entries)))

	return nil
}

// GetEntryByID returns a specific history entry by ID
func (h *HistoryTrackerImpl) GetEntryByID(id string) (*HistoryEntry, error) {
	h.logger.Debug("Getting history entry by ID", zap.String("id", id))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return nil, err
	}

	for _, entry := range h.entries {
		if entry.ID == id {
			return &entry, nil
		}
	}

	return nil, &PromptError{
		Type:    ErrorTypeHistory,
		Message: fmt.Sprintf("history entry not found: %s", id),
	}
}

// DeleteEntry removes a history entry
func (h *HistoryTrackerImpl) DeleteEntry(id string) error {
	h.logger.Debug("Deleting history entry", zap.String("id", id))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return err
	}

	// Find and remove entry
	var newEntries []HistoryEntry
	found := false
	for _, entry := range h.entries {
		if entry.ID != id {
			newEntries = append(newEntries, entry)
		} else {
			found = true
		}
	}

	if !found {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: fmt.Sprintf("history entry not found: %s", id),
		}
	}

	h.entries = newEntries

	// Save updated entries
	if err := h.saveToFile(); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to save history after deletion",
			Cause:   err,
		}
	}

	h.logger.Info("History entry deleted", zap.String("id", id))
	return nil
}

// UpdateEntry updates an existing history entry
func (h *HistoryTrackerImpl) UpdateEntry(entry *HistoryEntry) error {
	h.logger.Debug("Updating history entry", zap.String("id", entry.ID))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return err
	}

	// Validate entry
	if err := h.validateEntry(entry); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "invalid history entry",
			Cause:   err,
		}
	}

	// Find and update entry
	found := false
	for i, existingEntry := range h.entries {
		if existingEntry.ID == entry.ID {
			h.entries[i] = *entry
			found = true
			break
		}
	}

	if !found {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: fmt.Sprintf("history entry not found: %s", entry.ID),
		}
	}

	// Save updated entries
	if err := h.saveToFile(); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to save history after update",
			Cause:   err,
		}
	}

	h.logger.Info("History entry updated", zap.String("id", entry.ID))
	return nil
}

// ExportHistory exports history to a file
func (h *HistoryTrackerImpl) ExportHistory(filePath string, format string) error {
	h.logger.Info("Exporting history", zap.String("path", filePath), zap.String("format", format))

	// Ensure entries are loaded
	if err := h.ensureLoaded(); err != nil {
		return err
	}

	var data []byte
	var err error

	switch strings.ToLower(format) {
	case "json":
		data, err = json.MarshalIndent(h.entries, "", "  ")
	case "csv":
		data, err = h.exportAsCSV()
	default:
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: fmt.Sprintf("unsupported export format: %s", format),
		}
	}

	if err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to format history for export",
			Cause:   err,
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to write exported history",
			Cause:   err,
		}
	}

	h.logger.Info("History exported successfully",
		zap.String("path", filePath),
		zap.Int("entries", len(h.entries)))

	return nil
}

// Helper methods

func (h *HistoryTrackerImpl) ensureLoaded() error {
	if h.loaded {
		return nil
	}

	// Check if history file exists
	if _, err := os.Stat(h.historyFile); os.IsNotExist(err) {
		// File doesn't exist, start with empty history
		h.entries = make([]HistoryEntry, 0)
		h.loaded = true
		return nil
	}

	// Load from file
	data, err := os.ReadFile(h.historyFile)
	if err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to read history file",
			Cause:   err,
		}
	}

	if err := json.Unmarshal(data, &h.entries); err != nil {
		return &PromptError{
			Type:    ErrorTypeHistory,
			Message: "failed to parse history file",
			Cause:   err,
		}
	}

	h.loaded = true
	h.logger.Info("History loaded from file",
		zap.String("file", h.historyFile),
		zap.Int("entries", len(h.entries)))

	return nil
}

func (h *HistoryTrackerImpl) saveToFile() error {
	// Ensure directory exists
	dir := filepath.Dir(h.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(h.historyFile, data, 0644)
}

func (h *HistoryTrackerImpl) validateEntry(entry *HistoryEntry) error {
	if entry.Template == "" {
		return fmt.Errorf("template name is required")
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	if entry.Parameters == nil {
		entry.Parameters = make(map[string]string)
	}

	return nil
}

func (h *HistoryTrackerImpl) generateID() string {
	return fmt.Sprintf("hist_%d", time.Now().UnixNano())
}

func (h *HistoryTrackerImpl) filterEntries(filter string) []HistoryEntry {
	var filtered []HistoryEntry
	lowerFilter := strings.ToLower(filter)

	for _, entry := range h.entries {
		// Check various fields for the filter
		if strings.Contains(strings.ToLower(entry.Template), lowerFilter) ||
			strings.Contains(strings.ToLower(entry.Repository), lowerFilter) ||
			strings.Contains(strings.ToLower(entry.Language), lowerFilter) ||
			strings.Contains(strings.ToLower(entry.Framework), lowerFilter) ||
			strings.Contains(strings.ToLower(entry.AITool), lowerFilter) ||
			strings.Contains(strings.ToLower(entry.OutputMethod), lowerFilter) {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

func (h *HistoryTrackerImpl) getTopItems(counts map[string]int, limit int) []TemplateUsage {
	// Convert to slice for sorting
	var items []TemplateUsage
	for name, count := range counts {
		items = append(items, TemplateUsage{
			Name:  name,
			Count: count,
		})
	}

	// Sort by count (descending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

func (h *HistoryTrackerImpl) convertToLanguageUsage(counts map[string]int, limit int) []LanguageUsage {
	// Convert to slice for sorting
	var items []LanguageUsage
	for language, count := range counts {
		items = append(items, LanguageUsage{
			Language: language,
			Count:    count,
		})
	}

	// Sort by count (descending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

func (h *HistoryTrackerImpl) convertToRepositoryUsage(counts map[string]int, limit int) []RepositoryUsage {
	// Convert to slice for sorting
	var items []RepositoryUsage
	for repository, count := range counts {
		items = append(items, RepositoryUsage{
			Repository: repository,
			Count:      count,
		})
	}

	// Sort by count (descending)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	// Apply limit
	if len(items) > limit {
		items = items[:limit]
	}

	return items
}

func (h *HistoryTrackerImpl) exportAsCSV() ([]byte, error) {
	var lines []string
	
	// CSV header
	header := "ID,Template,Repository,Language,Framework,OutputMethod,AITool,Success,Timestamp,Duration,WordCount"
	lines = append(lines, header)

	// CSV rows
	for _, entry := range h.entries {
		row := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%t,%s,%s,%d",
			entry.ID,
			h.csvEscape(entry.Template),
			h.csvEscape(entry.Repository),
			h.csvEscape(entry.Language),
			h.csvEscape(entry.Framework),
			h.csvEscape(entry.OutputMethod),
			h.csvEscape(entry.AITool),
			entry.Success,
			entry.Timestamp.Format(time.RFC3339),
			entry.Duration.String(),
			entry.WordCount,
		)
		lines = append(lines, row)
	}

	return []byte(strings.Join(lines, "\n")), nil
}

func (h *HistoryTrackerImpl) csvEscape(field string) string {
	// Simple CSV escaping - wrap in quotes if contains comma or quote
	if strings.Contains(field, ",") || strings.Contains(field, "\"") {
		escaped := strings.ReplaceAll(field, "\"", "\"\"")
		return "\"" + escaped + "\""
	}
	return field
}