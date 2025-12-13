# Audit Logging Capabilities

This document details the comprehensive audit logging system implemented in the SilentRelay application, designed for security monitoring, compliance, and forensic analysis.

## Overview

The audit logging system provides enterprise-grade security event tracking with compliance-ready features including GDPR, HIPAA, and SOC 2 support. All security-relevant actions are logged with structured data for analysis and reporting.

## Architecture

### Core Components

**AuditLogger**: Main logging component with asynchronous batch processing
**AuditEvent**: Structured event representation with compliance metadata
**Database Storage**: PostgreSQL-based persistent storage with indexing
**Query Interface**: REST API for audit log retrieval and analysis

### Key Features

- **Asynchronous Processing**: Non-blocking logging with configurable batch sizes
- **Structured Data**: JSON-based event data with typed fields
- **Compliance Ready**: Built-in support for regulatory requirements
- **Performance Optimized**: Batch writing and connection pooling
- **Retention Management**: Configurable data retention policies

## Event Types

### Authentication Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `login_attempt` | User initiates login | Info | Every login attempt |
| `login_success` | Successful authentication | Medium | Successful login |
| `login_failed` | Failed authentication | High | Invalid credentials |
| `logout` | User session termination | Info | User logout |
| `pin_verified` | PIN verification success | Medium | PIN entry |
| `pin_failed` | PIN verification failure | High | Wrong PIN |
| `pin_locked` | PIN lockout triggered | Critical | Brute force detection |
| `session_created` | New session established | Medium | Device login |
| `session_revoked` | Session forcibly ended | High | Security action |

### Key Management Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `key_generated` | New cryptographic key created | Medium | Key generation |
| `key_rotated` | Key rotation performed | Medium | Scheduled rotation |
| `key_revoked` | Key revocation | Critical | Security incident |
| `prekeys_uploaded` | One-time pre-keys uploaded | Info | Device setup |
| `prekeys_low` | Pre-key count running low | Medium | < 10 pre-keys remaining |
| `recovery_key_viewed` | Recovery key accessed | High | User action |
| `recovery_key_used` | Recovery key utilized | Critical | Account recovery |

### Device Management Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `device_added` | New device linked | Medium | Device approval |
| `device_removed` | Device unlinked | Medium | User action |
| `device_suspicious` | Suspicious device activity | High | Anomaly detection |
| `device_approved` | Device approval granted | Medium | Primary device action |
| `device_rejected` | Device approval denied | Medium | Primary device action |

### Security Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `brute_force_blocked` | Brute force attempt blocked | Critical | Rate limiting |
| `suspicious_ip` | Suspicious IP activity | High | IP reputation |
| `replay_attempt` | Message replay detected | High | Nonce validation |
| `invalid_request` | Malformed request | Medium | Input validation |
| `rate_limited` | Rate limit exceeded | Medium | Abuse detection |
| `honeypot_triggered` | Honeypot system activated | Critical | Attack detection |
| `intrusion_detected` | IDS alert triggered | Critical | Pattern matching |

### Account Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `profile_updated` | User profile modified | Low | Profile changes |
| `privacy_changed` | Privacy settings updated | Low | Settings changes |
| `account_blocked` | Account suspended | Critical | Security action |
| `account_deleted` | Account permanently removed | Critical | User action |
| `account_created` | New account registration | Medium | Registration |
| `account_recovery` | Account recovery initiated | High | Recovery process |

### Administrative Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `admin_action` | Administrative operation | High | Admin actions |
| `config_changed` | System configuration modified | Critical | Config updates |
| `permission_grant` | Permission granted | High | Access control |
| `permission_revoke` | Permission revoked | High | Access control |

### Data Access Events

| Event Type | Description | Severity | Triggers |
|------------|-------------|----------|----------|
| `data_export` | Data export requested | Medium | GDPR compliance |
| `data_access` | Data accessed | Low | General access |
| `data_modified` | Data modified | Medium | Update operations |
| `data_deleted` | Data deleted | High | Deletion operations |

## Event Structure

### Core Fields

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "session_id": "session-uuid",
  "device_id": "device-uuid",
  "event_type": "login_success",
  "severity": "medium",
  "result": "success",
  "timestamp": "2025-12-04T07:00:00Z"
}
```

### Extended Fields

```json
{
  "resource": "user_profile",
  "resource_id": "user-uuid",
  "resource_type": "user",
  "action": "update",
  "event_data": {
    "field": "display_name",
    "old_value": "John",
    "new_value": "John Doe"
  },
  "description": "User updated display name",
  "ip_address": "192.168.1.100",
  "user_agent": "SecureMessenger/1.0",
  "request_id": "req-550e8400-e29b-41d4-a716-446655440000",
  "request_path": "/api/v1/users/me",
  "request_method": "PUT",
  "country": "United States",
  "region": "California",
  "city": "San Francisco",
  "duration_ms": 150,
  "compliance_flags": ["GDPR", "data_processing"],
  "data_category": "PII",
  "retention_days": 2555
}
```

## Security Features

### Data Protection

**Encryption at Rest**: All audit logs encrypted using AES-256-GCM
**Access Control**: Role-based access to audit logs
**Integrity Protection**: HMAC signatures for log integrity
**Tamper Detection**: Cryptographic verification of log entries

### Performance Optimization

**Asynchronous Logging**: Non-blocking event processing
**Batch Processing**: Configurable batch sizes (default: 100 events)
**Connection Pooling**: Efficient database connection management
**Indexing Strategy**: Optimized database indexes for query performance

### Reliability Features

**Durable Storage**: ACID-compliant database transactions
**Queue Overflow Protection**: Synchronous fallback when queue full
**Graceful Shutdown**: Proper draining of pending events
**Error Recovery**: Automatic retry mechanisms

## Compliance Support

### GDPR Compliance

**Data Subject Rights**:
- Right to access audit logs
- Right to data portability
- Right to erasure (with legal holds)
- Data processing transparency

**Audit Trail Requirements**:
```go
// GDPR-compliant data access logging
auditLogger.LogDataAccess(userID, "user_profile", userID.String(), "PII", map[string]any{
    "access_type": "read",
    "purpose": "user_request",
    "data_fields": []string{"email", "phone"},
})
```

### HIPAA Compliance

**Protected Health Information (PHI)**:
- PHI access logging
- Emergency access tracking
- Data breach notification logs
- Audit trail integrity

### SOC 2 Compliance

**Trust Service Criteria**:
- Security: Access controls and encryption
- Availability: System reliability and monitoring
- Processing Integrity: Accurate and timely processing
- Confidentiality: Data protection and privacy
- Privacy: Personal information handling

## Query and Analysis

### REST API Endpoints

**Get User Audit Events**:
```http
GET /api/v1/audit/events?user_id={uuid}&event_type={type}&limit={n}
Authorization: Bearer {token}
```

**Get Security Events**:
```http
GET /api/v1/audit/security-events?user_id={uuid}&since={timestamp}
Authorization: Bearer {token}
```

**Admin Audit Query**:
```http
GET /api/v1/admin/audit/events?event_type={type}&severity={level}&from={date}&to={date}
Authorization: Bearer {admin_token}
```

### Query Examples

**Recent Failed Logins**:
```sql
SELECT * FROM security_audit_log
WHERE user_id = $1
  AND event_type = 'login_failed'
  AND timestamp > NOW() - INTERVAL '24 hours'
ORDER BY timestamp DESC;
```

**Suspicious Activity Detection**:
```sql
SELECT ip_address, COUNT(*) as attempts
FROM security_audit_log
WHERE event_type IN ('login_failed', 'pin_failed')
  AND timestamp > NOW() - INTERVAL '1 hour'
GROUP BY ip_address
HAVING COUNT(*) >= 5;
```

**Compliance Reporting**:
```sql
SELECT event_type, severity, COUNT(*) as count
FROM security_audit_log
WHERE compliance_flags @> ARRAY['GDPR']
  AND timestamp >= '2025-01-01'
  AND timestamp < '2026-01-01'
GROUP BY event_type, severity
ORDER BY count DESC;
```

## Monitoring and Alerting

### Real-time Alerts

**Prometheus Metrics**:
```yaml
# Audit logging metrics
audit_events_total{event_type, severity, result}
audit_queue_size
audit_batch_write_duration
audit_error_total
```

**Alert Rules**:
```yaml
# High severity events
- alert: HighSeverityAuditEvents
  expr: rate(audit_events_total{severity="critical"}[5m]) > 5
  for: 5m
  labels:
    severity: critical

# Audit queue backlog
- alert: AuditQueueBacklog
  expr: audit_queue_size > 1000
  for: 5m
  labels:
    severity: warning
```

### Dashboard Integration

**Grafana Dashboards**:
- Security Events Overview
- Authentication Success/Failure Rates
- Geographic Access Patterns
- Compliance Reporting
- Performance Metrics

## Data Retention

### Retention Policies

| Data Category | Retention Period | Legal Basis |
|---------------|------------------|-------------|
| Authentication Events | 3 years | Security requirements |
| Security Incidents | 7 years | Regulatory compliance |
| Data Access Logs | 2 years | GDPR Article 5 |
| Administrative Actions | 7 years | Audit trail requirements |
| Debug Logs | 30 days | Operational needs |

### Automated Cleanup

```sql
-- Automatic retention enforcement
CREATE OR REPLACE FUNCTION cleanup_audit_logs() RETURNS void AS $$
BEGIN
    -- Delete expired events
    DELETE FROM security_audit_log
    WHERE timestamp < NOW() - INTERVAL '3 years'
      AND severity NOT IN ('critical');

    -- Archive critical events
    INSERT INTO audit_archive
    SELECT * FROM security_audit_log
    WHERE timestamp < NOW() - INTERVAL '7 years'
      AND severity = 'critical';

    DELETE FROM security_audit_log
    WHERE timestamp < NOW() - INTERVAL '7 years';
END;
$$ LANGUAGE plpgsql;
```

## Configuration

### Logger Configuration

```go
// Default configuration
auditLogger := NewAuditLogger(db)

// Custom configuration
auditLogger := NewAuditLoggerWithConfig(db, 200, 10*time.Second)
```

### Database Schema

```sql
CREATE TABLE security_audit_log (
    id UUID PRIMARY KEY,
    user_id UUID,
    session_id UUID,
    device_id UUID,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    result VARCHAR(20) NOT NULL,
    resource VARCHAR(100),
    resource_id VARCHAR(100),
    resource_type VARCHAR(50),
    action VARCHAR(100),
    event_data JSONB,
    description TEXT,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100),
    request_path VARCHAR(500),
    request_method VARCHAR(10),
    country VARCHAR(100),
    region VARCHAR(100),
    city VARCHAR(100),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duration_ms BIGINT,
    compliance_flags TEXT[],
    data_category VARCHAR(50),
    retention_days INTEGER
);

-- Performance indexes
CREATE INDEX idx_audit_user_timestamp ON security_audit_log(user_id, timestamp DESC);
CREATE INDEX idx_audit_event_type ON security_audit_log(event_type, timestamp DESC);
CREATE INDEX idx_audit_severity ON security_audit_log(severity, timestamp DESC);
CREATE INDEX idx_audit_ip ON security_audit_log(ip_address, timestamp DESC);
CREATE INDEX idx_audit_compliance ON security_audit_log USING GIN(compliance_flags);
```

## Usage Examples

### Basic Event Logging

```go
// Log successful login
auditLogger.LogSecurityEvent(ctx, AuditEventLoginSuccess, AuditResultSuccess,
    &userID, "User logged in successfully", map[string]any{
        "device_type": "web",
        "login_method": "password",
    })

// Log failed authentication
auditLogger.LogSecurityEvent(ctx, AuditEventLoginFailed, AuditResultFailure,
    &userID, "Invalid password provided", map[string]any{
        "attempt_count": 3,
        "lockout_remaining": 300,
    })
```

### Administrative Actions

```go
// Log admin action
auditLogger.LogAdminAction(adminID, "user_suspend", "user", userID.String(),
    map[string]any{
        "reason": "suspicious_activity",
        "duration": "7_days",
    })
```

### Data Access Logging

```go
// GDPR-compliant data access
auditLogger.LogDataAccess(userID, "messages", conversationID, "personal_data",
    map[string]any{
        "access_type": "export",
        "data_volume": 150,
        "purpose": "user_backup",
    })
```

## Support and Contact

**Security Team**: `security@silentrelay.com.au`
**Compliance Officer**: `compliance@silentrelay.com.au`
**Audit Support**: `audit@silentrelay.com.au`

**Emergency Contacts**:
- Security Incident: `+1-555-SECURITY`
- Data Breach: `+1-555-BREACH`
- Compliance Issue: `+1-555-COMPLIANCE`

---

*© 2025 SilentRelay. All rights reserved.*
