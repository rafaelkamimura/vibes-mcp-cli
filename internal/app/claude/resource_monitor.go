package claude

import (
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ResourceMonitor monitors resource usage of running processes
type ResourceMonitor struct {
	logger    *zap.Logger
	mu        sync.RWMutex
	monitors  map[string]*processMonitor
	ctx       context.Context
	cancel    context.CancelFunc
}

// processMonitor tracks resource usage for a single process
type processMonitor struct {
	processID string
	process   *os.Process
	startTime time.Time
	usage     *ResourceUsage
	ticker    *time.Ticker
	done      chan struct{}
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(logger *zap.Logger) *ResourceMonitor {
	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	return &ResourceMonitor{
		logger:   logger,
		monitors: make(map[string]*processMonitor),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// StartMonitoring starts monitoring a process
func (rm *ResourceMonitor) StartMonitoring(processID string, process *os.Process) {
	if process == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Don't monitor the same process twice
	if _, exists := rm.monitors[processID]; exists {
		return
	}

	monitor := &processMonitor{
		processID: processID,
		process:   process,
		startTime: time.Now(),
		usage: &ResourceUsage{
			PeakMemoryMB:  0,
			AvgCPUPercent: 0.0,
			DiskUsageMB:   0,
		},
		ticker: time.NewTicker(5 * time.Second), // Monitor every 5 seconds
		done:   make(chan struct{}),
	}

	rm.monitors[processID] = monitor

	// Start monitoring goroutine
	go rm.monitorProcess(monitor)

	rm.logger.Debug("started monitoring process",
		zap.String("process_id", processID),
		zap.Int("pid", process.Pid))
}

// StopMonitoring stops monitoring a process and returns final resource usage
func (rm *ResourceMonitor) StopMonitoring(processID string) *ResourceUsage {
	rm.mu.Lock()
	monitor, exists := rm.monitors[processID]
	if !exists {
		rm.mu.Unlock()
		return nil
	}
	delete(rm.monitors, processID)
	rm.mu.Unlock()

	// Stop the monitoring goroutine
	close(monitor.done)
	monitor.ticker.Stop()

	rm.logger.Debug("stopped monitoring process",
		zap.String("process_id", processID))

	return monitor.usage
}

// monitorProcess runs the monitoring loop for a single process
func (rm *ResourceMonitor) monitorProcess(monitor *processMonitor) {
	defer func() {
		if r := recover(); r != nil {
			rm.logger.Error("process monitoring panicked",
				zap.String("process_id", monitor.processID),
				zap.Any("panic", r))
		}
	}()

	sampleCount := 0
	cpuTotal := 0.0

	for {
		select {
		case <-monitor.done:
			return
		case <-rm.ctx.Done():
			return
		case <-monitor.ticker.C:
			// Simple monitoring - in a real implementation, this would
			// use platform-specific APIs to get actual resource usage
			if rm.updateResourceUsage(monitor) {
				sampleCount++
				if sampleCount > 0 {
					monitor.usage.AvgCPUPercent = cpuTotal / float64(sampleCount)
				}
			}
		}
	}
}

// updateResourceUsage updates resource usage statistics for a process
func (rm *ResourceMonitor) updateResourceUsage(monitor *processMonitor) bool {
	// Check if process is still running
	if monitor.process == nil {
		return false
	}

	// Try to signal the process to check if it's still alive
	err := monitor.process.Signal(os.Signal(nil))
	if err != nil {
		// Process is no longer running
		return false
	}

	// Simulate resource usage collection
	// In a real implementation, this would use:
	// - On Linux: /proc/[pid]/stat, /proc/[pid]/status
	// - On macOS: task_info() system calls
	// - On Windows: GetProcessMemoryInfo(), GetProcessTimes()
	
	currentMemoryMB := 50 + (time.Since(monitor.startTime).Minutes() * 2) // Simulate growing memory
	if int(currentMemoryMB) > monitor.usage.PeakMemoryMB {
		monitor.usage.PeakMemoryMB = int(currentMemoryMB)
	}

	return true
}

// GetActiveMonitors returns the number of active monitors
func (rm *ResourceMonitor) GetActiveMonitors() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.monitors)
}

// Close shuts down the resource monitor
func (rm *ResourceMonitor) Close() error {
	rm.cancel()

	// Stop all active monitors
	rm.mu.Lock()
	for processID, monitor := range rm.monitors {
		close(monitor.done)
		monitor.ticker.Stop()
		delete(rm.monitors, processID)
	}
	rm.mu.Unlock()

	rm.logger.Info("resource monitor closed")
	return nil
}