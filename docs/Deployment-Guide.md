# Deployment Guide

This comprehensive deployment guide covers production deployment, configuration management, monitoring, and maintenance for the vibes-mcp-cli Claude Code session manager. Whether you're deploying to a single server, container environment, or distributed system, this guide provides the necessary information for secure and reliable deployment.

## Table of Contents

- [Deployment Overview](#deployment-overview)
- [Single Server Deployment](#single-server-deployment)
- [Container Deployment](#container-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Configuration Management](#configuration-management)
- [Security Hardening](#security-hardening)
- [Monitoring and Logging](#monitoring-and-logging)
- [Backup and Recovery](#backup-and-recovery)
- [Performance Tuning](#performance-tuning)
- [Maintenance and Updates](#maintenance-and-updates)

## Deployment Overview

### Architecture Components

```
┌─────────────────── Load Balancer ──────────────────┐
│ • SSL/TLS termination                               │
│ • Request routing                                   │
│ • Rate limiting                                     │
└─────────────────────────────────────────────────────┘
                          │
┌─────────────────── Application Tier ───────────────┐
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │
│ │ vibes-mcp-  │ │ vibes-mcp-  │ │ vibes-mcp-  │   │
│ │ cli (TUI)   │ │ cli (HTTP)  │ │ cli (API)   │   │
│ └─────────────┘ └─────────────┘ └─────────────┘   │
└─────────────────────────────────────────────────────┘
                          │
┌─────────────────── Storage Tier ───────────────────┐
│ • Session storage (persistent volumes)             │
│ • Configuration files                               │
│ • Log storage                                       │
│ • Backup storage                                    │
└─────────────────────────────────────────────────────┘
```

### Deployment Patterns

1. **Single Server**: All components on one machine
2. **Multi-Server**: Distributed across multiple servers
3. **Container**: Docker/Podman containers
4. **Kubernetes**: Orchestrated container deployment
5. **Hybrid**: Mix of containers and traditional deployment

### Prerequisites

**System Requirements**:
- Linux/Unix-based OS (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- Go 1.21+ (for building from source)
- Docker 20.10+ (for container deployment)
- Kubernetes 1.24+ (for K8s deployment)

**Resource Requirements**:
- **Minimum**: 2 CPU cores, 4GB RAM, 10GB storage
- **Recommended**: 4 CPU cores, 8GB RAM, 50GB storage
- **High Load**: 8+ CPU cores, 16GB+ RAM, 100GB+ storage

## Single Server Deployment

### System Preparation

#### 1. Update System

```bash
# Ubuntu/Debian
sudo apt update && sudo apt upgrade -y

# CentOS/RHEL
sudo yum update -y

# Install required packages
sudo apt install -y curl wget git build-essential supervisor nginx
```

#### 2. Create Application User

```bash
# Create dedicated user
sudo useradd -r -m -s /bin/bash vibes-mcp
sudo mkdir -p /opt/vibes-mcp-cli
sudo chown vibes-mcp:vibes-mcp /opt/vibes-mcp-cli

# Create directories
sudo -u vibes-mcp mkdir -p /opt/vibes-mcp-cli/{bin,config,logs,sessions,backups}
```

#### 3. Install Application

```bash
# Download and install binary
cd /opt/vibes-mcp-cli
sudo -u vibes-mcp wget https://github.com/your-org/vibes-mcp-cli/releases/latest/download/vibes-mcp-cli-linux-amd64
sudo -u vibes-mcp mv vibes-mcp-cli-linux-amd64 bin/vibes-mcp-cli
sudo -u vibes-mcp chmod +x bin/vibes-mcp-cli

# Verify installation
sudo -u vibes-mcp ./bin/vibes-mcp-cli --version
```

### Configuration

#### 1. Production Configuration

```yaml
# /opt/vibes-mcp-cli/config/production.yaml
api_key: "${OPENAI_API_KEY}"
provider: "openai"
log_level: "info"

# Server configuration
server:
  host: "127.0.0.1"
  port: 8080
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/vibes-mcp-cli.crt"
    key_file: "/etc/ssl/private/vibes-mcp-cli.key"
  
# Security configuration
security:
  allowed_paths:
    - "/opt/vibes-mcp-cli/workspace"
    - "/tmp/vibes-workspace"
  forbidden_paths:
    - "/etc"
    - "/root"
    - "/sys"
    - "/proc"
  max_file_size: 10485760  # 10MB
  max_depth: 20
  allow_hidden: false

# Session management
session_manager:
  storage_path: "/opt/vibes-mcp-cli/sessions"
  max_sessions: 20
  session_timeout: "2h"
  cleanup_interval: "1h"
  auto_cleanup: true

# Logging
logging:
  level: "info"
  file: "/opt/vibes-mcp-cli/logs/app.log"
  max_size: 100  # MB
  max_backups: 10
  max_age: 30  # days
  audit_enabled: true
  audit_file: "/opt/vibes-mcp-cli/logs/audit.log"
```

#### 2. Environment Variables

```bash
# /opt/vibes-mcp-cli/config/.env
OPENAI_API_KEY=your-api-key-here
VIBES_CONFIG_FILE=/opt/vibes-mcp-cli/config/production.yaml
VIBES_LOG_LEVEL=info
VIBES_DATA_DIR=/opt/vibes-mcp-cli
```

### Service Configuration

#### 1. Systemd Service

```ini
# /etc/systemd/system/vibes-mcp-cli.service
[Unit]
Description=Vibes MCP CLI Session Manager
After=network.target
Wants=network.target

[Service]
Type=simple
User=vibes-mcp
Group=vibes-mcp
WorkingDirectory=/opt/vibes-mcp-cli
ExecStart=/opt/vibes-mcp-cli/bin/vibes-mcp-cli serve --config /opt/vibes-mcp-cli/config/production.yaml
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=vibes-mcp-cli

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/vibes-mcp-cli

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

#### 2. Start Service

```bash
# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable vibes-mcp-cli
sudo systemctl start vibes-mcp-cli

# Check status
sudo systemctl status vibes-mcp-cli
```

### Reverse Proxy Configuration

#### 1. Nginx Configuration

```nginx
# /etc/nginx/sites-available/vibes-mcp-cli
upstream vibes_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name your-domain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name your-domain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload";

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;
    limit_req zone=api burst=20 nodelay;

    # Proxy configuration
    location / {
        proxy_pass http://vibes_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://vibes_backend/health;
        access_log off;
    }
}
```

#### 2. Enable Site

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/vibes-mcp-cli /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Container Deployment

### Docker Configuration

#### 1. Production Dockerfile

```dockerfile
# Multi-stage build for optimized production image
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create build user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Build application
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o vibes-mcp-cli .

# Production image
FROM scratch

# Copy certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy user information
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy application
COPY --from=builder /src/vibes-mcp-cli /vibes-mcp-cli

# Create necessary directories
USER appuser
WORKDIR /app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/vibes-mcp-cli", "health-check"]

# Run application
ENTRYPOINT ["/vibes-mcp-cli"]
CMD ["serve"]
```

#### 2. Docker Compose

```yaml
# docker-compose.production.yml
version: '3.8'

services:
  vibes-mcp-cli:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: vibes-mcp-cli
    restart: unless-stopped
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - VIBES_LOG_LEVEL=info
      - VIBES_CONFIG_FILE=/app/config/production.yaml
    volumes:
      - ./config:/app/config:ro
      - vibes_sessions:/app/sessions
      - vibes_logs:/app/logs
    ports:
      - "8080:8080"
    networks:
      - vibes_network
    healthcheck:
      test: ["CMD", "/vibes-mcp-cli", "health-check"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  nginx:
    image: nginx:alpine
    container_name: vibes-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - vibes_logs:/var/log/nginx
    networks:
      - vibes_network
    depends_on:
      - vibes-mcp-cli

volumes:
  vibes_sessions:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/vibes-mcp-cli/sessions
  vibes_logs:
    driver: local

networks:
  vibes_network:
    driver: bridge
```

#### 3. Deploy with Docker Compose

```bash
# Create environment file
echo "OPENAI_API_KEY=your-api-key" > .env

# Deploy
docker-compose -f docker-compose.production.yml up -d

# Check status
docker-compose -f docker-compose.production.yml ps

# View logs
docker-compose -f docker-compose.production.yml logs -f vibes-mcp-cli
```

### Container Security

#### 1. Security Scanning

```bash
# Scan image for vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd):/root/.cache/ anchore/grype:latest \
  vibes-mcp-cli:latest

# Run security benchmark
docker run --rm --net host --pid host --userns host --cap-add audit_control \
  -v /etc:/etc:ro \
  -v /usr/bin/containerd:/usr/bin/containerd:ro \
  -v /usr/bin/runc:/usr/bin/runc:ro \
  -v /usr/lib/systemd:/usr/lib/systemd:ro \
  -v /var/lib:/var/lib:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --label docker_bench_security \
  docker/docker-bench-security
```

#### 2. Runtime Security

```yaml
# docker-compose.security.yml (additional security settings)
services:
  vibes-mcp-cli:
    security_opt:
      - no-new-privileges:true
      - apparmor:docker-default
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    read_only: true
    tmpfs:
      - /tmp:noexec,nosuid,size=100m
    ulimits:
      nproc: 65535
      nofile:
        soft: 20000
        hard: 40000
```

## Kubernetes Deployment

### Kubernetes Manifests

#### 1. Namespace and ConfigMap

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: vibes-mcp-cli
  labels:
    name: vibes-mcp-cli

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vibes-mcp-cli-config
  namespace: vibes-mcp-cli
data:
  production.yaml: |
    log_level: "info"
    server:
      host: "0.0.0.0"
      port: 8080
    security:
      allowed_paths:
        - "/app/workspace"
      forbidden_paths:
        - "/etc"
        - "/root"
      max_file_size: 10485760
      max_depth: 20
      allow_hidden: false
    session_manager:
      storage_path: "/app/sessions"
      max_sessions: 20
      session_timeout: "2h"
      cleanup_interval: "1h"
      auto_cleanup: true
    logging:
      level: "info"
      file: "/app/logs/app.log"
      audit_enabled: true
      audit_file: "/app/logs/audit.log"
```

#### 2. Secret for API Keys

```yaml
# k8s/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: vibes-mcp-cli-secrets
  namespace: vibes-mcp-cli
type: Opaque
data:
  openai-api-key: <base64-encoded-api-key>
```

#### 3. Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vibes-mcp-cli
  namespace: vibes-mcp-cli
  labels:
    app: vibes-mcp-cli
spec:
  replicas: 3
  selector:
    matchLabels:
      app: vibes-mcp-cli
  template:
    metadata:
      labels:
        app: vibes-mcp-cli
    spec:
      serviceAccountName: vibes-mcp-cli
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        runAsGroup: 1001
        fsGroup: 1001
      containers:
      - name: vibes-mcp-cli
        image: your-registry/vibes-mcp-cli:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: vibes-mcp-cli-secrets
              key: openai-api-key
        - name: VIBES_CONFIG_FILE
          value: "/app/config/production.yaml"
        volumeMounts:
        - name: config
          mountPath: /app/config
          readOnly: true
        - name: sessions
          mountPath: /app/sessions
        - name: logs
          mountPath: /app/logs
        - name: tmp
          mountPath: /tmp
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
      volumes:
      - name: config
        configMap:
          name: vibes-mcp-cli-config
      - name: sessions
        persistentVolumeClaim:
          claimName: vibes-mcp-cli-sessions
      - name: logs
        persistentVolumeClaim:
          claimName: vibes-mcp-cli-logs
      - name: tmp
        emptyDir: {}
```

#### 4. Service and Ingress

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: vibes-mcp-cli-service
  namespace: vibes-mcp-cli
spec:
  selector:
    app: vibes-mcp-cli
  ports:
  - name: http
    port: 80
    targetPort: 8080
  type: ClusterIP

---
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: vibes-mcp-cli-ingress
  namespace: vibes-mcp-cli
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/rate-limit: "100"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - your-domain.com
    secretName: vibes-mcp-cli-tls
  rules:
  - host: your-domain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: vibes-mcp-cli-service
            port:
              number: 80
```

#### 5. Persistent Volumes

```yaml
# k8s/pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: vibes-mcp-cli-sessions
  namespace: vibes-mcp-cli
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: vibes-mcp-cli-logs
  namespace: vibes-mcp-cli
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: standard
```

#### 6. Deploy to Kubernetes

```bash
# Deploy all components
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n vibes-mcp-cli
kubectl get services -n vibes-mcp-cli
kubectl get ingress -n vibes-mcp-cli

# View logs
kubectl logs -f deployment/vibes-mcp-cli -n vibes-mcp-cli
```

## Configuration Management

### Environment-Specific Configuration

#### 1. Configuration Structure

```
config/
├── base.yaml           # Base configuration
├── development.yaml    # Development overrides
├── staging.yaml        # Staging overrides
├── production.yaml     # Production overrides
└── secrets/
    ├── dev-secrets.yaml
    ├── staging-secrets.yaml
    └── prod-secrets.yaml
```

#### 2. Configuration Merging

```yaml
# config/base.yaml
log_level: "info"
server:
  port: 8080
  
session_manager:
  max_sessions: 10
  cleanup_interval: "1h"
  
security:
  max_file_size: 5242880  # 5MB
  max_depth: 15

---
# config/production.yaml (overrides)
log_level: "warn"
server:
  host: "0.0.0.0"
  tls:
    enabled: true
    
session_manager:
  max_sessions: 50
  
security:
  max_file_size: 10485760  # 10MB
  allowed_paths:
    - "/opt/workspace"
```

#### 3. Configuration Validation

```bash
# Validate configuration
vibes-mcp-cli config validate --config production.yaml

# Show merged configuration
vibes-mcp-cli config show --config production.yaml --format yaml

# Test configuration
vibes-mcp-cli config test --config production.yaml
```

### Secrets Management

#### 1. HashiCorp Vault Integration

```yaml
# config/vault.yaml
vault:
  address: "https://vault.company.com:8200"
  auth_method: "kubernetes"
  role: "vibes-mcp-cli"
  secrets:
    - path: "secret/vibes-mcp-cli/openai"
      key: "api_key"
      env: "OPENAI_API_KEY"
```

#### 2. Kubernetes Secrets

```bash
# Create secret from file
kubectl create secret generic vibes-mcp-cli-secrets \
  --from-file=openai-api-key=./secrets/openai-key.txt \
  -n vibes-mcp-cli

# Create secret from literal
kubectl create secret generic vibes-mcp-cli-config \
  --from-literal=database-password='S3cur3P@ssw0rd' \
  -n vibes-mcp-cli
```

## Security Hardening

### System Security

#### 1. Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw deny 8080/tcp  # Block direct access to app port
sudo ufw enable

# iptables
iptables -A INPUT -p tcp --dport 22 -j ACCEPT
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -j DROP
```

#### 2. SSL/TLS Configuration

```bash
# Generate self-signed certificate (development)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/ssl/private/vibes-mcp-cli.key \
  -out /etc/ssl/certs/vibes-mcp-cli.crt

# Let's Encrypt certificate (production)
sudo certbot --nginx -d your-domain.com
```

#### 3. File Permissions

```bash
# Set secure permissions
sudo chown -R vibes-mcp:vibes-mcp /opt/vibes-mcp-cli
sudo chmod 755 /opt/vibes-mcp-cli
sudo chmod 750 /opt/vibes-mcp-cli/config
sudo chmod 640 /opt/vibes-mcp-cli/config/*.yaml
sudo chmod 600 /opt/vibes-mcp-cli/config/.env
sudo chmod 755 /opt/vibes-mcp-cli/bin/vibes-mcp-cli
```

### Application Security

#### 1. Security Headers

```yaml
# config/security.yaml
server:
  security_headers:
    x_frame_options: "DENY"
    x_content_type_options: "nosniff"
    x_xss_protection: "1; mode=block"
    strict_transport_security: "max-age=31536000; includeSubDomains"
    content_security_policy: "default-src 'self'"
```

#### 2. Rate Limiting

```yaml
# config/rate-limit.yaml
rate_limiting:
  enabled: true
  requests_per_minute: 100
  burst_size: 20
  ip_whitelist:
    - "127.0.0.1"
    - "10.0.0.0/8"
```

## Monitoring and Logging

### Application Monitoring

#### 1. Health Check Endpoints

```go
// Implement health check endpoints
func (s *Server) setupHealthChecks() {
    s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
    s.router.HandleFunc("/ready", s.readinessCheck).Methods("GET")
    s.router.HandleFunc("/metrics", s.metricsHandler).Methods("GET")
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "version":   s.version,
        "uptime":    time.Since(s.startTime).String(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}
```

#### 2. Prometheus Metrics

```yaml
# docker-compose.monitoring.yml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

#### 3. Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'vibes-mcp-cli'
    static_configs:
      - targets: ['vibes-mcp-cli:8080']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

### Logging Configuration

#### 1. Structured Logging

```yaml
# config/logging.yaml
logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/opt/vibes-mcp-cli/logs/app.log"
  rotation:
    max_size: 100  # MB
    max_backups: 10
    max_age: 30    # days
    compress: true
  
  # Audit logging
  audit:
    enabled: true
    file: "/opt/vibes-mcp-cli/logs/audit.log"
    include_request_body: false
    include_response_body: false
```

#### 2. Log Aggregation

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /opt/vibes-mcp-cli/logs/*.log
  fields:
    application: vibes-mcp-cli
    environment: production
  fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "vibes-mcp-cli-%{+yyyy.MM.dd}"

setup.template.name: "vibes-mcp-cli"
setup.template.pattern: "vibes-mcp-cli-*"
```

## Backup and Recovery

### Backup Strategy

#### 1. Session Data Backup

```bash
#!/bin/bash
# scripts/backup-sessions.sh

BACKUP_DIR="/opt/vibes-mcp-cli/backups"
SESSION_DIR="/opt/vibes-mcp-cli/sessions"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="sessions_backup_${DATE}.tar.gz"

# Create backup
tar -czf "${BACKUP_DIR}/${BACKUP_FILE}" -C "${SESSION_DIR}" .

# Remove backups older than 30 days
find "${BACKUP_DIR}" -name "sessions_backup_*.tar.gz" -mtime +30 -delete

# Upload to S3 (optional)
aws s3 cp "${BACKUP_DIR}/${BACKUP_FILE}" "s3://your-backup-bucket/vibes-mcp-cli/"
```

#### 2. Configuration Backup

```bash
#!/bin/bash
# scripts/backup-config.sh

CONFIG_DIR="/opt/vibes-mcp-cli/config"
BACKUP_DIR="/opt/vibes-mcp-cli/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup configuration (excluding secrets)
tar -czf "${BACKUP_DIR}/config_backup_${DATE}.tar.gz" \
  --exclude="*.env" \
  --exclude="*secret*" \
  -C "${CONFIG_DIR}" .
```

#### 3. Automated Backup

```bash
# crontab -e (for vibes-mcp user)
# Backup sessions daily at 2 AM
0 2 * * * /opt/vibes-mcp-cli/scripts/backup-sessions.sh

# Backup configuration weekly on Sunday at 3 AM
0 3 * * 0 /opt/vibes-mcp-cli/scripts/backup-config.sh
```

### Recovery Procedures

#### 1. Session Recovery

```bash
#!/bin/bash
# scripts/restore-sessions.sh

if [ $# -ne 1 ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

BACKUP_FILE="$1"
SESSION_DIR="/opt/vibes-mcp-cli/sessions"

# Stop service
sudo systemctl stop vibes-mcp-cli

# Backup current sessions
mv "${SESSION_DIR}" "${SESSION_DIR}.backup.$(date +%Y%m%d_%H%M%S)"

# Restore from backup
mkdir -p "${SESSION_DIR}"
tar -xzf "${BACKUP_FILE}" -C "${SESSION_DIR}"
chown -R vibes-mcp:vibes-mcp "${SESSION_DIR}"

# Start service
sudo systemctl start vibes-mcp-cli
```

#### 2. Disaster Recovery

```bash
#!/bin/bash
# scripts/disaster-recovery.sh

echo "Starting disaster recovery process..."

# 1. Stop all services
sudo systemctl stop vibes-mcp-cli nginx

# 2. Restore from latest backup
LATEST_BACKUP=$(ls -t /opt/vibes-mcp-cli/backups/sessions_backup_*.tar.gz | head -1)
if [ -n "$LATEST_BACKUP" ]; then
    echo "Restoring from: $LATEST_BACKUP"
    ./restore-sessions.sh "$LATEST_BACKUP"
fi

# 3. Verify configuration
vibes-mcp-cli config validate --config /opt/vibes-mcp-cli/config/production.yaml

# 4. Start services
sudo systemctl start vibes-mcp-cli
sudo systemctl start nginx

echo "Disaster recovery completed"
```

## Performance Tuning

### System Optimization

#### 1. Kernel Parameters

```bash
# /etc/sysctl.d/99-vibes-mcp-cli.conf
# Increase file descriptor limits
fs.file-max = 2097152

# Network optimizations
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 1200

# Memory optimizations
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5

# Apply changes
sudo sysctl -p /etc/sysctl.d/99-vibes-mcp-cli.conf
```

#### 2. Process Limits

```bash
# /etc/security/limits.d/vibes-mcp-cli.conf
vibes-mcp soft nofile 65536
vibes-mcp hard nofile 65536
vibes-mcp soft nproc 8192
vibes-mcp hard nproc 8192
```

### Application Tuning

#### 1. Go Runtime Tuning

```bash
# Environment variables for Go runtime
export GOMAXPROCS=4              # Match CPU cores
export GOGC=100                  # GC target percentage
export GOMEMLIMIT=1GiB          # Memory limit
export GODEBUG=gctrace=1        # Enable GC tracing (development)
```

#### 2. Configuration Optimization

```yaml
# config/performance.yaml
session_manager:
  max_sessions: 50
  cleanup_interval: "30m"
  session_timeout: "4h"
  
# Connection pooling
http_client:
  max_idle_conns: 100
  max_idle_conns_per_host: 20
  idle_conn_timeout: "90s"
  
# File system optimization
file_system:
  cache_size: 1000
  cache_ttl: "5m"
  max_concurrent_operations: 10
```

## Maintenance and Updates

### Update Procedures

#### 1. Rolling Update

```bash
#!/bin/bash
# scripts/rolling-update.sh

NEW_VERSION="$1"
if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

echo "Starting rolling update to version $NEW_VERSION"

# 1. Download new version
wget "https://github.com/your-org/vibes-mcp-cli/releases/download/${NEW_VERSION}/vibes-mcp-cli-linux-amd64" \
  -O "/tmp/vibes-mcp-cli-${NEW_VERSION}"

# 2. Verify checksum
echo "Verifying checksum..."
# Add checksum verification here

# 3. Stop service
sudo systemctl stop vibes-mcp-cli

# 4. Backup current version
cp /opt/vibes-mcp-cli/bin/vibes-mcp-cli "/opt/vibes-mcp-cli/bin/vibes-mcp-cli.backup.$(date +%Y%m%d_%H%M%S)"

# 5. Install new version
sudo cp "/tmp/vibes-mcp-cli-${NEW_VERSION}" /opt/vibes-mcp-cli/bin/vibes-mcp-cli
sudo chmod +x /opt/vibes-mcp-cli/bin/vibes-mcp-cli
sudo chown vibes-mcp:vibes-mcp /opt/vibes-mcp-cli/bin/vibes-mcp-cli

# 6. Validate configuration
sudo -u vibes-mcp /opt/vibes-mcp-cli/bin/vibes-mcp-cli config validate \
  --config /opt/vibes-mcp-cli/config/production.yaml

# 7. Start service
sudo systemctl start vibes-mcp-cli

# 8. Health check
sleep 10
curl -f http://localhost:8080/health || {
    echo "Health check failed, rolling back..."
    sudo systemctl stop vibes-mcp-cli
    sudo cp /opt/vibes-mcp-cli/bin/vibes-mcp-cli.backup.* /opt/vibes-mcp-cli/bin/vibes-mcp-cli
    sudo systemctl start vibes-mcp-cli
    exit 1
}

echo "Update completed successfully"
```

#### 2. Kubernetes Rolling Update

```bash
# Update deployment image
kubectl set image deployment/vibes-mcp-cli \
  vibes-mcp-cli=your-registry/vibes-mcp-cli:v1.2.0 \
  -n vibes-mcp-cli

# Check rollout status
kubectl rollout status deployment/vibes-mcp-cli -n vibes-mcp-cli

# Rollback if necessary
kubectl rollout undo deployment/vibes-mcp-cli -n vibes-mcp-cli
```

### Maintenance Tasks

#### 1. Log Rotation

```bash
# /etc/logrotate.d/vibes-mcp-cli
/opt/vibes-mcp-cli/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        systemctl reload vibes-mcp-cli
    endscript
}
```

#### 2. Session Cleanup

```bash
#!/bin/bash
# scripts/cleanup-sessions.sh

SESSION_DIR="/opt/vibes-mcp-cli/sessions"
DAYS_OLD=30

echo "Cleaning up sessions older than $DAYS_OLD days..."

# Find and remove old session directories
find "$SESSION_DIR" -type d -name "session-*" -mtime +$DAYS_OLD -exec rm -rf {} +

# Cleanup empty directories
find "$SESSION_DIR" -type d -empty -delete

echo "Session cleanup completed"
```

#### 3. Health Monitoring

```bash
#!/bin/bash
# scripts/health-monitor.sh

# Check service status
if ! systemctl is-active --quiet vibes-mcp-cli; then
    echo "Service is not running, attempting restart..."
    systemctl start vibes-mcp-cli
    sleep 10
fi

# Check health endpoint
if ! curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "Health check failed"
    # Send alert
    echo "Health check failed at $(date)" | mail -s "Vibes MCP CLI Alert" admin@company.com
fi

# Check disk space
DISK_USAGE=$(df /opt/vibes-mcp-cli | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 80 ]; then
    echo "Disk usage is high: ${DISK_USAGE}%"
    # Trigger cleanup or send alert
fi
```

---

This comprehensive deployment guide covers all aspects of deploying the vibes-mcp-cli system in production environments. Regular maintenance, monitoring, and security updates are essential for maintaining a robust and secure deployment. For troubleshooting deployment issues, refer to the [Troubleshooting section](User-Guide.md#troubleshooting) in the User Guide.