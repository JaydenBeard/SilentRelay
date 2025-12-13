# Threat Model

> **📚 Related Documents:**
> - [Main Security Architecture](SECURITY.md#architecture-threat-model) - Security layers overview
> - [MITRE ATT&CK Mapping](MITRE_ATTCK_MAPPING.md) - Framework-specific threat coverage
> - [Red Team Checklist](REDTEAM_CHECKLIST.md) - Defensive capabilities assessment
> - [Security Certification Plan](SECURITY_CERTIFICATION_PLAN.md) - Compliance threat coverage

## [TARGET] Assets (What We're Protecting)

### Primary Assets
| Asset | Sensitivity | Impact if Compromised |
|-------|------------|----------------------|
| Message Content | **CRITICAL** | Privacy breach, legal liability |
| Identity Keys | **CRITICAL** | Impersonation, message decryption |
| Phone Numbers | HIGH | Spam, harassment, deanonymization |
| Social Graph | HIGH | Relationship analysis, targeting |
| Session Tokens | HIGH | Account takeover |
| Recovery Keys | **CRITICAL** | Full account access, backup decryption |

### Secondary Assets
| Asset | Sensitivity | Impact if Compromised |
|-------|------------|----------------------|
| Message Metadata | MEDIUM | Traffic analysis, pattern detection |
| Device Information | LOW | Fingerprinting |
| Timestamps | LOW | Activity patterns |

## [USER] Threat Actors

### 1. Script Kiddies
**Motivation**: Fun, reputation, minor profit
**Capabilities**: Automated tools, known exploits
**Targets**: Low-hanging fruit, unpatched systems

**Mitigations**:
- Keep dependencies updated
- Rate limiting
- Standard security headers

### 2. Cybercriminals
**Motivation**: Financial gain
**Capabilities**: Custom tools, social engineering
**Targets**: Credentials, payment info

**Mitigations**:
- No payment info stored
- E2EE protects message content
- Session security

### 3. Insiders (Malicious Employees)
**Motivation**: Profit, revenge
**Capabilities**: Direct system access
**Targets**: User data, system credentials

**Mitigations**:
- Zero-knowledge architecture
- Audit logging
- Principle of least privilege
- Background checks

### 4. Competitors
**Motivation**: Competitive advantage
**Capabilities**: Well-funded, patient
**Targets**: User base, trade secrets

**Mitigations**:
- E2EE prevents content snooping
- Rate limiting prevents scraping
- Abuse detection

### 5. Nation-State Actors
**Motivation**: Surveillance, intelligence
**Capabilities**: Unlimited resources, zero-days
**Targets**: High-value individuals

**Mitigations**:
- E2EE limits mass surveillance
- Warrant canary (optional)
- Minimal data retention
- Geographic distribution

### 6. Law Enforcement (with legal process)
**Motivation**: Crime investigation
**Capabilities**: Subpoenas, court orders
**Targets**: Specific users

**Response**:
- Comply with valid legal process
- Provide only data we have (metadata, not content)
- Transparency reports (annual)

## [UNLOCK] Attack Vectors

### Network Attacks

| Attack | Vector | Mitigation |
|--------|--------|------------|
| MITM | Intercept traffic | TLS 1.3 + Certificate Pinning |
| Eavesdropping | Passive sniffing | E2EE + TLS |
| Replay | Reuse old messages | Nonces + timestamps |
| DoS/DDoS | Overwhelm servers | Rate limiting + CDN |

### Application Attacks

| Attack | Vector | Mitigation |
|--------|--------|------------|
| SQL Injection | Malicious input | Parameterized queries |
| XSS | Script injection | CSP + output encoding |
| CSRF | Forged requests | CSRF tokens + SameSite cookies |
| Deserialization | Malicious objects | Input validation + typed parsing |

### Authentication Attacks

| Attack | Vector | Mitigation |
|--------|--------|------------|
| Credential Stuffing | Leaked passwords | No passwords (phone OTP) |
| Brute Force | Guess PIN | Rate limiting + lockout |
| Session Hijacking | Steal token | Device binding + rotation |
| SIM Swapping | Telco social engineering | PIN as second factor |

### Client Attacks

| Attack | Vector | Mitigation |
|--------|--------|------------|
| Malware | Compromised device | Out of scope (OS responsibility) |
| Key Extraction | Memory forensics | Secure memory handling |
| Screenshots | Capture app | Screenshot protection (mobile) |

### Key Attacks

| Attack | Vector | Mitigation |
|--------|--------|------------|
| Key Server Lies | Serve fake keys | Key Transparency Log |
| Key Compromise | Steal identity key | Alert on key change |
| Prekey Exhaustion | Use all prekeys | Monitoring + replenishment |

## [TREE] Attack Trees

### Attack Tree: Impersonate User

```
Impersonate User
 Steal Identity Key
    Compromise Device
       Physical Access
       Remote Exploit
    Exploit Key Export

 MITM Key Exchange
    Compromise CA
    Bypass Pinning
    Rogue Server

 Account Takeover
     SIM Swap
        Social Engineering Telco
     Steal Recovery Key
        Physical Access
        Cloud Backup Compromise
     Session Hijacking
         Steal Token
         Bypass Device Binding
```

### Attack Tree: Read Messages

```
Read Messages
 Compromise E2EE
    Break Cryptography
       Quantum Computer (future)
       Find Vulnerability
    Steal Keys
       [See Impersonate User]
    Protocol Downgrade
        Client Vulnerability

 Compromise Server
    [E2EE means no plaintext]

 Compromise Client
     Malware on Device
     Physical Access
     Social Engineering User
```

## [METRICS] Risk Assessment Matrix

| Risk | Likelihood | Impact | Overall | Status |
|------|------------|--------|---------|--------|
| SQL Injection | Low | High | Medium | PASS Mitigated |
| XSS | Low | Medium | Low | PASS Mitigated |
| MITM | Low | Critical | Medium | PASS Mitigated |
| Brute Force | High | Medium | Medium | PASS Mitigated |
| Key Server Lies | Low | Critical | Medium | PASS Mitigated |
| SIM Swapping | Medium | High | Medium | WARNING Partial |
| Device Compromise | Medium | Critical | High | FAIL Out of Scope |
| Insider Threat | Low | Critical | Medium | PASS Mitigated |

## [FUTURE] Future Threats

### Post-Quantum Cryptography
Current Signal Protocol uses:
- X25519 (ECDH) - vulnerable to quantum
- AES-256 (symmetric) - quantum-resistant
- SHA-256 - quantum-resistant

**Timeline**: 10-20 years (estimated)
**Mitigation Plan**: Monitor NIST PQC standardization, plan migration

### AI-Powered Attacks
- Deepfake voice for social engineering
- Automated vulnerability discovery
- Intelligent brute forcing

**Mitigation**: Continuous monitoring, AI-powered defense

## [CHECKLIST] Security Requirements

Based on this threat model:

1. **MUST** implement E2EE for all message content
2. **MUST** use TLS 1.3 with certificate pinning
3. **MUST** implement rate limiting on all endpoints
4. **MUST** log all security events
5. **MUST** hash/encrypt all sensitive data at rest
6. **SHOULD** implement Key Transparency
7. **SHOULD** support safety number verification
8. **SHOULD** implement sealed sender for metadata protection
9. **MAY** implement additional factors for high-risk actions
10. **MAY** implement warrant canary

## PASS Review Schedule

This threat model should be reviewed:
- After any security incident
- When adding major features
- At least annually
- When threat landscape changes significantly

Last Updated: November 2025
Next Review: November 2026

