package claude

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CommandOptions holds configuration for Claude Code execution
type CommandOptions struct {
	WorkingDir     string            // Working directory for the command
	Environment    map[string]string // Additional environment variables
	Timeout        time.Duration     // Command timeout
	InputText      string            // Initial input text to send
	Args           []string          // Additional arguments to pass to claude
	ResourceLimits *ResourceLimits   // Resource constraints
}

// ResourceLimits defines resource constraints for Claude processes
type ResourceLimits struct {
	MaxMemoryMB    int           // Maximum memory usage in MB
	MaxCPUPercent  float64       // Maximum CPU usage percentage
	MaxDiskUsageMB int           // Maximum disk usage in MB
	MaxDuration    time.Duration // Maximum execution duration
}

// DefaultResourceLimits returns sensible default resource limits
func DefaultResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxMemoryMB:    1024, // 1GB
		MaxCPUPercent:  80.0, // 80% CPU
		MaxDiskUsageMB: 512,  // 512MB
		MaxDuration:    time.Hour * 2, // 2 hours
	}
}

// ExecutionResult contains the result of a Claude Code execution
type ExecutionResult struct {
	ExitCode   int           // Process exit code
	Output     []byte        // Combined stdout/stderr output
	Duration   time.Duration // Execution duration
	Error      error         // Execution error if any
	ResourceUsage *ResourceUsage // Resource usage statistics
}

// ResourceUsage contains resource usage statistics
type ResourceUsage struct {
	PeakMemoryMB   int     // Peak memory usage in MB
	AvgCPUPercent  float64 // Average CPU usage percentage
	DiskUsageMB    int     // Disk usage in MB
	NetworkBytesTx int64   // Network bytes transmitted
	NetworkBytesRx int64   // Network bytes received
}

// Executor manages Claude Code process execution with proper resource management
type Executor struct {
	logger       *zap.Logger
	claudePath   string           // Path to claude executable
	mu           sync.RWMutex     // Protects concurrent access
	activeProcs  map[string]*Process // Active processes by ID
	resourceMon  *ResourceMonitor    // Resource usage monitor
	defaultOpts  *CommandOptions     // Default execution options
}

// NewExecutor creates a new Claude Code executor
func NewExecutor(logger *zap.Logger, claudePath string) (*Executor, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Verify claude executable exists and is executable
	if err := validateClaudeExecutable(claudePath); err != nil {
		return nil, fmt.Errorf("claude executable validation failed: %w", err)
	}

	executor := &Executor{
		logger:      logger,
		claudePath:  claudePath,
		activeProcs: make(map[string]*Process),
		resourceMon: NewResourceMonitor(logger),
		defaultOpts: &CommandOptions{
			Timeout:        time.Minute * 30,
			ResourceLimits: DefaultResourceLimits(),
			Environment:    make(map[string]string),
		},
	}

	return executor, nil
}

// validateClaudeExecutable checks if the claude executable is valid
func validateClaudeExecutable(path string) error {
	if path == "" {
		return fmt.Errorf("claude executable path is empty")
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("claude executable not found: %w", err)
	}

	// Check if it's executable
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("claude executable is not executable")
	}

	// Try to get version to verify it's actually claude
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude executable validation failed: %w", err)
	}

	return nil
}

// Execute runs a Claude Code command synchronously
func (e *Executor) Execute(ctx context.Context, opts *CommandOptions) (*ExecutionResult, error) {
	if opts == nil {
		opts = e.defaultOpts
	}

	// Merge with default options
	mergedOpts := e.mergeOptions(opts)

	// Create process
	process, err := e.createProcess(ctx, mergedOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	// Track the process
	e.mu.Lock()
	e.activeProcs[process.ID] = process
	e.mu.Unlock()

	// Ensure cleanup
	defer func() {
		e.mu.Lock()
		delete(e.activeProcs, process.ID)
		e.mu.Unlock()
	}()

	// Start execution
	return e.executeProcess(ctx, process, mergedOpts)
}

// ExecuteAsync runs a Claude Code command asynchronously
func (e *Executor) ExecuteAsync(ctx context.Context, opts *CommandOptions) (*Process, error) {
	if opts == nil {
		opts = e.defaultOpts
	}

	// Merge with default options
	mergedOpts := e.mergeOptions(opts)

	// Create process
	process, err := e.createProcess(ctx, mergedOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	// Track the process
	e.mu.Lock()
	e.activeProcs[process.ID] = process
	e.mu.Unlock()

	// Start process asynchronously
	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.activeProcs, process.ID)
			e.mu.Unlock()
		}()

		_, err := e.executeProcess(ctx, process, mergedOpts)
		if err != nil {
			e.logger.Error("async process execution failed",
				zap.String("process_id", process.ID),
				zap.Error(err))
		}
	}()

	return process, nil
}

// createProcess creates a new process with the given options
func (e *Executor) createProcess(ctx context.Context, opts *CommandOptions) (*Process, error) {
	// Build command arguments
	args := []string{}
	args = append(args, opts.Args...)

	// Create command
	cmd := exec.CommandContext(ctx, e.claudePath, args...)

	// Set working directory
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	// Set environment
	env := os.Environ()
	for key, value := range opts.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	// Create process
	process := NewProcess(cmd, e.logger)

	// Apply resource limits
	if opts.ResourceLimits != nil {
		process.SetResourceLimits(opts.ResourceLimits)
	}

	return process, nil
}

// executeProcess executes a process and handles I/O
func (e *Executor) executeProcess(ctx context.Context, process *Process, opts *CommandOptions) (*ExecutionResult, error) {
	startTime := time.Now()

	// Start resource monitoring
	if e.resourceMon != nil {
		e.resourceMon.StartMonitoring(process.ID, process.cmd.Process)
	}

	// Start the process
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Send initial input if provided
	if opts.InputText != "" {
		if err := process.SendInput(opts.InputText); err != nil {
			e.logger.Warn("failed to send initial input",
				zap.String("process_id", process.ID),
				zap.Error(err))
		}
	}

	// Wait for completion with timeout
	var result *ExecutionResult
	var err error

	if opts.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
		
		result, err = process.WaitWithContext(timeoutCtx)
	} else {
		result, err = process.Wait()
	}

	// Stop resource monitoring
	var resourceUsage *ResourceUsage
	if e.resourceMon != nil {
		resourceUsage = e.resourceMon.StopMonitoring(process.ID)
	}

	// Calculate execution duration
	duration := time.Since(startTime)

	// Build final result
	if result == nil {
		result = &ExecutionResult{
			ExitCode: -1,
			Error:    err,
		}
	}

	result.Duration = duration
	result.ResourceUsage = resourceUsage

	if err != nil {
		e.logger.Error("process execution failed",
			zap.String("process_id", process.ID),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		e.logger.Info("process execution completed",
			zap.String("process_id", process.ID),
			zap.Duration("duration", duration),
			zap.Int("exit_code", result.ExitCode))
	}

	return result, err
}

// mergeOptions merges user options with defaults
func (e *Executor) mergeOptions(opts *CommandOptions) *CommandOptions {
	merged := &CommandOptions{
		WorkingDir:     e.defaultOpts.WorkingDir,
		Environment:    make(map[string]string),
		Timeout:        e.defaultOpts.Timeout,
		ResourceLimits: e.defaultOpts.ResourceLimits,
	}

	// Copy default environment
	for k, v := range e.defaultOpts.Environment {
		merged.Environment[k] = v
	}

	// Apply user options
	if opts.WorkingDir != "" {
		merged.WorkingDir = opts.WorkingDir
	}
	if opts.Timeout > 0 {
		merged.Timeout = opts.Timeout
	}
	if opts.InputText != "" {
		merged.InputText = opts.InputText
	}
	if len(opts.Args) > 0 {
		merged.Args = make([]string, len(opts.Args))
		copy(merged.Args, opts.Args)
	}
	if opts.ResourceLimits != nil {
		merged.ResourceLimits = opts.ResourceLimits
	}

	// Merge environment variables
	for k, v := range opts.Environment {
		merged.Environment[k] = v
	}

	return merged
}

// GetActiveProcesses returns a list of currently active processes
func (e *Executor) GetActiveProcesses() []*Process {
	e.mu.RLock()
	defer e.mu.RUnlock()

	processes := make([]*Process, 0, len(e.activeProcs))
	for _, proc := range e.activeProcs {
		processes = append(processes, proc)
	}

	return processes
}

// KillProcess forcefully terminates a process
func (e *Executor) KillProcess(processID string) error {
	e.mu.RLock()
	process, exists := e.activeProcs[processID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	return process.Kill()
}

// KillAllProcesses forcefully terminates all active processes
func (e *Executor) KillAllProcesses() error {
	e.mu.RLock()
	processes := make([]*Process, 0, len(e.activeProcs))
	for _, proc := range e.activeProcs {
		processes = append(processes, proc)
	}
	e.mu.RUnlock()

	var lastErr error
	for _, proc := range processes {
		if err := proc.Kill(); err != nil {
			lastErr = err
			e.logger.Error("failed to kill process",
				zap.String("process_id", proc.ID),
				zap.Error(err))
		}
	}

	return lastErr
}

// SetDefaultOptions updates the default command options
func (e *Executor) SetDefaultOptions(opts *CommandOptions) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if opts != nil {
		e.defaultOpts = opts
	}
}

// Close shuts down the executor and cleans up resources
func (e *Executor) Close() error {
	// Kill all active processes
	if err := e.KillAllProcesses(); err != nil {
		e.logger.Error("failed to kill all processes during shutdown", zap.Error(err))
	}

	// Close resource monitor
	if e.resourceMon != nil {
		e.resourceMon.Close()
	}

	e.logger.Info("executor closed")
	return nil
}

// SendInputToProcess sends input to a specific active process
func (e *Executor) SendInputToProcess(processID, input string) error {
	e.mu.RLock()
	process, exists := e.activeProcs[processID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	return process.SendInput(input)
}

// GetProcessOutput gets accumulated output from a specific process
func (e *Executor) GetProcessOutput(processID string) ([]byte, error) {
	e.mu.RLock()
	process, exists := e.activeProcs[processID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process %s not found", processID)
	}

	return process.GetOutput(), nil
}

// SubscribeToOutput subscribes to real-time output from a process
func (e *Executor) SubscribeToOutput(processID string) (<-chan []byte, error) {
	e.mu.RLock()
	process, exists := e.activeProcs[processID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("process %s not found", processID)
	}

	return process.SubscribeToOutput(), nil
}