# Production Deployment

Guide to deploying APIGate in production.

---

## Quick Reference

See [[Tutorial-Production]] for a step-by-step deployment guide.

---

## Checklist

### Infrastructure

- [ ] SQLite database file on persistent storage (backed up)
- [ ] HTTPS enabled (ACME or manual certs)
- [ ] Reverse proxy (nginx/caddy) for SSL termination (optional)
- [ ] CDN for static assets (optional)

> **Note**: APIGate uses SQLite for simplicity and portability. For high-traffic deployments, ensure SQLite is on fast storage (SSD) and consider running multiple read replicas behind a load balancer.

### Security

- [ ] Database file permissions restricted
- [ ] Secure admin access
- [ ] Firewall configured
- [ ] Secrets stored via settings with `--encrypted`
- [ ] Regular backups of SQLite database

### Monitoring

- [ ] Health checks configured (`/health` endpoint)
- [ ] Prometheus metrics enabled (`/metrics` endpoint)
- [ ] Log aggregation
- [ ] Alerting for errors

---

## Environment Variables

Key production settings:

```bash
# Server
APIGATE_SERVER_HOST=0.0.0.0
APIGATE_SERVER_PORT=8080

# Database (SQLite path)
APIGATE_DATABASE_DSN=/data/apigate.db

# Upstream
APIGATE_UPSTREAM_URL=https://api.backend.com
APIGATE_UPSTREAM_TIMEOUT=30s

# Logging
APIGATE_LOG_LEVEL=info
APIGATE_LOG_FORMAT=json
```

### TLS/HTTPS Settings

Configure via settings:

```bash
# ACME (Let's Encrypt)
apigate settings set tls.enabled true
apigate settings set tls.mode acme
apigate settings set tls.domain "api.example.com"
apigate settings set tls.acme_email "admin@example.com"
```

---

## See Also

- [[Tutorial-Production]] - Step-by-step guide
- [[Database-Setup]] - Database configuration
- [[Security]] - Security best practices
- [[Configuration]] - All settings
