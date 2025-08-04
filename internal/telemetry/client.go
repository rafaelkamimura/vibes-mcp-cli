package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// httpClient implements the Client interface
type httpClient struct {
	config      *Config
	httpClient  *http.Client
	buffer      Buffer
	rateLimiter RateLimiter
	logger      *zap.Logger
	
	// Background processing
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// Shutdown coordination
	shutdownOnce sync.Once
	flushChan    chan struct{}
}

// NewClient creates a new telemetry client
func NewClient(config *Config, logger *zap.Logger) Client {
	if config == nil {
		config = DefaultConfig()
	}
	
	if logger == nil {
		logger = zap.NewNop()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &httpClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxConnsPerHost:     5,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
			},
		},
		buffer:      NewBuffer(config.BufferSize),
		rateLimiter: NewRateLimiter(),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		flushChan:   make(chan struct{}, 1),
	}
	
	if config.Enabled {
		client.startBackgroundWorker()
	}
	
	return client
}

// IsEnabled returns whether telemetry is currently enabled
func (c *httpClient) IsEnabled() bool {
	return c.config.Enabled
}

// Log sends a log entry to the telemetry system
func (c *httpClient) Log(ctx context.Context, entry LogEntry) error {
	if !c.config.Enabled {
		return nil
	}
	
	// Set client info and session info if not provided
	if entry.ClientName == "" {
		entry.ClientName = c.config.ClientName
	}
	if entry.ClientVersion == "" {
		entry.ClientVersion = c.config.ClientVersion
	}
	if entry.SessionID == "" {
		entry.SessionID = c.config.SessionID
	}
	
	// Add user info to metadata instead of deprecated UserID field
	if c.config.UserID != "" {
		if entry.Metadata == nil {
			entry.Metadata = make(map[string]interface{})
		}
		// Only add if not already set
		if _, exists := entry.Metadata["user_id"]; !exists {
			entry.Metadata["user_id"] = c.config.UserID
		}
		entry.UserID = c.config.UserID // Keep for internal use (not serialized)
	}
	
	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = c.generateID()
	}
	
	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	
	// Try to add to buffer
	if err := c.buffer.Add(entry); err != nil {
		if err == ErrBufferFull {
			// Buffer full, try to flush immediately
			c.triggerFlush()
			// Try again
			if err := c.buffer.Add(entry); err != nil {
				c.logger.Warn("Failed to buffer telemetry log", zap.Error(err))
				return err
			}
		} else {
			return err
		}
	}
	
	// Trigger flush if buffer is getting full
	if c.buffer.Size() >= c.config.BatchSize {
		c.triggerFlush()
	}
	
	return nil
}

// LogBatch sends multiple log entries in a single request
func (c *httpClient) LogBatch(ctx context.Context, entries []LogEntry) error {
	if !c.config.Enabled || len(entries) == 0 {
		return nil
	}
	
	// Process entries
	for i := range entries {
		if entries[i].ClientName == "" {
			entries[i].ClientName = c.config.ClientName
		}
		if entries[i].ClientVersion == "" {
			entries[i].ClientVersion = c.config.ClientVersion
		}
		if entries[i].SessionID == "" {
			entries[i].SessionID = c.config.SessionID
		}
		
		// Add user info to metadata instead of deprecated UserID field
		if c.config.UserID != "" {
			if entries[i].Metadata == nil {
				entries[i].Metadata = make(map[string]interface{})
			}
			// Only add if not already set
			if _, exists := entries[i].Metadata["user_id"]; !exists {
				entries[i].Metadata["user_id"] = c.config.UserID
			}
			entries[i].UserID = c.config.UserID // Keep for internal use (not serialized)
		}
		
		if entries[i].ID == "" {
			entries[i].ID = c.generateID()
		}
		if entries[i].Timestamp.IsZero() {
			entries[i].Timestamp = time.Now().UTC()
		}
	}
	
	return c.sendBatch(ctx, entries)
}

// Flush forces all buffered logs to be sent immediately
func (c *httpClient) Flush(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}
	
	entries := c.buffer.Flush()
	if len(entries) == 0 {
		return nil
	}
	
	return c.sendBatch(ctx, entries)
}

// Close gracefully shuts down the client, flushing remaining logs
func (c *httpClient) Close(ctx context.Context) error {
	var err error
	
	c.shutdownOnce.Do(func() {
		// Cancel background worker
		c.cancel()
		
		// Flush remaining logs
		if flushErr := c.Flush(ctx); flushErr != nil {
			err = flushErr
		}
		
		// Wait for background worker to finish
		c.wg.Wait()
		
		// Close HTTP client
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	})
	
	return err
}

// startBackgroundWorker starts the background goroutine for periodic flushing
func (c *httpClient) startBackgroundWorker() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		
		ticker := time.NewTicker(c.config.FlushInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.flushBuffered()
			case <-c.flushChan:
				c.flushBuffered()
			}
		}
	}()
}

// triggerFlush signals the background worker to flush immediately
func (c *httpClient) triggerFlush() {
	select {
	case c.flushChan <- struct{}{}:
	default:
		// Channel full, flush already pending
	}
}

// flushBuffered flushes buffered logs in the background
func (c *httpClient) flushBuffered() {
	entries := c.buffer.Flush()
	if len(entries) == 0 {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()
	
	if err := c.sendBatch(ctx, entries); err != nil {
		c.logger.Warn("Failed to send telemetry batch", zap.Error(err))
		
		// Put entries back in buffer if there's space
		for _, entry := range entries {
			if c.buffer.IsFull() {
				break
			}
			c.buffer.Add(entry)
		}
	}
}

// sendBatch sends a batch of log entries to the telemetry API
func (c *httpClient) sendBatch(ctx context.Context, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	
	// Split into smaller batches if needed
	for i := 0; i < len(entries); i += c.config.BatchSize {
		end := i + c.config.BatchSize
		if end > len(entries) {
			end = len(entries)
		}
		
		batch := entries[i:end]
		if err := c.sendSingleBatch(ctx, batch); err != nil {
			return err
		}
	}
	
	return nil
}

// sendSingleBatch sends a single batch with retry logic
func (c *httpClient) sendSingleBatch(ctx context.Context, entries []LogEntry) error {
	batchReq := BatchRequest{
		Logs: entries,
	}
	
	var lastErr error
	backoff := c.config.RetryBackoff
	
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Check rate limit
		if !c.rateLimiter.Allow() {
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return fmt.Errorf("rate limit wait failed: %w", err)
			}
		}
		
		// Send request
		if err := c.sendHTTPRequest(ctx, batchReq); err != nil {
			lastErr = err
			c.logger.Debug("Telemetry batch failed", 
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			
			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return err
			}
			
			// Exponential backoff
			if attempt < c.config.MaxRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					backoff *= 2
				}
			}
		} else {
			// Success
			c.logger.Debug("Telemetry batch sent successfully",
				zap.Int("logCount", len(entries)))
			return nil
		}
	}
	
	return fmt.Errorf("failed to send telemetry batch after %d attempts: %w", 
		c.config.MaxRetries+1, lastErr)
}

// sendHTTPRequest sends a single HTTP request to the telemetry API
func (c *httpClient) sendHTTPRequest(ctx context.Context, batchReq BatchRequest) error {
	jsonData, err := json.Marshal(batchReq)
	if err != nil {
		return fmt.Errorf("failed to marshal batch request: %w", err)
	}
	
	url := c.config.BaseURL + "/api/telemetry/logs"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "vibes-mcp-cli-telemetry/1.0")
	
	// Set authentication header - prefer API key, fallback to JWT
	if c.config.APIKey != "" {
		req.Header.Set("X-API-Key", c.config.APIKey)
	} else if c.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.AuthToken)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		// Response decode failed, but status was OK, so assume success
		c.logger.Debug("Failed to decode telemetry response", zap.Error(err))
		return nil
	}
	
	if batchResp.Count == 0 {
		return fmt.Errorf("batch processing failed: no logs were created")
	}
	
	return nil
}

// generateID generates a random ID for logs and batches
func (c *httpClient) generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}