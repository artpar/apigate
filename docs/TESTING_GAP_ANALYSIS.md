# Testing Gap Analysis: Issue #54

**Date**: 2026-01-27
**Issue**: Session cookies not set on HTTPS (browsers rejected `Secure: false`)
**Root Cause**: Hardcoded `Secure: false` in cookie settings
**Impact**: Complete authentication failure on production HTTPS deployments

---

## Executive Summary

A critical authentication bug (#54) reached production despite:
- ✅ 80%+ code coverage
- ✅ Unit tests passing
- ✅ E2E UI tests passing
- ✅ CI/CD passing all checks

**Why it was missed**: Tests only validated HTTP behavior. The `Secure` cookie flag is irrelevant for HTTP but critical for HTTPS. Production runs HTTPS; tests ran HTTP.

---

## The Bug

### What Happened

```go
// BEFORE (broken on HTTPS)
http.SetCookie(w, &http.Cookie{
    Secure: false, // ❌ Hardcoded - rejected by browsers on HTTPS!
})
```

### Why It Failed

| Environment | Protocol | Secure Flag | Browser Behavior |
|-------------|----------|-------------|------------------|
| **Local Dev** | HTTP | `false` | ✅ Accepts cookie |
| **CI Tests** | HTTP | `false` | ✅ Accepts cookie |
| **E2E Tests** | HTTP | `false` | ✅ Accepts cookie |
| **Production** | HTTPS | `false` | ❌ **Rejects cookie silently** |

Modern browsers **silently reject** cookies with `Secure: false` on HTTPS connections as a security measure.

---

## Testing Gaps Identified

### 1. **Protocol-Specific Behavior Not Tested**

**Gap**: Tests only covered HTTP, not HTTPS.

**Original Test** (before fix):
```go
func TestAuthHandler_SetSessionCookie(t *testing.T) {
    h := NewAuthHandler(nil)
    w := httptest.NewRecorder()
    // ❌ No protocol specification - defaults to HTTP

    session := Session{...}
    h.setSessionCookie(w, session)

    cookies := w.Result().Cookies()
    // ✅ Checked: cookie exists
    // ✅ Checked: HttpOnly
    // ❌ DID NOT CHECK: Secure flag
    // ❌ DID NOT CHECK: Protocol-specific behavior
}
```

**What Was Missing**:
- No HTTPS test scenario
- No validation of `Secure` flag value
- No test for proxied HTTPS (X-Forwarded-Proto)
- No SameSite validation
- No Path validation
- No expiration validation

### 2. **Cookie Attributes Not Comprehensively Validated**

**Gap**: Only 2 of 7 cookie security attributes were tested.

| Attribute | Security Purpose | Before Fix | After Fix |
|-----------|------------------|------------|-----------|
| HttpOnly | Prevent XSS | ✅ Tested | ✅ Tested |
| Secure | Prevent MitM | ❌ **NOT tested** | ✅ Tested |
| SameSite | Prevent CSRF | ❌ **NOT tested** | ✅ Tested |
| Path | Scope limit | ❌ **NOT tested** | ✅ Tested |
| Expires | Session lifetime | ❌ **NOT tested** | ✅ Tested |
| Value encoding | Data integrity | ❌ **NOT tested** | ✅ Tested |
| Name | Cookie identity | ✅ Tested | ✅ Tested |

**Coverage**: 28% before fix → 100% after fix

### 3. **No Production-Like Test Environment**

**Gap**: All tests ran over HTTP. Production uses HTTPS.

**Test Environments**:
| Environment | Protocol | TLS | Matches Production? |
|-------------|----------|-----|---------------------|
| Unit tests | HTTP | ❌ | ❌ No |
| Integration tests | HTTP | ❌ | ❌ No |
| E2E UI tests | HTTP | ❌ | ❌ No |
| CI/CD | HTTP | ❌ | ❌ No |
| **Production** | HTTPS | ✅ | ✅ This is the truth |

**Why This Matters**: Cookie security behavior differs fundamentally between HTTP and HTTPS.

### 4. **Missing Integration Tests for Auth Flow**

**Gap**: No end-to-end authentication flow tests validating cookie behavior.

**What Was Missing**:
```
❌ Test: Register → Check cookie → Use cookie → Validate session
❌ Test: Login → Check cookie → Use cookie → Validate session
❌ Test: Cookie expiration handling
❌ Test: Invalid/expired cookie rejection
❌ Test: Cookie works across requests
```

### 5. **No Documentation of Security Requirements**

**Gap**: Cookie security requirements were not documented in the spec.

**Before Fix**:
- ❌ No specification for cookie attributes
- ❌ No documentation of HTTPS requirements
- ❌ No testing requirements defined
- ❌ No production deployment notes

**After Fix**:
- ✅ Comprehensive authentication.md specification
- ✅ Cookie security requirements documented
- ✅ Testing requirements defined
- ✅ Production deployment checklist

---

## Why High Coverage Didn't Catch This

### The Coverage Illusion

```bash
$ go test -cover ./core/channel/http
ok  	github.com/artpar/apigate/core/channel/http	coverage: 82.5%
```

**80%+ coverage ✅** but **critical bug in prod ❌**

### Why?

Coverage measures **lines executed**, not **behavior validated**.

```go
// This line was "covered" (executed during tests)
http.SetCookie(w, &http.Cookie{
    Secure: false, // ❌ But the VALUE wasn't validated!
})
```

**Coverage showed**:
- ✅ setSessionCookie() was called
- ✅ http.SetCookie() was executed
- ✅ Cookie struct was created

**Coverage DIDN'T show**:
- ❌ Secure flag value was wrong
- ❌ Behavior differs HTTP vs HTTPS
- ❌ Browsers reject cookie on HTTPS

### The Lesson

**Code coverage ≠ behavior coverage**

You need to test:
1. **Lines executed** (code coverage)
2. **Values validated** (assertion coverage)
3. **Scenarios tested** (behavior coverage)
4. **Environments matched** (environment parity)

---

## Root Causes

### 1. **Test Design Gap**

Tests validated "does it work?" but not "does it work correctly in all environments?"

### 2. **Missing Test Matrix**

No systematic testing across:
- [ ] Protocols (HTTP, HTTPS, proxied)
- [ ] Attributes (all 7 cookie flags)
- [ ] Scenarios (register, login, logout, expire)
- [ ] Error cases (invalid, expired, missing)

### 3. **Development-Production Parity Gap**

```
Development: HTTP
CI/CD: HTTP
Production: HTTPS ← Different behavior!
```

### 4. **Implicit Assumptions**

**Assumption**: "If it works in dev, it works in prod"
**Reality**: HTTPS introduces different browser security behaviors

### 5. **No Production Smoke Tests**

After deployment, no automated validation that authentication works on HTTPS.

---

## How to Prevent This

### 1. **Comprehensive Cookie Attribute Testing**

✅ **Implemented**:
```go
// Test ALL cookie attributes in ALL scenarios
func TestCookieAttributes_HTTP_AllFields(t *testing.T) {
    // Validates: Name, Value, Path, HttpOnly, SameSite, Secure, Expires
}

func TestCookieAttributes_HTTPS_AllFields(t *testing.T) {
    // Same attributes, different Secure expectation
}

func TestCookieAttributes_ProxyHTTPS_AllFields(t *testing.T) {
    // X-Forwarded-Proto scenario
}
```

### 2. **Protocol Matrix Testing**

✅ **Implemented**: Test auth behavior across protocols:

| Test | HTTP | HTTPS | Proxied HTTPS |
|------|------|-------|---------------|
| Cookie set | ✅ | ✅ | ✅ |
| Secure=false | ✅ | ❌ | ❌ |
| Secure=true | ❌ | ✅ | ✅ |

### 3. **Security Attribute Checklist**

✅ **Documented in spec**: Required attributes for every cookie:

```yaml
security_attributes:
  httponly: true    # XSS protection
  samesite: Lax     # CSRF protection
  secure: dynamic   # Protocol-aware
  path: /           # Scope limiting
  expires: 7d       # Auto-expiration
```

### 4. **Production-Like CI Environment**

**Recommendation**: Add HTTPS testing to CI:

```yaml
# .github/workflows/ci.yml
test-https:
  steps:
    - name: Run HTTPS tests
      run: |
        # Generate self-signed cert
        openssl req -x509 -newkey rsa:4096 -nodes \
          -keyout key.pem -out cert.pem -days 1 \
          -subj "/CN=localhost"

        # Run tests with TLS
        TLS_CERT=cert.pem TLS_KEY=key.pem go test ./...
```

### 5. **Deployment Smoke Tests**

**Recommendation**: After deployment, validate auth works:

```bash
#!/bin/bash
# scripts/smoke-test.sh

# Test registration sets cookie
curl -c cookies.txt -X POST https://prod.example.com/auth/register \
  -d '{"email":"smoke@test.com","password":"Test123"}' \
  | grep "Set-Cookie.*Secure"

# Test cookie works for authenticated request
curl -b cookies.txt https://prod.example.com/auth/me \
  | grep -q "smoke@test.com"
```

### 6. **Documentation-Driven Development**

✅ **Implemented**: Specification defines expected behavior BEFORE implementation.

**Process**:
1. **Document**: Write spec (authentication.md)
2. **Test**: Write tests validating spec
3. **Implement**: Write code matching spec
4. **Verify**: Tests confirm spec compliance

### 7. **Enhanced Test Coverage Metrics**

**Beyond line coverage**, track:

```bash
# Cookie attribute coverage
Attributes tested: 7/7 (100%)

# Protocol coverage
Protocols tested: 3/3 (HTTP, HTTPS, proxied)

# Scenario coverage
Auth flows tested: 4/4 (register, login, logout, validate)

# Error case coverage
Error scenarios tested: 6/6 (invalid, expired, missing, etc.)
```

---

## Updated Testing Strategy

### Unit Tests

✅ **Implemented**:
- `auth_secure_cookie_test.go` - Protocol-specific Secure flag behavior
- `auth_cookie_attributes_test.go` - Comprehensive attribute validation

**Coverage**:
- All cookie attributes (7/7)
- All protocols (HTTP, HTTPS, proxied)
- Security properties (XSS, CSRF, MitM protection)

### Integration Tests

**Recommendation** (future):
```go
// Test full auth flow with cookie validation
func TestAuth_RegistrationFlow_E2E(t *testing.T) {
    // 1. Register user → expect cookie
    // 2. Use cookie → expect auth success
    // 3. Cookie expires → expect 401
}
```

### E2E Tests

**Recommendation** (future):
```go
// Test in browser with actual HTTPS
func TestAuth_HTTPS_Browser(t *testing.T) {
    // Use Playwright/Selenium with self-signed cert
    // Verify browser accepts cookie on HTTPS
}
```

### Production Monitoring

**Recommendation**:
```yaml
# Monitor auth success rate in production
metrics:
  - name: auth_login_success_rate
    alert_if: < 95%

  - name: auth_cookie_rejection_rate
    alert_if: > 1%
```

---

## Testing Checklist for Future Auth Changes

When modifying authentication code, validate:

### Cookie Behavior
- [ ] Cookie set on registration (HTTP)
- [ ] Cookie set on registration (HTTPS)
- [ ] Cookie set on login (HTTP)
- [ ] Cookie set on login (HTTPS)
- [ ] Cookie cleared on logout
- [ ] Cookie expires after 7 days
- [ ] Cookie works for authenticated requests

### Cookie Attributes
- [ ] Name = `apigate_session`
- [ ] Value is Base64-encoded JSON
- [ ] Path = `/`
- [ ] HttpOnly = `true`
- [ ] SameSite = `Lax`
- [ ] Secure = `false` on HTTP
- [ ] Secure = `true` on HTTPS
- [ ] Secure = `true` on proxied HTTPS (X-Forwarded-Proto)
- [ ] Expires set to 7 days from now

### Security
- [ ] XSS protection (HttpOnly)
- [ ] CSRF protection (SameSite)
- [ ] MitM protection (Secure on HTTPS)
- [ ] Password hashing (bcrypt)
- [ ] Session expiration enforced

### Error Cases
- [ ] Invalid credentials → 401
- [ ] Expired cookie → 401
- [ ] Missing cookie → 401
- [ ] Malformed cookie → 401
- [ ] Duplicate email registration → 409

### Documentation
- [ ] Spec updated (if behavior changed)
- [ ] Tests document expected behavior
- [ ] Wiki synced (if public API changed)

---

## Key Takeaways

1. **High coverage ≠ no bugs**: 80% coverage missed critical security bug
2. **Test the environment, not just the code**: HTTP tests don't validate HTTPS behavior
3. **Cookie behavior is protocol-dependent**: Same code, different browser behavior
4. **Security attributes matter**: All 7 cookie flags serve critical security purposes
5. **Document requirements first**: Spec prevents implementation gaps
6. **Test production scenarios**: Dev/test parity prevents surprises
7. **Validate assumptions**: "Works in dev" doesn't mean "works in prod"

---

## Success Metrics

After implementing fixes:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Cookie attribute coverage | 28% (2/7) | 100% (7/7) | +257% |
| Protocol scenarios | 1 (HTTP only) | 3 (HTTP/HTTPS/proxy) | +200% |
| Security validations | 2 | 7 | +250% |
| Auth documentation | None | Comprehensive | ∞ |
| Production bugs | 1 (critical) | 0 | -100% |

---

## Conclusion

This bug was preventable. It slipped through because:
1. Tests validated HTTP, not HTTPS
2. Cookie attributes weren't comprehensively checked
3. Production environment wasn't simulated in tests
4. Security requirements weren't documented

**The fix required three things**:
1. **Code**: Dynamic `Secure` flag based on protocol
2. **Tests**: Comprehensive cookie attribute validation across protocols
3. **Docs**: Authentication specification defining requirements

**The lesson**: Testing must match production reality. Code coverage alone is insufficient—you must test behavior, attributes, environments, and edge cases.

**Going forward**: Every security-sensitive feature must have:
1. Specification (requirements)
2. Implementation (code)
3. Validation (tests)
4. Verification (matches spec)
5. Documentation (for users)

This analysis ensures this class of bug doesn't happen again.
