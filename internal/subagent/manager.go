package subagent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"openai-cli/internal/mcp"
)

// TaskType represents different types of tasks that subagents can handle
type TaskType string

const (
	TaskTypeCodeAnalysis    TaskType = "code_analysis"
	TaskTypeCodeReview     TaskType = "code_review"
	TaskTypeCodeSummary    TaskType = "code_summary"
	TaskTypeBugAnalysis    TaskType = "bug_analysis"
	TaskTypeDocSummary     TaskType = "doc_summary"
	TaskTypeDataAnalysis   TaskType = "data_analysis"
	TaskTypeLogAnalysis    TaskType = "log_analysis"
	TaskTypeConfigReview   TaskType = "config_review"
	TaskTypeGeneralSummary TaskType = "general_summary"
)

// SubagentCapability defines what a subagent can do
type SubagentCapability struct {
	TaskType     TaskType `json:"task_type"`
	Languages    []string `json:"languages"`
	MaxTokens    int      `json:"max_tokens"`
	MinTokens    int      `json:"min_tokens"`
	Confidence   float64  `json:"confidence"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// SubagentDefinition defines a subagent's properties and capabilities
type SubagentDefinition struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Version      string                `json:"version"`
	Capabilities []SubagentCapability  `json:"capabilities"`
	Endpoint     string                `json:"endpoint"`
	Status       SubagentStatus        `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// SubagentStatus represents the current status of a subagent
type SubagentStatus string

const (
	StatusActive      SubagentStatus = "active"
	StatusInactive    SubagentStatus = "inactive"
	StatusMaintenance SubagentStatus = "maintenance"
	StatusError       SubagentStatus = "error"
)

// TaskRequest represents a request to process a task
type TaskRequest struct {
	ID          string                 `json:"id"`
	TaskType    TaskType              `json:"task_type"`
	Input       string                `json:"input"`
	Context     []mcp.ContextChunk    `json:"context"`
	Parameters  map[string]interface{} `json:"parameters"`
	MaxTokens   int                   `json:"max_tokens"`
	Priority    int                   `json:"priority"`
	Timeout     time.Duration         `json:"timeout"`
	RequestedBy string                `json:"requested_by"`
	Timestamp   time.Time             `json:"timestamp"`
}

// TaskResponse represents the response from a subagent
type TaskResponse struct {
	ID             string                 `json:"id"`
	RequestID      string                 `json:"request_id"`
	SubagentID     string                 `json:"subagent_id"`
	Summary        string                 `json:"summary"`
	KeyPoints      []string               `json:"key_points"`
	Analysis       string                 `json:"analysis"`
	TokensProcessed int                   `json:"tokens_processed"`
	TokensSaved    int                   `json:"tokens_saved"`
	Confidence     float64                `json:"confidence"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Metadata       map[string]interface{} `json:"metadata"`
	Success        bool                   `json:"success"`
	Error          string                 `json:"error,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// SubagentManager manages the lifecycle and routing of subagents
type SubagentManager struct {
	subagents    map[string]*SubagentDefinition
	mcpClient    *mcp.Client
	router       *TaskRouter
	metrics      *ManagerMetrics
	config       *ManagerConfig
	mu           sync.RWMutex
}

// ManagerConfig contains configuration for the subagent manager
type ManagerConfig struct {
	MaxConcurrentTasks int           `json:"max_concurrent_tasks"`
	DefaultTimeout     time.Duration `json:"default_timeout"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryDelay         time.Duration `json:"retry_delay"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	EnableLoadBalancing bool          `json:"enable_load_balancing"`
	EnableFallback     bool          `json:"enable_fallback"`
}

// ManagerMetrics tracks performance metrics for the manager
type ManagerMetrics struct {
	TotalRequests      int64                    `json:"total_requests"`
	SuccessfulRequests int64                    `json:"successful_requests"`
	FailedRequests     int64                    `json:"failed_requests"`
	TotalTokensSaved   int64                    `json:"total_tokens_saved"`
	AverageProcessingTime time.Duration        `json:"average_processing_time"`
	SubagentPerformance map[string]*SubagentMetrics `json:"subagent_performance"`
	LastUpdated        time.Time                `json:"last_updated"`
}

// SubagentMetrics tracks performance for individual subagents
type SubagentMetrics struct {
	RequestCount       int64         `json:"request_count"`
	SuccessRate        float64       `json:"success_rate"`
	AverageConfidence  float64       `json:"average_confidence"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	TokensSaved        int64         `json:"tokens_saved"`
	LastUsed           time.Time     `json:"last_used"`
}

// TaskRouter handles intelligent routing of tasks to appropriate subagents
type TaskRouter struct {
	manager *SubagentManager
	rules   []RoutingRule
}

// RoutingRule defines how tasks should be routed
type RoutingRule struct {
	Priority    int                    `json:"priority"`
	Conditions  map[string]interface{} `json:"conditions"`
	SubagentID  string                 `json:"subagent_id"`
	Weight      float64                `json:"weight"`
}

// NewSubagentManager creates a new subagent manager
func NewSubagentManager(mcpClient *mcp.Client, config *ManagerConfig) *SubagentManager {
	if config == nil {
		config = &ManagerConfig{
			MaxConcurrentTasks:  10,
			DefaultTimeout:      30 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          1 * time.Second,
			HealthCheckInterval: 5 * time.Minute,
			EnableLoadBalancing: true,
			EnableFallback:      true,
		}
	}

	manager := &SubagentManager{
		subagents: make(map[string]*SubagentDefinition),
		mcpClient: mcpClient,
		config:    config,
		metrics: &ManagerMetrics{
			SubagentPerformance: make(map[string]*SubagentMetrics),
		},
	}

	manager.router = &TaskRouter{
		manager: manager,
		rules:   manager.createDefaultRoutingRules(),
	}

	// Register default subagents
	manager.registerDefaultSubagents()

	// Start background tasks
	go manager.startHealthChecks()
	go manager.startMetricsCollection()

	return manager
}

// RegisterSubagent registers a new subagent
func (sm *SubagentManager) RegisterSubagent(def *SubagentDefinition) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if def.ID == "" {
		return fmt.Errorf("subagent ID cannot be empty")
	}

	def.CreatedAt = time.Now()
	def.UpdatedAt = time.Now()
	def.Status = StatusActive

	sm.subagents[def.ID] = def
	sm.metrics.SubagentPerformance[def.ID] = &SubagentMetrics{}

	return nil
}

// ProcessTask processes a task using the most appropriate subagent
func (sm *SubagentManager) ProcessTask(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	sm.metrics.TotalRequests++
	startTime := time.Now()

	// Route task to appropriate subagent
	subagentID, err := sm.router.RouteTask(request)
	if err != nil {
		sm.metrics.FailedRequests++
		return nil, fmt.Errorf("task routing failed: %w", err)
	}

	// Process with selected subagent
	response, err := sm.processWithSubagent(ctx, subagentID, request)
	if err != nil {
		sm.metrics.FailedRequests++
		
		// Try fallback if enabled
		if sm.config.EnableFallback {
			if fallbackResponse := sm.tryFallback(ctx, request); fallbackResponse != nil {
				sm.metrics.SuccessfulRequests++
				return fallbackResponse, nil
			}
		}
		
		return nil, fmt.Errorf("task processing failed: %w", err)
	}

	// Update metrics
	sm.updateMetrics(subagentID, response, time.Since(startTime))
	sm.metrics.SuccessfulRequests++

	return response, nil
}

// BatchProcessTasks processes multiple tasks concurrently
func (sm *SubagentManager) BatchProcessTasks(ctx context.Context, requests []*TaskRequest) ([]*TaskResponse, error) {
	if len(requests) == 0 {
		return []*TaskResponse{}, nil
	}

	responses := make([]*TaskResponse, len(requests))
	semaphore := make(chan struct{}, sm.config.MaxConcurrentTasks)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, request := range requests {
		wg.Add(1)
		go func(index int, req *TaskRequest) {
			defer wg.Done()
			
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			response, err := sm.ProcessTask(ctx, req)
			
			mu.Lock()
			if err != nil {
				responses[index] = &TaskResponse{
					ID:        fmt.Sprintf("error_%d", index),
					RequestID: req.ID,
					Success:   false,
					Error:     err.Error(),
					Timestamp: time.Now(),
				}
			} else {
				responses[index] = response
			}
			mu.Unlock()
		}(i, request)
	}

	wg.Wait()
	return responses, nil
}

// GetOptimalSubagent finds the best subagent for a given task type
func (sm *SubagentManager) GetOptimalSubagent(taskType TaskType, context []mcp.ContextChunk) (*SubagentDefinition, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	candidates := make([]*SubagentDefinition, 0)
	
	// Find subagents that can handle the task type
	for _, subagent := range sm.subagents {
		if subagent.Status != StatusActive {
			continue
		}
		
		for _, capability := range subagent.Capabilities {
			if capability.TaskType == taskType {
				candidates = append(candidates, subagent)
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no active subagent found for task type: %s", taskType)
	}

	// Score candidates based on performance metrics and context
	scored := sm.scoreSubagents(candidates, taskType, context)
	
	if len(scored) == 0 {
		return candidates[0], nil // Fallback to first candidate
	}

	return scored[0].subagent, nil
}

// GetSubagentMetrics returns performance metrics for a specific subagent
func (sm *SubagentManager) GetSubagentMetrics(subagentID string) (*SubagentMetrics, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics, exists := sm.metrics.SubagentPerformance[subagentID]
	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", subagentID)
	}

	return metrics, nil
}

// GetManagerMetrics returns overall manager metrics
func (sm *SubagentManager) GetManagerMetrics() *ManagerMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *sm.metrics
	return &metrics
}

// ListSubagents returns all registered subagents
func (sm *SubagentManager) ListSubagents() []*SubagentDefinition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subagents := make([]*SubagentDefinition, 0, len(sm.subagents))
	for _, subagent := range sm.subagents {
		subagents = append(subagents, subagent)
	}

	return subagents
}

// UpdateSubagentStatus updates the status of a subagent
func (sm *SubagentManager) UpdateSubagentStatus(subagentID string, status SubagentStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	subagent, exists := sm.subagents[subagentID]
	if !exists {
		return fmt.Errorf("subagent not found: %s", subagentID)
	}

	subagent.Status = status
	subagent.UpdatedAt = time.Now()

	return nil
}

// Private methods

func (sm *SubagentManager) registerDefaultSubagents() {
	// Code analyzer subagent
	codeAnalyzer := &SubagentDefinition{
		ID:          "code_analyzer",
		Name:        "Code Analyzer",
		Description: "Analyzes code for structure, patterns, and potential issues",
		Version:     "1.0.0",
		Capabilities: []SubagentCapability{
			{
				TaskType:       TaskTypeCodeAnalysis,
				Languages:      []string{"go", "python", "javascript", "typescript", "java"},
				MaxTokens:      3000,
				MinTokens:      100,
				Confidence:     0.85,
				ProcessingTime: 2 * time.Second,
			},
			{
				TaskType:       TaskTypeCodeReview,
				Languages:      []string{"go", "python", "javascript", "typescript"},
				MaxTokens:      2500,
				MinTokens:      200,
				Confidence:     0.80,
				ProcessingTime: 3 * time.Second,
			},
		},
		Status: StatusActive,
	}

	// Document summarizer subagent
	docSummarizer := &SubagentDefinition{
		ID:          "doc_summarizer",
		Name:        "Document Summarizer",
		Description: "Summarizes documentation, README files, and text content",
		Version:     "1.0.0",
		Capabilities: []SubagentCapability{
			{
				TaskType:       TaskTypeDocSummary,
				Languages:      []string{"markdown", "text", "rst"},
				MaxTokens:      2000,
				MinTokens:      50,
				Confidence:     0.90,
				ProcessingTime: 1 * time.Second,
			},
		},
		Status: StatusActive,
	}

	// Data processor subagent
	dataProcessor := &SubagentDefinition{
		ID:          "data_processor",
		Name:        "Data Processor",
		Description: "Processes and analyzes structured data, logs, and configurations",
		Version:     "1.0.0",
		Capabilities: []SubagentCapability{
			{
				TaskType:       TaskTypeDataAnalysis,
				Languages:      []string{"json", "yaml", "csv", "xml"},
				MaxTokens:      1500,
				MinTokens:      20,
				Confidence:     0.75,
				ProcessingTime: 1500 * time.Millisecond,
			},
			{
				TaskType:       TaskTypeLogAnalysis,
				Languages:      []string{"log", "text"},
				MaxTokens:      2000,
				MinTokens:      100,
				Confidence:     0.70,
				ProcessingTime: 2 * time.Second,
			},
		},
		Status: StatusActive,
	}

	sm.RegisterSubagent(codeAnalyzer)
	sm.RegisterSubagent(docSummarizer)
	sm.RegisterSubagent(dataProcessor)
}

func (sm *SubagentManager) processWithSubagent(ctx context.Context, subagentID string, request *TaskRequest) (*TaskResponse, error) {
	_, exists := sm.subagents[subagentID]
	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", subagentID)
	}

	// Prepare MCP subagent call parameters
	params := mcp.SubagentCallParams{
		SubagentID: subagentID,
		TaskType:   string(request.TaskType),
		Input:      request.Input,
		Context:    request.Context,
		Parameters: request.Parameters,
		MaxTokens:  request.MaxTokens,
		Timeout:    int(sm.config.DefaultTimeout.Seconds()),
	}

	traceID := fmt.Sprintf("subagent_%s_%s", subagentID, request.ID)
	
	// Make MCP call with timeout
	callCtx, cancel := context.WithTimeout(ctx, sm.config.DefaultTimeout)
	defer cancel()

	mcpResponse, err := sm.mcpClient.CallSubagent(callCtx, params, traceID)
	if err != nil {
		return nil, fmt.Errorf("MCP subagent call failed: %w", err)
	}

	// Convert MCP response to TaskResponse
	response := &TaskResponse{
		ID:             mcpResponse.ID,
		RequestID:      request.ID,
		SubagentID:     subagentID,
		Summary:        mcpResponse.Summary,
		KeyPoints:      mcpResponse.KeyPoints,
		Analysis:       mcpResponse.Analysis,
		TokensProcessed: mcpResponse.TokensProcessed,
		TokensSaved:    mcpResponse.TokensSaved,
		Confidence:     mcpResponse.Confidence,
		ProcessingTime: mcpResponse.ProcessingTime,
		Metadata:       mcpResponse.Metadata,
		Success:        mcpResponse.Success,
		Error:          mcpResponse.Error,
		Timestamp:      time.Now(),
	}

	return response, nil
}

func (sm *SubagentManager) tryFallback(ctx context.Context, request *TaskRequest) *TaskResponse {
	// Simple fallback: return a basic summary
	summary := sm.createBasicSummary(request.Input)
	
	return &TaskResponse{
		ID:             fmt.Sprintf("fallback_%s", request.ID),
		RequestID:      request.ID,
		SubagentID:     "fallback",
		Summary:        summary,
		KeyPoints:      []string{"Processed with fallback mechanism"},
		TokensProcessed: len(request.Input) / 4,
		TokensSaved:    0,
		Confidence:     0.5,
		ProcessingTime: 100 * time.Millisecond,
		Success:        true,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"fallback": true,
		},
	}
}

func (sm *SubagentManager) createBasicSummary(input string) string {
	// Very basic summarization
	lines := strings.Split(input, "\n")
	if len(lines) <= 3 {
		return input
	}
	
	return fmt.Sprintf("Content summary: %d lines, starting with: %s", 
		len(lines), truncate(lines[0], 100))
}

func (sm *SubagentManager) updateMetrics(subagentID string, response *TaskResponse, duration time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update manager metrics
	sm.metrics.TotalTokensSaved += int64(response.TokensSaved)
	sm.metrics.LastUpdated = time.Now()

	// Update subagent metrics
	subagentMetrics, exists := sm.metrics.SubagentPerformance[subagentID]
	if !exists {
		subagentMetrics = &SubagentMetrics{}
		sm.metrics.SubagentPerformance[subagentID] = subagentMetrics
	}

	subagentMetrics.RequestCount++
	subagentMetrics.TokensSaved += int64(response.TokensSaved)
	subagentMetrics.LastUsed = time.Now()

	// Update success rate
	if response.Success {
		subagentMetrics.SuccessRate = (subagentMetrics.SuccessRate*float64(subagentMetrics.RequestCount-1) + 1.0) / float64(subagentMetrics.RequestCount)
	} else {
		subagentMetrics.SuccessRate = (subagentMetrics.SuccessRate * float64(subagentMetrics.RequestCount-1)) / float64(subagentMetrics.RequestCount)
	}

	// Update average confidence
	subagentMetrics.AverageConfidence = (subagentMetrics.AverageConfidence*float64(subagentMetrics.RequestCount-1) + response.Confidence) / float64(subagentMetrics.RequestCount)

	// Update average processing time
	subagentMetrics.AverageProcessingTime = time.Duration((int64(subagentMetrics.AverageProcessingTime)*int64(subagentMetrics.RequestCount-1) + int64(duration)) / int64(subagentMetrics.RequestCount))
}

type scoredSubagent struct {
	subagent *SubagentDefinition
	score    float64
}

func (sm *SubagentManager) scoreSubagents(candidates []*SubagentDefinition, taskType TaskType, context []mcp.ContextChunk) []scoredSubagent {
	scored := make([]scoredSubagent, len(candidates))
	
	for i, candidate := range candidates {
		score := sm.calculateSubagentScore(candidate, taskType, context)
		scored[i] = scoredSubagent{
			subagent: candidate,
			score:    score,
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored
}

func (sm *SubagentManager) calculateSubagentScore(subagent *SubagentDefinition, taskType TaskType, context []mcp.ContextChunk) float64 {
	score := 0.0

	// Base capability score
	for _, capability := range subagent.Capabilities {
		if capability.TaskType == taskType {
			score += capability.Confidence * 0.4 // 40% weight for confidence
			break
		}
	}

	// Performance metrics score
	if metrics, exists := sm.metrics.SubagentPerformance[subagent.ID]; exists {
		score += metrics.SuccessRate * 0.3        // 30% weight for success rate
		score += metrics.AverageConfidence * 0.2  // 20% weight for historical confidence
		
		// Penalize slow subagents
		if metrics.AverageProcessingTime > 5*time.Second {
			score -= 0.1
		}
	}

	// Context relevance score
	contextScore := sm.calculateContextRelevance(subagent, context)
	score += contextScore * 0.1 // 10% weight for context relevance

	return score
}

func (sm *SubagentManager) calculateContextRelevance(subagent *SubagentDefinition, context []mcp.ContextChunk) float64 {
	if len(context) == 0 {
		return 0.5 // Neutral score for no context
	}

	relevanceScore := 0.0
	totalChunks := float64(len(context))

	for _, chunk := range context {
		for _, capability := range subagent.Capabilities {
			for _, tag := range chunk.Tags {
				for _, lang := range capability.Languages {
					if strings.EqualFold(tag, lang) {
						relevanceScore += 1.0
						break
					}
				}
			}
		}
	}

	return relevanceScore / totalChunks
}

func (sm *SubagentManager) createDefaultRoutingRules() []RoutingRule {
	return []RoutingRule{
		{
			Priority: 1,
			Conditions: map[string]interface{}{
				"task_type": TaskTypeCodeAnalysis,
				"has_tags":  []string{"go", "python", "javascript"},
			},
			SubagentID: "code_analyzer",
			Weight:     1.0,
		},
		{
			Priority: 2,
			Conditions: map[string]interface{}{
				"task_type": TaskTypeDocSummary,
				"has_tags":  []string{"markdown", "documentation"},
			},
			SubagentID: "doc_summarizer",
			Weight:     1.0,
		},
		{
			Priority: 3,
			Conditions: map[string]interface{}{
				"task_type": TaskTypeDataAnalysis,
				"has_tags":  []string{"json", "yaml", "csv"},
			},
			SubagentID: "data_processor",
			Weight:     1.0,
		},
	}
}

func (sm *SubagentManager) startHealthChecks() {
	ticker := time.NewTicker(sm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.performHealthChecks()
		}
	}
}

func (sm *SubagentManager) performHealthChecks() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, subagent := range sm.subagents {
		// Simple health check: mark as error if not used recently
		if metrics, exists := sm.metrics.SubagentPerformance[id]; exists {
			if time.Since(metrics.LastUsed) > 24*time.Hour && subagent.Status == StatusActive {
				subagent.Status = StatusInactive
				subagent.UpdatedAt = time.Now()
			}
		}
	}
}

func (sm *SubagentManager) startMetricsCollection() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.updateAggregateMetrics()
		}
	}
}

func (sm *SubagentManager) updateAggregateMetrics() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Calculate average processing time across all subagents
	totalTime := time.Duration(0)
	count := int64(0)

	for _, metrics := range sm.metrics.SubagentPerformance {
		totalTime += metrics.AverageProcessingTime * time.Duration(metrics.RequestCount)
		count += metrics.RequestCount
	}

	if count > 0 {
		sm.metrics.AverageProcessingTime = totalTime / time.Duration(count)
	}

	sm.metrics.LastUpdated = time.Now()
}

// RouteTask routes a task to the most appropriate subagent
func (tr *TaskRouter) RouteTask(request *TaskRequest) (string, error) {
	// Find the best subagent using the routing rules and scoring
	subagent, err := tr.manager.GetOptimalSubagent(request.TaskType, request.Context)
	if err != nil {
		return "", err
	}

	return subagent.ID, nil
}

// Utility functions
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}