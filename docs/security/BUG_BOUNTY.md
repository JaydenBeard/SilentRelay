# Bug Bounty Program

## Overview

We take security seriously. If you discover a vulnerability, we want to hear about it and reward you fairly.

## Scope

### In Scope

| Target | Description |
|--------|-------------|
| API Endpoints | All `/api/v1/*` endpoints |
| WebSocket | Real-time messaging infrastructure |
| Authentication | Login, PIN, recovery flows |
| Encryption | Client-side E2EE implementation |
| Key Management | Key generation, rotation, distribution |
| Session Management | Token handling, device binding |
| Mobile Apps | iOS and Android applications |
| Web Application | Browser-based client |

### Out of Scope

| Target | Reason |
|--------|--------|
| Third-party services | Report directly to vendor |
| Social engineering | Testing on employees not permitted |
| Physical attacks | Not applicable |
| DDoS | Don't test this on production |
| Spam/phishing | Creates user harm |

## Vulnerability Classification

### Critical ($$$$)
- Remote code execution
- Authentication bypass
- E2EE key compromise
- Mass data exposure
- Full account takeover without credentials

### High ($$$)
- SQL injection
- Stored XSS
- CSRF on sensitive actions
- Privilege escalation
- Session hijacking
- Information disclosure of keys/secrets

### Medium ($$)
- Reflected XSS
- IDOR (limited scope)
- Rate limit bypass
- Missing security headers (if exploitable)
- Insecure direct object references

### Low ($)
- Missing security best practices
- Information disclosure (non-sensitive)
- Clickjacking (without demonstrated impact)
- CSRF on non-sensitive actions

## Reward Ranges

| Severity | Range |
|----------|-------|
| Critical | $5,000 - $20,000 |
| High | $1,000 - $5,000 |
| Medium | $250 - $1,000 |
| Low | $50 - $250 |

*Actual rewards depend on impact, quality of report, and exploitability.*

## Rules of Engagement

### Do

- Test on your own accounts
- Use test/staging environments when available
- Stop and report immediately if you access real user data
- Provide detailed reproduction steps
- Give us reasonable time to fix (90 days for critical, 180 for others)
- Encrypt sensitive communications with our PGP key

### Don't

- Access, modify, or delete data you don't own
- Execute DoS/DDoS attacks
- Social engineer employees
- Test on accounts you don't control
- Publicly disclose before we've fixed the issue
- Chain exploits to maximize reward (report each separately)

## Reporting

### Required Information

```markdown
## Vulnerability Report

**Title**: [Brief description]

**Severity**: [Critical/High/Medium/Low]

**Type**: [XSS/SQLi/AuthBypass/etc.]

**Affected Component**: [API/Web/Mobile/etc.]

**Steps to Reproduce**:
1. [Step 1]
2. [Step 2]
3. [Step 3]

**Expected vs Actual Behavior**:
- Expected: [What should happen]
- Actual: [What does happen]

**Impact**: [What can an attacker do?]

**Proof of Concept**: [Code/screenshots/video]

**Suggested Fix**: [Optional but appreciated]

**Environment**:
- Browser/Device:
- OS:
- App Version:
```

### Where to Report

**Email**: security@silentrelay.com.au

**PGP Key**: [Key ID: 0xABCDEF12]

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
[Your PGP public key here]
-----END PGP PUBLIC KEY BLOCK-----
```

## Response Timeline

| Stage | Timeline |
|-------|----------|
| Acknowledgment | 24 hours |
| Initial Assessment | 72 hours |
| Status Update | Weekly |
| Fix Deployed | Varies by severity |
| Reward Issued | Within 30 days of fix |

## Hall of Fame

We publicly thank researchers (with permission):

| Researcher | Findings | Date |
|------------|----------|------|
| *Be the first!* | | |

## Legal Safe Harbor

We will not pursue legal action against researchers who:
- Follow the rules of engagement
- Act in good faith
- Report vulnerabilities responsibly

This commitment is limited to security research and does not authorize violations of law.

## FAQ

**Q: Can I test on production?**
A: Only with your own accounts. Never access other users' data.

**Q: What if I accidentally access real data?**
A: Stop immediately, don't save/share it, and report it to us.

**Q: Can I report the same bug found in multiple places?**
A: Report each unique instance. We'll determine if they're duplicates.

**Q: How long do I have to wait before disclosure?**
A: 90 days for critical, 180 days otherwise. We may extend if actively working on a fix.

**Q: What if someone else reports the same bug?**
A: First reporter gets the reward. We may offer a smaller reward for duplicates that provide new information.

---

**Thank you for helping keep our users safe!**


