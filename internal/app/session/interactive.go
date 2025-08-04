package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"openai-cli/internal/app/claude"
)

// InteractionResponse represents a response from an interactive session
type InteractionResponse struct {
	ID        string                 `json:"id"`         // Unique response ID
	SessionID string                 `json:"session_id"` // Associated session ID
	Type      ResponseType           `json:"type"`       // Response type
	Content   string                 `json:"content"`    // Response content
	Metadata  map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
	Timestamp time.Time              `json:"timestamp"`  // Response timestamp
	Duration  time.Duration          `json:"duration"`   // Processing duration
	Error     error                  `json:"error,omitempty"` // Error if any
	Tokens    int                    `json:"tokens,omitempty"` // Token count if applicable
}

// ResponseType represents the type of interaction response
type ResponseType string

const (
	ResponseTypeOutput    ResponseType = "output"
	ResponseTypeError     ResponseType = "error"
	ResponseTypeComplete  ResponseType = "complete"
	ResponseTypePartial   ResponseType = "partial"
	ResponseTypeSystem    ResponseType = "system"
	ResponseTypeStatus    ResponseType = "status"
)

// InteractionContext provides context for interactive session operations
type InteractionContext struct {
	SessionID   string
	RequestID   string
	Timeout     time.Duration
	Metadata    map[string]interface{}
	CancelFunc  context.CancelFunc
	ResponseCh  chan *InteractionResponse
	StartTime   time.Time
}

// NewInteractionContext creates a new interaction context
func NewInteractionContext(sessionID string, timeout time.Duration) *InteractionContext {
	_, cancel := context.WithTimeout(context.Background(), timeout)
	
	return &InteractionContext{
		SessionID:  sessionID,
		RequestID:  fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Timeout:    timeout,
		Metadata:   make(map[string]interface{}),
		CancelFunc: cancel,
		ResponseCh: make(chan *InteractionResponse, 100), // Buffered channel
		StartTime:  time.Now(),
	}
}

// InteractiveController manages interactive session operations
type InteractiveController struct {
	manager        *Manager
	logger         *zap.Logger
	mu             sync.RWMutex
	activeRequests map[string]*InteractionContext // RequestID -> Context
	historyManager *HistoryManager
}

// NewInteractiveController creates a new interactive controller
func NewInteractiveController(manager *Manager, historyManager *HistoryManager, logger *zap.Logger) *InteractiveController {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &InteractiveController{
		manager:        manager,
		logger:         logger,
		activeRequests: make(map[string]*InteractionContext),
		historyManager: historyManager,
	}
}

// SendInteractiveInput sends input to a session with real-time response handling
func (ic *InteractiveController) SendInteractiveInput(ctx context.Context, sessionID, input string, options *InteractionOptions) (*InteractionContext, error) {
	// Validate session exists and is active
	session, err := ic.manager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if !session.IsActive() {
		return nil, fmt.Errorf("session is not active: %s", session.GetState().String())
	}

	// Create interaction context
	timeout := time.Minute * 5 // Default timeout
	if options != nil && options.Timeout > 0 {
		timeout = options.Timeout
	}

	interactionCtx := NewInteractionContext(sessionID, timeout)
	
	// Add metadata if provided
	if options != nil && options.Metadata != nil {
		for k, v := range options.Metadata {
			interactionCtx.Metadata[k] = v
		}
	}

	// Store active request
	ic.mu.Lock()
	ic.activeRequests[interactionCtx.RequestID] = interactionCtx
	ic.mu.Unlock()

	// Log input to conversation history
	if ic.historyManager != nil {
		history := ic.historyManager.GetHistory(sessionID)
		history.AddInputEntry(input, map[string]interface{}{
			"request_id": interactionCtx.RequestID,
			"timestamp":  interactionCtx.StartTime,
		})
	}

	// Start monitoring output in background
	go ic.monitorSessionOutput(interactionCtx, session)

	// Send input to session
	if err := ic.manager.SendInput(sessionID, input); err != nil {
		// Clean up on send failure
		ic.cleanupRequest(interactionCtx.RequestID)
		return nil, fmt.Errorf("failed to send input: %w", err)
	}

	ic.logger.Debug("interactive input sent",
		zap.String("session_id", sessionID),
		zap.String("request_id", interactionCtx.RequestID),
		zap.Int("input_length", len(input)))

	return interactionCtx, nil
}

// InteractionOptions provides options for interactive operations
type InteractionOptions struct {
	Timeout          time.Duration          `json:"timeout"`
	Metadata         map[string]interface{} `json:"metadata"`
	StreamResponse   bool                   `json:"stream_response"`
	CollectComplete  bool                   `json:"collect_complete"`
	MaxResponseSize  int                    `json:"max_response_size"`
	ResponseFilters  []string               `json:"response_filters"`
}

// monitorSessionOutput monitors session output and sends responses
func (ic *InteractiveController) monitorSessionOutput(interactionCtx *InteractionContext, session *claude.Session) {
	defer ic.cleanupRequest(interactionCtx.RequestID)
	defer close(interactionCtx.ResponseCh)

	// Subscribe to session output
	outputCh, err := session.SubscribeToOutput()
	if err != nil {
		ic.sendErrorResponse(interactionCtx, fmt.Errorf("failed to subscribe to output: %w", err))
		return
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), interactionCtx.Timeout)
	defer cancel()

	var completeOutput []byte
	responseCount := 0

	for {
		select {
		case output, ok := <-outputCh:
			if !ok {
				// Output channel closed, send final response
				if len(completeOutput) > 0 {
					ic.sendCompleteResponse(interactionCtx, string(completeOutput), responseCount)
				}
				return
			}

			completeOutput = append(completeOutput, output...)
			responseCount++

			// Send streaming response
			response := &InteractionResponse{
				ID:        fmt.Sprintf("%s-resp-%d", interactionCtx.RequestID, responseCount),
				SessionID: interactionCtx.SessionID,
				Type:      ResponseTypePartial,
				Content:   string(output),
				Timestamp: time.Now(),
				Duration:  time.Since(interactionCtx.StartTime),
			}

			// Send response non-blocking
			select {
			case interactionCtx.ResponseCh <- response:
			default:
				ic.logger.Warn("response channel full, dropping response",
					zap.String("session_id", interactionCtx.SessionID),
					zap.String("request_id", interactionCtx.RequestID))
			}

		case <-ctx.Done():
			// Timeout or cancellation
			ic.sendErrorResponse(interactionCtx, ctx.Err())
			return
		}
	}
}

// sendCompleteResponse sends a complete response
func (ic *InteractiveController) sendCompleteResponse(interactionCtx *InteractionContext, content string, responseCount int) {
	response := &InteractionResponse{
		ID:        fmt.Sprintf("%s-complete", interactionCtx.RequestID),
		SessionID: interactionCtx.SessionID,
		Type:      ResponseTypeComplete,
		Content:   content,
		Timestamp: time.Now(),
		Duration:  time.Since(interactionCtx.StartTime),
		Metadata: map[string]interface{}{
			"response_count": responseCount,
			"total_length":   len(content),
		},
	}

	// Log output to conversation history
	if ic.historyManager != nil {
		history := ic.historyManager.GetHistory(interactionCtx.SessionID)
		history.AddOutputEntry(content, map[string]interface{}{
			"request_id":     interactionCtx.RequestID,
			"response_count": responseCount,
			"duration":       response.Duration,
		})
	}

	select {
	case interactionCtx.ResponseCh <- response:
	default:
		ic.logger.Error("failed to send complete response, channel full",
			zap.String("session_id", interactionCtx.SessionID),
			zap.String("request_id", interactionCtx.RequestID))
	}
}

// sendErrorResponse sends an error response
func (ic *InteractiveController) sendErrorResponse(interactionCtx *InteractionContext, err error) {
	response := &InteractionResponse{
		ID:        fmt.Sprintf("%s-error", interactionCtx.RequestID),
		SessionID: interactionCtx.SessionID,
		Type:      ResponseTypeError,
		Content:   fmt.Sprintf("Error: %v", err),
		Timestamp: time.Now(),
		Duration:  time.Since(interactionCtx.StartTime),
		Error:     err,
	}

	// Log error to conversation history
	if ic.historyManager != nil {
		history := ic.historyManager.GetHistory(interactionCtx.SessionID)
		history.AddErrorEntry(response.Content, err, map[string]interface{}{
			"request_id": interactionCtx.RequestID,
			"duration":   response.Duration,
		})
	}

	select {
	case interactionCtx.ResponseCh <- response:
	default:
		ic.logger.Error("failed to send error response, channel full",
			zap.String("session_id", interactionCtx.SessionID),
			zap.String("request_id", interactionCtx.RequestID),
			zap.Error(err))
	}
}

// ExecuteCommand executes a command in a session and waits for completion
func (ic *InteractiveController) ExecuteCommand(ctx context.Context, sessionID, command string, options *InteractionOptions) (*InteractionResponse, error) {
	interactionCtx, err := ic.SendInteractiveInput(ctx, sessionID, command, options)
	if err != nil {
		return nil, err
	}

	// Wait for complete response or timeout
	timeout := time.Minute * 5
	if options != nil && options.Timeout > 0 {
		timeout = options.Timeout
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var lastResponse *InteractionResponse

	for {
		select {
		case response, ok := <-interactionCtx.ResponseCh:
			if !ok {
				// Channel closed
				if lastResponse != nil && lastResponse.Type == ResponseTypeComplete {
					return lastResponse, nil
				}
				return nil, fmt.Errorf("response channel closed unexpectedly")
			}

			lastResponse = response

			// Return on complete or error
			if response.Type == ResponseTypeComplete || response.Type == ResponseTypeError {
				return response, response.Error
			}

		case <-timer.C:
			ic.CancelRequest(interactionCtx.RequestID)
			return nil, fmt.Errorf("command execution timeout after %v", timeout)

		case <-ctx.Done():
			ic.CancelRequest(interactionCtx.RequestID)
			return nil, ctx.Err()
		}
	}
}

// StreamResponse streams responses from an active interaction
func (ic *InteractiveController) StreamResponse(requestID string) (<-chan *InteractionResponse, error) {
	ic.mu.RLock()
	interactionCtx, exists := ic.activeRequests[requestID]
	ic.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("request not found: %s", requestID)
	}

	return interactionCtx.ResponseCh, nil
}

// CancelRequest cancels an active interaction request
func (ic *InteractiveController) CancelRequest(requestID string) error {
	ic.mu.RLock()
	interactionCtx, exists := ic.activeRequests[requestID]
	ic.mu.RUnlock()

	if !exists {
		return fmt.Errorf("request not found: %s", requestID)
	}

	// Cancel the context
	if interactionCtx.CancelFunc != nil {
		interactionCtx.CancelFunc()
	}

	ic.logger.Info("interaction request cancelled",
		zap.String("session_id", interactionCtx.SessionID),
		zap.String("request_id", requestID))

	return nil
}

// GetActiveRequests returns a list of active interaction requests
func (ic *InteractiveController) GetActiveRequests() []string {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	requests := make([]string, 0, len(ic.activeRequests))
	for requestID := range ic.activeRequests {
		requests = append(requests, requestID)
	}

	return requests
}

// GetRequestInfo returns information about an active request
func (ic *InteractiveController) GetRequestInfo(requestID string) (*InteractionContext, error) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	interactionCtx, exists := ic.activeRequests[requestID]
	if !exists {
		return nil, fmt.Errorf("request not found: %s", requestID)
	}

	// Return a copy to prevent external modification
	info := &InteractionContext{
		SessionID: interactionCtx.SessionID,
		RequestID: interactionCtx.RequestID,
		Timeout:   interactionCtx.Timeout,
		StartTime: interactionCtx.StartTime,
		Metadata:  make(map[string]interface{}),
	}

	// Copy metadata
	for k, v := range interactionCtx.Metadata {
		info.Metadata[k] = v
	}

	return info, nil
}

// CleanupExpiredRequests removes expired interaction requests
func (ic *InteractiveController) CleanupExpiredRequests() int {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	var expired []string
	now := time.Now()

	for requestID, interactionCtx := range ic.activeRequests {
		if now.Sub(interactionCtx.StartTime) > interactionCtx.Timeout {
			expired = append(expired, requestID)
		}
	}

	for _, requestID := range expired {
		if interactionCtx, exists := ic.activeRequests[requestID]; exists {
			if interactionCtx.CancelFunc != nil {
				interactionCtx.CancelFunc()
			}
			delete(ic.activeRequests, requestID)
		}
	}

	if len(expired) > 0 {
		ic.logger.Info("cleaned up expired interaction requests",
			zap.Int("count", len(expired)))
	}

	return len(expired)
}

// cleanupRequest removes a request from active requests
func (ic *InteractiveController) cleanupRequest(requestID string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if interactionCtx, exists := ic.activeRequests[requestID]; exists {
		if interactionCtx.CancelFunc != nil {
			interactionCtx.CancelFunc()
		}
		delete(ic.activeRequests, requestID)
	}
}

// SendSystemMessage sends a system message to a session
func (ic *InteractiveController) SendSystemMessage(sessionID, message string) error {
	// Log system message to conversation history
	if ic.historyManager != nil {
		history := ic.historyManager.GetHistory(sessionID)
		history.AddSystemEntry(message, map[string]interface{}{
			"timestamp": time.Now(),
			"type":      "system_message",
		})
	}

	ic.logger.Info("system message sent to session",
		zap.String("session_id", sessionID),
		zap.String("message", message))

	return nil
}

// GetInteractionHistory returns the interaction history for a session
func (ic *InteractiveController) GetInteractionHistory(sessionID string, limit int) ([]*ConversationEntry, error) {
	if ic.historyManager == nil {
		return nil, fmt.Errorf("history manager not available")
	}

	history := ic.historyManager.GetHistory(sessionID)
	entries := history.GetEntries()

	// Apply limit if specified
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// GetInteractionStats returns interaction statistics for a session
func (ic *InteractiveController) GetInteractionStats(sessionID string) (*ConversationStats, error) {
	if ic.historyManager == nil {
		return nil, fmt.Errorf("history manager not available")
	}

	history := ic.historyManager.GetHistory(sessionID)
	return history.GetStats(), nil
}

// Close shuts down the interactive controller
func (ic *InteractiveController) Close() error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Cancel all active requests
	for requestID, interactionCtx := range ic.activeRequests {
		if interactionCtx.CancelFunc != nil {
			interactionCtx.CancelFunc()
		}
		delete(ic.activeRequests, requestID)
	}

	ic.logger.Info("interactive controller closed")
	return nil
}