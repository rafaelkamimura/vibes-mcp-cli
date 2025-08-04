package main

import (
	"context"
	"log"
	"time"

	"openai-cli/internal/context"
	"openai-cli/internal/integration"
	"openai-cli/internal/mcp"
	"openai-cli/internal/metrics"
	"openai-cli/internal/subagent"
	"openai-cli/internal/ui"
)

// Example implementation showing how to integrate all context optimization components
func main() {
	// Configuration
	agentBackendURL := "http://localhost:8000"
	authToken := "your-jwt-token-here"
	userID := "user123"
	sessionID := "session456"

	// Initialize MCP client
	mcpClient := mcp.NewClient(agentBackendURL, authToken)

	// Initialize context optimizer with strategy
	optimizationStrategy := context.OptimizationStrategy{
		MaxTokens:         4000,
		SummaryThreshold:  1000,
		CompressionRatio:  0.7,
		RetainRecent:      5,
		PriorityWeighting: true,
	}
	
	contextOptimizer := context.NewContextOptimizer(mcpClient, optimizationStrategy)

	// Initialize subagent manager
	subagentConfig := &subagent.ManagerConfig{
		MaxConcurrentTasks:  5,
		DefaultTimeout:      30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
		HealthCheckInterval: 5 * time.Minute,
		EnableLoadBalancing: true,
		EnableFallback:      true,
	}
	
	subagentManager := subagent.NewSubagentManager(mcpClient, subagentConfig)

	// Initialize token monitor with in-memory storage
	tokenStorage := &InMemoryTokenStorage{}
	alertHandler := &ConsoleAlertHandler{}
	monitorConfig := &metrics.MonitorConfig{
		MaxEventsInMemory:    1000,
		PersistInterval:      5 * time.Minute,
		MetricsInterval:      1 * time.Minute,
		EnableRealTimeAlerts: true,
		DefaultDailyLimit:    100000,
		DefaultMonthlyLimit:  2000000,
	}
	
	tokenMonitor := metrics.NewTokenMonitor(tokenStorage, alertHandler, monitorConfig)

	// Initialize context-aware MCP client
	contextAwareMCPClient := mcp.NewContextAwareMCPClient(
		agentBackendURL,
		authToken,
		contextOptimizer,
	)

	// Initialize backend adapter for integration
	backendAdapter := integration.NewBackendAdapter(agentBackendURL, authToken)

	// Register subagents from collection
	ctx := context.Background()
	if err := backendAdapter.RegisterSubagentCollection(ctx, "./subagents-collection"); err != nil {
		log.Printf("Warning: Failed to register subagent collection: %v", err)
	}

	// Create optimized UI
	optimizedUI := ui.NewOptimizedChatUI(
		contextAwareMCPClient,
		contextOptimizer,
		subagentManager,
		tokenMonitor,
		userID,
		sessionID,
	)

	// Set up user budget
	if err := tokenMonitor.SetBudget(userID, 10000, 200000); err != nil {
		log.Printf("Warning: Failed to set budget: %v", err)
	}

	// Start background services
	go startBackgroundTasks(ctx, backendAdapter, subagentManager, tokenMonitor)

	// Run the optimized UI
	log.Println("Starting Optimized Chat UI...")
	log.Println("Features enabled:")
	log.Println("  • Context optimization with subagents")
	log.Println("  • Real-time token usage monitoring") 
	log.Println("  • Intelligent content summarization")
	log.Println("  • Cost tracking and budgets")
	log.Println("  • Performance metrics")
	log.Println("")
	log.Println("Controls:")
	log.Println("  • F1: Help")
	log.Println("  • F2: Subagent Status")
	log.Println("  • F3: Detailed Metrics")
	log.Println("  • F5: Clear Conversation")
	log.Println("  • Ctrl+O: Toggle Optimization")
	log.Println("  • Ctrl+M: Toggle Metrics Panel")
	log.Println("  • Ctrl+C: Exit")

	if err := optimizedUI.Run(); err != nil {
		log.Fatalf("UI error: %v", err)
	}
}

// Background tasks for maintaining system health and performance
func startBackgroundTasks(
	ctx context.Context,
	backendAdapter *integration.BackendAdapter,
	subagentManager *subagent.SubagentManager,
	tokenMonitor *metrics.TokenMonitor,
) {
	// Sync metrics with backend every 5 minutes
	metricsTicker := time.NewTicker(5 * time.Minute)
	defer metricsTicker.Stop()

	// Send telemetry every 10 minutes
	telemetryTicker := time.NewTicker(10 * time.Minute)
	defer telemetryTicker.Stop()

	// Check for optimization recommendations every 15 minutes
	recommendationsTicker := time.NewTicker(15 * time.Minute)
	defer recommendationsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-metricsTicker.C:
			// Sync subagent metrics
			managerMetrics := subagentManager.GetManagerMetrics()
			if err := backendAdapter.SyncSubagentMetrics(ctx, managerMetrics.SubagentPerformance); err != nil {
				log.Printf("Failed to sync metrics: %v", err)
			}

		case <-telemetryTicker.C:
			// Send telemetry data
			realTimeMetrics := tokenMonitor.GetRealTimeMetrics()
			log.Printf("Telemetry: %d active users, %d total tokens, %.1f%% optimization rate",
				realTimeMetrics.ActiveUserCount,
				realTimeMetrics.TotalTokens,
				realTimeMetrics.OptimizationRate*100,
			)

		case <-recommendationsTicker.C:
			// This would get recommendations for active users
			log.Println("Checking for optimization recommendations...")
		}
	}
}

// Simple implementations for the example

// InMemoryTokenStorage provides in-memory storage for token usage events
type InMemoryTokenStorage struct {
	events  []metrics.TokenUsageEvent
	budgets map[string]*metrics.TokenBudget
}

func (s *InMemoryTokenStorage) StoreEvent(event metrics.TokenUsageEvent) error {
	s.events = append(s.events, event)
	return nil
}

func (s *InMemoryTokenStorage) GetEvents(userID string, timeRange metrics.TimeRange) ([]metrics.TokenUsageEvent, error) {
	var userEvents []metrics.TokenUsageEvent
	for _, event := range s.events {
		if event.UserID == userID &&
			event.Timestamp.After(timeRange.Start) &&
			event.Timestamp.Before(timeRange.End) {
			userEvents = append(userEvents, event)
		}
	}
	return userEvents, nil
}

func (s *InMemoryTokenStorage) GetBudget(userID string) (*metrics.TokenBudget, error) {
	if s.budgets == nil {
		s.budgets = make(map[string]*metrics.TokenBudget)
	}
	
	budget, exists := s.budgets[userID]
	if !exists {
		return nil, fmt.Errorf("budget not found for user: %s", userID)
	}
	return budget, nil
}

func (s *InMemoryTokenStorage) UpdateBudget(budget *metrics.TokenBudget) error {
	if s.budgets == nil {
		s.budgets = make(map[string]*metrics.TokenBudget)
	}
	s.budgets[budget.UserID] = budget
	return nil
}

// ConsoleAlertHandler provides console-based alerts
type ConsoleAlertHandler struct{}

func (h *ConsoleAlertHandler) SendAlert(userID string, message string, severity metrics.AlertSeverity) error {
	log.Printf("[%s] Alert for user %s: %s", severity, userID, message)
	return nil
}

// Add missing import
import "fmt"