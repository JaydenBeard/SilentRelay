#!/bin/bash
#
# Docker Secrets Setup Script
# Run this on the server to initialize secrets
#

set -e

SECRETS_DIR="/opt/secrets"

echo "🔐 Setting up Docker Secrets"
echo "=============================="

# Create secrets directory
mkdir -p "$SECRETS_DIR"
chmod 700 "$SECRETS_DIR"

# Function to create a secret
create_secret() {
    local name=$1
    local description=$2
    local default=$3
    local file="$SECRETS_DIR/$name"
    
    if [ -f "$file" ]; then
        echo "✓ $name already exists"
        return
    fi
    
    echo ""
    echo "$description"
    if [ -n "$default" ]; then
        echo "Press Enter to use existing value from .env, or type a new value:"
        read -r value
        if [ -z "$value" ]; then
            value="$default"
        fi
    else
        echo "Enter value (or press Enter to generate random):"
        read -r value
        if [ -z "$value" ]; then
            value=$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)
            echo "Generated: $value"
        fi
    fi
    
    echo -n "$value" > "$file"
    chmod 444 "$file"  # Readable by all (secrets are in protected directory)
    echo "✓ Created $name"
}

# Load existing values from .env if present
if [ -f ".env" ]; then
    echo "Loading existing values from .env..."
    source .env 2>/dev/null || true
fi

# Create secrets
create_secret "jwt_secret" "JWT Secret (for authentication tokens)" "${JWT_SECRET:-}"
create_secret "hmac_secret" "HMAC Secret (for WebSocket message signing)" "${HMAC_SECRET:-}"
create_secret "postgres_password" "PostgreSQL Password" "${POSTGRES_PASSWORD:-}"
create_secret "clicksend_api_key" "ClickSend API Key (for SMS)" "${CLICKSEND_API_KEY:-}"
create_secret "turn_secret" "TURN Server Secret (for WebRTC)" "${TURN_SECRET:-}"
create_secret "minio_root_password" "MinIO Root Password" "${MINIO_ROOT_PASSWORD:-}"

echo ""
echo "=============================="
echo "✅ Secrets setup complete!"
echo ""
echo "Secrets stored in: $SECRETS_DIR"
ls -la "$SECRETS_DIR"
echo ""
echo "Next steps:"
echo "1. Pull the latest code: git pull"
echo "2. Restart services: docker compose down && docker compose up -d"
