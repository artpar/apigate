# APIGate User Journeys

This directory contains comprehensive documentation of all user journeys through the APIGate platform. Each journey is documented from both **UX** and **business** perspectives to ensure we build features that delight users while driving revenue.

---

## Our Business Model

APIGate is a **B2B2C API monetization platform**:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        APIGate Business Model                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   API Sellers (Our Customers)          API Buyers (Their Customers)     │
│   ─────────────────────────            ───────────────────────────      │
│   • Buy APIGate license                • Pay API Sellers for access     │
│   • Configure their API backend        • Use API via APIGate proxy      │
│   • Set pricing & plans                • Self-serve through portal      │
│   • Collect revenue                    • Monitor their usage            │
│                                                                         │
│                    ┌─────────────────┐                                  │
│   API Seller ───── │    APIGate      │ ───── API Buyer                  │
│                    │  (Our Product)  │                                  │
│                    └─────────────────┘                                  │
│                            │                                            │
│                            ▼                                            │
│                    API Seller's Backend                                 │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Revenue Streams

| Stream | Description | Journey Impact |
|--------|-------------|----------------|
| **License Sales** | API Sellers purchase APIGate | J1 (Setup) must be frictionless |
| **Seller MRR** | API Sellers collect from their customers | J2, J8 (Plans, Upgrades) |
| **Platform Fees** | Potential % of transactions | J8 (Payment flow) |

---

## User Personas

### Primary: API Seller (Admin)

| Attribute | Description |
|-----------|-------------|
| **Role** | Indie hacker, SaaS founder, agency developer |
| **Goal** | Monetize their API without building billing infrastructure |
| **Pain Points** | Don't want to spend months building auth, billing, portals |
| **Success Metric** | Time to first paying customer |
| **Technical Level** | Developer, comfortable with APIs and deployment |

**Key Journeys:** [J1](admin/j1-first-time-setup.md), [J2](admin/j2-plan-management.md), [J3](admin/j3-monitoring.md), [J4](admin/j4-platform-config.md)

### Secondary: API Buyer (Customer)

| Attribute | Description |
|-----------|-------------|
| **Role** | Developer integrating the API Seller's product |
| **Goal** | Get API access quickly and start building |
| **Pain Points** | Slow onboarding, unclear documentation, surprise billing |
| **Success Metric** | Time to first successful API call |
| **Technical Level** | Developer, varies from junior to senior |

**Key Journeys:** [J5](customer/j5-onboarding.md), [J6](customer/j6-api-access.md), [J7](customer/j7-usage-monitoring.md), [J8](customer/j8-plan-upgrade.md), [J9](customer/j9-documentation.md)

---

## Journey Map Overview

```
                                    API SELLER JOURNEYS
    ┌─────────────────────────────────────────────────────────────────────────┐
    │                                                                         │
    │  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐              │
    │  │   J1    │───▶│   J2    │───▶│   J3    │◀──▶│   J4    │              │
    │  │ Setup   │    │ Plans   │    │ Monitor │    │ Config  │              │
    │  └─────────┘    └─────────┘    └─────────┘    └─────────┘              │
    │       │              │              ▲                                   │
    │       │              │              │                                   │
    │       ▼              ▼              │                                   │
    │  ┌─────────────────────────────────┴───────────────────┐               │
    │  │              ADMIN DASHBOARD (Hub)                   │               │
    │  └──────────────────────────────────────────────────────┘               │
    │                                                                         │
    └─────────────────────────────────────────────────────────────────────────┘

                                   API BUYER JOURNEYS
    ┌─────────────────────────────────────────────────────────────────────────┐
    │                                                                         │
    │  ┌─────────┐    ┌─────────┐    ┌─────────┐                             │
    │  │   J5    │───▶│   J6    │───▶│   J7    │                             │
    │  │Onboard  │    │API Keys │    │ Usage   │                             │
    │  └─────────┘    └─────────┘    └─────────┘                             │
    │       │              │              │                                   │
    │       │              ▼              ▼                                   │
    │       │         ┌─────────┐    ┌─────────┐                             │
    │       │         │   J9    │    │   J8    │                             │
    │       │         │  Docs   │    │ Upgrade │                             │
    │       │         └─────────┘    └─────────┘                             │
    │       ▼                                                                 │
    │  ┌──────────────────────────────────────────────────────┐               │
    │  │             CUSTOMER PORTAL (Hub)                    │               │
    │  └──────────────────────────────────────────────────────┘               │
    │                                                                         │
    └─────────────────────────────────────────────────────────────────────────┘
```

---

## Journey Index

### Admin Journeys

| Journey | Name | Business Impact | Doc |
|---------|------|-----------------|-----|
| **J1** | First-Time Setup | License activation, first impression | [View](admin/j1-first-time-setup.md) |
| **J2** | Plan Management | Revenue model configuration | [View](admin/j2-plan-management.md) |
| **J3** | Monitoring & Management | Operational visibility | [View](admin/j3-monitoring.md) |
| **J4** | Platform Configuration | Payment & email setup | [View](admin/j4-platform-config.md) |

### Customer Journeys

| Journey | Name | Business Impact | Doc |
|---------|------|-----------------|-----|
| **J5** | Customer Onboarding | User acquisition funnel | [View](customer/j5-onboarding.md) |
| **J6** | Get API Access | Activation milestone | [View](customer/j6-api-access.md) |
| **J7** | Usage Monitoring | Retention & upgrade triggers | [View](customer/j7-usage-monitoring.md) |
| **J8** | Plan Upgrade | Revenue expansion | [View](customer/j8-plan-upgrade.md) |
| **J9** | Documentation | Self-service enablement | [View](customer/j9-documentation.md) |

### Error Journeys

| Journey | Name | Doc |
|---------|------|-----|
| **E1** | Authentication Errors | [View](errors/authentication-errors.md) |
| **E2** | Rate Limiting | [View](errors/rate-limiting.md) |
| **E3** | Quota Exceeded | [View](errors/quota-exceeded.md) |

---

## Journey Document Structure

Each journey document follows this structure:

```markdown
# Journey Title

## Business Context
- Why this journey matters to revenue
- Key conversion/retention metrics
- Business success criteria

## User Context
- Who is the user at this point
- What triggered this journey
- User goals and expectations

## The Journey
- Step-by-step flow with screenshots
- Decision points and branches
- Success and failure paths

## UX Analysis
- Cognitive load assessment
- Friction points
- Accessibility considerations
- Mobile experience

## Emotional Map
- User's emotional state progression
- Anxiety triggers
- Delight opportunities

## Metrics & KPIs
- What to measure
- Target benchmarks
- Drop-off monitoring

## Edge Cases & Errors
- What can go wrong
- Recovery paths
- Support escalation points

## Screenshot Automation
- Capture points
- GIF sequences
- Naming conventions
```

---

## Key Metrics Dashboard

### Acquisition (J1, J5)
- Setup completion rate
- Time to complete setup
- Signup conversion rate
- Account activation rate

### Activation (J6, J9)
- Time to first API key
- Time to first API call
- Documentation engagement
- Try-it usage

### Retention (J3, J7)
- Daily/Weekly active users
- API call volume trends
- Usage pattern consistency

### Revenue (J2, J8)
- Plan distribution
- Upgrade rate
- Average revenue per user
- Churn rate

---

## Automation

See [automation/](automation/) for:
- [Screenshot Capture Guide](automation/screenshot-guide.md)
- [GIF Generation](automation/gif-generation.md)
- [CI Integration](automation/ci-integration.md)

---

## Contributing

When adding or modifying journeys:

1. **Start with business impact** - Why does this matter?
2. **Map the emotional journey** - How does the user feel?
3. **Document every step** - Screenshots and descriptions
4. **Identify friction** - Where might users drop off?
5. **Define success metrics** - How do we know it's working?
6. **Capture screenshots** - Run automation after changes
