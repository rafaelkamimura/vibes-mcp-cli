package session

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"openai-cli/internal/app/claude"
)

// SearchCriteria defines search parameters for sessions
type SearchCriteria struct {
	// Text search
	Query       string   `json:"query,omitempty"`        // Text query for name, description, or content
	QueryFields []string `json:"query_fields,omitempty"` // Fields to search in (name, description, content, tags)
	CaseSensitive bool   `json:"case_sensitive"`         // Case-sensitive search
	UseRegex    bool     `json:"use_regex"`              // Use regular expressions
	
	// State filtering
	States      []claude.SessionState `json:"states,omitempty"`      // Filter by session states
	ExcludeStates []claude.SessionState `json:"exclude_states,omitempty"` // Exclude specific states
	
	// Time filtering
	CreatedAfter  *time.Time `json:"created_after,omitempty"`  // Sessions created after this time
	CreatedBefore *time.Time `json:"created_before,omitempty"` // Sessions created before this time
	UpdatedAfter  *time.Time `json:"updated_after,omitempty"`  // Sessions updated after this time
	UpdatedBefore *time.Time `json:"updated_before,omitempty"` // Sessions updated before this time
	LastActiveAfter *time.Time `json:"last_active_after,omitempty"` // Sessions active after this time
	
	// Tags and labels
	Tags         []string          `json:"tags,omitempty"`         // Must have all these tags
	AnyTags      []string          `json:"any_tags,omitempty"`     // Must have any of these tags
	ExcludeTags  []string          `json:"exclude_tags,omitempty"` // Must not have these tags
	Labels       map[string]string `json:"labels,omitempty"`       // Must have these label key-value pairs
	HasLabels    []string          `json:"has_labels,omitempty"`   // Must have these label keys (any value)
	
	// Priority and category
	Priorities   []SessionPriority `json:"priorities,omitempty"`   // Filter by priorities
	Categories   []string          `json:"categories,omitempty"`   // Filter by categories
	
	// Statistics filtering
	MinInputCount    *int           `json:"min_input_count,omitempty"`    // Minimum input count
	MaxInputCount    *int           `json:"max_input_count,omitempty"`    // Maximum input count
	MinOutputCount   *int           `json:"min_output_count,omitempty"`   // Minimum output count
	MaxOutputCount   *int           `json:"max_output_count,omitempty"`   // Maximum output count
	MinDuration      *time.Duration `json:"min_duration,omitempty"`       // Minimum session duration
	MaxDuration      *time.Duration `json:"max_duration,omitempty"`       // Maximum session duration
	MinTokens        *int           `json:"min_tokens,omitempty"`         // Minimum token usage
	MaxTokens        *int           `json:"max_tokens,omitempty"`         // Maximum token usage
	MinErrorCount    *int           `json:"min_error_count,omitempty"`    // Minimum error count
	MaxErrorCount    *int           `json:"max_error_count,omitempty"`    // Maximum error count
	
	// Resource usage filtering
	MinMemoryMB      *int64   `json:"min_memory_mb,omitempty"`      // Minimum memory usage
	MaxMemoryMB      *int64   `json:"max_memory_mb,omitempty"`      // Maximum memory usage
	MinCPUPercent    *float64 `json:"min_cpu_percent,omitempty"`    // Minimum CPU usage
	MaxCPUPercent    *float64 `json:"max_cpu_percent,omitempty"`    // Maximum CPU usage
	
	// Custom metadata filtering
	CustomFilters    map[string]interface{} `json:"custom_filters,omitempty"` // Custom metadata filters
}

// SortCriteria defines sorting parameters for search results
type SortCriteria struct {
	Field     string    `json:"field"`               // Field to sort by
	Direction SortDirection `json:"direction"`       // Sort direction
	Secondary *SortCriteria `json:"secondary,omitempty"` // Secondary sort criteria
}

// SortDirection defines sort direction
type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

// Common sort fields
const (
	SortFieldCreatedAt    = "created_at"
	SortFieldUpdatedAt    = "updated_at"
	SortFieldLastActiveAt = "last_active_at"
	SortFieldName         = "name"
	SortFieldState        = "state"
	SortFieldPriority     = "priority"
	SortFieldDuration     = "duration"
	SortFieldInputCount   = "input_count"
	SortFieldOutputCount  = "output_count"
	SortFieldErrorCount   = "error_count"
	SortFieldTokensUsed   = "tokens_used"
	SortFieldMemoryUsage  = "memory_usage"
	SortFieldCPUUsage     = "cpu_usage"
)

// SearchOptions provides additional search configuration
type SearchOptions struct {
	Limit         int           `json:"limit,omitempty"`          // Maximum number of results
	Offset        int           `json:"offset,omitempty"`         // Number of results to skip
	IncludeStats  bool          `json:"include_stats"`            // Include detailed statistics
	IncludeHistory bool         `json:"include_history"`          // Include conversation history
	Sort          *SortCriteria `json:"sort,omitempty"`          // Sort criteria
}

// SearchResult represents a search result
type SearchResult struct {
	Sessions    []*EnhancedSessionMetadata `json:"sessions"`
	TotalCount  int                        `json:"total_count"`
	HasMore     bool                       `json:"has_more"`
	SearchTime  time.Duration              `json:"search_time"`
	Query       *SearchCriteria            `json:"query,omitempty"`
}

// SessionSearcher provides session search and filtering capabilities
type SessionSearcher struct {
	metadataTracker *MetadataTracker
	historyManager  *HistoryManager
	registry        *Registry
}

// NewSessionSearcher creates a new session searcher
func NewSessionSearcher(metadataTracker *MetadataTracker, historyManager *HistoryManager, registry *Registry) *SessionSearcher {
	return &SessionSearcher{
		metadataTracker: metadataTracker,
		historyManager:  historyManager,
		registry:        registry,
	}
}

// Search searches for sessions based on criteria
func (ss *SessionSearcher) Search(criteria *SearchCriteria, options *SearchOptions) (*SearchResult, error) {
	startTime := time.Now()
	
	// Get all session metadata
	allSessions := ss.metadataTracker.ListMetadata()
	
	// Apply filters
	filteredSessions := ss.applyFilters(allSessions, criteria)
	
	// Apply sorting
	if options != nil && options.Sort != nil {
		ss.applySorting(filteredSessions, options.Sort)
	} else {
		// Default sorting by updated_at desc
		ss.applySorting(filteredSessions, &SortCriteria{
			Field:     SortFieldUpdatedAt,
			Direction: SortDirectionDesc,
		})
	}
	
	totalCount := len(filteredSessions)
	
	// Apply pagination
	var paginatedSessions []*EnhancedSessionMetadata
	if options != nil {
		start := options.Offset
		end := start + options.Limit
		
		if start < 0 {
			start = 0
		}
		if options.Limit <= 0 {
			end = len(filteredSessions)
		}
		if end > len(filteredSessions) {
			end = len(filteredSessions)
		}
		if start < len(filteredSessions) {
			paginatedSessions = filteredSessions[start:end]
		}
	} else {
		paginatedSessions = filteredSessions
	}
	
	// Determine if there are more results
	hasMore := false
	if options != nil && options.Limit > 0 {
		hasMore = options.Offset+options.Limit < totalCount
	}
	
	searchTime := time.Since(startTime)
	
	return &SearchResult{
		Sessions:   paginatedSessions,
		TotalCount: totalCount,
		HasMore:    hasMore,
		SearchTime: searchTime,
		Query:      criteria,
	}, nil
}

// applyFilters applies search criteria filters to sessions
func (ss *SessionSearcher) applyFilters(sessions []*EnhancedSessionMetadata, criteria *SearchCriteria) []*EnhancedSessionMetadata {
	if criteria == nil {
		return sessions
	}
	
	var filtered []*EnhancedSessionMetadata
	
	for _, session := range sessions {
		if ss.matchesSession(session, criteria) {
			filtered = append(filtered, session)
		}
	}
	
	return filtered
}

// matchesSession checks if a session matches the search criteria
func (ss *SessionSearcher) matchesSession(session *EnhancedSessionMetadata, criteria *SearchCriteria) bool {
	// Text query matching
	if criteria.Query != "" {
		if !ss.matchesTextQuery(session, criteria) {
			return false
		}
	}
	
	// State filtering
	if len(criteria.States) > 0 {
		found := false
		for _, state := range criteria.States {
			if session.State == state {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Exclude states
	if len(criteria.ExcludeStates) > 0 {
		for _, state := range criteria.ExcludeStates {
			if session.State == state {
				return false
			}
		}
	}
	
	// Time filtering
	if criteria.CreatedAfter != nil && session.CreatedAt.Before(*criteria.CreatedAfter) {
		return false
	}
	if criteria.CreatedBefore != nil && session.CreatedAt.After(*criteria.CreatedBefore) {
		return false
	}
	if criteria.UpdatedAfter != nil && session.UpdatedAt.Before(*criteria.UpdatedAfter) {
		return false
	}
	if criteria.UpdatedBefore != nil && session.UpdatedAt.After(*criteria.UpdatedBefore) {
		return false
	}
	if criteria.LastActiveAfter != nil && session.LastActiveAt != nil && session.LastActiveAt.Before(*criteria.LastActiveAfter) {
		return false
	}
	
	// Tag filtering
	if len(criteria.Tags) > 0 {
		if !ss.hasAllTags(session.Tags, criteria.Tags) {
			return false
		}
	}
	if len(criteria.AnyTags) > 0 {
		if !ss.hasAnyTags(session.Tags, criteria.AnyTags) {
			return false
		}
	}
	if len(criteria.ExcludeTags) > 0 {
		if ss.hasAnyTags(session.Tags, criteria.ExcludeTags) {
			return false
		}
	}
	
	// Label filtering
	if len(criteria.Labels) > 0 {
		if !ss.hasAllLabels(session.Labels, criteria.Labels) {
			return false
		}
	}
	if len(criteria.HasLabels) > 0 {
		if !ss.hasLabelKeys(session.Labels, criteria.HasLabels) {
			return false
		}
	}
	
	// Priority filtering
	if len(criteria.Priorities) > 0 {
		found := false
		for _, priority := range criteria.Priorities {
			if session.Priority == priority {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Category filtering
	if len(criteria.Categories) > 0 {
		found := false
		for _, category := range criteria.Categories {
			if session.Category == category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Statistics filtering
	if session.Stats != nil {
		if criteria.MinInputCount != nil && session.Stats.InputCount < *criteria.MinInputCount {
			return false
		}
		if criteria.MaxInputCount != nil && session.Stats.InputCount > *criteria.MaxInputCount {
			return false
		}
		if criteria.MinOutputCount != nil && session.Stats.OutputCount < *criteria.MinOutputCount {
			return false
		}
		if criteria.MaxOutputCount != nil && session.Stats.OutputCount > *criteria.MaxOutputCount {
			return false
		}
		if criteria.MinDuration != nil && session.Stats.TotalDuration < *criteria.MinDuration {
			return false
		}
		if criteria.MaxDuration != nil && session.Stats.TotalDuration > *criteria.MaxDuration {
			return false
		}
		if criteria.MinTokens != nil && session.Stats.TotalTokensUsed < *criteria.MinTokens {
			return false
		}
		if criteria.MaxTokens != nil && session.Stats.TotalTokensUsed > *criteria.MaxTokens {
			return false
		}
		if criteria.MinErrorCount != nil && session.Stats.ErrorCount < *criteria.MinErrorCount {
			return false
		}
		if criteria.MaxErrorCount != nil && session.Stats.ErrorCount > *criteria.MaxErrorCount {
			return false
		}
	}
	
	// Resource usage filtering
	if session.ResourceUsage != nil {
		if criteria.MinMemoryMB != nil && session.ResourceUsage.PeakMemoryMB < *criteria.MinMemoryMB {
			return false
		}
		if criteria.MaxMemoryMB != nil && session.ResourceUsage.PeakMemoryMB > *criteria.MaxMemoryMB {
			return false
		}
		if criteria.MinCPUPercent != nil && session.ResourceUsage.PeakCPUPercent < *criteria.MinCPUPercent {
			return false
		}
		if criteria.MaxCPUPercent != nil && session.ResourceUsage.PeakCPUPercent > *criteria.MaxCPUPercent {
			return false
		}
	}
	
	// Custom metadata filtering
	if len(criteria.CustomFilters) > 0 {
		if !ss.matchesCustomFilters(session.CustomData, criteria.CustomFilters) {
			return false
		}
	}
	
	return true
}

// matchesTextQuery checks if session matches text query
func (ss *SessionSearcher) matchesTextQuery(session *EnhancedSessionMetadata, criteria *SearchCriteria) bool {
	query := criteria.Query
	if !criteria.CaseSensitive {
		query = strings.ToLower(query)
	}
	
	// Determine which fields to search
	searchFields := criteria.QueryFields
	if len(searchFields) == 0 {
		searchFields = []string{"name", "description", "tags"}
	}
	
	for _, field := range searchFields {
		var content string
		
		switch field {
		case "name":
			content = session.Name
		case "description":
			content = session.Description
		case "tags":
			content = strings.Join(session.Tags, " ")
		case "category":
			content = session.Category
		case "content":
			// Search in conversation history if available
			if ss.historyManager != nil {
				history := ss.historyManager.GetHistory(session.ID)
				for _, entry := range history.GetEntries() {
					if ss.matchesText(entry.Content, query, criteria) {
						return true
					}
				}
			}
			continue
		default:
			continue
		}
		
		if ss.matchesText(content, query, criteria) {
			return true
		}
	}
	
	return false
}

// matchesText checks if text matches query with regex/case sensitivity options
func (ss *SessionSearcher) matchesText(text, query string, criteria *SearchCriteria) bool {
	if !criteria.CaseSensitive {
		text = strings.ToLower(text)
	}
	
	if criteria.UseRegex {
		flags := ""
		if !criteria.CaseSensitive {
			flags = "(?i)"
		}
		regex, err := regexp.Compile(flags + query)
		if err != nil {
			// Fall back to simple string search
			return strings.Contains(text, query)
		}
		return regex.MatchString(text)
	}
	
	return strings.Contains(text, query)
}

// hasAllTags checks if session has all required tags
func (ss *SessionSearcher) hasAllTags(sessionTags, requiredTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range sessionTags {
		tagSet[tag] = true
	}
	
	for _, required := range requiredTags {
		if !tagSet[required] {
			return false
		}
	}
	
	return true
}

// hasAnyTags checks if session has any of the specified tags
func (ss *SessionSearcher) hasAnyTags(sessionTags, anyTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range sessionTags {
		tagSet[tag] = true
	}
	
	for _, tag := range anyTags {
		if tagSet[tag] {
			return true
		}
	}
	
	return false
}

// hasAllLabels checks if session has all required labels
func (ss *SessionSearcher) hasAllLabels(sessionLabels, requiredLabels map[string]string) bool {
	for key, value := range requiredLabels {
		if sessionValue, exists := sessionLabels[key]; !exists || sessionValue != value {
			return false
		}
	}
	return true
}

// hasLabelKeys checks if session has all required label keys
func (ss *SessionSearcher) hasLabelKeys(sessionLabels map[string]string, requiredKeys []string) bool {
	for _, key := range requiredKeys {
		if _, exists := sessionLabels[key]; !exists {
			return false
		}
	}
	return true
}

// matchesCustomFilters checks if session matches custom metadata filters
func (ss *SessionSearcher) matchesCustomFilters(customData, filters map[string]interface{}) bool {
	for key, expectedValue := range filters {
		actualValue, exists := customData[key]
		if !exists {
			return false
		}
		
		// Simple equality check (could be enhanced with type-specific comparisons)
		if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}
	return true
}

// applySorting sorts sessions based on sort criteria
func (ss *SessionSearcher) applySorting(sessions []*EnhancedSessionMetadata, sortCriteria *SortCriteria) {
	sort.Slice(sessions, func(i, j int) bool {
		return ss.compareSessionsWithCriteria(sessions[i], sessions[j], sortCriteria) < 0
	})
}

// compareSessionsWithCriteria compares two sessions based on sort criteria
func (ss *SessionSearcher) compareSessionsWithCriteria(a, b *EnhancedSessionMetadata, criteria *SortCriteria) int {
	result := ss.compareSessionsByField(a, b, criteria.Field)
	
	if result == 0 && criteria.Secondary != nil {
		// Use secondary sort criteria for tie-breaking
		result = ss.compareSessionsWithCriteria(a, b, criteria.Secondary)
	}
	
	if criteria.Direction == SortDirectionDesc {
		return -result
	}
	return result
}

// compareSessionsByField compares sessions by a specific field
func (ss *SessionSearcher) compareSessionsByField(a, b *EnhancedSessionMetadata, field string) int {
	switch field {
	case SortFieldCreatedAt:
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	case SortFieldUpdatedAt:
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return -1
		} else if a.UpdatedAt.After(b.UpdatedAt) {
			return 1
		}
		return 0
	case SortFieldLastActiveAt:
		aTime := a.LastActiveAt
		bTime := b.LastActiveAt
		if aTime == nil && bTime == nil {
			return 0
		}
		if aTime == nil {
			return -1
		}
		if bTime == nil {
			return 1
		}
		if aTime.Before(*bTime) {
			return -1
		} else if aTime.After(*bTime) {
			return 1
		}
		return 0
	case SortFieldName:
		return strings.Compare(a.Name, b.Name)
	case SortFieldState:
		return strings.Compare(a.State.String(), b.State.String())
	case SortFieldPriority:
		return strings.Compare(string(a.Priority), string(b.Priority))
	case SortFieldDuration:
		if a.Stats != nil && b.Stats != nil {
			if a.Stats.TotalDuration < b.Stats.TotalDuration {
				return -1
			} else if a.Stats.TotalDuration > b.Stats.TotalDuration {
				return 1
			}
		}
		return 0
	case SortFieldInputCount:
		if a.Stats != nil && b.Stats != nil {
			return a.Stats.InputCount - b.Stats.InputCount
		}
		return 0
	case SortFieldOutputCount:
		if a.Stats != nil && b.Stats != nil {
			return a.Stats.OutputCount - b.Stats.OutputCount
		}
		return 0
	case SortFieldErrorCount:
		if a.Stats != nil && b.Stats != nil {
			return a.Stats.ErrorCount - b.Stats.ErrorCount
		}
		return 0
	case SortFieldTokensUsed:
		if a.Stats != nil && b.Stats != nil {
			return a.Stats.TotalTokensUsed - b.Stats.TotalTokensUsed
		}
		return 0
	case SortFieldMemoryUsage:
		if a.ResourceUsage != nil && b.ResourceUsage != nil {
			if a.ResourceUsage.PeakMemoryMB < b.ResourceUsage.PeakMemoryMB {
				return -1
			} else if a.ResourceUsage.PeakMemoryMB > b.ResourceUsage.PeakMemoryMB {
				return 1
			}
		}
		return 0
	case SortFieldCPUUsage:
		if a.ResourceUsage != nil && b.ResourceUsage != nil {
			if a.ResourceUsage.PeakCPUPercent < b.ResourceUsage.PeakCPUPercent {
				return -1
			} else if a.ResourceUsage.PeakCPUPercent > b.ResourceUsage.PeakCPUPercent {
				return 1
			}
		}
		return 0
	default:
		return 0
	}
}

// QuickSearch provides a simple search interface for common use cases
func (ss *SessionSearcher) QuickSearch(query string, limit int) (*SearchResult, error) {
	criteria := &SearchCriteria{
		Query:       query,
		QueryFields: []string{"name", "description", "tags", "content"},
	}
	
	options := &SearchOptions{
		Limit: limit,
		Sort: &SortCriteria{
			Field:     SortFieldUpdatedAt,
			Direction: SortDirectionDesc,
		},
	}
	
	return ss.Search(criteria, options)
}

// SearchByTags searches sessions by tags
func (ss *SessionSearcher) SearchByTags(tags []string, matchAll bool) (*SearchResult, error) {
	criteria := &SearchCriteria{}
	
	if matchAll {
		criteria.Tags = tags
	} else {
		criteria.AnyTags = tags
	}
	
	options := &SearchOptions{
		Sort: &SortCriteria{
			Field:     SortFieldUpdatedAt,
			Direction: SortDirectionDesc,
		},
	}
	
	return ss.Search(criteria, options)
}

// SearchByState searches sessions by state
func (ss *SessionSearcher) SearchByState(states []claude.SessionState) (*SearchResult, error) {
	criteria := &SearchCriteria{
		States: states,
	}
	
	options := &SearchOptions{
		Sort: &SortCriteria{
			Field:     SortFieldLastActiveAt,
			Direction: SortDirectionDesc,
		},
	}
	
	return ss.Search(criteria, options)
}

// SearchActiveSessionss returns all active sessions
func (ss *SessionSearcher) SearchActiveSessions() (*SearchResult, error) {
	return ss.SearchByState([]claude.SessionState{claude.SessionStateActive})
}

// SearchRecentSessions returns recently updated sessions
func (ss *SessionSearcher) SearchRecentSessions(since time.Duration, limit int) (*SearchResult, error) {
	cutoff := time.Now().Add(-since)
	
	criteria := &SearchCriteria{
		UpdatedAfter: &cutoff,
	}
	
	options := &SearchOptions{
		Limit: limit,
		Sort: &SortCriteria{
			Field:     SortFieldUpdatedAt,
			Direction: SortDirectionDesc,
		},
	}
	
	return ss.Search(criteria, options)
}