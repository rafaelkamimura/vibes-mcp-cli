package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"openai-cli/internal/client"
	contextpkg "openai-cli/internal/context"
	"openai-cli/internal/mcp"
	"openai-cli/internal/metrics"
	"openai-cli/internal/subagent"
	"openai-cli/internal/telemetry"
	"openai-cli/internal/ui"
)

// MCPOptimizationDemo demonstrates the full MCP integration with context optimization
func main() {
	fmt.Println("🌊 Vibes MCP CLI - Context Optimization Demo")
	fmt.Println(repeatString("=", 50))

	// Setup configuration
	config := &DemoConfig{
		AgentURL:        "http://localhost:8000", // vibes-agent-backend
		AuthToken:       "demo-token",
		TelemetryAPIKey: "vibes-telemetry-api-key-2024",
		MaxTokens:       4000,
		SummaryThreshold: 500,
		RetainRecent:    5,
		UseSubagents:    true,
		EnableOptimization: true,
	}

	// Initialize telemetry
	telemetryClient, err := telemetry.SetupTelemetryForAgentBackend(
		config.AgentURL,
		config.AuthToken,
		config.TelemetryAPIKey,
		nil,
	)
	if err != nil {
		log.Printf("Warning: Failed to setup telemetry: %v", err)
		telemetryClient = nil
	}
	defer func() {
		if telemetryClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			telemetryClient.Close(ctx)
		}
	}()

	logger, err := telemetry.SetupTelemetryLogger(telemetryClient, "mcp_demo", "info")
	if err != nil {
		log.Printf("Warning: Failed to setup telemetry logger: %v", err)
	}

	// Initialize MCP client
	mcpClient := mcp.NewClient(config.AgentURL, config.AuthToken)
	fmt.Printf("✅ MCP Client initialized: %s\n", config.AgentURL)

	// Initialize context optimizer
	strategy := mcp.OptimizationStrategy{
		MaxTokens:         config.MaxTokens,
		SummaryThreshold:  config.SummaryThreshold,
		CompressionRatio:  0.7,
		RetainRecent:      config.RetainRecent,
		PriorityWeighting: true,
		UseSubagents:      config.UseSubagents,
	}
	contextOptimizer := contextpkg.NewContextOptimizer(mcpClient, strategy)
	fmt.Printf("✅ Context Optimizer initialized (max tokens: %d)\n", config.MaxTokens)

	// Initialize subagent manager
	managerConfig := &subagent.ManagerConfig{
		MaxConcurrentTasks:  5,
		DefaultTimeout:      30 * time.Second,
		RetryAttempts:       2,
		RetryDelay:          1 * time.Second,
		HealthCheckInterval: 5 * time.Minute,
		EnableLoadBalancing: true,
		EnableFallback:      true,
	}
	subagentManager := subagent.NewSubagentManager(mcpClient, managerConfig)
	fmt.Printf("✅ Subagent Manager initialized (%d subagents)\n", len(subagentManager.ListSubagents()))

	// Initialize optimization tracker
	optimizationTracker := metrics.NewOptimizationTracker(telemetryClient, logger)
	defer optimizationTracker.Close()
	fmt.Printf("✅ Optimization Tracker initialized\n")

	// Test scenarios
	fmt.Println("\n📊 Running Optimization Tests...")
	
	// Test 1: Basic context optimization
	runBasicOptimizationTest(contextOptimizer, optimizationTracker)
	
	// Test 2: Subagent routing
	runSubagentRoutingTest(contextOptimizer, optimizationTracker)
	
	// Test 3: Performance metrics
	runPerformanceTest(optimizationTracker)

	// Test 4: UI component integration
	runUIComponentTest(mcpClient, telemetryClient, logger)

	// Final metrics report
	fmt.Println("\n📈 Final Optimization Report:")
	fmt.Println(optimizationTracker.GetSummaryReport())

	fmt.Println("\n🎉 MCP Integration Demo Complete!")
}

// DemoConfig holds configuration for the demo
type DemoConfig struct {
	AgentURL           string
	AuthToken          string
	TelemetryAPIKey    string
	MaxTokens          int
	SummaryThreshold   int
	RetainRecent       int
	UseSubagents       bool
	EnableOptimization bool
}

// runBasicOptimizationTest tests basic context optimization
func runBasicOptimizationTest(optimizer *contextpkg.ContextOptimizer, tracker *metrics.OptimizationTracker) {
	fmt.Println("\n🔧 Test 1: Basic Context Optimization")
	
	// Create a long conversation to test optimization
	messages := []client.ChatMessage{
		{Role: "user", Content: "What is machine learning? Please explain in detail with examples and applications."},
		{Role: "assistant", Content: "Machine learning is a subset of artificial intelligence (AI) that provides systems the ability to automatically learn and improve from experience without being explicitly programmed. It focuses on the development of computer programs that can access data and use it to learn for themselves. The process of learning begins with observations or data, such as examples, direct experience, or instruction, in order to look for patterns in data and make better decisions in the future based on the examples that we provide. The primary aim is to allow the computers learn automatically without human intervention or assistance and adjust actions accordingly."},
		{Role: "user", Content: "Can you give me specific examples of machine learning algorithms?"},
		{Role: "assistant", Content: "Certainly! Here are some specific examples of machine learning algorithms: 1. Linear Regression - Used for predicting continuous values, 2. Decision Trees - Used for both classification and regression tasks, 3. Random Forest - An ensemble method that combines multiple decision trees, 4. Support Vector Machines (SVM) - Effective for high-dimensional data, 5. K-Means Clustering - Used for unsupervised learning to group similar data points, 6. Neural Networks - Inspired by the human brain, used for complex pattern recognition, 7. Naive Bayes - A probabilistic classifier based on Bayes' theorem."},
		{Role: "user", Content: "How does deep learning differ from traditional machine learning?"},
		{Role: "assistant", Content: "Deep learning differs from traditional machine learning in several key ways: 1. Architecture: Deep learning uses neural networks with multiple hidden layers (hence 'deep'), while traditional ML often uses simpler algorithms. 2. Feature Engineering: Traditional ML requires manual feature selection and engineering, while deep learning can automatically learn features from raw data. 3. Data Requirements: Deep learning typically requires much larger datasets to perform well. 4. Computational Resources: Deep learning is more computationally intensive and often requires specialized hardware like GPUs. 5. Interpretability: Traditional ML models are often more interpretable, while deep learning models are often considered 'black boxes'. 6. Performance: For complex tasks like image recognition and natural language processing, deep learning often outperforms traditional methods."},
		{Role: "user", Content: "What are the main challenges in implementing machine learning in real-world applications?"},
	}

	// Create additional context
	additionalContext := []mcp.ContextChunk{
		{
			ID:         "context_1",
			Type:       "text",
			Content:    "Previous discussion about data science fundamentals and statistical methods.",
			TokenCount: 50,
			Priority:   3,
			Timestamp:  time.Now().Add(-1 * time.Hour),
			Source:     "previous_session",
			Tags:       []string{"data_science", "statistics"},
		},
		{
			ID:         "context_2",
			Type:       "code",
			Content:    "import pandas as pd\nimport numpy as np\nfrom sklearn.model_selection import train_test_split\nfrom sklearn.linear_model import LogisticRegression",
			TokenCount: 30,
			Priority:   6,
			Timestamp:  time.Now().Add(-30 * time.Minute),
			Source:     "code_example",
			Tags:       []string{"python", "sklearn", "code"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	optimizedMessages, result, err := optimizer.OptimizeContext(ctx, messages, additionalContext)
	processingTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Optimization failed: %v\n", err)
		tracker.TrackError("optimization", err.Error(), "demo_user", "demo_session")
		return
	}

	fmt.Printf("✅ Optimization successful!\n")
	fmt.Printf("   Original messages: %d\n", len(messages))
	fmt.Printf("   Optimized messages: %d\n", len(optimizedMessages))
	fmt.Printf("   Original tokens: %d\n", result.OriginalTokens)
	fmt.Printf("   Optimized tokens: %d\n", result.OptimizedTokens)
	fmt.Printf("   Tokens saved: %d (%.1f%%)\n", result.TokensSaved, 
		float64(result.TokensSaved)/float64(result.OriginalTokens)*100)
	fmt.Printf("   Processing time: %v\n", processingTime)

	// Track the optimization
	tracker.TrackOptimization(result, processingTime, "demo_user", "demo_session")
}

// runSubagentRoutingTest tests subagent routing functionality
func runSubagentRoutingTest(optimizer *contextpkg.ContextOptimizer, tracker *metrics.OptimizationTracker) {
	fmt.Println("\n🤖 Test 2: Subagent Routing")

	testCases := []struct {
		taskType string
		input    string
		expected string
	}{
		{
			taskType: "code_analysis",
			input:    "function calculateTotal(items) { return items.reduce((sum, item) => sum + item.price, 0); }",
			expected: "code_analyzer",
		},
		{
			taskType: "doc_summary",
			input:    "# API Documentation\n\nThis API provides endpoints for managing user accounts and processing payments.",
			expected: "doc_summarizer",
		},
		{
			taskType: "data_analysis",
			input:    `{"users": [{"id": 1, "name": "Alice", "age": 30}, {"id": 2, "name": "Bob", "age": 25}]}`,
			expected: "data_processor",
		},
	}

	for i, tc := range testCases {
		fmt.Printf("   Test case %d: %s\n", i+1, tc.taskType)
		
		contextChunks := []mcp.ContextChunk{
			{
				ID:         fmt.Sprintf("test_chunk_%d", i),
				Type:       tc.taskType,
				Content:    tc.input,
				TokenCount: len(tc.input) / 4,
				Priority:   5,
				Timestamp:  time.Now(),
				Source:     "test",
				Tags:       []string{tc.taskType},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		
		startTime := time.Now()
		response, err := optimizer.RouteTaskToSubagent(ctx, tc.taskType, tc.input, contextChunks)
		processingTime := time.Since(startTime)
		
		cancel()

		if err != nil {
			fmt.Printf("   ❌ Routing failed: %v\n", err)
			tracker.TrackError("subagent_routing", err.Error(), "demo_user", "demo_session")
			continue
		}

		fmt.Printf("   ✅ Routed to: %s\n", response.SubagentID)
		fmt.Printf("   📊 Confidence: %.2f\n", response.Confidence)
		fmt.Printf("   ⏱️  Processing time: %v\n", processingTime)

		// Track the subagent call
		tracker.TrackSubagentCall(response.SubagentID, tc.taskType, response, "demo_user", "demo_session")
	}
}

// runPerformanceTest evaluates overall performance metrics
func runPerformanceTest(tracker *metrics.OptimizationTracker) {
	fmt.Println("\n📊 Test 3: Performance Metrics")
	
	metrics := tracker.GetMetrics()
	
	fmt.Printf("   Total optimizations: %d\n", metrics.TotalOptimizations)
	fmt.Printf("   Success rate: %.1f%%\n", 
		float64(metrics.SuccessfulOptimizations)/float64(metrics.TotalOptimizations)*100)
	fmt.Printf("   Total tokens saved: %d\n", metrics.TotalTokensSaved)
	fmt.Printf("   Average compression ratio: %.1f%%\n", metrics.AverageCompressionRatio*100)
	fmt.Printf("   Average processing time: %v\n", metrics.AverageProcessingTime)
	
	if len(metrics.SubagentUsage) > 0 {
		fmt.Printf("   Subagent usage:\n")
		for subagent, count := range metrics.SubagentUsage {
			successRate := metrics.SubagentSuccessRates[subagent] * 100
			fmt.Printf("     - %s: %d calls (%.1f%% success)\n", subagent, count, successRate)
		}
	}
	
	fmt.Printf("   Estimated cost savings: $%.4f\n", metrics.EstimatedCostSavings)
}

// runUIComponentTest tests the optimized chat UI component
func runUIComponentTest(mcpClient *mcp.Client, telemetryClient telemetry.Client, logger *telemetry.TelemetryLogger) {
	fmt.Println("\n🖥️  Test 4: UI Component Integration")
	
	// Configure the optimized chat component
	config := &ui.OptimizedChatConfig{
		MaxTokens:          4000,
		SummaryThreshold:   500,
		RetainRecent:       5,
		UseSubagents:       true,
		EnableOptimization: true,
		CompressionRatio:   0.7,
		TelemetryClient:    telemetryClient,
		Logger:             logger,
	}
	
	// Create the optimized chat component
	chatComponent := ui.NewOptimizedChatComponent(mcpClient, config)
	
	fmt.Printf("   ✅ Optimized chat component created\n")
	fmt.Printf("   📱 Component ready for TUI integration\n")
	
	// Get initial stats
	stats := chatComponent.GetStats()
	fmt.Printf("   📊 Initial stats: %d messages, %d optimizations\n", 
		stats.TotalMessages, stats.TotalOptimizations)
		
	fmt.Printf("   🎮 Keyboard shortcuts:\n")
	fmt.Printf("      Ctrl+O: Toggle optimization\n")
	fmt.Printf("      Ctrl+S: Toggle subagent usage\n")
	fmt.Printf("      Ctrl+R: Refresh subagents\n")
}

// Utility function to repeat a string (Go doesn't have built-in string multiplication)
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}