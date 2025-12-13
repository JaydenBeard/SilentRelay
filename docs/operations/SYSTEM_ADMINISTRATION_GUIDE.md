# System Administration Guide

**Version**: 2025.12.04
**Last Updated**: 2025-12-04
**Status**: Active
**Owner**: Operations Team

## Table of Contents

1. [Introduction](#introduction)
2. [System Architecture Overview](#system-architecture-overview)
3. [Prerequisites and Requirements](#prerequisites-and-requirements)
4. [System Setup and Configuration](#system-setup-and-configuration)
5. [Service Management](#service-management)
6. [User Management](#user-management)
7. [Configuration Management](#configuration-management)
8. [Security Administration](#security-administration)
9. [Troubleshooting and Diagnostics](#troubleshooting-and-diagnostics)
10. [Best Practices](#best-practices)
11. [Cross-References](#cross-references)

## Introduction

This guide provides comprehensive procedures for administering the SilentRelay system. It covers all aspects of system setup, configuration, management, and maintenance.

### Scope

This document applies to:
- System administrators
- Operations team members
- DevOps engineers
- Technical support staff

### Target Audience

- **Primary**: System administrators responsible for day-to-day operations
- **Secondary**: DevOps engineers managing deployment and scaling
- **Tertiary**: Security team members needing system context

## System Architecture Overview

### High-Level Architecture

```mermaid
graph TD
    A[Clients] -->|HTTPS/WebSocket| B[Load Balancer]
    B --> C[Chat Server]
    B --> D[Group Service]
    B --> E[Presence Service]
    B --> F[Notification Service]
    C --> G[Redis]
    C --> H[PostgreSQL]
    C --> I[MinIO]
    J[Scheduler] --> H
    K[Monitoring] --> C
    K --> D
    K --> E
    K --> F
    K --> G
    K --> H
```

### Key Components

| Component | Purpose | Technology |
|-----------|---------|------------|
| Chat Server | Core messaging service | Go, WebSocket |
| Group Service | Group chat management | Go, gRPC |
| Presence Service | User presence tracking | Go, Redis |
| Notification Service | Push notifications | Go, Firebase |
| Scheduler | Maintenance tasks | Go, Cron |
| Redis | Real-time data store | Redis 6.x |
| PostgreSQL | Persistent storage | PostgreSQL 14+ |
| MinIO | Media storage | S3-compatible |
| Prometheus | Monitoring | Prometheus 2.x |
| Grafana | Visualization | Grafana 9.x |
| Loki | Logging | Loki 2.x |

### Service Dependencies

```mermaid
graph LR
    ChatServer --> Redis
    ChatServer --> PostgreSQL
    ChatServer --> MinIO
    GroupService --> Redis
    GroupService --> PostgreSQL
    PresenceService --> Redis
    NotificationService --> PostgreSQL
    Scheduler --> PostgreSQL
    Scheduler --> Redis
```

## Prerequisites and Requirements

### Hardware Requirements

| Component | Minimum | Recommended | Production |
|-----------|---------|-------------|------------|
| CPU | 2 cores | 4 cores | 8+ cores |
| RAM | 4GB | 8GB | 16+ GB |
| Storage | 50GB | 100GB | 500+ GB |
| Network | 1Gbps | 10Gbps | 10Gbps+ |

### Software Requirements

| Component | Version | Notes |
|-----------|---------|-------|
| Docker | 20.10+ | Container runtime |
| Docker Compose | 1.29+ | Orchestration |
| Go | 1.20+ | Backend services |
| Node.js | 18+ | Frontend build |
| PostgreSQL | 14+ | Database |
| Redis | 6.2+ | Cache/presence |
| Prometheus | 2.40+ | Monitoring |
| Grafana | 9.0+ | Visualization |
| Loki | 2.8+ | Logging |

### Network Requirements

- **Ports**: 80 (HTTP), 443 (HTTPS), 8080-8085 (services), 9090 (Prometheus), 3000 (Grafana)
- **Firewall**: Allow outbound for updates, inbound only for required services
- **DNS**: Proper DNS resolution for all service names

## System Setup and Configuration

### Initial Setup Procedure

1. **Environment Preparation**
   ```bash
   # Clone repository
   git clone https://github.com/secure-messenger/messaging-app.git
   cd messaging-app

   # The .env file is already included with development defaults
   # Review and customize if needed (values prefixed with 'dev-' are placeholders)
   cat .env
   ```

2. **Dependency Installation**
   ```bash
   # Install Go dependencies
   go mod download

   # Install frontend dependencies
   cd web && npm install && cd ..

   # Set up Docker network
   docker network create messenger-network
   ```

3. **Configuration Files**
   - [`docker-compose.yml`](../docker-compose.yml) - Main orchestration
   - [`internal/config/config.go`](../internal/config/config.go) - Application configuration
   - [`infrastructure/haproxy/haproxy.cfg`](../infrastructure/haproxy/haproxy.cfg) - Load balancer
   - [`infrastructure/prometheus/prometheus.yml`](../infrastructure/prometheus/prometheus.yml) - Monitoring

### Configuration Parameters

#### Core Configuration (`config.go`)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `ServerPort` | 8080 | Main server port |
| `WebSocketPort` | 8081 | WebSocket port |
| `MaxMessageSize` | 10MB | Maximum message size |
| `RateLimit` | 100/rps | Request rate limit |
| `JWTSecret` | (random) | JWT signing secret |
| `SessionTimeout` | 24h | User session timeout |

#### Database Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `PostgresURL` | `postgres://messaging:messaging@localhost:5432/messaging` | PostgreSQL connection |
| `RedisURL` | `localhost:6379` | Redis connection |
| `MaxConnections` | 50 | Database pool size |
| `ConnectionTimeout` | 30s | Connection timeout |

### Service Configuration Files

#### Chat Server Configuration

```yaml
# Example chat server configuration
server:
  port: 8080
  websocket_port: 8081
  max_connections: 10000
  read_timeout: 30s
  write_timeout: 30s

database:
  postgres_url: "postgres://messaging:messaging@postgres:5432/messaging"
  redis_url: "redis:6379"
  max_connections: 50

security:
  jwt_secret: "your-very-secure-secret-here"
  session_timeout: "24h"
  rate_limit: 100
```

## Service Management

### Service Startup Procedure

1. **Start all services**
   ```bash
   docker-compose up -d
   ```

2. **Start individual services**
   ```bash
   # Chat server
   go run cmd/chatserver/main.go

   # Group service
   go run cmd/groupservice/main.go

   # Presence service
   go run cmd/presence/main.go
   ```

### Service Health Checks

```bash
# Check service status
docker-compose ps

# View logs for specific service
docker-compose logs chat-server

# Check service health endpoint
curl -s http://localhost:8080/health | jq
```

### Service Scaling

```bash
# Scale chat servers
docker-compose up -d --scale chat-server=3

# Scale presence service
docker-compose up -d --scale presence-service=2
```

### Service Restart Procedures

1. **Graceful restart**
   ```bash
   docker-compose restart chat-server
   ```

2. **Rolling restart**
   ```bash
   # For zero-downtime updates
   docker-compose up -d --build --no-deps chat-server
   ```

## User Management

### User Provisioning

```sql
-- Create new user (via admin interface)
INSERT INTO users (user_id, phone_number, username, created_at)
VALUES (gen_random_uuid(), '+1234567890', 'newuser', NOW());

-- Set user as admin
UPDATE users SET is_admin = true WHERE user_id = 'user-uuid-here';
```

### User Deprovisioning

```sql
-- Disable user account
UPDATE users SET is_active = false, disabled_at = NOW()
WHERE user_id = 'user-uuid-here';

-- Full user deletion (compliance with data retention policies)
DELETE FROM users WHERE user_id = 'user-uuid-here';
```

### Access Control Management

```bash
# Add user to admin group (example)
curl -X POST http://localhost:8080/api/v1/admin/users/grant-admin \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-uuid-here"}'
```

## Configuration Management

### Configuration Update Procedure

1. **Modify configuration file**
   ```bash
   # Edit configuration
   vim internal/config/config.go

   # Or use environment variables
   export SERVER_PORT=8082
   ```

2. **Validate configuration**
   ```bash
   # Test configuration syntax
   go run cmd/chatserver/main.go --validate-config

   # Check environment variables
   env | grep MESSAGING
   ```

3. **Apply configuration**
   ```bash
   # Restart service to apply changes
   docker-compose restart chat-server
   ```

### Configuration Versioning

```bash
# Commit configuration changes
git add internal/config/config.go
git commit -m "Update rate limits to 150 req/s"
git tag config/v1.2.3
git push origin config/v1.2.3
```

## Security Administration

### Certificate Management

```bash
# Generate new SSL certificates
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout infrastructure/certs/server.key \
  -out infrastructure/certs/server.crt \
  -subj "/CN=messenger.example.com"

# Update HAProxy configuration
vim infrastructure/haproxy/haproxy-ssl.cfg

# Restart HAProxy
docker-compose restart loadbalancer
```

### Security Policy Updates

```bash
# Update CORS policy
vim internal/middleware/auth.go

# Update rate limiting rules
vim internal/middleware/ratelimit.go

# Apply security updates
docker-compose build --no-cache chat-server
docker-compose up -d chat-server
```

## Troubleshooting and Diagnostics

### Common Issues and Solutions

| Issue | Symptom | Solution |
|-------|---------|----------|
| Service not starting | Container exits immediately | Check logs: `docker-compose logs chat-server` |
| Database connection failed | "connection refused" errors | Verify PostgreSQL is running: `docker-compose ps postgres` |
| WebSocket failures | Clients can't connect | Check HAProxy config: `docker-compose logs loadbalancer` |
| High latency | Slow message delivery | Check Redis health: `redis-cli ping` |
| Authentication failures | Login errors | Verify JWT secret: `grep JWTSecret .env` |

### Diagnostic Commands

```bash
# Check system resource usage
docker stats

# Check network connectivity
docker network inspect messenger-network

# Check database health
psql -h localhost -U messaging -c "SELECT version();"

# Check Redis health
redis-cli -h redis info
```

### Log Analysis

```bash
# View recent logs
docker-compose logs --tail=100 chat-server

# Follow logs in real-time
docker-compose logs -f chat-server

# Search logs for errors
docker-compose logs chat-server | grep ERROR
```

## Best Practices

### Operational Best Practices

1. **Configuration Management**
   - Use environment variables for sensitive data
   - Version all configuration changes
   - Test configurations in staging before production

2. **Service Management**
   - Implement health checks for all services
   - Use rolling deployments for zero-downtime updates
   - Monitor service metrics continuously

3. **Security Administration**
   - Rotate certificates quarterly
   - Review access controls monthly
   - Audit security configurations after changes

4. **Troubleshooting**
   - Document all incident resolutions
   - Create runbooks for common issues
   - Implement automated alerting for known problems

### Performance Optimization Tips

1. **Database Optimization**
   - Regularly vacuum and analyze PostgreSQL tables
   - Monitor and optimize slow queries
   - Maintain appropriate connection pool sizes

2. **Caching Strategies**
   - Use Redis for frequently accessed data
   - Implement proper cache invalidation
   - Monitor cache hit ratios

3. **Resource Management**
   - Set appropriate resource limits
   - Monitor memory and CPU usage
   - Scale services based on load patterns

## Cross-References

### Related Documentation

- [Monitoring Setup Guide](MONITORING_SETUP_GUIDE.md) - Monitoring infrastructure setup
- [Maintenance Procedures](MAINTENANCE_PROCEDURES.md) - Scheduled maintenance tasks
- [Incident Response Playbook](INCIDENT_RESPONSE_PLAYBOOK.md) - Emergency procedures
- [Backup Strategy Guide](BACKUP_STRATEGY_GUIDE.md) - Data protection procedures
- [Performance Monitoring Guide](PERFORMANCE_MONITORING_GUIDE.md) - Performance optimization procedures

### Configuration Files

- [`docker-compose.yml`](../docker-compose.yml) - Service orchestration
- [`internal/config/config.go`](../internal/config/config.go) - Application configuration
- [`infrastructure/prometheus/prometheus.yml`](../infrastructure/prometheus/prometheus.yml) - Monitoring setup
- [`cmd/scheduler/main.go`](../cmd/scheduler/main.go) - Scheduler service implementation

### Tools and Utilities

- **Configuration Validation**: `go run cmd/chatserver/main.go --validate-config`
- **Service Health Check**: `curl http://localhost:8080/health`
- **Resource Monitoring**: `docker stats`
- **Log Analysis**: `docker-compose logs --tail=100 --grep ERROR`
- **Performance Analysis**: `docker-compose top`

### Operational Workflows

1. **System Setup**: Follow [System Setup and Configuration](#system-setup-and-configuration) for initial deployment
2. **Monitoring Configuration**: Refer to [Monitoring Setup Guide](MONITORING_SETUP_GUIDE.md) for comprehensive monitoring
3. **Maintenance Operations**: See [Maintenance Procedures](MAINTENANCE_PROCEDURES.md) for scheduled tasks
4. **Emergency Response**: Use [Incident Response Playbook](INCIDENT_RESPONSE_PLAYBOOK.md) for incident handling
5. **Performance Optimization**: Consult [Performance Monitoring Guide](PERFORMANCE_MONITORING_GUIDE.md) for tuning

### Quick Reference

| Task | Documentation | Tools |
|------|---------------|-------|
| System setup | [System Setup and Configuration](#system-setup-and-configuration) | `docker-compose up -d` |
| Service management | [Service Management](#service-management) | `docker-compose restart` |
| User management | [User Management](#user-management) | `psql -U messaging` |
| Monitoring setup | [Monitoring Setup Guide](MONITORING_SETUP_GUIDE.md) | `curl http://localhost:9090` |
| Incident response | [Incident Response Playbook](INCIDENT_RESPONSE_PLAYBOOK.md) | Security dashboard |
| Performance tuning | [Performance Monitoring Guide](PERFORMANCE_MONITORING_GUIDE.md) | `docker-compose top` |
| Backup procedures | [Backup Strategy Guide](BACKUP_STRATEGY_GUIDE.md) | `pg_dump`, `mc mirror` |

### Documentation Navigation

```mermaid
graph TD
    A[System Administration Guide] --> B[Monitoring Setup Guide]
    A --> C[Maintenance Procedures]
    A --> D[Incident Response Playbook]
    B --> E[Performance Monitoring Guide]
    C --> F[Backup Strategy Guide]
    D --> G[Security Documentation]
    E --> H[Capacity Planning]
    F --> I[Disaster Recovery]
```

### Cross-Document References

- **Monitoring Setup**: For detailed monitoring configuration, see [Monitoring Setup Guide](MONITORING_SETUP_GUIDE.md#prometheus-setup)
- **Maintenance Tasks**: Scheduled maintenance procedures are documented in [Maintenance Procedures](MAINTENANCE_PROCEDURES.md#scheduled-maintenance-tasks)
- **Incident Handling**: Comprehensive incident response procedures in [Incident Response Playbook](INCIDENT_RESPONSE_PLAYBOOK.md#incident-response-procedures)
- **Performance Optimization**: Performance tuning techniques in [Performance Monitoring Guide](PERFORMANCE_MONITORING_GUIDE.md#performance-tuning)
- **Backup and Recovery**: Data protection strategies in [Backup Strategy Guide](BACKUP_STRATEGY_GUIDE.md#database-backup-procedures)

## Change Log

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 2025.12.04 | 2025-12-04 | Initial document creation | Operations Team |
| | | Added system architecture overview | |
| | | Added service management procedures | |
| | | Added troubleshooting guide | |

## Next Steps

1. **Review** this guide and provide feedback
2. **Test** all procedures in staging environment
3. **Update** with any missing or corrected information
4. **Integrate** with existing documentation and training materials

This System Administration Guide provides the foundation for all operational procedures. Refer to the [Operational Documentation Index](OPERATIONAL_DOCUMENTATION_INDEX.md) for the complete documentation suite.