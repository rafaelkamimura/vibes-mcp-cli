package claude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ProcessState represents the current state of a process
type ProcessState int

const (
	ProcessStateCreated ProcessState = iota
	ProcessStateStarting
	ProcessStateRunning
	ProcessStateFinished
	ProcessStateKilled
	ProcessStateError
)

// String returns the string representation of the process state
func (ps ProcessState) String() string {
	switch ps {
	case ProcessStateCreated:
		return "created"
	case ProcessStateStarting:
		return "starting"
	case ProcessStateRunning:
		return "running"
	case ProcessStateFinished:
		return "finished"
	case ProcessStateKilled:
		return "killed"
	case ProcessStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Process represents a managed Claude Code process with I/O handling
type Process struct {
	ID             string             // Unique process identifier
	cmd            *exec.Cmd          // Underlying command
	logger         *zap.Logger        // Logger instance
	state          ProcessState       // Current process state
	mu             sync.RWMutex       // Protects concurrent access
	stdin          io.WriteCloser     // Process stdin
	stdout         io.ReadCloser      // Process stdout
	stderr         io.ReadCloser      // Process stderr
	outputBuffer   *bytes.Buffer      // Accumulated output buffer
	outputChan     chan []byte        // Real-time output channel
	subscribers    []chan []byte      // Output subscribers
	resourceLimits *ResourceLimits    // Resource constraints
	startTime      time.Time          // Process start time
	endTime        time.Time          // Process end time
	exitCode       int                // Process exit code
	err            error              // Process error
	done           chan struct{}      // Completion signal
	cancel         context.CancelFunc // Context cancellation
}

// NewProcess creates a new managed process
func NewProcess(cmd *exec.Cmd, logger *zap.Logger) *Process {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Process{
		ID:           generateProcessID(),
		cmd:          cmd,
		logger:       logger,
		state:        ProcessStateCreated,
		outputBuffer: &bytes.Buffer{},
		outputChan:   make(chan []byte, 100), // Buffered channel
		subscribers:  make([]chan []byte, 0),
		done:         make(chan struct{}),
		exitCode:     -1,
	}
}

// generateProcessID generates a unique process identifier
func generateProcessID() string {
	return fmt.Sprintf("claude-proc-%d", time.Now().UnixNano())
}

// SetResourceLimits sets resource constraints for the process
func (p *Process) SetResourceLimits(limits *ResourceLimits) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resourceLimits = limits
}

// GetState returns the current process state
func (p *Process) GetState() ProcessState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// setState updates the process state
func (p *Process) setState(state ProcessState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state

	p.logger.Debug("process state changed",
		zap.String("process_id", p.ID),
		zap.String("state", state.String()))
}

// Start starts the process and sets up I/O handling
func (p *Process) Start() error {
	p.setState(ProcessStateStarting)

	// Set up pipes for I/O
	var err error

	p.stdin, err = p.cmd.StdinPipe()
	if err != nil {
		p.setState(ProcessStateError)
		p.err = fmt.Errorf("failed to create stdin pipe: %w", err)
		return p.err
	}

	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		p.setState(ProcessStateError)
		p.err = fmt.Errorf("failed to create stdout pipe: %w", err)
		return p.err
	}

	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		p.setState(ProcessStateError)
		p.err = fmt.Errorf("failed to create stderr pipe: %w", err)
		return p.err
	}

	// Apply resource limits if set
	if p.resourceLimits != nil {
		p.applyResourceLimits()
	}

	// Start the process
	p.startTime = time.Now()
	if err := p.cmd.Start(); err != nil {
		p.setState(ProcessStateError)
		p.err = fmt.Errorf("failed to start process: %w", err)
		return p.err
	}

	p.setState(ProcessStateRunning)

	// Start I/O goroutines
	go p.handleOutput()
	go p.handleCompletion()

	p.logger.Info("process started",
		zap.String("process_id", p.ID),
		zap.Int("pid", p.cmd.Process.Pid))

	return nil
}

// applyResourceLimits applies resource constraints to the process
func (p *Process) applyResourceLimits() {
	if p.resourceLimits == nil {
		return
	}

	// Set process group to allow killing child processes
	p.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Note: Memory and CPU limits would typically be enforced by:
	// 1. Using cgroups on Linux
	// 2. Using process limits on other systems
	// 3. External monitoring and enforcement
	// For now, we'll implement monitoring and reactive enforcement
}

// handleOutput manages stdout and stderr output
func (p *Process) handleOutput() {
	defer close(p.outputChan)

	// Create combined output reader
	combinedOutput := io.MultiReader(p.stdout, p.stderr)
	scanner := bufio.NewScanner(combinedOutput)

	// Set scanner buffer size to handle large lines
	const maxScanTokenSize = 64 * 1024 // 64KB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Make a copy since scanner.Bytes() returns a slice that may be overwritten
		output := make([]byte, len(line)+1) // +1 for newline
		copy(output, line)
		output[len(line)] = '\n'

		// Add to output buffer
		p.mu.Lock()
		p.outputBuffer.Write(output)
		p.mu.Unlock()

		// Send to output channel (non-blocking)
		select {
		case p.outputChan <- output:
		default:
			// Channel is full, skip this output to prevent blocking
			p.logger.Warn("output channel full, dropping output",
				zap.String("process_id", p.ID))
		}

		// Send to subscribers
		p.mu.RLock()
		for _, subscriber := range p.subscribers {
			select {
			case subscriber <- output:
			default:
				// Subscriber channel is full, skip
			}
		}
		p.mu.RUnlock()
	}

	if err := scanner.Err(); err != nil {
		p.logger.Error("output scanning error",
			zap.String("process_id", p.ID),
			zap.Error(err))
	}
}

// handleCompletion waits for process completion
func (p *Process) handleCompletion() {
	defer close(p.done)

	// Wait for process to finish
	err := p.cmd.Wait()
	p.endTime = time.Now()

	p.mu.Lock()
	defer p.mu.Unlock()

	if err != nil {
		p.err = err
		if exitError, ok := err.(*exec.ExitError); ok {
			p.exitCode = exitError.ExitCode()
		}
		p.setState(ProcessStateError)
	} else {
		p.exitCode = 0
		p.setState(ProcessStateFinished)
	}

	duration := p.endTime.Sub(p.startTime)
	p.logger.Info("process completed",
		zap.String("process_id", p.ID),
		zap.Duration("duration", duration),
		zap.Int("exit_code", p.exitCode))
}

// SendInput sends input to the process stdin
func (p *Process) SendInput(input string) error {
	p.mu.RLock()
	state := p.state
	stdin := p.stdin
	p.mu.RUnlock()

	if state != ProcessStateRunning {
		return fmt.Errorf("process is not running (state: %s)", state.String())
	}

	if stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	_, err := stdin.Write([]byte(input))
	if err != nil {
		p.logger.Error("failed to send input",
			zap.String("process_id", p.ID),
			zap.Error(err))
		return fmt.Errorf("failed to send input: %w", err)
	}

	return nil
}

// SendInputLine sends a line of input to the process stdin
func (p *Process) SendInputLine(input string) error {
	return p.SendInput(input + "\n")
}

// CloseInput closes the stdin pipe
func (p *Process) CloseInput() error {
	p.mu.RLock()
	stdin := p.stdin
	p.mu.RUnlock()

	if stdin == nil {
		return nil
	}

	return stdin.Close()
}

// GetOutput returns the accumulated output
func (p *Process) GetOutput() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.outputBuffer.Bytes()
}

// GetOutputString returns the accumulated output as a string
func (p *Process) GetOutputString() string {
	return string(p.GetOutput())
}

// SubscribeToOutput creates a new subscription to real-time output
func (p *Process) SubscribeToOutput() <-chan []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	subscriber := make(chan []byte, 100) // Buffered channel
	p.subscribers = append(p.subscribers, subscriber)
	return subscriber
}

// Wait waits for the process to complete
func (p *Process) Wait() (*ExecutionResult, error) {
	<-p.done

	p.mu.RLock()
	defer p.mu.RUnlock()

	result := &ExecutionResult{
		ExitCode: p.exitCode,
		Output:   p.outputBuffer.Bytes(),
		Duration: p.endTime.Sub(p.startTime),
		Error:    p.err,
	}

	return result, p.err
}

// WaitWithContext waits for the process to complete with context cancellation
func (p *Process) WaitWithContext(ctx context.Context) (*ExecutionResult, error) {
	select {
	case <-p.done:
		return p.Wait()
	case <-ctx.Done():
		// Context cancelled, kill the process
		if err := p.Kill(); err != nil {
			p.logger.Error("failed to kill process after context cancellation",
				zap.String("process_id", p.ID),
				zap.Error(err))
		}
		return nil, ctx.Err()
	}
}

// Kill forcefully terminates the process
func (p *Process) Kill() error {
	p.mu.RLock()
	state := p.state
	cmd := p.cmd
	p.mu.RUnlock()

	if state == ProcessStateFinished || state == ProcessStateKilled {
		return nil // Already finished
	}

	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	// Kill the process group to ensure child processes are also killed
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		// Kill the entire process group
		syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		// Fallback to killing just the main process
		cmd.Process.Kill()
	}

	p.setState(ProcessStateKilled)

	p.logger.Info("process killed",
		zap.String("process_id", p.ID),
		zap.Int("pid", cmd.Process.Pid))

	return nil
}

// Signal sends a signal to the process
func (p *Process) Signal(sig os.Signal) error {
	p.mu.RLock()
	state := p.state
	cmd := p.cmd
	p.mu.RUnlock()

	if state != ProcessStateRunning {
		return fmt.Errorf("process is not running (state: %s)", state.String())
	}

	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("process not available")
	}

	return cmd.Process.Signal(sig)
}

// GetPID returns the process ID
func (p *Process) GetPID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return -1
}

// GetStartTime returns the process start time
func (p *Process) GetStartTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.startTime
}

// GetDuration returns the process duration (running time or total time if finished)
func (p *Process) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.state == ProcessStateFinished || p.state == ProcessStateKilled || p.state == ProcessStateError {
		return p.endTime.Sub(p.startTime)
	}

	if !p.startTime.IsZero() {
		return time.Since(p.startTime)
	}

	return 0
}

// GetExitCode returns the process exit code (only valid after completion)
func (p *Process) GetExitCode() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitCode
}

// GetError returns any process error
func (p *Process) GetError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.err
}

// IsRunning returns true if the process is currently running
func (p *Process) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state == ProcessStateRunning
}

// IsFinished returns true if the process has finished (successfully or with error)
func (p *Process) IsFinished() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state == ProcessStateFinished || p.state == ProcessStateError || p.state == ProcessStateKilled
}

// Close cleans up process resources
func (p *Process) Close() error {
	// Kill process if still running
	if p.IsRunning() {
		p.Kill()
	}

	// Close I/O pipes
	var lastErr error

	if p.stdin != nil {
		if err := p.stdin.Close(); err != nil {
			lastErr = err
		}
	}

	if p.stdout != nil {
		if err := p.stdout.Close(); err != nil {
			lastErr = err
		}
	}

	if p.stderr != nil {
		if err := p.stderr.Close(); err != nil {
			lastErr = err
		}
	}

	// Close subscriber channels
	p.mu.Lock()
	for _, subscriber := range p.subscribers {
		close(subscriber)
	}
	p.subscribers = nil
	p.mu.Unlock()

	return lastErr
}
