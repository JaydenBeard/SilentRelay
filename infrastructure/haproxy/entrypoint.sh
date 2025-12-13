#!/bin/bash

# HAProxy Entrypoint Script with Certificate Validation
# This script ensures HAProxy starts with fallback config when SSL certificates are not available

set -e

# Configuration
CERT_FILE="/etc/ssl/certs/haproxy.pem"
HAPROXY_CONFIG="/usr/local/etc/haproxy/haproxy.cfg"
HAPROXY_FALLBACK_CONFIG="/usr/local/etc/haproxy/haproxy-fallback.cfg"
MAX_RETRIES=60
RETRY_INTERVAL=10
FALLBACK_TIMEOUT=300  # 5 minutes in fallback mode before giving up

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [ENTRYPOINT] $1"
}

# Function to validate certificate
validate_certificate() {
    if [ ! -f "$CERT_FILE" ]; then
        log "Certificate file $CERT_FILE does not exist"
        return 1
    fi

    if [ ! -s "$CERT_FILE" ]; then
        log "Certificate file $CERT_FILE is empty"
        return 1
    fi

    # Check if file contains valid PEM format
    if ! grep -q "BEGIN CERTIFICATE" "$CERT_FILE" && ! grep -q "BEGIN RSA PRIVATE KEY" "$CERT_FILE"; then
        log "Certificate file $CERT_FILE does not appear to be in valid PEM format"
        return 1
    fi

    # Try to validate the certificate using openssl
    if ! openssl x509 -in "$CERT_FILE" -noout >/dev/null 2>&1; then
        log "Certificate file $CERT_FILE is not valid"
        return 1
    fi

    log "✓ Certificate validation successful"
    return 0
}

# Function to check if HAProxy config is valid
validate_haproxy_config() {
    local config_file=$1

    if [ ! -f "$config_file" ]; then
        log "HAProxy configuration file $config_file does not exist"
        return 1
    fi

    # Test HAProxy configuration syntax
    if haproxy -c -f "$config_file"; then
        log "✓ HAProxy configuration $config_file is valid"
        return 0
    else
        log "HAProxy configuration $config_file is invalid"
        return 1
    fi
}

# Function to start HAProxy with fallback configuration
start_fallback_mode() {
    log "Starting HAProxy in fallback mode (HTTP only)"

    if [ -f "$HAPROXY_FALLBACK_CONFIG" ]; then
        if validate_haproxy_config "$HAPROXY_FALLBACK_CONFIG"; then
            log "Starting HAProxy with fallback configuration..."
            exec haproxy -f "$HAPROXY_FALLBACK_CONFIG"
        else
            log "Fallback configuration is invalid, cannot start HAProxy"
            return 1
        fi
    else
        log "Fallback configuration not found, cannot start HAProxy"
        return 1
    fi
}

# Main execution
log "Starting HAProxy entrypoint script"

# First, check if we have a fallback configuration
if [ -f "$HAPROXY_FALLBACK_CONFIG" ]; then
    log "Fallback configuration found"
else
    log "No fallback configuration found, will wait for SSL certificates"
fi

# Check if certificates are already available
if validate_certificate && validate_haproxy_config "$HAPROXY_CONFIG"; then
    log "All validations passed, starting HAProxy with SSL..."
    exec haproxy -f "$HAPROXY_CONFIG"
else
    log "SSL certificates not available, entering retry mode..."

    retry_count=0
    fallback_started=false
    fallback_start_time=0

    while [ $retry_count -lt $MAX_RETRIES ]; do
        if validate_certificate && validate_haproxy_config "$HAPROXY_CONFIG"; then
            log "Certificates now available, restarting HAProxy with SSL configuration..."

            # If we were in fallback mode, we need to restart the container
            # For now, just start with SSL config
            exec haproxy -f "$HAPROXY_CONFIG"
        else
            # If we have fallback config and haven't started it yet, start it
            if [ "$fallback_started" = false ] && [ -f "$HAPROXY_FALLBACK_CONFIG" ]; then
                log "Starting fallback mode while waiting for certificates..."
                if start_fallback_mode; then
                    # This should not return if exec works
                    fallback_started=true
                    fallback_start_time=$(date +%s)
                fi
            fi

            # Check if we've been in fallback mode too long
            if [ "$fallback_started" = true ]; then
                current_time=$(date +%s)
                elapsed_time=$((current_time - fallback_start_time))

                if [ "$elapsed_time" -ge "$FALLBACK_TIMEOUT" ]; then
                    log "ERROR: Timeout reached in fallback mode, giving up"
                    exit 1
                fi
            fi

            retry_count=$((retry_count + 1))
            if [ $retry_count -eq $MAX_RETRIES ]; then
                log "ERROR: Maximum retries reached, giving up"
                exit 1
            fi

            if [ "$fallback_started" = false ]; then
                log "Retry $retry_count/$MAX_RETRIES - waiting $RETRY_INTERVAL seconds for certificates..."
                sleep $RETRY_INTERVAL
            fi
        fi
    done
fi

# Should never reach here
exit 1