# MITRE ATT&CK Framework Mapping

## Overview

This document maps our security controls to the [MITRE ATT&CK](https://attack.mitre.org/) framework, ensuring comprehensive coverage against known adversary tactics, techniques, and procedures (TTPs).

> **📚 Related Documents:**
> - [Threat Model](THREAT_MODEL.md) - Comprehensive threat analysis
> - [Red Team Checklist](REDTEAM_CHECKLIST.md) - Defensive capabilities assessment
> - [Intrusion Detection System](INTRUSION_DETECTION_SYSTEM.md) - Attack detection implementation
> - [Security Documentation Index](SECURITY_DOCUMENTATION_INDEX.md) - Complete documentation navigation

## Tactics & Techniques Coverage

### Initial Access (TA0001)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Phishing | T1566 | Phone-based auth (no email links) | PASS |
| Drive-by Compromise | T1189 | CSP headers, no inline scripts | PASS |
| Exploit Public-Facing App | T1190 | WAF, input validation, rate limiting | PASS |
| Valid Accounts | T1078 | Device binding, PIN, session rotation | PASS |
| Trusted Relationship | T1199 | Zero trust - verify everything | PASS |

### Execution (TA0002)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Command Interpreter | T1059 | No shell execution, sandboxed | PASS |
| Exploitation for Execution | T1203 | Memory-safe Go, input validation | PASS |
| User Execution | T1204 | No executable content served | PASS |
| Native API | T1106 | Strict API contracts, validation | PASS |

### Persistence (TA0003)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Create Account | T1136 | Admin approval, audit logging | PASS |
| Account Manipulation | T1098 | Immutable audit trail, alerts | PASS |
| Web Shell | T1505.003 | No file uploads to webroot | PASS |
| Scheduled Task | T1053 | No user-controlled scheduling | PASS |
| Valid Accounts | T1078 | Session expiry, device binding | PASS |

### Privilege Escalation (TA0004)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Abuse Elevation Control | T1548 | Least privilege, RBAC | PASS |
| Access Token Manipulation | T1134 | Token binding, rotation | PASS |
| Exploitation for Privilege Escalation | T1068 | Regular patching, minimal surface | PASS |
| Valid Accounts | T1078 | Role separation, MFA | PASS |

### Defense Evasion (TA0005)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Indicator Removal | T1070 | Immutable append-only logs | PASS |
| Impair Defenses | T1562 | Integrity monitoring, alerts | PASS |
| Masquerading | T1036 | Strict content types, validation | PASS |
| Obfuscated Files | T1027 | No execution of uploads | PASS |
| Rootkit | T1014 | Container isolation, read-only FS | PASS |
| Timestomp | T1070.006 | Server-side timestamps only | PASS |

### Credential Access (TA0006)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Brute Force | T1110 | Rate limiting, progressive lockout | PASS |
| Credential Dumping | T1003 | No plaintext storage, HSM | PASS |
| MitM | T1557 | Certificate pinning, E2EE | PASS |
| Steal App Access Token | T1528 | Token rotation, device binding | PASS |
| Unsecured Credentials | T1552 | Vault integration, no env secrets | PASS |
| Network Sniffing | T1040 | TLS 1.3, E2EE | PASS |

### Discovery (TA0007)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Account Discovery | T1087 | Privacy-preserving contact discovery | PASS |
| Network Scanning | T1046 | Rate limiting, honeypots | PASS |
| System Info Discovery | T1082 | Minimal info in responses | PASS |
| Permission Groups Discovery | T1069 | No group enumeration | PASS |

### Lateral Movement (TA0008)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Internal Spearphishing | T1534 | E2EE prevents content inspection | PASS |
| Use of Valid Accounts | T1078 | Per-device sessions, no sharing | PASS |
| Remote Services | T1021 | Network segmentation | PASS |

### Collection (TA0009)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Automated Collection | T1119 | E2EE - server can't read content | PASS |
| Data from Local System | T1005 | Client-side encryption | PASS |
| Input Capture | T1056 | N/A (client responsibility) | WARNING |
| Screen Capture | T1113 | Screenshot protection (mobile) | WARNING |

### Exfiltration (TA0010)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Exfil Over C2 Channel | T1041 | E2EE - nothing to exfil | PASS |
| Exfil Over Web Service | T1567 | No sensitive data on server | PASS |
| Data Encrypted | T1022 | Already encrypted by design | PASS |

### Impact (TA0040)

| Technique | ID | Our Defense | Status |
|-----------|-----|-------------|--------|
| Data Destruction | T1485 | Backups, soft deletes | PASS |
| Defacement | T1491 | Integrity checks, immutable deploy | PASS |
| DoS | T1498/T1499 | Rate limiting, CDN, auto-scaling | PASS |
| Resource Hijacking | T1496 | Container limits, monitoring | PASS |

## Detection Coverage

### SIEM Rules Needed

```yaml
# Brute Force Detection
- name: "Multiple Failed Logins"
  condition: count(login_failed) > 5 in 5m by source_ip
  severity: high
  mitre: T1110

# Credential Access
- name: "Session Hijacking Attempt"
  condition: fingerprint_mismatch AND valid_token
  severity: critical
  mitre: T1528

# Discovery
- name: "Endpoint Scanning"
  condition: count(unique_endpoints) > 50 in 1m by source_ip
  severity: medium
  mitre: T1046

# Defense Evasion
- name: "WAF Bypass Attempt"
  condition: blocked_request AND retry_count > 10
  severity: high
  mitre: T1562
```

## Gap Analysis

### Covered
- 45+ techniques with active defenses
- All high-priority techniques addressed

### Partial Coverage
- T1056 Input Capture - Client-side protection needed
- T1113 Screen Capture - Mobile app feature

### Out of Scope
- Physical access attacks
- Supply chain compromise (addressed separately)
- OS-level attacks (client responsibility)

## Continuous Improvement

This mapping is reviewed:
- After every security incident
- Quarterly against MITRE updates
- During annual security audits

Last Updated: November 2025
MITRE ATT&CK Version: v14

