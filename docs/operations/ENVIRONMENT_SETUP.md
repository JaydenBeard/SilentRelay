# Environment Setup Guide

This guide will help you set up all the necessary keys, secrets, and configuration for the messaging app.

## Security First

**IMPORTANT:** Secrets management is critical for security:
- **`.env` files contain REAL SECRETS and are NEVER committed to git**
- **`.env.example` is the template with placeholder values (tracked in git)**
- **Copy `.env.example` to `.env` and replace ALL placeholders with real secrets**
- **Use environment-specific files: `.env.development`, `.env.production`**
- **Rotate production secrets regularly (every 90 days recommended)**

---

## What's Been Created

The following files are provided:

1. **`.env.example`** - Template with placeholder values (tracked in git)
2. **`.env`** - Your local secrets file (git-ignored, create by copying .env.example)
3. **`.gitignore`** - Prevents accidental commits of secret files
4. **`ENVIRONMENT_SETUP.md`** - This guide

**Note:** Never commit `.env` files. Use `.env.example` as the template for new developers.

---

## Generated Keys

All keys have been securely generated using `openssl rand -base64`:

### JWT Secret
- **Purpose:** Signs authentication tokens
- **Length:** 64 characters (Base64 encoded)
- **Security:** Must be at least 32 characters
- **Rotation:** Every 90 days recommended
- **Current:** Already set in `.env`

### MinIO Access Key
- **Purpose:** S3-compatible object storage access
- **Usage:** Like AWS Access Key ID
- **Current:** Already set in `.env`

### MinIO Secret Key
- **Purpose:** S3-compatible object storage authentication
- **Usage:** Like AWS Secret Access Key
- **Current:** Already set in `.env`

---

## Quick Start

### 1. Set Up Environment File
```bash
# Copy the template to create your local .env file
cp .env.example .env

# Edit .env and replace ALL placeholder values with real secrets
nano .env  # or your preferred editor

# Verify .env is properly ignored by git
git check-ignore .env
# Should output: .env
```

### 2. Start Infrastructure Services

Make sure you have the required services running:

```bash
# Start all services with Docker Compose
docker-compose up -d

# Verify services are running
docker-compose ps
```

Required services:
- PostgreSQL (database)
- Redis (pub/sub, caching)
- MinIO (object storage)
- Consul (service discovery)

### 3. Initialize Database

```bash
# Run database migrations (if you have them)
# psql -U messaging -d messaging -f scripts/schema.sql

# Or connect to PostgreSQL and verify
psql postgresql://messaging:messaging@localhost:5432/messaging
```

### 4. Start the Chat Server

```bash
# Build and run
go run cmd/chatserver/main.go

# Or build first
go build -o chatserver cmd/chatserver/main.go
./chatserver
```

**Expected output:**
```
[DEPLOY] Starting Chat Server: chat-server-1
[NETWORK] Chat Server listening on port 8080
```

**If you see this error:**
```
FATAL: JWT_SECRET environment variable is required
```
Make sure your `.env` file is in the project root.

---

## Configuration Details

### Current Settings (Development)

| Variable | Value | Notes |
|----------|-------|-------|
| DEV_MODE | `true` | Set to `false` in production |
| SERVER_PORT | `8080` | Chat server HTTP port |
| POSTGRES_URL | `localhost:5432` | Local PostgreSQL |
| REDIS_URL | `localhost:6379` | Local Redis |
| CONSUL_URL | `localhost:8500` | Local Consul |
| MINIO_URL | `localhost:9000` | Local MinIO |

### Environment Variables Explained

#### `JWT_SECRET` (REQUIRED)
- Used to sign and verify JWT tokens
- Must be minimum 32 characters
- Application will fail to start if not set or too short
- **Never** use the example value in production

#### `DEV_MODE` (REQUIRED)
- `true`: Development mode (returns verification codes in API)
- `false`: Production mode (codes sent via SMS only)
- **Must be `false` in production**

#### `ALLOWED_ORIGINS` (REQUIRED for WebSocket)
- Comma-separated list of allowed origins
- Used for CORS and WebSocket origin validation
- Example: `https://app.example.com,https://www.example.com`

---

## Rotating Secrets

### When to Rotate:
- Every 90 days (recommended)
- After a security incident
- When team member with access leaves
- Before going to production

### How to Rotate JWT Secret:

```bash
# Generate new JWT secret
openssl rand -base64 48

# Update .env file
# Old sessions will be invalidated
```

### How to Rotate MinIO Keys:

```bash
# Generate new keys
openssl rand -base64 32  # Access Key
openssl rand -base64 32  # Secret Key

# Update .env file
# Update MinIO configuration
docker exec -it minio mc admin user add myminio NEW_ACCESS_KEY NEW_SECRET_KEY
```

---

## Environment-Specific Configuration

The application supports environment-specific configuration files:
- `.env.development` - Development environment settings
- `.env.production` - Production environment settings
- `.env` - Default/fallback settings

Environment files are loaded in order: `.env` → `.env.{NODE_ENV}` → `.env.local`

### Development (.env.development)
```bash
NODE_ENV=development
DEV_MODE=true
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
POSTGRES_URL=postgres://messaging:messaging@localhost:5432/messaging?sslmode=disable
LOG_LEVEL=debug
ENABLE_DEBUG_ENDPOINTS=true
```

### Production (.env.production)
```bash
NODE_ENV=production
DEV_MODE=false
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
POSTGRES_URL=postgres://user:pass@prod-db:5432/messaging?sslmode=require
LOG_LEVEL=info
ENABLE_DEBUG_ENDPOINTS=false
```

### Setting Environment
```bash
# Development
NODE_ENV=development go run cmd/chatserver/main.go

# Production
NODE_ENV=production go run cmd/chatserver/main.go
```

---

## Production Security Checklist

Before deploying to production:

- [ ] Copy `.env.example` to `.env` and replace ALL placeholders with real secrets
- [ ] Generate new cryptographically secure secrets (use `openssl rand -hex 32`)
- [ ] Create `.env.production` with production-specific settings
- [ ] Set `NODE_ENV=production` and `DEV_MODE=false`
- [ ] Update `CORS_ALLOWED_ORIGINS` to production domains only
- [ ] Use strong PostgreSQL password (not default values)
- [ ] Enable SSL for PostgreSQL (`sslmode=require`)
- [ ] Use Redis password authentication
- [ ] Secure MinIO with strong credentials
- [ ] Set file permissions: `chmod 600 .env*`
- [ ] Use HTTPS/TLS for all connections
- [ ] Enable firewall rules
- [ ] Set up monitoring and alerting
- [ ] Configure backup strategy
- [ ] Document incident response plan
- [ ] Verify `.env` is properly git-ignored

---

## Docker Environment Variables

If using Docker, pass environment variables:

```yaml
# docker-compose.yml
services:
  chatserver:
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - DEV_MODE=${DEV_MODE}
      - POSTGRES_URL=${POSTGRES_URL}
    env_file:
      - .env
```

Or pass directly:

```bash
docker run -e JWT_SECRET="your-secret" \
           -e DEV_MODE=false \
           messaging-app:latest
```

---

## Testing Your Setup

### Verify JWT Secret
```bash
# This should start successfully
go run cmd/chatserver/main.go

# This should fail with JWT_SECRET error
JWT_SECRET= go run cmd/chatserver/main.go
```

### Verify Database Connection
```bash
psql "$POSTGRES_URL" -c "SELECT version();"
```

### Verify Redis Connection
```bash
redis-cli -h localhost -p 6379 ping
# Should return: PONG
```

### Verify MinIO Connection
```bash
# Using MinIO client
mc alias set myminio http://localhost:9000 \
  "YOUR_ACCESS_KEY" "YOUR_SECRET_KEY"
mc admin info myminio
```

---

## Troubleshooting

### "FATAL: JWT_SECRET environment variable is required"
**Solution:** Make sure `.env` file exists in the project root (copy from `.env.example`) and contains `JWT_SECRET=...` with a real secret, not a placeholder.

### "JWT_SECRET must be at least 32 characters"
**Solution:** Generate a new key: `openssl rand -base64 48`

### "Database connection failed"
**Solution:**
1. Check PostgreSQL is running: `docker-compose ps`
2. Verify credentials in `.env`
3. Test connection: `psql "$POSTGRES_URL"`

### "Redis connection refused"
**Solution:**
1. Check Redis is running: `docker-compose ps redis`
2. Test connection: `redis-cli ping`

### "MinIO connection failed"
**Solution:**
1. Check MinIO is running: `docker-compose ps minio`
2. Verify keys in `.env` match MinIO configuration
3. Access MinIO console: http://localhost:9001

---

## Additional Resources

- [Security Best Practices](SECURITY_FIXES_COMPLETED.md)
- [Deployment Guide](DEPLOY.md)
- [Quick Start](QUICKSTART.md)
- [API Documentation](docs/)

---

## Advanced Secrets Management

### Production Secrets Architecture

**Multi-Layer Secrets Management**:
- **Application Layer**: Environment variables for runtime config
- **Infrastructure Layer**: Vault/KMS for sensitive credentials
- **Certificate Layer**: ACM/Let's Encrypt for TLS certificates
- **Key Layer**: HSM/KMS for cryptographic keys

### HashiCorp Vault Integration

**Vault Setup for Production**:
```bash
# Initialize Vault
vault operator init

# Unseal Vault
vault operator unseal

# Enable KV secrets engine
vault secrets enable -path=secret kv-v2

# Create policy for application
vault policy write messaging-app - <<EOF
path "secret/data/messaging/*" {
  capabilities = ["read"]
}
EOF

# Create application token
vault token create -policy=messaging-app
```

**Application Integration**:
```go
// Vault client initialization
func NewVaultClient() (*api.Client, error) {
    config := api.DefaultConfig()
    config.Address = os.Getenv("VAULT_ADDR")

    client, err := api.NewClient(config)
    if err != nil {
        return nil, err
    }

    client.SetToken(os.Getenv("VAULT_TOKEN"))
    return client, nil
}

// Retrieve secrets at runtime
func getDatabaseCredentials(vault *api.Client) (string, error) {
    secret, err := vault.Logical().Read("secret/data/messaging/database")
    if err != nil {
        return "", err
    }

    data := secret.Data["data"].(map[string]interface{})
    return data["password"].(string), nil
}
```

### AWS Secrets Manager Integration

**Secrets Manager Setup**:
```bash
# Create secret
aws secretsmanager create-secret \
  --name messaging/database \
  --secret-string '{"username":"messaging","password":"secure-password"}'

# Rotate secret automatically
aws secretsmanager rotate-secret \
  --secret-id messaging/database
```

**Application Integration**:
```go
// AWS SDK integration
func getSecretFromAWS(secretName string) (string, error) {
    svc := secretsmanager.New(session.New())

    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String(secretName),
    })
    if err != nil {
        return "", err
    }

    return *result.SecretString, nil
}
```

### Azure Key Vault Integration

**Key Vault Setup**:
```bash
# Create key vault
az keyvault create --name messaging-kv --resource-group messaging-rg

# Add secret
az keyvault secret set --vault-name messaging-kv --name jwt-secret --value "secure-jwt-secret"

# Set access policy
az keyvault set-policy --name messaging-kv --object-id <app-object-id> --secret-permissions get list
```

### Certificate Management

**Automated Certificate Rotation**:
```bash
# Let's Encrypt with certbot
certbot certonly --standalone -d api.yourdomain.com

# AWS Certificate Manager
aws acm request-certificate \
  --domain-name api.yourdomain.com \
  --validation-method DNS

# Automatic renewal with Lambda
# Configure CloudWatch Events to trigger renewal
```

### Key Rotation Procedures

**JWT Secret Rotation**:
```bash
# Generate new secret
NEW_SECRET=$(openssl rand -base64 48)

# Update in secrets manager
vault kv put secret/messaging/jwt secret=$NEW_SECRET

# Deploy application (maintains backward compatibility)
kubectl rollout restart deployment/messaging-app

# Clean up old secrets after grace period
```

**Database Credential Rotation**:
```bash
# Create new credentials
NEW_PASSWORD=$(openssl rand -base64 32)

# Update database user
psql -c "ALTER USER messaging PASSWORD '$NEW_PASSWORD';"

# Update secrets manager
aws secretsmanager update-secret \
  --secret-id messaging/database \
  --secret-string "{\"password\":\"$NEW_PASSWORD\"}"

# Deploy with new credentials
```

## Security Best Practices

### Secrets Lifecycle Management

1. **Generation**
   - Use cryptographically secure random generators
   - Minimum 256-bit entropy for keys
   - Unique secrets per environment

2. **Storage**
   - Encrypted at rest in secrets managers
   - Access logging enabled
   - Regular backup and recovery testing

3. **Usage**
   - Never log or expose in error messages
   - Use short-lived tokens where possible
   - Implement proper access controls

4. **Rotation**
   - Automatic rotation for short-lived secrets
   - Manual rotation for long-lived secrets
   - 90-day maximum lifetime for production secrets

### Monitoring and Alerting

**Secret Access Monitoring**:
```yaml
# Prometheus alerting rules
- alert: HighSecretAccessRate
  expr: rate(vault_secret_access_total[5m]) > 100
  for: 5m
  labels:
    severity: warning

- alert: UnauthorizedSecretAccess
  expr: vault_secret_access_denied_total > 0
  for: 1m
  labels:
    severity: critical
```

**Audit Logging**:
- All secret access attempts logged
- Failed access attempts alerted
- Regular audit reviews conducted
- Compliance reporting automated

### Disaster Recovery

**Secrets Backup Strategy**:
- Encrypted backups of secrets managers
- Multi-region replication
- Manual recovery procedures documented
- Regular recovery testing

**Emergency Access**:
- Break-glass procedures for emergency access
- Dual authorization required
- Audit trail maintained
- Automatic revocation after use

---

## Support

If you encounter issues:
1. Check this guide first
2. Review error messages carefully
3. Verify all services are running
4. Check logs: `docker-compose logs`
5. Consult the security documentation

---

**Remember:** Your `.env` file contains sensitive secrets. Treat it like a password! Never commit it to version control. Use `.env.example` as the template for new team members.
