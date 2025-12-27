# APIGate

**Drop-in API monetization. One binary. Five minute setup.**

Turn any HTTP API into a paid product with authentication, rate limiting, usage metering, and Stripe billing.

```
Your Users → APIGate → Your API
               │
               ├─ Validates API keys
               ├─ Enforces rate limits
               ├─ Tracks usage
               └─ Bills via Stripe
```

---

## Quick Start

```bash
# 1. Configure
cp configs/apigate.example.yaml apigate.yaml
# Edit: set your upstream URL and plans

# 2. Run
docker compose up -d

# 3. Access
# Proxy:  http://localhost:8080  (your users hit this)
# Admin:  http://localhost:8081  (you manage users here)
# Portal: http://localhost:8082  (users self-service here)
```

---

## Features

| Feature | Description |
|---------|-------------|
| **Reverse Proxy** | Transparent forwarding with header injection |
| **API Key Auth** | Secure, hashed keys with prefix identification |
| **Rate Limiting** | Per-key limits with sliding window algorithm |
| **Usage Metering** | Async batched recording, per-endpoint multipliers |
| **Tiered Plans** | Define plans in YAML, quota enforcement |
| **Stripe Billing** | Subscriptions + usage-based overage |
| **Admin Dashboard** | Manage users, keys, view analytics |
| **Developer Portal** | Self-service signup, key management |

---

## Configuration

```yaml
# apigate.yaml

# Your backend API
upstream:
  url: "http://your-api:3000"
  timeout: 30s

# Define pricing plans
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
    price_monthly: 2900  # $29.00
    overage_price: 50    # $0.0050 per extra request
    stripe_price_id: "price_xxx"

  - id: pro
    name: "Pro"
    requests_per_month: 500000
    rate_limit_per_minute: 300
    price_monthly: 9900  # $99.00
    overage_price: 30    # $0.0030
    stripe_price_id: "price_yyy"

# Per-endpoint pricing
endpoints:
  - path: "/v1/generate"
    method: "POST"
    cost_multiplier: 10  # Counts as 10 requests

# Stripe integration
billing:
  enabled: true
  stripe_secret_key: "${STRIPE_SECRET_KEY}"
  stripe_webhook_secret: "${STRIPE_WEBHOOK_SECRET}"
```

See [configs/apigate.example.yaml](configs/apigate.example.yaml) for full options.

---

## API Reference

### Proxy (port 8080)

All requests are forwarded to upstream. Send API key in header:

```bash
curl -H "X-API-Key: ak_your_key_here" http://localhost:8080/your/endpoint
```

**Response Headers:**
- `X-RateLimit-Remaining`: Requests left in window
- `X-RateLimit-Reset`: When limit resets (RFC3339)

**Injected Headers (to upstream):**
- `X-User-ID`: Authenticated user ID
- `X-Plan-ID`: User's plan
- `X-Key-ID`: API key ID

### Admin API (port 8081)

Requires `X-Admin-Secret` header.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/stats` | Dashboard statistics |
| GET | `/api/users` | List all users |
| POST | `/api/users` | Create user (returns API key) |
| GET | `/api/users/:id` | Get user details |
| PUT | `/api/users/:id` | Update user |
| GET | `/api/users/:id/keys` | List user's API keys |
| POST | `/api/users/:id/keys` | Create new API key |
| DELETE | `/api/keys/:id` | Revoke API key |
| GET | `/api/plans` | List available plans |

### Portal API (port 8082)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/signup` | None | Create account |
| GET | `/api/plans` | None | List plans |
| GET | `/api/me` | Key | Get profile + usage |
| GET | `/api/keys` | Key | List my keys |
| POST | `/api/keys` | Key | Create new key |
| DELETE | `/api/keys/:id` | Key | Revoke my key |
| GET | `/api/usage` | Key | Usage statistics |
| GET | `/api/invoices` | Key | Billing history |
| POST | `/api/subscription` | Key | Change plan |

---

## Deployment

### Docker (Recommended)

```bash
docker run -d \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  -v $(pwd)/apigate.yaml:/app/apigate.yaml:ro \
  -v apigate-data:/app/data \
  -e APIGATE_ADMIN_SECRET=your-secret \
  -e STRIPE_SECRET_KEY=sk_live_xxx \
  ghcr.io/artpar/apigate:latest
```

### Binary

```bash
# Download for your platform
curl -L https://github.com/artpar/apigate/releases/latest/download/apigate-$(uname -s)-$(uname -m) -o apigate
chmod +x apigate

# Run
./apigate -config apigate.yaml
```

### Production (PostgreSQL)

```yaml
database:
  driver: postgres
  dsn: "postgres://user:pass@localhost:5432/apigate?sslmode=require"
```

---

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

**Key Design Principles:**

1. **Values as Boundaries**: Pure business logic in `domain/`, I/O at edges
2. **Dependency Injection**: All external systems accessed via interfaces
3. **100% Testable**: No mocks needed for domain tests
4. **Single Binary**: Everything embedded, no external dependencies

```
domain/          Pure functions, value types (no I/O)
ports/           Interface definitions
adapters/        I/O implementations (DB, HTTP, Stripe)
app/             Orchestration layer
cmd/apigate/     Composition root
```

---

## Development

See [docs/DEVELOPER_KT.md](docs/DEVELOPER_KT.md) for developer knowledge transfer.

```bash
# Run tests (no external deps needed)
go test ./domain/...     # Pure unit tests
go test ./app/...        # With in-memory adapters
go test ./...            # Everything

# Run locally
make dev

# Build
make build

# Build Docker image
make docker
```

---

## License

Commercial license required for production use. Contact for pricing.

For evaluation and development: MIT license.
