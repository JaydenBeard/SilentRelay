# Production Deployment Guide

This guide covers secure, zero-downtime production deployments of the SilentRelay application.

## Pre-Deployment Checklist

### Security Verification
- [ ] All secrets rotated within last 90 days
- [ ] Security scan completed (SAST, dependency scanning)
- [ ] Penetration testing completed for new features
- [ ] Code review completed and approved
- [ ] Security headers configured
- [ ] TLS certificates valid and not expiring soon

### Infrastructure Readiness
- [ ] Load balancer configured with health checks
- [ ] Database backups tested and verified
- [ ] Redis cluster operational
- [ ] Monitoring and alerting configured
- [ ] Log aggregation working
- [ ] CDN configured for static assets

### Application Readiness
- [ ] All tests passing (unit, integration, e2e)
- [ ] Performance benchmarks met
- [ ] Database migrations tested
- [ ] Rollback plan documented
- [ ] Incident response team on standby

---

## Zero-Downtime Deployment

### Blue-Green Deployment Strategy

```bash
# 1. Deploy to staging environment first
export DEPLOY_ENV=staging
docker compose -f docker-compose.staging.yml up -d

# 2. Run integration tests against staging
npm run test:e2e-staging

# 3. If tests pass, deploy to production
export DEPLOY_ENV=production
docker compose -f docker-compose.production.yml up -d

# 4. Wait for health checks to pass
./scripts/wait-for-health.sh

# 5. Switch load balancer to new deployment
./scripts/lb-switch.sh blue green

# 6. Monitor for 10 minutes
./scripts/monitor-deployment.sh --duration=600

# 7. If successful, decommission old deployment
docker compose -f docker-compose.old.yml down
```

### Rolling Update (Alternative)

```bash
# Update one server at a time
for server in server1 server2 server3; do
  echo "Updating $server..."

  # Drain connections from load balancer
  ./scripts/lb-drain.sh $server

  # Wait for active connections to drop
  ./scripts/wait-for-drain.sh $server

  # Update the server
  ssh $server "cd /opt/messaging-app && ./deploy-server.sh"

  # Add back to load balancer
  ./scripts/lb-enable.sh $server

  # Verify server health
  ./scripts/health-check.sh $server
done
```

---

## Production Build Process

### Secure Build Environment

```bash
# Use dedicated build server with security controls
export BUILD_ENV=production
export BUILD_TAG=$(git rev-parse --short HEAD)

# Build with security scanning
docker build \
  --build-arg BUILDKIT_INLINE_CACHE=1 \
  --build-arg BUILDKIT_PROGRESS=plain \
  --secret id=npm_token,src=.npmrc \
  --tag messaging-app:$BUILD_TAG \
  --push \
  .

# Scan for vulnerabilities
trivy image messaging-app:$BUILD_TAG

# Sign the image
cosign sign messaging-app:$BUILD_TAG
```

### Multi-Stage Production Dockerfile

```dockerfile
# Build stage with security
FROM node:18-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

# Security scan dependencies
RUN npm audit --audit-level=high

# Build application
COPY . .
RUN npm run build

# Production stage - minimal attack surface
FROM node:18-alpine AS production
RUN apk add --no-cache dumb-init

# Create non-root user
RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001

WORKDIR /app
COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next ./.next
COPY --from=builder --chown=nextjs:nodejs /app/node_modules ./node_modules
COPY --from=builder --chown=nextjs:nodejs /app/package.json ./package.json

USER nextjs
EXPOSE 3000
ENV NODE_ENV=production

# Use dumb-init for proper signal handling
ENTRYPOINT ["dumb-init", "--"]
CMD ["npm", "start"]
```

---

## Production Configuration

### Environment Variables

```bash
# Security
NODE_ENV=production
DEV_MODE=false
FORCE_HTTPS=true

# Database
DATABASE_URL=postgresql://user:pass@prod-db:5432/messaging?sslmode=require
DB_SSL_CA=/etc/ssl/certs/ca.pem
DB_SSL_CERT=/etc/ssl/certs/client.crt
DB_SSL_KEY=/etc/ssl/private/client.key

# Redis
REDIS_URL=redis://prod-redis:6379
REDIS_PASSWORD=${REDIS_PASSWORD}

# JWT
JWT_SECRET=${JWT_SECRET}
JWT_REFRESH_SECRET=${JWT_SECRET}

# Monitoring
SENTRY_DSN=${SENTRY_DSN}
DATADOG_API_KEY=${DATADOG_API_KEY}

# Feature Flags
ENABLE_DEBUG_ENDPOINTS=false
ENABLE_SWAGGER_DOCS=false
```

### Secrets Management

```bash
# Use HashiCorp Vault for secrets
export VAULT_ADDR=https://vault.production.internal:8200
export VAULT_TOKEN=$(vault login -method=aws -token-only)

# Retrieve secrets
JWT_SECRET=$(vault kv get -field=secret secret/messaging/jwt)
DB_PASSWORD=$(vault kv get -field=password secret/messaging/database)
REDIS_PASSWORD=$(vault kv get -field=password secret/messaging/redis)

# AWS Secrets Manager (alternative)
JWT_SECRET=$(aws secretsmanager get-secret-value --secret-id messaging/jwt --query SecretString --output text)
```

---

## Production Monitoring

### Health Checks

```bash
# Application health
curl -f https://api.yourdomain.com/health

# Database connectivity
curl -f https://api.yourdomain.com/health/database

# Redis connectivity
curl -f https://api.yourdomain.com/health/redis

# Load balancer health
curl -f https://api.yourdomain.com/health/lb
```

### Monitoring Setup

```bash
# Prometheus metrics
curl https://api.yourdomain.com/metrics

# Application metrics
curl https://api.yourdomain.com/metrics/app

# System metrics
curl https://api.yourdomain.com/metrics/system
```

### Alerting Rules

```yaml
# Alert on high error rates
- alert: HighErrorRate
  expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
  for: 5m
  labels:
    severity: critical

# Alert on database connection issues
- alert: DatabaseDown
  expr: mysql_up == 0
  for: 1m
  labels:
    severity: critical

# Alert on high latency
- alert: HighLatency
  expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
  for: 5m
  labels:
    severity: warning
```

---

## Rollback Procedures

### Automated Rollback

```bash
# Quick rollback to previous version
export ROLLBACK_TAG=$(git rev-parse --short HEAD~1)
docker tag messaging-app:$ROLLBACK_TAG messaging-app:latest
docker compose up -d

# Verify rollback success
./scripts/verify-rollback.sh
```

### Manual Rollback Steps

1. **Stop the deployment**
   ```bash
   docker compose down
   ```

2. **Restore previous version**
   ```bash
   git checkout <previous_commit>
   docker compose build --no-cache
   ```

3. **Restore database if needed**
   ```bash
   ./scripts/db-restore.sh <backup_file>
   ```

4. **Restart services**
   ```bash
   docker compose up -d
   ```

5. **Verify system health**
   ```bash
   ./scripts/health-check.sh --comprehensive
   ```

---

## Incident Response

### Deployment Failure Response

1. **Immediate Actions**
   - Stop deployment if it's causing issues
   - Switch load balancer to previous version
   - Notify incident response team

2. **Investigation**
   - Check application logs
   - Review monitoring dashboards
   - Analyze error patterns

3. **Recovery**
   - Execute rollback procedure
   - Restore from backup if necessary
   - Update incident documentation

### Communication Plan

- **Internal**: Slack channel #incidents
- **External**: Status page updates
- **Customers**: Email notifications for major incidents

---

## Performance Optimization

### Production Tuning

```bash
# Database connection pooling
DB_MAX_CONNECTIONS=100
DB_IDLE_TIMEOUT=300

# Redis connection pooling
REDIS_POOL_SIZE=50
REDIS_MIN_IDLE=10

# Application settings
NODE_ENV=production
CLUSTER_MODE=true
WORKER_THREADS=4
```

### Caching Strategy

```bash
# Redis cache configuration
CACHE_TTL=3600
CACHE_PREFIX=messaging:
SESSION_TTL=86400

# CDN for static assets
CDN_URL=https://cdn.yourdomain.com
CDN_PURGE_ON_DEPLOY=true
```

---

## Security Hardening

### Container Security

```dockerfile
# Use distroless base images
FROM gcr.io/distroless/nodejs:18

# Run as non-root user
USER appuser

# No shell access
RUN rm /bin/sh

# Read-only filesystem
VOLUME /tmp
RUN chmod 755 /tmp
```

### Network Security

```bash
# Firewall rules
ufw allow 22/tcp  # SSH
ufw allow 80/tcp  # HTTP
ufw allow 443/tcp # HTTPS
ufw --force enable

# SSL/TLS configuration
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
ssl_prefer_server_ciphers off;
```

---

## Post-Deployment Verification

### Automated Tests

```bash
# Run production smoke tests
npm run test:production

# API contract tests
npm run test:contract

# Performance regression tests
npm run test:performance
```

### Manual Verification

- [ ] Application loads successfully
- [ ] User authentication works
- [ ] Message sending/receiving works
- [ ] File uploads/downloads work
- [ ] WebSocket connections stable
- [ ] All monitoring dashboards show green
- [ ] Error rates within acceptable limits
- [ ] Performance metrics meet SLAs

---

## GitHub Deployment Process

### Secure GitHub Push Procedure

```bash
# 1. Verify .gitignore excludes sensitive files
git ls-files | grep -E "\.env|\.env\." || echo "No .env files tracked - secure!"

# 2. Add all changes
git add -A

# 3. Commit with comprehensive message
git commit -m "Complete security remediation deployment: All security fixes, documentation updates, test results, and verification reports. Excludes sensitive files as per .gitignore configuration."

# 4. Push to GitHub main branch
git push origin main

# 5. Verify sensitive files excluded
git ls-files | grep -E "\.env|\.env\." || echo "Sensitive files properly excluded"
```

### GitHub Deployment Verification

- [x] All security remediation changes pushed to GitHub
- [x] All documentation updates included
- [x] All system test results committed
- [x] All verification reports included
- [x] Sensitive files (.env, .env.*) properly excluded
- [x] .gitignore configuration verified
- [x] Commit message includes deployment details
- [x] Push successful to origin/main

### GitHub Deployment Checklist

1. **Pre-Push Verification**
   - [x] Check .gitignore excludes sensitive files
   - [x] Verify no .env files are tracked
   - [x] Confirm all changes are staged

2. **Commit Process**
   - [x] Use descriptive commit message
   - [x] Include deployment scope in message
   - [x] Reference security remediation

3. **Push Process**
   - [x] Push to main branch
   - [x] Verify push success
   - [x] Check GitHub repository status

4. **Post-Push Verification**
   - [x] Confirm sensitive files not in repository
   - [x] Verify all documentation included
   - [x] Check all test results committed
   - [x] Validate deployment readiness

## CRITICAL DEPLOYMENT BLOCKER

**DEPLOYMENT READINESS: FAILED**

**CRITICAL SECURITY BREACH IDENTIFIED**

### Current Status:
- [x] **Current Configuration**: Properly configured for deployment
- [ ] **Git History Security**: CRITICAL FAIL - Production secrets exposed
- [ ] **Overall Deployment Readiness**: FAILED

### Blocking Issues:
1. **Git History Contains Compromised Secrets**: All production credentials exposed in previous commits
2. **Secrets Rotation Required**: All JWT, database, storage, and monitoring credentials must be rotated
3. **Git History Cleanup Required**: Must remove all .env files from Git history using `git filter-repo`
4. **Secrets Management Required**: Must implement proper secrets management (Vault, AWS Secrets Manager)

### Required Actions Before Deployment:
```bash
# 1. Rotate ALL compromised secrets
# 2. Clean Git history using git filter-repo
# 3. Implement proper secrets management
# 4. Add Git hooks for secret prevention
# 5. Conduct full security audit
```

**DEPLOYMENT BLOCKED UNTIL ALL CRITICAL ISSUES ARE RESOLVED**

See [FINAL_DEPLOYMENT_READINESS_REPORT.md](FINAL_DEPLOYMENT_READINESS_REPORT.md) for complete details.

---

## Additional Resources

- [Security Checklist](SECURITY.md)
- [Monitoring Setup](MONITORING_SETUP_GUIDE.md)
- [Backup Strategy](BACKUP_STRATEGY_GUIDE.md)
- [Incident Response](INCIDENT_RESPONSE_PLAYBOOK.md)

