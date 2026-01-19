# APIGate Documentation

> **Self-hosted API monetization platform** - Turn your API into a revenue stream in minutes.

---

## What is APIGate?

APIGate is a complete API gateway and monetization platform that helps developers and businesses:

- **Proxy & Protect** - Route requests to your backend with authentication and rate limiting
- **Monetize** - Create pricing plans, manage subscriptions, collect payments
- **Self-Serve** - Give customers a portal to sign up, get API keys, and track usage
- **Scale** - Handle rate limiting, quotas, and usage metering automatically

```
┌─────────────────────────────────────────────────────────────────┐
│                     Your Customers                               │
│         (Developers using your API)                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                       APIGate                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Customer    │  │   Proxy     │  │   Admin     │             │
│  │ Portal      │  │   Engine    │  │   Dashboard │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ┌────────────────────────────────────────────────────┐         │
│  │  Auth │ Rate Limit │ Quota │ Usage │ Billing      │         │
│  └────────────────────────────────────────────────────┘         │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Your API Backend                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Links

### Getting Started
- [[Installation]] - Deploy APIGate in 5 minutes
- [[Quick-Start]] - Your first API proxy
- [[First-Customer]] - Onboard your first paying customer

### Core Concepts
- [[Architecture]] - How APIGate works
- [[Upstreams]] - Configure backend services
- [[Routes]] - Define API endpoints
- [[API-Keys]] - Authentication system
- [[Plans]] - Pricing and quotas
- [[Users]] - Customer management

### Features
- [[Proxying]] - Request routing and forwarding
- [[Rate-Limiting]] - Protect your API from abuse
- [[Quotas]] - Monthly usage limits
- [[Usage-Metering]] - Track every request
- [[Transformations]] - Modify requests/responses
- [[Protocols]] - HTTP, SSE, WebSocket support

### Guides & Tutorials
- [[Tutorial-Basic-API]] - Proxy your first API
- [[Tutorial-Monetization]] - Set up paid plans
- [[Tutorial-Stripe]] - Integrate Stripe payments
- [[Tutorial-Custom-Portal]] - Customize the customer portal
- [[Tutorial-Production]] - Production deployment

### Reference
- [[API-Reference]] - REST API documentation
- [[CLI-Reference]] - Command-line interface
- [[Configuration]] - Environment variables
- [[Troubleshooting]] - Common issues

---

## Key Features

| Feature | Description |
|---------|-------------|
| **API Proxying** | Route requests to any HTTP backend with path rewriting |
| **Authentication** | API key authentication |
| **Rate Limiting** | Token bucket algorithm, per-key limits |
| **Quota Management** | Monthly request/byte quotas with grace periods |
| **Usage Tracking** | Detailed metrics for every API call |
| **Customer Portal** | Self-service signup, API keys, usage dashboard |
| **Admin Dashboard** | Manage users, plans, routes, and settings |
| **Auto Documentation** | Generate API docs from your routes |
| **Payment Integration** | Stripe, Paddle, LemonSqueezy support |
| **Multi-Protocol** | HTTP, Server-Sent Events, WebSocket |

---

## Architecture Overview

APIGate follows a clean architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                            │
│  Portal │ Admin UI │ REST API │ CLI │ Docs │ Proxy Handler      │
├─────────────────────────────────────────────────────────────────┤
│                    APPLICATION LAYER                             │
│  ProxyService │ RouteService │ AuthService │ UsageService       │
├─────────────────────────────────────────────────────────────────┤
│                      DOMAIN LAYER                                │
│  User │ Key │ Plan │ Route │ Upstream │ Usage │ Quota │ Settings│
├─────────────────────────────────────────────────────────────────┤
│                   INFRASTRUCTURE LAYER                           │
│  SQLite │ HTTP Client │ Stripe │ Paddle │ SMTP                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Who Uses APIGate?

**API Sellers (You)**
- Indie hackers monetizing side projects
- SaaS companies offering API access
- Agencies building API products for clients
- Enterprises with internal API marketplaces

**API Buyers (Your Customers)**
- Developers integrating your API
- Companies building on your platform
- Partners accessing your data

---

## Getting Help

- **GitHub Issues**: [Report bugs and request features](https://github.com/artpar/apigate/issues)
- **Documentation**: You're here!
- **Source Code**: [GitHub Repository](https://github.com/artpar/apigate)

---

## Documentation Index

### Setup & Configuration
1. [[Installation]]
2. [[Quick-Start]]
3. [[Configuration]]
4. [[Database-Setup]]

### Core Concepts
1. [[Architecture]]
2. [[Request-Lifecycle]]
3. [[Upstreams]]
4. [[Routes]]
5. [[API-Keys]]
6. [[Users]]
7. [[Plans]]

### Features
1. [[Proxying]]
2. [[Authentication]]
3. [[Rate-Limiting]]
4. [[Quotas]]
5. [[Usage-Metering]]
6. [[Transformations]]
7. [[Protocols]]
8. [[Webhooks]]

### Integrations
1. [[Payment-Stripe]]
2. [[Payment-Paddle]]
3. [[Payment-LemonSqueezy]]
4. [[Email-Configuration]]

### Tutorials
1. [[Tutorial-Basic-API]]
2. [[Tutorial-Monetization]]
3. [[Tutorial-Stripe]]
4. [[Tutorial-Custom-Portal]]
5. [[Tutorial-Production]]

### Reference
1. [[API-Reference]]
2. [[CLI-Reference]]
3. [[Module-System]]
4. [[Error-Codes]]
5. [[JSON-API-Format]]

### Troubleshooting
1. [[Troubleshooting]]
2. [[FAQ]]
