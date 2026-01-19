# Groups

**Groups** enable team-based API access, allowing organizations to share API keys and usage quotas across multiple team members.

---

## Overview

Groups are the core entity for multi-user API access:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Group Structure                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│                        ┌──────────────┐                          │
│                        │    Group     │                          │
│                        │              │                          │
│                        │ • name       │                          │
│                        │ • slug       │                          │
│                        │ • plan       │                          │
│                        └──────┬───────┘                          │
│                               │                                  │
│           ┌───────────────────┼───────────────────┐              │
│           │                   │                   │              │
│           ▼                   ▼                   ▼              │
│    ┌──────────┐       ┌──────────┐       ┌──────────┐           │
│    │ Members  │       │ API Keys │       │ Invites  │           │
│    │          │       │          │       │          │           │
│    │ • owner  │       │ • shared │       │ • email  │           │
│    │ • admin  │       │ • quotas │       │ • role   │           │
│    │ • member │       │          │       │ • token  │           │
│    └──────────┘       └──────────┘       └──────────┘           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Group Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Group display name (required) |
| `slug` | string | URL-friendly identifier (required, unique) |
| `description` | string | Group description |
| `owner_id` | string | User who owns the group (required) |
| `plan_id` | string | Group's subscription plan |
| `billing_email` | string | Email for billing notifications |
| `stripe_customer_id` | string | Stripe customer ID (internal) |
| `status` | enum | active, suspended |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

---

## Use Cases

### Team API Access

Allow multiple developers to share API access:

```
Company: Acme Corp
├── Group: "Acme Engineering"
│   ├── Member: alice@acme.com (owner)
│   ├── Member: bob@acme.com (admin)
│   └── Member: charlie@acme.com (member)
│   └── API Keys:
│       ├── "Production Backend"
│       └── "Staging Environment"
```

### Organization-Level Billing

Bill at the organization level, not per-user:

- Group has a plan with quota
- All members share the quota
- Single invoice for the group

### Partner Access

Grant API access to external partners:

- Create group for each partner
- Invite partner contacts
- Track usage per partner

---

## Creating Groups

### Admin UI

1. Go to **Groups** in sidebar
2. Click **New Group**
3. Fill in:
   - **Name**: Group display name
   - **Slug**: URL identifier (auto-generated from name)
   - **Description**: Optional description
   - **Plan**: Select a plan (or inherit from owner)
4. Click **Create**

### CLI

```bash
# Create a group
apigate groups create \
  --name "Acme Engineering" \
  --slug "acme-engineering" \
  --owner "user-id-here"

# With plan
apigate groups create \
  --name "Partner Portal" \
  --slug "partner-portal" \
  --owner "user-id-here" \
  --plan "enterprise"
```

### REST API

```bash
curl -X POST http://localhost:8080/admin/groups \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Engineering",
    "slug": "acme-engineering",
    "owner_id": "user-id-here",
    "plan_id": "plan-id-here",
    "billing_email": "billing@acme.com"
  }'
```

---

## Member Roles

### Owner

- Full control over group
- Can delete the group
- Can manage all members
- Can manage billing
- Cannot be removed

### Admin

- Can manage members
- Can create/revoke API keys
- Can invite new members
- Cannot delete group

### Member

- Can use group API keys
- Can view group usage
- Cannot modify group settings
- Cannot manage other members

---

## Managing Members

Members are managed via the `group-members` CLI command.

### Add Member

```bash
# CLI
apigate group-members create \
  --group "group-id-here" \
  --user "user-id-here" \
  --role member

# API
curl -X POST http://localhost:8080/api/groups/<group-id>/members \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-id-here",
    "role": "member"
  }'
```

### Update Member Role

```bash
# CLI
apigate group-members change_role <member-id> --role admin

# API
curl -X PATCH http://localhost:8080/api/groups/<group-id>/members/<member-id>/role \
  -H "Content-Type: application/json" \
  -d '{"role": "admin"}'
```

### Remove Member

```bash
# CLI
apigate group-members delete <member-id>

# API
curl -X DELETE http://localhost:8080/api/groups/<group-id>/members/<member-id>
```

### List Group Members

```bash
# List members of a group
apigate group-members list --group <group-id>

# List groups a user belongs to
apigate group-members list-user --user <user-id>
```

---

## Group Invites

Invite users to join a group via email. Invites are managed via the `group-invites` CLI command.

### Create Invite

```bash
# CLI
apigate group-invites create \
  --group "group-id-here" \
  --email "newmember@example.com" \
  --role member \
  --inviter "admin-user-id"

# API
curl -X POST http://localhost:8080/api/groups/<group-id>/invites \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newmember@example.com",
    "role": "member"
  }'
```

### Invite Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier |
| `group_id` | string | Target group |
| `email` | string | Invitee's email (required) |
| `role` | enum | admin, member |
| `invited_by` | string | User who sent invite |
| `token` | string | Secret invite token (internal) |
| `expires_at` | timestamp | When invite expires |
| `created_at` | timestamp | When invite was created |

### Invite Flow

1. Admin creates invite with email and role
2. Invite email sent with unique link
3. User clicks link to accept
4. If logged in: Immediately added to group
5. If not logged in: Redirected to login, then added

### List Invites

```bash
apigate group-invites list --group <group-id>
```

### Revoke Invite

```bash
# CLI
apigate group-invites revoke <invite-id>

# API
curl -X DELETE http://localhost:8080/api/groups/<group-id>/invites/<invite-id>
```

---

## Group API Keys

API keys can belong to groups instead of individual users:

### Create Group Key

```bash
# CLI
apigate keys create \
  --group "group-id-here" \
  --name "Shared Production Key"

# API
curl -X POST http://localhost:8080/admin/keys \
  -H "Content-Type: application/json" \
  -d '{
    "group_id": "group-id-here",
    "name": "Shared Production Key"
  }'
```

### Key Behavior

- Group keys use the **group's plan** for rate limits and quotas
- Usage is tracked against the **group**, not individual users
- All group members can view group key prefixes
- Only owners/admins can create/revoke group keys

---

## Group Plans and Billing

### Assign Plan to Group

```bash
apigate groups update <id> --plan "enterprise"
```

### Billing Flow

1. Group owner sets `billing_email`
2. Subscription created for group (not owner)
3. Invoices sent to billing email
4. Usage quota shared across all members

### View Group Usage

Group usage can be viewed in the Admin UI under **Groups** > **[Group Name]** > **Usage**.

---

## Admin UI Pages

### Groups List (`/groups`)

View all groups with:
- Name and slug
- Member count
- Plan name
- Status

### Group Detail (`/groups/:id`)

View group details:
- Members list with roles
- API keys
- Pending invites
- Usage statistics

### Group Edit (`/groups/:id/edit`)

Edit group settings:
- Name and description
- Plan assignment
- Billing email

---

## Best Practices

### 1. Use Meaningful Slugs

```bash
# Good - descriptive
apigate groups create --slug "acme-production"

# Bad - generic
apigate groups create --slug "group1"
```

### 2. Set Billing Email

Always set a billing email separate from the owner:

```bash
apigate groups update <id> --billing-email "billing@company.com"
```

### 3. Limit Admin Count

Keep admin count minimal for security:
- 1 owner
- 1-2 admins
- Everyone else as members

### 4. Use Groups for Partners

Create separate groups for each integration partner:

```bash
apigate groups create --name "Partner: Acme" --slug "partner-acme"
apigate groups create --name "Partner: Foo Inc" --slug "partner-foo"
```

---

## See Also

- [[API-Keys]] - Creating group keys
- [[Users]] - User management
- [[Plans]] - Group plans
- [[Usage-Tracking]] - Group usage metrics
