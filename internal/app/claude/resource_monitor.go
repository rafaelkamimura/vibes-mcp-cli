package claude

import (
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ResourceMonitor tracks resource usage for processes
type ResourceMonitor struct {
	logger     *zap.Logger
	mu         sync.RWMutex
	monitoring map[string]*processStats // Process ID -> stats
	stopChan   chan struct{}
	ticker     *time.Ticker
}

// processStats holds resource usage statistics for a process
type processStats struct {
	PID            int
	process        *os.Process
	startTime      time.Time
	lastCPUTime    time.Duration
	lastSampleTime time.Time
	peakMemoryMB   int
	totalCPUTime   time.Duration
	sampleCount    int
	cpuSamples     []float64
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(logger *zap.Logger) *ResourceMonitor {
	if logger == nil {
		logger = zap.NewNop()
	}

	rm := &ResourceMonitor{
		logger:     logger,
		monitoring: make(map[string]*processStats),
		stopChan:   make(chan struct{}),
		ticker:     time.NewTicker(time.Second * 2), // Sample every 2 seconds
	}

	// Start monitoring goroutine
	go rm.monitorLoop()

	return rm
}

// StartMonitoring starts monitoring a process
func (rm *ResourceMonitor) StartMonitoring(processID string, process *os.Process) {
	if process == nil {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.monitoring[processID] = &processStats{
		PID:            process.Pid,
		process:        process,
		startTime:      time.Now(),
		lastSampleTime: time.Now(),
		cpuSamples:     make([]float64, 0, 100), // Pre-allocate for 100 samples
	}

	rm.logger.Debug("started monitoring process",
		zap.String("process_id", processID),
		zap.Int("pid", process.Pid))
}

// StopMonitoring stops monitoring a process and returns final statistics
func (rm *ResourceMonitor) StopMonitoring(processID string) *ResourceUsage {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	stats, exists := rm.monitoring[processID]
	if !exists {
		return nil
	}

	// Calculate final statistics
	usage := &ResourceUsage{
		PeakMemoryMB:  stats.peakMemoryMB,
		AvgCPUPercent: rm.calculateAverageCPU(stats),
	}

	// Clean up
	delete(rm.monitoring, processID)

	rm.logger.Debug("stopped monitoring process",
		zap.String("process_id", processID),
		zap.Int("peak_memory_mb", usage.PeakMemoryMB),
		zap.Float64("avg_cpu_percent", usage.AvgCPUPercent))

	return usage
}

// monitorLoop runs the resource monitoring loop
func (rm *ResourceMonitor) monitorLoop() {
	defer rm.ticker.Stop()

	for {
		select {
		case <-rm.ticker.C:
			rm.sampleResources()
		case <-rm.stopChan:
			return
		}
	}
}

// sampleResources samples resource usage for all monitored processes
func (rm *ResourceMonitor) sampleResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for processID, stats := range rm.monitoring {
		if err := rm.sampleProcessResources(processID, stats); err != nil {
			rm.logger.Debug("failed to sample process resources",
				zap.String("process_id", processID),
				zap.Error(err))
			// Don't delete here, let the caller decide when to stop monitoring
		}
	}
}

// sampleProcessResources samples resource usage for a single process
func (rm *ResourceMonitor) sampleProcessResources(processID string, stats *processStats) error {
	// Get process usage information
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_CHILDREN, &rusage); err != nil {
		// Fallback to less accurate method
		return rm.sampleProcessResourcesFallback(processID, stats)
	}

	// Calculate memory usage (in MB)
	memoryMB := int(rusage.Maxrss / 1024) // rusage.Maxrss is in KB on most systems
	if memoryMB > stats.peakMemoryMB {
		stats.peakMemoryMB = memoryMB
	}

	// Calculate CPU usage
	currentTime := time.Now()
	currentCPUTime := time.Duration(rusage.Utime.Sec)*time.Second + time.Duration(rusage.Utime.Usec)*time.Microsecond +
		time.Duration(rusage.Stime.Sec)*time.Second + time.Duration(rusage.Stime.Usec)*time.Microsecond

	if !stats.lastSampleTime.IsZero() {
		timeDelta := currentTime.Sub(stats.lastSampleTime)
		cpuDelta := currentCPUTime - stats.lastCPUTime

		if timeDelta > 0 {
			cpuPercent := float64(cpuDelta) / float64(timeDelta) * 100.0
			stats.cpuSamples = append(stats.cpuSamples, cpuPercent)

			// Keep only recent samples to prevent unbounded memory growth
			if len(stats.cpuSamples) > 100 {
				stats.cpuSamples = stats.cpuSamples[1:]
			}
		}
	}

	stats.lastCPUTime = currentCPUTime
	stats.lastSampleTime = currentTime
	stats.sampleCount++

	return nil
}

// sampleProcessResourcesFallback provides a fallback resource sampling method
func (rm *ResourceMonitor) sampleProcessResourcesFallback(processID string, stats *processStats) error {
	// This is a simplified fallback that doesn't provide accurate measurements
	// In a production system, you would implement platform-specific resource gathering

	// For now, just record that we attempted sampling
	stats.sampleCount++

	// Use runtime memory stats as a rough approximation
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Very rough approximation - not accurate for individual processes
	memoryMB := int(memStats.Alloc / 1024 / 1024)
	if memoryMB > stats.peakMemoryMB {
		stats.peakMemoryMB = memoryMB
	}

	return nil
}

// calculateAverageCPU calculates the average CPU usage from samples
func (rm *ResourceMonitor) calculateAverageCPU(stats *processStats) float64 {
	if len(stats.cpuSamples) == 0 {
		return 0.0
	}

	var total float64
	for _, sample := range stats.cpuSamples {
		total += sample
	}

	return total / float64(len(stats.cpuSamples))
}

// GetCurrentStats returns current resource statistics for a process
func (rm *ResourceMonitor) GetCurrentStats(processID string) *ResourceUsage {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats, exists := rm.monitoring[processID]
	if !exists {
		return nil
	}

	return &ResourceUsage{
		PeakMemoryMB:  stats.peakMemoryMB,
		AvgCPUPercent: rm.calculateAverageCPU(stats),
	}
}

// GetAllStats returns resource statistics for all monitored processes
func (rm *ResourceMonitor) GetAllStats() map[string]*ResourceUsage {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	result := make(map[string]*ResourceUsage)
	for processID, stats := range rm.monitoring {
		result[processID] = &ResourceUsage{
			PeakMemoryMB:  stats.peakMemoryMB,
			AvgCPUPercent: rm.calculateAverageCPU(stats),
		}
	}

	return result
}

// Close shuts down the resource monitor
func (rm *ResourceMonitor) Close() {
	close(rm.stopChan)

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Clear monitoring data
	rm.monitoring = make(map[string]*processStats)

	rm.logger.Debug("resource monitor closed")
}
