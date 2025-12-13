# Security Testing Procedures

This document outlines comprehensive security testing procedures for the SilentRelay application, including automated tests, manual testing, and penetration testing guidelines.

## Overview

Security testing is performed at multiple levels to ensure comprehensive coverage:

- **Automated Security Tests**: Run on every commit via CI/CD
- **Manual Security Reviews**: Performed before major releases
- **Penetration Testing**: Conducted quarterly by external security firms
- **Red Team Exercises**: Annual adversarial simulations

## Automated Security Testing

### CI/CD Security Pipeline

All security tests run automatically on every code change:

```yaml
# .github/workflows/security.yml
name: Security Tests
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Security Tests
        run: ./scripts/run-security-tests.sh
      - name: Security Audit
        run: ./scripts/security-check.sh
```

### Static Analysis Tests

**SAST (Static Application Security Testing)**:

```bash
# Run comprehensive static analysis
./scripts/run-security-tests.sh

# Individual tools
go vet ./...                    # Go static analysis
staticcheck ./...              # Advanced Go linting
golint ./...                   # Go style and security
npm audit --audit-level=high   # Frontend dependency audit
```

**Test Coverage**:
- [x] Null pointer dereferences
- [x] SQL injection vectors
- [x] Hardcoded secrets detection
- [x] Unsafe cryptographic usage
- [x] Race conditions
- [x] Resource leaks

### Secrets Detection

**Automated Secret Scanning**:

```bash
# Check for hardcoded secrets
! grep -rn 'password\s*=\s*["'\'']' --include='*.go' --include='*.ts' .

# AWS key detection
! grep -rn 'AKIA[0-9A-Z]{16}' .

# Private key detection
! grep -rn 'BEGIN.*PRIVATE KEY' . | grep -v '.md'

# JWT secret detection
! grep -rn 'jwt.*secret.*=.*["'\'']' --include='*.go' .
```

**GitLeaks Integration**:
```yaml
# .gitleaks.toml
[allowlist]
paths = [
  '''.*_test\.go''',
  '''.*\.md''',
  '''go\.sum''',
]
```

### Dependency Vulnerability Scanning

**Go Dependencies**:
```bash
# Vulnerability checking
govulncheck ./...

# Dependency verification
go mod verify

# License compliance
go-licenses check ./...
```

**JavaScript Dependencies**:
```bash
# Security audit
npm audit --audit-level=high

# Outdated packages
npm outdated

# License checking
license-checker --failOn MIT
```

### Cryptographic Security Validation

**Algorithm Validation**:
```bash
# No weak algorithms
! grep -rn 'crypto/md5\|crypto/sha1\|crypto/des\|crypto/rc4' --include='*.go' .

# Proper random usage
! grep -rn 'math/rand' --include='*.go' internal/security/ internal/auth/

# Secure key sizes
# Verified in code: AES-256, X25519, Ed25519, Argon2id
```

### SQL Injection Prevention

**Query Analysis**:
```bash
# No string concatenation in SQL
! grep -rn 'fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT' --include='*.go' .

# No raw SQL execution
! grep -rn 'db.Exec.*\+.*"' --include='*.go' .

# Parameterized queries verification
grep -rn 'db.QueryRow\|db.Exec' --include='*.go' | head -10
```

## Unit and Integration Security Tests

### Cryptographic Tests

**Signal Protocol Tests**:
```go
func TestSignalProtocol(t *testing.T) {
    sp := NewSignalProtocol()

    // Test key exchange
    aliceKey, _ := sp.GenerateKeyPair()
    bobKey, _ := sp.GenerateKeyPair()

    aliceShared, _ := sp.SharedSecret(aliceKey.PrivateKey, bobKey.PublicKey)
    bobShared, _ := sp.SharedSecret(bobKey.PrivateKey, aliceKey.PublicKey)

    assert.Equal(t, aliceShared, bobShared)
}

func TestAESGCMEncryption(t *testing.T) {
    sp := NewSignalProtocol()
    plaintext := []byte("Hello, World!")
    key := make([]byte, 32) // 256-bit key
    rand.Read(key)

    ciphertext, _ := sp.EncryptAESGCM(plaintext, key)
    decrypted, _ := sp.DecryptAESGCM(ciphertext, key)

    assert.Equal(t, plaintext, decrypted)
}
```

**JWT Security Tests**:
```go
func TestJWTSecurity(t *testing.T) {
    // Test token expiration
    expiredToken := createExpiredToken()
    _, err := validateToken(expiredToken)
    assert.Error(t, err)

    // Test invalid signature
    tamperedToken := tamperToken(validToken)
    _, err = validateToken(tamperedToken)
    assert.Error(t, err)
}
```

### Authentication Security Tests

**PIN Security Tests**:
```go
func TestPINSecurity(t *testing.T) {
    // Test Argon2id parameters
    pin := "1234"
    hash, _ := hashPIN(pin)

    // Verify hash format
    assert.Contains(t, hash, "$argon2id$")

    // Test verification
    valid := verifyPIN(pin, hash)
    assert.True(t, valid)

    // Test wrong PIN
    invalid := verifyPIN("wrong", hash)
    assert.False(t, invalid)
}

func TestPINLockout(t *testing.T) {
    // Test progressive lockout
    for i := 0; i < 5; i++ {
        err := verifyPIN("wrong", correctHash)
        if i >= 3 {
            assert.Error(t, err) // Should be locked
        }
    }
}
```

### Rate Limiting Tests

**Enhanced Rate Limiting Tests**:
```go
func TestEnhancedRateLimiting(t *testing.T) {
    limiter := NewEnhancedRateLimiter(config, redisClient)

    // Test normal mode
    for i := 0; i < 60; i++ {
        allowed := limiter.Allow("test-endpoint", "192.168.1.1")
        if i < 59 {
            assert.True(t, allowed)
        } else {
            assert.False(t, allowed) // Should be limited
        }
    }

    // Test abuse detection
    for i := 0; i < 150; i++ {
        limiter.RecordAttempt("192.168.1.1", "")
    }
    // Should trigger penalty mode
    assert.True(t, limiter.IsInPenaltyMode("192.168.1.1"))
}
```

## Manual Security Testing

### Code Review Checklist

**Security Code Review**:
- [ ] All inputs validated and sanitized
- [ ] Authentication required for sensitive operations
- [ ] Authorization checks implemented
- [ ] Cryptographic operations use approved algorithms
- [ ] Secrets never logged or exposed
- [ ] Error messages don't leak sensitive information
- [ ] SQL queries use parameterized statements
- [ ] File uploads validated for type and size
- [ ] HTTPS enforced for all connections
- [ ] Security headers properly configured

### Penetration Testing Procedures

**Authentication Testing**:
```bash
# Brute force protection
for i in {1..10}; do
    curl -X POST /api/v1/auth/pin/verify \
         -d '{"pin": "1234"}' \
         -H "X-Forwarded-For: $ATTACKER_IP"
done

# Expected: Rate limited after 5 attempts

# Session management
curl -X POST /api/v1/auth/login \
     -d '{"phone": "+1234567890"}' \
     -H "User-Agent: Old Browser"

# Expected: Session invalidated on suspicious activity
```

**Authorization Testing**:
```bash
# IDOR testing
curl -X GET /api/v1/messages?user_id=$OTHER_USER_ID \
     -H "Authorization: Bearer $USER_TOKEN"

# Expected: 403 Forbidden

# Privilege escalation
curl -X POST /api/v1/admin/users \
     -H "Authorization: Bearer $USER_TOKEN"

# Expected: 403 Forbidden
```

**Input Validation Testing**:
```bash
# SQL injection
curl -X GET "/api/v1/users/search?q=' OR '1'='1"

# Expected: Sanitized, no results leaked

# XSS testing
curl -X PUT /api/v1/users/me \
     -d '{"display_name": "<script>alert(1)</script>"}' \
     -H "Authorization: Bearer $TOKEN"

# Expected: Sanitized output
```

### WebSocket Security Testing

**Connection Security**:
```javascript
// Test unauthenticated access
const ws = new WebSocket('wss://api.example.com/ws');

// Expected: Connection rejected

// Test token validation
const ws = new WebSocket('wss://api.example.com/ws?token=invalid');

// Expected: Connection rejected
```

**Message Injection Testing**:
```javascript
// Test message type validation
ws.send(JSON.stringify({
    type: 'invalid_type',
    payload: { malicious: 'data' }
}));

// Expected: Message rejected
```

## Security Monitoring and Alerting

### Runtime Security Monitoring

**Application Metrics**:
```yaml
# Prometheus metrics
security_failed_auth_total
security_rate_limit_hits_total
security_suspicious_activity_total
crypto_operation_errors_total
```

**Alerting Rules**:
```yaml
# High error rate alert
- alert: HighSecurityErrorRate
  expr: rate(security_failed_auth_total[5m]) > 10
  for: 5m
  labels:
    severity: critical

# Rate limit abuse
- alert: RateLimitAbuse
  expr: rate(security_rate_limit_hits_total[1m]) > 100
  for: 2m
  labels:
    severity: warning
```

### Log Analysis

**Security Event Logging**:
```go
// Structured security logging
auditLogger.LogSecurityEvent(ctx, AuditEventFailedAuth, AuditResultFailure, &userID,
    "Authentication failed", map[string]any{
        "ip_address": ip,
        "user_agent": userAgent,
        "failure_reason": "invalid_pin",
    })
```

**Log Analysis Queries**:
```bash
# Failed authentication attempts
grep "failed_auth" /var/log/messaging/security.log | jq '.ip_address' | sort | uniq -c | sort -nr

# Rate limit hits by endpoint
grep "rate_limit" /var/log/messaging/security.log | jq '.endpoint' | sort | uniq -c | sort -nr

# Suspicious IP addresses
grep "suspicious" /var/log/messaging/security.log | jq '.ip_address' | sort | uniq -c | sort -nr | head -10
```

## Red Team Exercises

### Annual Red Team Testing

**Objectives**:
- Test defense-in-depth controls
- Identify unknown vulnerabilities
- Validate incident response procedures
- Assess security monitoring effectiveness

**Rules of Engagement**:
- Scope limited to production-like environment
- No disruption to actual production systems
- 48-hour exercise duration
- Full access to source code and documentation

**Success Criteria**:
- Time to detect initial compromise
- Time to contain breach
- Data exfiltration prevention
- Incident response effectiveness

## Continuous Security Improvement

### Security Metrics Tracking

**Key Performance Indicators**:
- Mean Time To Detect (MTTD) security incidents
- Mean Time To Respond (MTTR) to security alerts
- False positive rate of security monitoring
- Security test coverage percentage
- Vulnerability remediation time

### Regular Security Assessments

**Monthly Activities**:
- Automated security test execution
- Dependency vulnerability scanning
- Security log review
- Configuration drift detection

**Quarterly Activities**:
- Penetration testing
- Code security review
- Architecture security assessment
- Third-party vendor security review

**Annual Activities**:
- Red team exercise
- Comprehensive security audit
- Disaster recovery testing
- Security awareness training

## Security Testing Tools

### Recommended Tools

| Category | Tool | Purpose |
|----------|------|---------|
| SAST | SonarQube, Semgrep | Static code analysis |
| DAST | OWASP ZAP, Burp Suite | Dynamic application testing |
| SCA | Snyk, Dependabot | Software composition analysis |
| Container | Trivy, Clair | Container vulnerability scanning |
| Secrets | GitLeaks, TruffleHog | Secret detection |
| Crypto | CryptCheck, SSL Labs | Cryptographic assessment |

### Integration Examples

**GitHub Actions Security Workflow**:
```yaml
- name: Security Scan
  uses: github/super-linter@v4
  env:
    DEFAULT_BRANCH: main
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

- name: Dependency Check
  uses: dependency-check/Dependency-Check_Action@main

- name: Secrets Scan
  uses: zricethezav/gitleaks-action@main
```

## Security Testing Contacts

**Security Team**: `security@silentrelay.com.au`
**Penetration Testing**: `pentest@silentrelay.com.au`
**Bug Bounty**: `bounty@silentrelay.com.au`

**Emergency Contacts**:
- Security Incident: `+1-555-SECURITY`
- After Hours: `security-emergency@silentrelay.com.au`

---

*© 2025 SilentRelay. All rights reserved.*
