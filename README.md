# APIGate

**Turn any API into a revenue stream. In 5 minutes.**

Stop building billing infrastructure. Start making money.

[![License](https://img.shields.io/badge/license-Commercial-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/docker-ready-2496ED.svg)](https://hub.docker.com/r/artpar/apigate)

---

## The Problem

You built an amazing API. Now you need to monetize it.

But that means building:
- User authentication and API key management
- Rate limiting that actually works
- Usage tracking and metering
- Subscription billing with Stripe
- A customer portal for self-service
- An admin dashboard for you

**That's 3-6 months of work.** Or you could use APIGate.

---

## The Solution

APIGate is a single binary that sits between your customers and your API:

```
Customers → APIGate → Your API
              │
              ├── Authenticates requests (API keys)
              ├── Enforces rate limits (per plan)
              ├── Tracks every request (usage metering)
              ├── Bills automatically (Stripe/Paddle/LemonSqueezy)
              └── Lets customers self-serve (portal)
```

**You focus on your API. APIGate handles the business.**

---

## See It In Action

### Setup Wizard (2 minutes)
![Setup Wizard](docs/screenshots/setup-wizard.png)
*Connect your API, create admin account, set pricing - done.*

### Admin Dashboard
![Admin Dashboard](docs/screenshots/dashboard.png)
*Real-time stats: users, API keys, requests, revenue.*

### Customer Portal
![Customer Portal](docs/screenshots/customer-portal.png)
*Your customers manage their own keys and usage.*

---

## Features

### For You (API Seller)

| Feature | What It Does |
|---------|--------------|
| **5-Minute Setup** | Guided wizard - no config files to edit |
| **Flexible Pricing** | Free tiers, paid plans, overage billing |
| **Multiple Payment Providers** | Stripe, Paddle, LemonSqueezy |
| **Real-Time Dashboard** | Users, keys, requests, MRR at a glance |
| **Usage Analytics** | See who's using what, and how much |
| **Email Notifications** | SMTP or SendGrid for transactional emails |

### For Your Customers (API Buyers)

| Feature | What It Does |
|---------|--------------|
| **Self-Service Portal** | Sign up, manage keys, view usage |
| **Instant API Keys** | Generate keys immediately after signup |
| **Clear Usage Tracking** | Know exactly how much they've used |
| **Plan Upgrades** | Switch plans without contacting support |
| **Password Reset** | Standard auth flows, no friction |

### Under The Hood

| Feature | What It Does |
|---------|--------------|
| **Reverse Proxy** | Transparent forwarding, header injection |
| **Rate Limiting** | Sliding window algorithm, per-key limits |
| **High Performance** | Written in Go, handles thousands of req/sec |
| **Single Binary** | No dependencies, runs anywhere |
| **SQLite or Postgres** | Start simple, scale when needed |

---

## Quick Start

### Option 1: Binary (Simplest)

```bash
# Download
curl -L https://github.com/artpar/apigate/releases/latest/download/apigate-$(uname -s)-$(uname -m) -o apigate
chmod +x apigate

# Run
./apigate

# Open http://localhost:8080 - setup wizard guides you through
```

### Option 2: Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v apigate-data:/data \
  ghcr.io/artpar/apigate:latest
```

### Option 3: Docker Compose

```yaml
version: '3.8'
services:
  apigate:
    image: ghcr.io/artpar/apigate:latest
    ports:
      - "8080:8080"
    volumes:
      - apigate-data:/data
    environment:
      - DATABASE_DSN=/data/apigate.db

volumes:
  apigate-data:
```

---

## How It Works

### 1. Connect Your API
Point APIGate at your backend. It becomes the front door.

### 2. Create Pricing Plans
Define tiers: Free (1K requests), Pro ($29/mo, 10K requests), etc.

### 3. Share Your Portal
Customers sign up at `your-domain.com/portal/`

### 4. Get Paid
Stripe/Paddle handles billing. You see MRR in your dashboard.

---

## Pricing Configuration Example

```yaml
plans:
  - id: free
    name: "Free"
    requests_per_month: 1000
    rate_limit_per_minute: 10
    price_monthly: 0

  - id: starter
    name: "Starter"
    requests_per_month: 50000
    rate_limit_per_minute: 60
    price_monthly: 29.00
    overage_price_per_1000: 0.50

  - id: pro
    name: "Pro"
    requests_per_month: 500000
    rate_limit_per_minute: 300
    price_monthly: 99.00
    overage_price_per_1000: 0.30
```

---

## Who Is This For?

**Indie Hackers** building AI wrappers, data APIs, or developer tools. You want to launch fast and focus on product, not billing infrastructure.

**SaaS Founders** adding API access to existing products. You need enterprise-grade rate limiting and usage tracking without building it yourself.

**Agencies** building APIs for clients. You need a turnkey monetization solution you can deploy and forget.

---

## Comparison

| Feature | APIGate | Build It Yourself | API Gateway (AWS/Kong) |
|---------|---------|-------------------|------------------------|
| Setup Time | 5 minutes | 3-6 months | Days to weeks |
| Billing Built-in | Yes | No | No |
| Customer Portal | Yes | Build it | No |
| Self-Hosted | Yes | Yes | Depends |
| Cost | One-time | Dev time | Usage-based |
| Complexity | Low | High | Medium-High |

---

## API Reference

### Making Requests (Your Customers)

```bash
curl -H "X-API-Key: ak_your_key_here" \
  https://your-apigate-domain.com/your-endpoint
```

**Response Headers:**
- `X-RateLimit-Remaining` - Requests left in window
- `X-RateLimit-Reset` - When limit resets

### Injected Headers (To Your Backend)

APIGate adds these headers to requests it forwards:
- `X-User-ID` - Authenticated user ID
- `X-Plan-ID` - User's current plan
- `X-Key-ID` - Which API key was used

---

## Production Deployment

### With PostgreSQL

```bash
DATABASE_DSN="postgres://user:pass@localhost:5432/apigate" ./apigate
```

### Behind Nginx

```nginx
server {
    listen 443 ssl;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_DSN` | Database connection string | `apigate.db` (SQLite) |
| `APIGATE_PORT` | HTTP port | `8080` |
| `STRIPE_SECRET_KEY` | Stripe API key | - |
| `SENDGRID_API_KEY` | SendGrid API key | - |

---

## Support & Resources

- **Documentation**: [docs.apigate.dev](https://docs.apigate.dev)
- **GitHub Issues**: [github.com/artpar/apigate/issues](https://github.com/artpar/apigate/issues)
- **Discord Community**: [discord.gg/apigate](https://discord.gg/apigate)

---

## License

**Commercial License** required for production use.

Free for:
- Local development
- Testing and evaluation
- Open source projects

[Contact for pricing](mailto:hello@apigate.dev) | [View License](LICENSE)

---

## Built With

- **Go** - Fast, reliable, single binary
- **SQLite/PostgreSQL** - Your choice of database
- **HTMX** - Lightweight, no JavaScript framework bloat

---

<p align="center">
  <b>Stop building billing. Start shipping.</b>
  <br><br>
  <a href="https://github.com/artpar/apigate/releases">Download APIGate</a> ·
  <a href="https://docs.apigate.dev">Documentation</a> ·
  <a href="https://discord.gg/apigate">Community</a>
</p>
