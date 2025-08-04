package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"openai-cli/internal/prompt"
)

// PromptMCPServer provides MCP server capabilities for prompt operations
type PromptMCPServer struct {
	handler    *PromptMCPHandler
	logger     *zap.Logger
	
	// Server configuration
	host       string
	port       int
	enableWS   bool
	enableAuth bool
	
	// WebSocket connections (simplified for this implementation)
	wsClients  map[string]bool // Using string keys instead of conn pointers
	clientsMux sync.RWMutex
	
	// Event broadcasting
	eventChan chan *MCPEvent
	shutdown  chan struct{}
	
	// Metrics
	metrics *ServerMetrics
}

// MCPEvent represents an event to broadcast to clients
type MCPEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ServerMetrics tracks server performance metrics
type ServerMetrics struct {
	StartTime        time.Time
	RequestCount     int64
	ErrorCount       int64
	ActiveClients    int64
	LastRequestTime  time.Time
	AverageResponse  time.Duration
	mutex           sync.RWMutex
}

// NewPromptMCPServer creates a new MCP server for prompt operations
func NewPromptMCPServer(promptManager prompt.Manager, logger *zap.Logger) *PromptMCPServer {
	handler := NewPromptMCPHandler(promptManager, logger)
	
	server := &PromptMCPServer{
		handler:    handler,
		logger:     logger,
		host:       "localhost",
		port:       8081,
		enableWS:   true,
		enableAuth: false,
		wsClients: make(map[string]bool),
		eventChan: make(chan *MCPEvent, 100),
		shutdown:  make(chan struct{}),
		metrics: &ServerMetrics{
			StartTime: time.Now(),
		},
	}
	
	// Start event broadcaster
	go server.eventBroadcaster()
	
	return server
}

// SetConfiguration sets server configuration
func (pms *PromptMCPServer) SetConfiguration(host string, port int, enableWS, enableAuth bool) {
	pms.host = host
	pms.port = port
	pms.enableWS = enableWS
	pms.enableAuth = enableAuth
	
	pms.logger.Info("MCP server configuration updated",
		zap.String("host", host),
		zap.Int("port", port),
		zap.Bool("websocket", enableWS),
		zap.Bool("auth", enableAuth))
}

// Start starts the MCP server
func (pms *PromptMCPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// Register HTTP endpoints
	pms.registerHTTPRoutes(mux)
	
	// Add middleware
	var handler http.Handler = mux
	handler = pms.metricsMiddleware(handler)
	handler = pms.corsMiddleware(handler)
	handler = pms.handler.WithLogging(handler)
	
	if pms.enableAuth {
		handler = pms.handler.WithAuth(handler)
	}
	
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", pms.host, pms.port),
		Handler: handler,
	}
	
	pms.logger.Info("Starting MCP server",
		zap.String("address", server.Addr),
		zap.Bool("websocket", pms.enableWS),
		zap.Bool("auth", pms.enableAuth))
	
	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pms.logger.Error("MCP server error", zap.Error(err))
		}
	}()
	
	// Wait for context cancellation
	<-ctx.Done()
	
	pms.logger.Info("Shutting down MCP server")
	close(pms.shutdown)
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return server.Shutdown(shutdownCtx)
}

// registerHTTPRoutes registers all HTTP routes
func (pms *PromptMCPServer) registerHTTPRoutes(mux *http.ServeMux) {
	// Main MCP endpoint
	mux.HandleFunc("/mcp", pms.handler.ServeHTTP)
	
	// Health check
	mux.HandleFunc("/health", pms.handleHealth)
	
	// Metrics endpoint
	mux.HandleFunc("/metrics", pms.handleMetrics)
	
	// Server info
	mux.HandleFunc("/info", pms.handleInfo)
	
	// WebSocket endpoint
	if pms.enableWS {
		mux.HandleFunc("/ws", pms.handleWebSocket)
	}
	
	// Static file serving for docs (optional)
	mux.HandleFunc("/docs/", pms.handleDocs)
}

// handleHealth handles health check requests
func (pms *PromptMCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(pms.metrics.StartTime).String(),
		"version":   "1.0.0",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleMetrics handles metrics requests
func (pms *PromptMCPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	pms.metrics.mutex.RLock()
	metrics := map[string]interface{}{
		"start_time":        pms.metrics.StartTime,
		"uptime":           time.Since(pms.metrics.StartTime).String(),
		"request_count":    pms.metrics.RequestCount,
		"error_count":      pms.metrics.ErrorCount,
		"active_clients":   pms.metrics.ActiveClients,
		"last_request":     pms.metrics.LastRequestTime,
		"average_response": pms.metrics.AverageResponse.String(),
		"websocket_enabled": pms.enableWS,
		"auth_enabled":     pms.enableAuth,
	}
	pms.metrics.mutex.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleInfo handles server info requests
func (pms *PromptMCPServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	info := map[string]interface{}{
		"name":        "Prompt MCP Server",
		"version":     "1.0.0",
		"description": "Model Context Protocol server for prompt template operations",
		"capabilities": map[string]interface{}{
			"resources": []string{
				"prompt://templates/*",
			},
			"tools": []string{
				"generate_prompt",
				"validate_template",
				"detect_context",
				"suggest_templates",
				"get_history",
				"template_stats",
				"workspace_analysis",
				"quality_check",
			},
			"ai_integration": []string{
				"claude",
				"gpt",
				"local",
			},
		},
		"endpoints": map[string]string{
			"mcp":     "/mcp",
			"health":  "/health",
			"metrics": "/metrics",
			"info":    "/info",
			"ws":      "/ws",
		},
		"features": []string{
			"template_resources",
			"prompt_generation",
			"context_detection",
			"ai_integration",
			"real_time_updates",
			"metrics_tracking",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleWebSocket handles WebSocket connections (placeholder implementation)
func (pms *PromptMCPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !pms.enableWS {
		http.Error(w, "WebSocket not enabled", http.StatusNotFound)
		return
	}
	
	// This is a placeholder implementation
	// In a real implementation, you would upgrade to WebSocket here
	pms.logger.Info("WebSocket endpoint called",
		zap.String("remote_addr", r.RemoteAddr))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "WebSocket endpoint - upgrade to WebSocket protocol required",
		"status":  "placeholder",
	})
}

// handleWebSocketMessage handles individual WebSocket messages (placeholder)
func (pms *PromptMCPServer) handleWebSocketMessage(clientID string, message map[string]interface{}) {
	// This would handle WebSocket messages in a real implementation
	pms.logger.Debug("WebSocket message received", 
		zap.String("client", clientID),
		zap.Any("message", message))
}

// handleDocs handles documentation requests
func (pms *PromptMCPServer) handleDocs(w http.ResponseWriter, r *http.Request) {
	// Simple documentation page
	docs := `
<!DOCTYPE html>
<html>
<head>
    <title>Prompt MCP Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        pre { background: #f5f5f5; padding: 10px; border-radius: 5px; }
        .endpoint { margin: 20px 0; }
    </style>
</head>
<body>
    <h1>Prompt MCP Server</h1>
    <p>Model Context Protocol server for prompt template operations</p>
    
    <h2>Endpoints</h2>
    <div class="endpoint">
        <h3>POST /mcp</h3>
        <p>Main MCP JSON-RPC endpoint</p>
    </div>
    
    <div class="endpoint">
        <h3>GET /health</h3>
        <p>Health check endpoint</p>
    </div>
    
    <div class="endpoint">
        <h3>GET /metrics</h3>
        <p>Server metrics and statistics</p>
    </div>
    
    <div class="endpoint">
        <h3>GET /info</h3>
        <p>Server information and capabilities</p>
    </div>
    
    <div class="endpoint">
        <h3>WS /ws</h3>
        <p>WebSocket endpoint for real-time communication</p>
    </div>
    
    <h2>Example MCP Request</h2>
    <pre>
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "tools/call",
  "params": {
    "name": "generate_prompt",
    "arguments": {
      "template_name": "golang-function",
      "parameters": {
        "function_name": "calculateSum",
        "description": "Calculate sum of two integers"
      }
    }
  }
}
    </pre>
</body>
</html>
`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(docs))
}

// Middleware

// metricsMiddleware tracks request metrics
func (pms *PromptMCPServer) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		pms.metrics.mutex.Lock()
		pms.metrics.RequestCount++
		pms.metrics.LastRequestTime = start
		pms.metrics.mutex.Unlock()
		
		next.ServeHTTP(w, r)
		
		duration := time.Since(start)
		pms.metrics.mutex.Lock()
		pms.metrics.AverageResponse = (pms.metrics.AverageResponse + duration) / 2
		pms.metrics.mutex.Unlock()
	})
}

// corsMiddleware adds CORS headers
func (pms *PromptMCPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Event handling

// eventBroadcaster broadcasts events to all WebSocket clients
func (pms *PromptMCPServer) eventBroadcaster() {
	for {
		select {
		case event := <-pms.eventChan:
			pms.broadcastEvent(event)
		case <-pms.shutdown:
			return
		}
	}
}

// broadcastEvent sends an event to all connected WebSocket clients (placeholder)
func (pms *PromptMCPServer) broadcastEvent(event *MCPEvent) {
	pms.clientsMux.RLock()
	defer pms.clientsMux.RUnlock()
	
	pms.logger.Debug("Broadcasting event to clients", 
		zap.String("type", event.Type),
		zap.Int("client_count", len(pms.wsClients)))
	
	// In a real implementation, this would send to actual WebSocket connections
	for clientID := range pms.wsClients {
		pms.logger.Debug("Event would be sent to client", zap.String("client", clientID))
	}
}

// sendToClient sends a message to a specific WebSocket client (placeholder)
func (pms *PromptMCPServer) sendToClient(clientID string, message interface{}) {
	pms.logger.Debug("Sending message to client",
		zap.String("client", clientID),
		zap.Any("message", message))
}

// sendError sends an error message to a WebSocket client (placeholder)
func (pms *PromptMCPServer) sendError(clientID string, message string, code int, id string) {
	pms.logger.Debug("Sending error to client",
		zap.String("client", clientID),
		zap.String("error", message),
		zap.Int("code", code))
}

// Public methods for event emission

// EmitTemplateEvent emits a template-related event
func (pms *PromptMCPServer) EmitTemplateEvent(eventType, templateName string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["template"] = templateName
	
	event := &MCPEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
	
	select {
	case pms.eventChan <- event:
	default:
		pms.logger.Warn("Event channel full, dropping event", zap.String("type", eventType))
	}
}

// EmitGenerationEvent emits a prompt generation event
func (pms *PromptMCPServer) EmitGenerationEvent(templateName string, success bool, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["template"] = templateName
	data["success"] = success
	
	eventType := "generation_success"
	if !success {
		eventType = "generation_error"
	}
	
	pms.EmitTemplateEvent(eventType, templateName, data)
}

// EmitContextEvent emits a context detection event
func (pms *PromptMCPServer) EmitContextEvent(context map[string]interface{}) {
	event := &MCPEvent{
		Type:      "context_detected",
		Timestamp: time.Now(),
		Data:      context,
	}
	
	select {
	case pms.eventChan <- event:
	default:
		pms.logger.Warn("Event channel full, dropping context event")
	}
}

// GetMetrics returns current server metrics
func (pms *PromptMCPServer) GetMetrics() *ServerMetrics {
	pms.metrics.mutex.RLock()
	defer pms.metrics.mutex.RUnlock()
	
	// Return a copy
	return &ServerMetrics{
		StartTime:       pms.metrics.StartTime,
		RequestCount:    pms.metrics.RequestCount,
		ErrorCount:      pms.metrics.ErrorCount,
		ActiveClients:   pms.metrics.ActiveClients,
		LastRequestTime: pms.metrics.LastRequestTime,
		AverageResponse: pms.metrics.AverageResponse,
	}
}