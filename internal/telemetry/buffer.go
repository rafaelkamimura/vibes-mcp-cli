package telemetry

import (
	"errors"
	"sync"
)

var (
	ErrBufferFull = errors.New("buffer is full")
)

// ringBuffer implements a thread-safe circular buffer for log entries
type ringBuffer struct {
	mu       sync.RWMutex
	entries  []LogEntry
	capacity int
	size     int
	head     int
	tail     int
}

// NewBuffer creates a new log buffer with the specified capacity
func NewBuffer(capacity int) Buffer {
	return &ringBuffer{
		entries:  make([]LogEntry, capacity),
		capacity: capacity,
	}
}

// Add adds a log entry to the buffer
func (b *ringBuffer) Add(entry LogEntry) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.size >= b.capacity {
		return ErrBufferFull
	}
	
	b.entries[b.tail] = entry
	b.tail = (b.tail + 1) % b.capacity
	b.size++
	
	return nil
}

// Flush returns all buffered entries and clears the buffer
func (b *ringBuffer) Flush() []LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.size == 0 {
		return nil
	}
	
	entries := make([]LogEntry, b.size)
	
	// Copy entries from head to tail
	for i := 0; i < b.size; i++ {
		idx := (b.head + i) % b.capacity
		entries[i] = b.entries[idx]
	}
	
	// Reset buffer
	b.size = 0
	b.head = 0
	b.tail = 0
	
	return entries
}

// Size returns the current number of buffered entries
func (b *ringBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size
}

// IsFull returns whether the buffer is at capacity
func (b *ringBuffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.size >= b.capacity
}

// Clear removes all entries from the buffer
func (b *ringBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.size = 0
	b.head = 0
	b.tail = 0
}