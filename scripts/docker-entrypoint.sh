#!/bin/sh
# Docker secrets to environment variables helper
# This script reads secrets from /run/secrets/ and exports them as env vars

# Function to read secret file if it exists
read_secret() {
    secret_name=$1
    env_var_name=$2
    secret_file="/run/secrets/${secret_name}"
    
    if [ -f "$secret_file" ]; then
        # Read the secret, trim whitespace
        value=$(cat "$secret_file" | tr -d '\n')
        export "$env_var_name=$value"
        echo "✓ Loaded secret: $env_var_name"
    fi
}

# Load all secrets
read_secret "jwt_secret" "JWT_SECRET"
read_secret "hmac_secret" "HMAC_SECRET"
read_secret "postgres_password" "POSTGRES_PASSWORD"
read_secret "clicksend_api_key" "CLICKSEND_API_KEY"
read_secret "turn_secret" "TURN_SECRET"
read_secret "minio_root_password" "MINIO_ROOT_PASSWORD"

# Build POSTGRES_URL if password is set from secret
if [ -n "$POSTGRES_PASSWORD" ]; then
    export POSTGRES_URL="postgres://messaging:${POSTGRES_PASSWORD}@${POSTGRES_HOST:-postgres}:5432/messaging?sslmode=disable"
fi

# Execute the main command
exec "$@"
