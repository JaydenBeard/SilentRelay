#!/bin/bash
#
# SECRETS ROTATION SCRIPT
# 
# IMPORTANT: Run this script after .env was exposed in git history.
# This script generates new secure secrets that must be deployed.
#
# Usage: ./scripts/rotate_secrets.sh
#

set -e

echo "🔐 SilentRelay Secrets Rotation Script"
echo "======================================="
echo ""
echo "⚠️  WARNING: This will generate NEW secrets."
echo "   All existing sessions will be invalidated."
echo ""

# Generate cryptographically secure secrets
generate_secret() {
    openssl rand -base64 $1 | tr -d '\n'
}

generate_hex() {
    openssl rand -hex $1
}

echo "Generating new secrets..."
echo ""

# JWT Secret (64 bytes = 512 bits)
JWT_SECRET=$(generate_hex 64)
echo "JWT_SECRET=$JWT_SECRET"
echo ""

# HMAC Secret (32 bytes = 256 bits)
HMAC_SECRET=$(generate_hex 32)
echo "HMAC_SECRET=$HMAC_SECRET"
echo ""

# PostgreSQL password
POSTGRES_PASS=$(generate_secret 24)
echo "POSTGRES_PASSWORD=$POSTGRES_PASS"
echo ""

# Redis password
REDIS_PASS=$(generate_secret 24)
echo "REDIS_PASSWORD=$REDIS_PASS"
echo ""

# MinIO credentials
MINIO_ACCESS=$(generate_hex 16)
MINIO_SECRET=$(generate_secret 32)
echo "MINIO_ACCESS_KEY=$MINIO_ACCESS"
echo "MINIO_SECRET_KEY=$MINIO_SECRET"
echo ""

# ClickSend API Key placeholder
echo "CLICKSEND_API_KEY=<get from ClickSend dashboard>"
echo ""

echo "======================================="
echo "✅ New secrets generated!"
echo ""
echo "NEXT STEPS:"
echo "1. Update your .env file with these new values"
echo "2. Update PostgreSQL user password:"
echo "   ALTER USER messaging PASSWORD '<new_password>';"
echo "3. Update Redis password in redis.conf"
echo "4. Update MinIO credentials via mc admin"
echo "5. Restart all services"
echo ""
echo "⚠️  IMPORTANT: After rotating secrets, purge git history:"
echo "   git filter-repo --path .env --invert-paths"
echo "   git push origin --force --all"
echo ""
