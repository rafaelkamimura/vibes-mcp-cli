# Security Guide

This comprehensive security guide covers all security features, patterns, and best practices for the vibes-mcp-cli Claude Code session manager. The system implements enterprise-grade security measures to protect against common vulnerabilities and ensure safe operation in production environments.

## Table of Contents

- [Security Architecture](#security-architecture)
- [File System Security](#file-system-security)
- [Process Security](#process-security)
- [Session Security](#session-security)
- [Network Security](#network-security)
- [Configuration Security](#configuration-security)
- [Audit and Monitoring](#audit-and-monitoring)
- [Security Best Practices](#security-best-practices)
- [Incident Response](#incident-response)

## Security Architecture

### Defense in Depth

The system implements multiple layers of security controls:

```
┌─────────────────── User Interface Layer ───────────────────┐
│ • Input validation                                          │
│ • XSS prevention                                            │
│ • UI security controls                                      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────── Application Layer ──────────────────────┐
│ • Authentication & authorization                            │
│ • Business logic security                                   │
│ • Session management security                               │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────── File System Layer ──────────────────────┐
│ • Path traversal prevention                                 │
│ • Access control enforcement                                │
│ • File operation validation                                 │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────── Process Layer ───────────────────────────┐
│ • Process isolation                                          │
│ • Resource limits                                            │
│ • Execution sandboxing                                       │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────── System Layer ────────────────────────────┐
│ • OS-level security                                          │
│ • Network isolation                                          │
│ • Hardware security features                                 │
└─────────────────────────────────────────────────────────────┘
```

### Security Principles

1. **Principle of Least Privilege**: Minimal necessary permissions
2. **Defense in Depth**: Multiple layers of security controls
3. **Fail Secure**: Secure defaults and safe failure modes
4. **Security by Design**: Security integrated from architecture phase
5. **Zero Trust**: Verify all operations and access requests

## File System Security

### Path Traversal Prevention

The system implements comprehensive protection against path traversal attacks:

#### Security Validator Implementation

```go
type SecurityValidator struct {
    config           *SecurityConfig
    allowedPathsAbs  []string    // Resolved absolute paths
    forbiddenPathsAbs []string   // Resolved absolute paths
    mu               sync.RWMutex
}

func (sv *SecurityValidator) ValidatePath(path string) error {
    // 1. Clean and normalize path
    cleanPath := filepath.Clean(path)
    
    // 2. Resolve to absolute path
    absPath, err := filepath.Abs(cleanPath)
    if err != nil {
        return &SecurityError{Path: path, Reason: "path resolution failed"}
    }
    
    // 3. Check for path traversal sequences
    if strings.Contains(path, "..") || strings.Contains(path, "~") {
        return &SecurityError{Path: path, Reason: "path traversal attempt"}
    }
    
    // 4. URL decode check (prevent encoded traversal)
    if decoded, err := url.QueryUnescape(path); err == nil && decoded != path {
        if strings.Contains(decoded, "..") {
            return &SecurityError{Path: path, Reason: "encoded path traversal attempt"}
        }
    }
    
    // 5. Validate against allowed/forbidden paths
    return sv.validatePathAccess(absPath)
}
```

#### Attack Vector Protection

**Protected Against**:
- `../../../etc/passwd` - Classic path traversal
- `..%2F..%2F..%2Fetc%2Fpasswd` - URL-encoded traversal
- `....//....//etc/passwd` - Double-dot traversal
- `/var/www/../../../etc/passwd` - Absolute path traversal
- `~/../../etc/passwd` - Home directory traversal
- Symlink-based traversal attacks

#### Path Access Control

```go
type SecurityConfig struct {
    // Allowed base paths (allowlist)
    AllowedPaths   []string `json:"allowed_paths"`
    
    // Explicitly forbidden paths (denylist)
    ForbiddenPaths []string `json:"forbidden_paths"`
    
    // Maximum directory traversal depth
    MaxDepth       int      `json:"max_depth"`
    
    // Allow access to hidden files/directories
    AllowHidden    bool     `json:"allow_hidden"`
    
    // Maximum file size for operations (bytes)
    MaxFileSize    int64    `json:"max_file_size"`
}
```

**Default Security Configuration**:
```yaml
security:
  allowed_paths:
    - "~/projects"
    - "~/workspace"
    - "/tmp"
  forbidden_paths:
    - "/etc"
    - "/root"
    - "/boot"
    - "/sys"
    - "/proc"
    - "~/.ssh"
    - "~/.gnupg"
  max_depth: 20
  allow_hidden: false
  max_file_size: 10485760  # 10MB
```

### Symlink Security

#### Symlink Target Validation

```go
func (sv *SecurityValidator) ValidateSymlink(linkPath string) error {
    // 1. Check if file is a symlink
    fileInfo, err := os.Lstat(linkPath)
    if err != nil {
        return err
    }
    
    if fileInfo.Mode()&os.ModeSymlink == 0 {
        return nil // Not a symlink, regular validation applies
    }
    
    // 2. Resolve symlink target
    target, err := os.Readlink(linkPath)
    if err != nil {
        return &SecurityError{Path: linkPath, Reason: "symlink resolution failed"}
    }
    
    // 3. Validate symlink target
    if !filepath.IsAbs(target) {
        // Relative symlink - resolve relative to symlink directory
        target = filepath.Join(filepath.Dir(linkPath), target)
    }
    
    // 4. Recursively validate target path
    return sv.ValidatePath(target)
}
```

#### Symlink Attack Prevention

**Protected Against**:
- Symlinks pointing outside allowed directories
- Circular symlink references
- Symlinks to privileged system files
- Time-of-check-time-of-use (TOCTOU) attacks via symlinks

### File Operation Security

#### Secure File Reading

```go
func (n *Navigator) ReadFile(path string) ([]byte, error) {
    // 1. Validate path access
    if err := n.validator.ValidatePath(path); err != nil {
        return nil, err
    }
    
    // 2. Check file size before reading
    fileInfo, err := os.Stat(path)
    if err != nil {
        return nil, &FileOperationError{Path: path, Operation: "stat", Cause: err}
    }
    
    if fileInfo.Size() > n.validator.config.MaxFileSize {
        return nil, &SecurityError{
            Path:   path,
            Reason: fmt.Sprintf("file too large: %d bytes", fileInfo.Size()),
        }
    }
    
    // 3. Prevent TOCTOU attacks
    file, err := os.Open(path)
    if err != nil {
        return nil, &FileOperationError{Path: path, Operation: "open", Cause: err}
    }
    defer file.Close()
    
    // 4. Verify file hasn't changed since stat
    currentInfo, err := file.Stat()
    if err != nil {
        return nil, err
    }
    
    if !os.SameFile(fileInfo, currentInfo) {
        return nil, &SecurityError{Path: path, Reason: "file changed during access"}
    }
    
    // 5. Read with size limit
    return ioutil.ReadAll(io.LimitReader(file, n.validator.config.MaxFileSize))
}
```

### Resource Protection

#### Directory Size Limits

```go
func (n *Navigator) LoadChildren(node *FileNode) error {
    // 1. Check directory depth
    depth := n.calculateDepth(node.Path)
    if depth > n.validator.config.MaxDepth {
        return &SecurityError{
            Path:   node.Path,
            Reason: fmt.Sprintf("directory depth exceeds limit: %d", depth),
        }
    }
    
    // 2. Limit number of children loaded
    entries, err := ioutil.ReadDir(node.Path)
    if err != nil {
        return err
    }
    
    const maxChildren = 1000
    if len(entries) > maxChildren {
        entries = entries[:maxChildren]
        // Log warning about truncation
    }
    
    // 3. Process children with resource monitoring
    return n.processDirectoryEntries(node, entries)
}
```

## Process Security

### Execution Sandboxing

#### Process Isolation

```go
type Executor struct {
    resourceMonitor *ResourceMonitor
    processes       map[int]*Process
    limits          *ResourceLimits
    logger          *zap.Logger
    mu              sync.RWMutex
}

type ResourceLimits struct {
    MaxMemory     int64         // Maximum memory usage (bytes)
    MaxCPU        time.Duration // Maximum CPU time
    MaxProcesses  int           // Maximum number of processes
    MaxFileSize   int64         // Maximum file size
    MaxOpenFiles  int           // Maximum open file descriptors
    Timeout       time.Duration // Process timeout
}
```

#### Secure Process Execution

```go
func (e *Executor) StartProcess(config *ProcessConfig) (*Process, error) {
    // 1. Validate process configuration
    if err := e.validateProcessConfig(config); err != nil {
        return nil, err
    }
    
    // 2. Check resource limits
    if len(e.processes) >= e.limits.MaxProcesses {
        return nil, fmt.Errorf("maximum process limit reached: %d", e.limits.MaxProcesses)
    }
    
    // 3. Prepare secure execution environment
    cmd := exec.Command(config.Command, config.Args...)
    cmd.Dir = config.WorkingDir
    cmd.Env = e.sanitizeEnvironment(config.Environment)
    
    // 4. Set resource limits (Unix systems)
    if runtime.GOOS != "windows" {
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Setpgid: true, // Create new process group
        }
    }
    
    // 5. Start process with monitoring
    process := &Process{
        cmd:     cmd,
        config:  config,
        monitor: e.resourceMonitor,
        logger:  e.logger,
    }
    
    return process, process.Start()
}
```

### Resource Monitoring

#### Real-Time Resource Tracking

```go
type ResourceMonitor struct {
    processes map[int]*ProcessInfo
    limits    *ResourceLimits
    alerts    chan ResourceAlert
    mu        sync.RWMutex
}

func (rm *ResourceMonitor) MonitorProcess(pid int) {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        usage, err := rm.getResourceUsage(pid)
        if err != nil {
            continue
        }
        
        // Check memory limit
        if usage.Memory > rm.limits.MaxMemory {
            rm.alerts <- ResourceAlert{
                PID:     pid,
                Type:    AlertMemoryLimit,
                Value:   usage.Memory,
                Limit:   rm.limits.MaxMemory,
            }
        }
        
        // Check CPU limit
        if usage.CPUTime > rm.limits.MaxCPU {
            rm.alerts <- ResourceAlert{
                PID:     pid,
                Type:    AlertCPULimit,
                Value:   int64(usage.CPUTime),
                Limit:   int64(rm.limits.MaxCPU),
            }
        }
    }
}
```

#### Process Termination Controls

```go
func (p *Process) Kill() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.cmd == nil || p.cmd.Process == nil {
        return nil
    }
    
    // 1. Try graceful termination first
    if err := p.cmd.Process.Signal(syscall.SIGTERM); err == nil {
        // Wait for graceful shutdown
        done := make(chan error, 1)
        go func() {
            done <- p.cmd.Wait()
        }()
        
        select {
        case <-done:
            return nil
        case <-time.After(5 * time.Second):
            // Fall through to force kill
        }
    }
    
    // 2. Force kill if graceful termination failed
    if runtime.GOOS != "windows" {
        // Kill entire process group
        return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
    }
    
    return p.cmd.Process.Kill()
}
```

## Session Security

### Session Isolation

#### Session Sandboxing

```go
type Session struct {
    id          string
    name        string
    sandbox     *SessionSandbox
    limits      *SessionLimits
    security    *SessionSecurity
}

type SessionSandbox struct {
    WorkingDir    string            // Isolated working directory
    Environment   map[string]string // Sanitized environment variables
    FileLimits    *FileLimits      // File access restrictions
    NetworkLimits *NetworkLimits   // Network access restrictions
}
```

#### Environment Variable Sanitization

```go
func (e *Executor) sanitizeEnvironment(env map[string]string) []string {
    // Start with minimal safe environment
    safeEnv := []string{
        "PATH=/usr/local/bin:/usr/bin:/bin",
        "HOME=" + os.Getenv("HOME"),
        "USER=" + os.Getenv("USER"),
        "LANG=en_US.UTF-8",
    }
    
    // Sanitize provided environment variables
    for key, value := range env {
        // Block dangerous environment variables
        if isDangerousEnvVar(key) {
            continue
        }
        
        // Sanitize value
        if sanitizedValue := sanitizeEnvValue(value); sanitizedValue != "" {
            safeEnv = append(safeEnv, key+"="+sanitizedValue)
        }
    }
    
    return safeEnv
}

func isDangerousEnvVar(key string) bool {
    dangerous := []string{
        "LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
        "PYTHONPATH", "RUBYLIB", "PERL5LIB",
        "SSH_AUTH_SOCK", "GPG_AGENT_INFO",
    }
    
    for _, d := range dangerous {
        if strings.ToUpper(key) == d {
            return true
        }
    }
    return false
}
```

### Session Storage Security

#### Secure Session Persistence

```go
func (s *Session) Save() error {
    sessionData := &SessionData{
        ID:          s.id,
        Name:        s.name,
        Config:      s.config,
        History:     s.sanitizeHistory(),
        CreatedAt:   s.createdAt,
        UpdatedAt:   time.Now(),
    }
    
    // 1. Encrypt sensitive data
    encryptedData, err := s.encrypt(sessionData)
    if err != nil {
        return fmt.Errorf("failed to encrypt session data: %w", err)
    }
    
    // 2. Write to secure storage
    sessionPath := filepath.Join(s.storagePath, s.id+".session")
    
    // Create with restrictive permissions
    file, err := os.OpenFile(sessionPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
    if err != nil {
        return err
    }
    defer file.Close()
    
    return json.NewEncoder(file).Encode(encryptedData)
}
```

#### History Sanitization

```go
func (s *Session) sanitizeHistory() []HistoryEntry {
    var sanitized []HistoryEntry
    
    for _, entry := range s.history {
        // Remove sensitive patterns
        sanitizedInput := s.removeSensitiveData(entry.Input)
        sanitizedOutput := s.removeSensitiveData(entry.Output)
        
        sanitized = append(sanitized, HistoryEntry{
            Timestamp: entry.Timestamp,
            Input:     sanitizedInput,
            Output:    sanitizedOutput,
        })
    }
    
    return sanitized
}

func (s *Session) removeSensitiveData(text string) string {
    // Patterns for sensitive data
    patterns := []*regexp.Regexp{
        regexp.MustCompile(`password\s*[=:]\s*\S+`),
        regexp.MustCompile(`api[_-]?key\s*[=:]\s*\S+`),
        regexp.MustCompile(`token\s*[=:]\s*\S+`),
        regexp.MustCompile(`secret\s*[=:]\s*\S+`),
        regexp.MustCompile(`\b[A-Za-z0-9+/]{20,}={0,2}\b`), // Base64 tokens
    }
    
    result := text
    for _, pattern := range patterns {
        result = pattern.ReplaceAllString(result, "[REDACTED]")
    }
    
    return result
}
```

## Network Security

### HTTP Server Security

#### Request Validation

```go
func (s *Server) validateRequest(r *http.Request) error {
    // 1. Check request size
    r.Body = http.MaxBytesReader(nil, r.Body, 10<<20) // 10MB limit
    
    // 2. Validate headers
    if err := s.validateHeaders(r.Header); err != nil {
        return err
    }
    
    // 3. Check for malicious patterns
    if err := s.checkMaliciousPatterns(r); err != nil {
        return err
    }
    
    return nil
}

func (s *Server) validateHeaders(headers http.Header) error {
    // Check for dangerous headers
    dangerous := []string{
        "X-Forwarded-For", "X-Real-IP", "X-Original-URL",
    }
    
    for _, header := range dangerous {
        if value := headers.Get(header); value != "" {
            if !s.isTrustedProxy(value) {
                return fmt.Errorf("untrusted proxy header: %s", header)
            }
        }
    }
    
    return nil
}
```

#### Rate Limiting

```go
type RateLimiter struct {
    clients map[string]*ClientLimiter
    mu      sync.RWMutex
}

type ClientLimiter struct {
    tokens    int
    lastRefill time.Time
    mu        sync.Mutex
}

func (rl *RateLimiter) Allow(clientID string) bool {
    rl.mu.RLock()
    client, exists := rl.clients[clientID]
    rl.mu.RUnlock()
    
    if !exists {
        rl.mu.Lock()
        client = &ClientLimiter{
            tokens:    100, // Initial tokens
            lastRefill: time.Now(),
        }
        rl.clients[clientID] = client
        rl.mu.Unlock()
    }
    
    return client.consumeToken()
}
```

### TLS Configuration

#### Secure TLS Settings

```go
func (s *Server) configureTLS() *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
        PreferServerCipherSuites: true,
        CurvePreferences: []tls.CurveID{
            tls.CurveP256,
            tls.X25519,
        },
    }
}
```

## Configuration Security

### Secure Configuration Management

#### Configuration Validation

```go
func (c *Config) Validate() error {
    // 1. Validate security settings
    if err := c.validateSecurityConfig(); err != nil {
        return fmt.Errorf("security configuration invalid: %w", err)
    }
    
    // 2. Check for insecure settings
    if err := c.checkInsecureSettings(); err != nil {
        return fmt.Errorf("insecure configuration detected: %w", err)
    }
    
    // 3. Validate paths and permissions
    return c.validatePaths()
}

func (c *Config) validateSecurityConfig() error {
    if len(c.Security.AllowedPaths) == 0 {
        return errors.New("no allowed paths configured")
    }
    
    if c.Security.MaxFileSize <= 0 {
        return errors.New("invalid max file size")
    }
    
    if c.Security.MaxDepth <= 0 || c.Security.MaxDepth > 100 {
        return errors.New("invalid max depth")
    }
    
    return nil
}
```

#### Secrets Management

```go
type SecretsManager struct {
    keyring   Keyring
    encrypted map[string][]byte
    mu        sync.RWMutex
}

func (sm *SecretsManager) StoreSecret(key, value string) error {
    // 1. Encrypt the secret
    encrypted, err := sm.encrypt([]byte(value))
    if err != nil {
        return err
    }
    
    // 2. Store encrypted value
    sm.mu.Lock()
    sm.encrypted[key] = encrypted
    sm.mu.Unlock()
    
    // 3. Clear plaintext from memory
    for i := range value {
        value = value[:i] + "\x00" + value[i+1:]
    }
    
    return nil
}
```

## Audit and Monitoring

### Security Event Logging

#### Audit Log Implementation

```go
type AuditLogger struct {
    logger *zap.Logger
    file   *os.File
    mu     sync.Mutex
}

type AuditEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    EventType   string    `json:"event_type"`
    UserID      string    `json:"user_id,omitempty"`
    SessionID   string    `json:"session_id,omitempty"`
    Path        string    `json:"path,omitempty"`
    Action      string    `json:"action"`
    Result      string    `json:"result"`
    Risk        string    `json:"risk_level"`
    Details     string    `json:"details,omitempty"`
}

func (al *AuditLogger) LogSecurityEvent(event AuditEvent) {
    al.mu.Lock()
    defer al.mu.Unlock()
    
    event.Timestamp = time.Now()
    
    // Log to structured logger
    al.logger.Info("security_event",
        zap.String("type", event.EventType),
        zap.String("action", event.Action),
        zap.String("result", event.Result),
        zap.String("risk", event.Risk),
    )
    
    // Write to audit file
    json.NewEncoder(al.file).Encode(event)
}
```

#### Security Metrics

```go
type SecurityMetrics struct {
    PathTraversalAttempts   int64
    FileAccessDenials       int64
    ResourceLimitViolations int64
    ProcessKills            int64
    SessionSecurityEvents   int64
    TotalSecurityEvents     int64
}

func (sm *SecurityMetrics) RecordEvent(eventType SecurityEventType) {
    atomic.AddInt64(&sm.TotalSecurityEvents, 1)
    
    switch eventType {
    case PathTraversalAttempt:
        atomic.AddInt64(&sm.PathTraversalAttempts, 1)
    case FileAccessDenial:
        atomic.AddInt64(&sm.FileAccessDenials, 1)
    case ResourceLimitViolation:
        atomic.AddInt64(&sm.ResourceLimitViolations, 1)
    // ... other cases
    }
}
```

### Intrusion Detection

#### Anomaly Detection

```go
type AnomalyDetector struct {
    baselines map[string]*Baseline
    alerter   *SecurityAlerter
    mu        sync.RWMutex
}

func (ad *AnomalyDetector) CheckAnomaly(userID string, activity UserActivity) {
    baseline := ad.getBaseline(userID)
    
    // Check for suspicious patterns
    if activity.FileAccessRate > baseline.NormalFileAccessRate*3 {
        ad.alerter.Alert(SecurityAlert{
            Type:        AnomalousFileAccess,
            UserID:      userID,
            Severity:    HighSeverity,
            Description: "Unusual file access pattern detected",
            Metrics:     map[string]interface{}{
                "access_rate": activity.FileAccessRate,
                "baseline":    baseline.NormalFileAccessRate,
            },
        })
    }
    
    // Update baseline
    baseline.Update(activity)
}
```

## Security Best Practices

### Development Security

#### Secure Coding Practices

1. **Input Validation**: Validate all user inputs
```go
func validateInput(input string) error {
    if len(input) > 1000 {
        return errors.New("input too long")
    }
    
    // Check for malicious patterns
    if matched, _ := regexp.MatchString(`[<>&"']`, input); matched {
        return errors.New("potentially malicious input")
    }
    
    return nil
}
```

2. **Error Handling**: Don't leak sensitive information
```go
func (n *Navigator) ReadFile(path string) ([]byte, error) {
    if err := n.validator.ValidatePath(path); err != nil {
        // Don't reveal exact path in error
        return nil, errors.New("file access denied")
    }
    
    // ... rest of implementation
}
```

3. **Resource Management**: Always clean up resources
```go
func (s *Session) processFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close() // Always close
    
    // Process file...
    return nil
}
```

### Deployment Security

#### Production Configuration

```yaml
# production-security.yaml
security:
  # Strict path controls
  allowed_paths:
    - "/app/workspace"
  forbidden_paths:
    - "/etc"
    - "/root"
    - "/sys"
    - "/proc"
  
  # Conservative limits
  max_file_size: 5242880  # 5MB
  max_depth: 10
  allow_hidden: false

session_manager:
  # Resource limits
  max_sessions: 5
  session_timeout: "1h"
  cleanup_interval: "30m"

# Logging configuration
logging:
  level: "info"
  audit_enabled: true
  audit_file: "/var/log/vibes-mcp-cli/audit.log"
  
# TLS configuration  
server:
  tls_min_version: "1.2"
  cipher_suites:
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
```

#### Container Security

```dockerfile
# Dockerfile with security best practices
FROM golang:1.21-alpine AS builder

# Create non-root user for build
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Build application
COPY . /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o vibes-mcp-cli

# Production image
FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /src/vibes-mcp-cli /vibes-mcp-cli

# Run as non-root user
USER appuser

# Expose only necessary port
EXPOSE 8080

ENTRYPOINT ["/vibes-mcp-cli"]
```

### Operational Security

#### Monitoring and Alerting

1. **Set up security monitoring**:
```bash
# Monitor audit logs
tail -f /var/log/vibes-mcp-cli/audit.log | \
  jq 'select(.risk_level == "HIGH")'

# Alert on security events
grep "security_violation" /var/log/vibes-mcp-cli/app.log | \
  mail -s "Security Alert" admin@company.com
```

2. **Regular security reviews**:
```bash
# Weekly security report
vibes-mcp-cli security-report \
  --start-date $(date -d '1 week ago' +%Y-%m-%d) \
  --format json > weekly-security-report.json
```

3. **Access control auditing**:
```bash
# Review file access patterns
vibes-mcp-cli audit file-access \
  --user-id all \
  --suspicious-only
```

## Incident Response

### Security Incident Types

#### Classification

- **Level 1 - Low**: Policy violations, minor security events
- **Level 2 - Medium**: Suspicious activity, failed intrusion attempts  
- **Level 3 - High**: Successful intrusion, data access violations
- **Level 4 - Critical**: System compromise, data breach

#### Response Procedures

**Immediate Response**:
1. Isolate affected systems
2. Preserve evidence
3. Assess scope of impact
4. Notify stakeholders

**Investigation**:
1. Analyze audit logs
2. Identify attack vectors
3. Determine data accessed
4. Document findings

**Recovery**:
1. Patch vulnerabilities
2. Update security controls  
3. Restore from clean backups
4. Monitor for re-attack

### Incident Response Commands

```bash
# Emergency session termination
vibes-mcp-cli emergency shutdown --reason "security incident"

# Lock user account
vibes-mcp-cli user lock --user-id suspicious-user

# Export audit logs
vibes-mcp-cli audit export \
  --start-date 2024-01-01 \
  --end-date 2024-01-31 \
  --format json \
  --output incident-logs.json

# Security status check
vibes-mcp-cli security status --verbose
```

---

This comprehensive security guide covers all aspects of security in the vibes-mcp-cli system. Regular review and updates of security practices are essential to maintain a strong security posture. For implementation details, see the [API Reference](API-Reference.md) and [Architecture](Architecture.md) documentation.