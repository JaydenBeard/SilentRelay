# SilentRelay Documentation

This directory contains all technical documentation for the SilentRelay secure messaging platform.

## Documentation Structure

### [api/](./api/)
API reference documentation for all endpoints.

| Document | Description |
|----------|-------------|
| [API_AUTHENTICATION.md](./api/API_AUTHENTICATION.md) | Authentication flows, JWT, OTP |
| [API_BEST_PRACTICES.md](./api/API_BEST_PRACTICES.md) | API usage guidelines |
| [API_CHANGELOG.md](./api/API_CHANGELOG.md) | API version history |
| [API_DEVICES.md](./api/API_DEVICES.md) | Device management endpoints |
| [API_DEVICE_APPROVAL.md](./api/API_DEVICE_APPROVAL.md) | Multi-device approval flows |
| [API_DOCUMENTATION_INDEX.md](./api/API_DOCUMENTATION_INDEX.md) | Complete API index |
| [API_GROUPS.md](./api/API_GROUPS.md) | Group messaging endpoints |
| [API_MEDIA.md](./api/API_MEDIA.md) | Media upload/download |
| [API_MESSAGES.md](./api/API_MESSAGES.md) | Messaging endpoints |
| [API_PIN.md](./api/API_PIN.md) | PIN protection |
| [API_PRIVACY.md](./api/API_PRIVACY.md) | Privacy settings |
| [API_SECURITY.md](./api/API_SECURITY.md) | Security-related endpoints |
| [API_USERS.md](./api/API_USERS.md) | User management |
| [API_WEBSOCKET.md](./api/API_WEBSOCKET.md) | WebSocket protocol |

### [security/](./security/)
Security architecture, implementations, and threat analysis.

| Document | Description |
|----------|-------------|
| [SECURITY.md](./security/SECURITY.md) | Security overview & architecture |
| [SECRETS_MANAGEMENT.md](./security/SECRETS_MANAGEMENT.md) | **SOPS encryption for .env files** |
| [THREAT_MODEL.md](./security/THREAT_MODEL.md) | Threat modeling documentation |
| [CRYPTO_IMPLEMENTATION.md](./security/CRYPTO_IMPLEMENTATION.md) | Cryptographic implementations |
| [KEY_ROTATION_IMPLEMENTATION.md](./security/KEY_ROTATION_IMPLEMENTATION.md) | Key rotation strategies |
| [SEALED_SENDER_IMPLEMENTATION.md](./security/SEALED_SENDER_IMPLEMENTATION.md) | Sealed sender protocol |
| [POST_QUANTUM_MIGRATION_PLAN.md](./security/POST_QUANTUM_MIGRATION_PLAN.md) | Post-quantum crypto roadmap |
| [INTRUSION_DETECTION_SYSTEM.md](./security/INTRUSION_DETECTION_SYSTEM.md) | IDS documentation |
| [HONEYPOT_SYSTEM_DOCUMENTATION.md](./security/HONEYPOT_SYSTEM_DOCUMENTATION.md) | Honeypot implementation |
| [SUPPLY_CHAIN_SECURITY.md](./security/SUPPLY_CHAIN_SECURITY.md) | Supply chain protection |
| [MITRE_ATTCK_MAPPING.md](./security/MITRE_ATTCK_MAPPING.md) | MITRE ATT&CK coverage |
| [REDTEAM_CHECKLIST.md](./security/REDTEAM_CHECKLIST.md) | Red team testing guide |
| [BUG_BOUNTY.md](./security/BUG_BOUNTY.md) | Bug bounty program |
| [SECURITY_TESTING_PROCEDURES.md](./security/SECURITY_TESTING_PROCEDURES.md) | Security testing guide |
| [USER_SECURITY_GUIDE.md](./security/USER_SECURITY_GUIDE.md) | End-user security guide |
| [COMPREHENSIVE_SECURITY_FIXES_DOCUMENTATION.md](./security/COMPREHENSIVE_SECURITY_FIXES_DOCUMENTATION.md) | Security fix history |

### [operations/](./operations/)
Operational guides for deployment, monitoring, and maintenance.

| Document | Description |
|----------|-------------|
| [ENVIRONMENT_SETUP.md](./operations/ENVIRONMENT_SETUP.md) | Environment configuration |
| [SYSTEM_ADMINISTRATION_GUIDE.md](./operations/SYSTEM_ADMINISTRATION_GUIDE.md) | Admin guide |
| [MONITORING_SETUP_GUIDE.md](./operations/MONITORING_SETUP_GUIDE.md) | Prometheus/Grafana setup |
| [PERFORMANCE_MONITORING_GUIDE.md](./operations/PERFORMANCE_MONITORING_GUIDE.md) | Performance monitoring |
| [BACKUP_STRATEGY_GUIDE.md](./operations/BACKUP_STRATEGY_GUIDE.md) | Backup procedures |
| [DATABASE_OPTIMIZATION_GUIDE.md](./operations/DATABASE_OPTIMIZATION_GUIDE.md) | PostgreSQL optimization |
| [MAINTENANCE_PROCEDURES.md](./operations/MAINTENANCE_PROCEDURES.md) | Maintenance runbooks |
| [INCIDENT_RESPONSE_PLAYBOOK.md](./operations/INCIDENT_RESPONSE_PLAYBOOK.md) | Incident response |
| [AUDIT_LOGGING.md](./operations/AUDIT_LOGGING.md) | Audit log configuration |
| [CERTIFICATE_TROUBLESHOOTING_GUIDE.md](./operations/CERTIFICATE_TROUBLESHOOTING_GUIDE.md) | TLS/SSL troubleshooting |
| [PROTOCOL_ADAPTER_DOCUMENTATION.md](./operations/PROTOCOL_ADAPTER_DOCUMENTATION.md) | Protocol adapter docs |

### [architecture/](./architecture/)
System architecture and design documentation.

| Document | Description |
|----------|-------------|
| [architecture_diagram.md](./architecture/architecture_diagram.md) | System architecture diagrams |
| [codebase_summary.md](./architecture/codebase_summary.md) | Codebase overview |

### [reports/](./reports/)
Verification and audit reports.

| Document | Description |
|----------|-------------|
| [FINAL_COMPREHENSIVE_VERIFICATION_REPORT.md](./reports/FINAL_COMPREHENSIVE_VERIFICATION_REPORT.md) | Final security audit report |

## Quick Links

- **Getting Started**: [../QUICKSTART.md](../QUICKSTART.md)
- **Deployment**: [../DEPLOY.md](../DEPLOY.md)
- **User Documentation**: [../user-docs/](../user-docs/)

## Technology Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.23, Gorilla (mux/websocket), PostgreSQL 16, Redis 7 |
| Frontend | React 18, TypeScript 5.6, Vite 7, Tailwind CSS, Zustand |
| Infrastructure | Docker, HAProxy 2.9, Consul, MinIO, Prometheus/Grafana |
| Cryptography | Signal Protocol via @matrix-org/olm, AES-256-GCM, Ed25519 |
