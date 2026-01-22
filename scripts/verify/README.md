# Documentation Verification System

Tools for verifying LLM-generated documentation against the actual codebase.

## Quick Start

```bash
# 1. Run automated checks (generates baseline report)
./scripts/verify/verify-all.sh

# 2. Start a verification session
./scripts/verify/session-tracker.sh start "Initial Verification"

# 3. Check a specific document
./scripts/verify/verify-doc.sh docs/spec/error-codes.md

# 4. Mark documents as verified or log issues
./scripts/verify/session-tracker.sh complete docs/spec/error-codes.md
./scripts/verify/session-tracker.sh issue docs/spec/pagination.md "Wrong default page size" high

# 5. Check progress
./scripts/verify/session-tracker.sh status

# 6. See what's next to verify
./scripts/verify/session-tracker.sh next
```

## Workflow for Multi-Session Verification

### Starting a Session

```bash
# Start with a focus area
./scripts/verify/session-tracker.sh start "API Spec Verification"
```

### Verification Process

For each document:

1. **Read the documentation file**
2. **Run automated checks**: `./scripts/verify/verify-doc.sh <file>`
3. **Manual verification** for items that couldn't be auto-verified
4. **Mark completion or log issues**

### Verification Checklist Per Document

When verifying a document, check:

| Check | How to Verify |
|-------|---------------|
| API endpoints exist | `grep -rn "endpoint" adapters/http/` |
| Error codes match | Compare with `pkg/jsonapi/errors.go` |
| Config options exist | `grep -rn "ENV_VAR" --include="*.go"` |
| Module fields match | Compare with `core/modules/*.yaml` |
| Code examples work | Try running them |
| Screenshots current | Compare with running app |

### Ending a Session

```bash
# Generate report
./scripts/verify/session-tracker.sh report

# Check what's left
./scripts/verify/session-tracker.sh next
```

## Files

| File | Purpose |
|------|---------|
| `verify-all.sh` | Run all automated verification checks |
| `verify-doc.sh` | Verify a single documentation file |
| `session-tracker.sh` | Track progress across sessions |
| `README.md` | This file |

## Reports Location

All reports are saved to `docs/verification-reports/`:

- `report_YYYYMMDD_HHMMSS.md` - Automated verification reports
- `verification-state.json` - Session state (verified files, issues)
- `summary_YYYYMMDD.md` - Summary reports

## Issue Severity Levels

| Level | When to Use |
|-------|-------------|
| `critical` | Documentation is completely wrong, would cause errors |
| `high` | Misleading information, could cause confusion |
| `medium` | Inaccurate but not harmful |
| `low` | Minor discrepancy, typo |

## Common Verification Patterns

### Verifying API Endpoints

```bash
# Extract endpoints from a doc
grep -oE '(GET|POST|PUT|PATCH|DELETE) /[a-zA-Z0-9/_-]+' docs/spec/wiki/API-Reference.md

# Check if they exist in code
grep -rn '"/admin/users"' adapters/http/
```

### Verifying Error Codes

```bash
# List all error codes in docs
grep -oE '`[a-z_]+`' docs/spec/error-codes.md | sort | uniq

# List all error codes in code
grep 'NewError' pkg/jsonapi/errors.go
```

### Verifying Module Fields

```bash
# Get fields from YAML
grep -A 50 'fields:' core/modules/user.yaml

# Check if documented correctly
grep -A 20 '## User' docs/spec/resource-types.md
```

### Verifying Config Options

```bash
# Find all env vars in code
grep -rh 'os.Getenv\|viper.Get' --include="*.go" . | grep -oE '"[A-Z_]+"'

# Check if documented
grep -E '^[A-Z_]+' docs/spec/wiki/Configuration.md
```

## Integration with DOCUMENTATION_VERIFICATION.md

The main checklist is in `docs/DOCUMENTATION_VERIFICATION.md`. Use it alongside these scripts:

1. Open `DOCUMENTATION_VERIFICATION.md` in an editor
2. Run verification scripts
3. Update checklist statuses `[ ]` → `[x]` or `[!]`
4. Add notes for any issues found

## Recommended Verification Order

1. **High Priority** - These define core behavior:
   - `docs/spec/error-codes.md`
   - `docs/spec/json-api.md`
   - `docs/spec/resource-types.md`

2. **Medium Priority** - Feature documentation:
   - `docs/spec/wiki/` files
   - `docs/user_journeys/`

3. **Lower Priority** - Supporting docs:
   - `docs/SYSTEM_ARCHITECTURE.md`
   - `docs/TECHNICAL_FEATURES.md`
   - `docs/USER_GUIDE.md`
