# Production Deployment

Guide to deploying APIGate in production.

---

## Quick Reference

See [[Tutorial-Production]] for a step-by-step deployment guide.

---

## Checklist

### Infrastructure

- [ ] PostgreSQL for database (not SQLite)
- [ ] Redis for rate limiting and caching
- [ ] HTTPS enabled (ACME or manual certs)
- [ ] Load balancer for multiple instances
- [ ] CDN for static assets (optional)

### Security

- [ ] Strong database password
- [ ] Secure admin access
- [ ] Firewall configured
- [ ] Secrets in environment variables
- [ ] Regular backups

### Monitoring

- [ ] Health checks configured
- [ ] Prometheus metrics enabled
- [ ] Log aggregation
- [ ] Alerting for errors

---

## Environment Variables

Key production settings:

```bash
# Database
DATABASE_URL=postgres://user:pass@db:5432/apigate

# Redis
REDIS_URL=redis://redis:6379

# HTTPS
TLS_ACME_ENABLED=true
TLS_ACME_EMAIL=admin@example.com

# Security
ADMIN_SECRET=<strong-random-string>
```

---

## See Also

- [[Tutorial-Production]] - Step-by-step guide
- [[Database-Setup]] - Database configuration
- [[Security]] - Security best practices
- [[Configuration]] - All settings
