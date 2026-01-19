# Tutorial: Production Deployment

Deploy APIGate to production with security, reliability, and scalability.

---

## Overview

Production deployment checklist:
1. ✅ Secure configuration
2. ✅ TLS/HTTPS setup
3. ✅ Database backup
4. ✅ Monitoring and logging
5. ✅ High availability
6. ✅ Performance tuning

---

## Prerequisites

- Linux server (Ubuntu 22.04 recommended)
- Domain name pointing to your server
- SSL certificate (or use Let's Encrypt)

---

## Step 1: Prepare Server

### System Updates

```bash
sudo apt update && sudo apt upgrade -y
```

### Create Service User

```bash
sudo useradd -r -s /bin/false apigate
sudo mkdir -p /opt/apigate /var/lib/apigate /var/log/apigate
sudo chown -R apigate:apigate /opt/apigate /var/lib/apigate /var/log/apigate
```

### Install APIGate

```bash
# Download latest release
curl -L https://github.com/yourorg/apigate/releases/latest/download/apigate-linux-amd64 \
  -o /opt/apigate/apigate

chmod +x /opt/apigate/apigate
```

---

## Step 2: Secure Configuration

### Generate Secrets

```bash
# Generate encryption key
openssl rand -hex 32 > /opt/apigate/secret.key
chmod 600 /opt/apigate/secret.key

# Generate session secret
openssl rand -hex 32 > /opt/apigate/session.key
chmod 600 /opt/apigate/session.key
```

### Create Configuration File

Create `/opt/apigate/config.yaml`:

```yaml
server:
  host: 127.0.0.1  # Bind to localhost, use reverse proxy
  port: 8080
  base_url: https://api.yourdomain.com

database:
  path: /var/lib/apigate/apigate.db

security:
  secret_key: ${file:/opt/apigate/secret.key}
  session_secret: ${file:/opt/apigate/session.key}
  session_duration: 24h
  cors_origins:
    - https://app.yourdomain.com
    - https://admin.yourdomain.com

proxy:
  default_timeout_ms: 30000
  max_idle_conns: 200
  idle_conn_timeout_ms: 90000

logging:
  level: info
  format: json
  file: /var/log/apigate/apigate.log
```

Set permissions:
```bash
chmod 600 /opt/apigate/config.yaml
chown apigate:apigate /opt/apigate/config.yaml
```

---

## Step 3: Set Up TLS

### Option A: Reverse Proxy (Recommended)

Use nginx as reverse proxy with TLS termination.

Install nginx and certbot:
```bash
sudo apt install nginx certbot python3-certbot-nginx -y
```

Create `/etc/nginx/sites-available/apigate`:

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 60s;
    }
}
```

Enable site and get certificate:
```bash
sudo ln -s /etc/nginx/sites-available/apigate /etc/nginx/sites-enabled/
sudo certbot --nginx -d api.yourdomain.com
sudo systemctl reload nginx
```

### Option B: Direct TLS

APIGate can terminate TLS directly:

```yaml
# In config.yaml
server:
  host: 0.0.0.0
  port: 443
  tls:
    cert: /etc/ssl/certs/apigate.crt
    key: /etc/ssl/private/apigate.key
```

---

## Step 4: Create Systemd Service

Create `/etc/systemd/system/apigate.service`:

```ini
[Unit]
Description=APIGate API Gateway
After=network.target

[Service]
Type=simple
User=apigate
Group=apigate
WorkingDirectory=/opt/apigate
ExecStart=/opt/apigate/apigate serve --config /opt/apigate/config.yaml
Restart=always
RestartSec=5

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/var/lib/apigate /var/log/apigate

# Resource limits
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable apigate
sudo systemctl start apigate
```

Check status:
```bash
sudo systemctl status apigate
sudo journalctl -u apigate -f
```

---

## Step 5: Database Backup

### Automatic Backups

Create `/opt/apigate/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/apigate"
DATE=$(date +%Y%m%d_%H%M%S)
DB_PATH="/var/lib/apigate/apigate.db"

mkdir -p $BACKUP_DIR

# Create backup
sqlite3 $DB_PATH ".backup $BACKUP_DIR/apigate_$DATE.db"

# Compress
gzip $BACKUP_DIR/apigate_$DATE.db

# Keep only last 30 days
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete

# Optional: Upload to S3
# aws s3 cp $BACKUP_DIR/apigate_$DATE.db.gz s3://your-bucket/backups/
```

Add cron job:
```bash
chmod +x /opt/apigate/backup.sh
echo "0 2 * * * root /opt/apigate/backup.sh" | sudo tee /etc/cron.d/apigate-backup
```

### Restore from Backup

```bash
sudo systemctl stop apigate
gunzip /var/backups/apigate/apigate_20250119.db.gz
cp /var/backups/apigate/apigate_20250119.db /var/lib/apigate/apigate.db
chown apigate:apigate /var/lib/apigate/apigate.db
sudo systemctl start apigate
```

---

## Step 6: Monitoring

### Health Check Endpoint

APIGate exposes `/health`:

```bash
curl https://api.yourdomain.com/health
```

Response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "72h15m32s",
  "database": "ok"
}
```

### Prometheus Metrics

Enable metrics endpoint:

```yaml
# In config.yaml
metrics:
  enabled: true
  path: /metrics
```

Scrape with Prometheus:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'apigate'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

### Log Aggregation

Ship logs to your log aggregator:

**Filebeat configuration:**
```yaml
filebeat.inputs:
  - type: log
    paths:
      - /var/log/apigate/apigate.log
    json.keys_under_root: true
    json.add_error_key: true

output.elasticsearch:
  hosts: ["localhost:9200"]
```

### Alerting

Set up alerts for:
- High error rate (5xx > 1%)
- High latency (p99 > 5s)
- Database connection errors
- Disk space low
- Memory usage high

---

## Step 7: Performance Tuning

### System Limits

Edit `/etc/security/limits.conf`:

```
apigate soft nofile 65535
apigate hard nofile 65535
apigate soft nproc 4096
apigate hard nproc 4096
```

### Kernel Parameters

Edit `/etc/sysctl.conf`:

```
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1
```

Apply: `sudo sysctl -p`

### APIGate Tuning

```yaml
# In config.yaml
proxy:
  max_idle_conns: 500          # Higher for many upstreams
  idle_conn_timeout_ms: 60000  # Lower for high connection churn

rate_limit:
  enabled: true
  storage: sqlite              # Default, works for single node

usage:
  batch_size: 200              # Batch more records
  flush_interval_ms: 2000      # Write less frequently
```

---

## Step 8: High Availability (Optional)

For high availability, run multiple APIGate instances behind a load balancer.

### Load Balancer Setup

**nginx.conf:**
```nginx
upstream apigate {
    least_conn;
    server 10.0.1.10:8080;
    server 10.0.1.11:8080;
    server 10.0.1.12:8080;
}

server {
    location / {
        proxy_pass http://apigate;
    }
}
```

### Shared Database

For multiple instances, use PostgreSQL (when supported) or shared SQLite via NFS:

```yaml
# Each instance points to same database
database:
  path: /mnt/shared/apigate.db
```

### Session Sharing

For distributed sessions, configure Redis:

```yaml
session:
  store: redis
  redis_url: redis://redis-cluster:6379
```

---

## Step 9: Security Checklist

### Network Security

- [ ] Firewall allows only ports 80, 443
- [ ] Admin UI accessible only from internal network
- [ ] SSH key-only authentication
- [ ] Fail2ban configured

### Application Security

- [ ] HTTPS enforced
- [ ] CORS restricted to known origins
- [ ] Rate limiting enabled
- [ ] Strong session secrets
- [ ] Sensitive data encrypted at rest

### Updates

- [ ] Subscribe to security advisories
- [ ] Regular dependency updates
- [ ] Test updates in staging first

---

## Step 10: Go-Live Checklist

Before going live:

- [ ] DNS pointing to production server
- [ ] TLS certificate valid
- [ ] Admin account secured with strong password
- [ ] Backup tested and verified
- [ ] Monitoring alerts configured
- [ ] Load tested under expected traffic
- [ ] Runbook for common operations
- [ ] Incident response plan

### Smoke Test

```bash
# Health check
curl -f https://api.yourdomain.com/health

# API request
curl -H "X-API-Key: ak_test_xxx" https://api.yourdomain.com/api/test

# Admin UI accessible
curl -I https://api.yourdomain.com/ui

# Portal accessible
curl -I https://api.yourdomain.com/portal
```

---

## Production Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Production Architecture                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Internet                                                        │
│     │                                                            │
│     ▼                                                            │
│  ┌─────────────────────────────────────────────────────┐        │
│  │              Load Balancer / CDN                     │        │
│  │         (Cloudflare, AWS ALB, etc.)                 │        │
│  └─────────────────────────────────────────────────────┘        │
│     │                                                            │
│     ▼                                                            │
│  ┌─────────────────────────────────────────────────────┐        │
│  │                  nginx (TLS)                         │        │
│  └─────────────────────────────────────────────────────┘        │
│     │                                                            │
│     ▼                                                            │
│  ┌─────────────────────────────────────────────────────┐        │
│  │                   APIGate                            │        │
│  │  • Authentication    • Rate limiting                 │        │
│  │  • Routing          • Usage tracking                │        │
│  └─────────────────────────────────────────────────────┘        │
│     │                        │                                   │
│     ▼                        ▼                                   │
│  ┌──────────────┐     ┌──────────────┐                          │
│  │   SQLite DB  │     │   Upstreams  │                          │
│  │   (backed up)│     │   (your APIs)│                          │
│  └──────────────┘     └──────────────┘                          │
│                                                                  │
│  Monitoring: Prometheus + Grafana                                │
│  Logging: ELK / Loki                                             │
│  Backups: S3 / GCS                                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Maintenance

### Regular Tasks

| Task | Frequency |
|------|-----------|
| Check logs for errors | Daily |
| Review monitoring dashboards | Daily |
| Verify backups | Weekly |
| Apply security updates | Weekly |
| Rotate secrets | Monthly |
| Review access logs | Monthly |
| Load test | Quarterly |

### Updates

```bash
# Download new version
curl -L https://github.com/yourorg/apigate/releases/latest/download/apigate-linux-amd64 \
  -o /opt/apigate/apigate.new

# Test new version
/opt/apigate/apigate.new version

# Stop, update, start
sudo systemctl stop apigate
mv /opt/apigate/apigate /opt/apigate/apigate.old
mv /opt/apigate/apigate.new /opt/apigate/apigate
chmod +x /opt/apigate/apigate
sudo systemctl start apigate

# Verify
curl https://api.yourdomain.com/health
```

---

## Summary

You've deployed APIGate to production with:

1. ✅ Secure configuration with encrypted secrets
2. ✅ TLS/HTTPS via nginx reverse proxy
3. ✅ Systemd service for reliability
4. ✅ Automated database backups
5. ✅ Monitoring and logging
6. ✅ Performance tuning
7. ✅ Security hardening

Your API gateway is production-ready!
