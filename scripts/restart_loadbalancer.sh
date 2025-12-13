#!/bin/bash

# =============================================================================
# Load Balancer Restart Script for SSL Certificate Updates
# =============================================================================
#
# This script handles the proper restart of the HAProxy load balancer after
# SSL certificate updates. It includes comprehensive validation, error
# handling, and feedback mechanisms.
#
# Usage: ./restart_loadbalancer.sh [options]
#
# Options:
#   --help, -h          Show this help message
#   --dry-run, -n       Perform validation but don't actually restart
#   --force, -f         Force restart even if certificates haven't changed
#   --verbose, -v       Enable verbose output
#
# Exit Codes:
#   0   - Success
#   1   - General error
#   2   - Certificate validation failed
#   3   - Container not running
#   4   - Docker command failed
#   5   - Invalid arguments
#
# =============================================================================

# Configuration
CERT_DIR="./infrastructure/certs"
REQUIRED_CERTS=("haproxy.pem")
DOCKER_COMPOSE_FILE="docker-compose.yml"
LOADBALANCER_SERVICE="loadbalancer"
VERBOSE=false
DRY_RUN=false
FORCE_RESTART=false

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to display help
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --help, -h          Show this help message"
    echo "  --dry-run, -n       Perform validation but don't actually restart"
    echo "  --force, -f         Force restart even if certificates haven't changed"
    echo "  --verbose, -v       Enable verbose output"
    echo ""
    echo "This script validates SSL certificates and restarts the HAProxy load balancer."
    echo "It ensures proper certificate deployment and minimal service disruption."
}

# Function to log messages
log_message() {
    local level=$1
    local message=$2
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")

    case $level in
        "INFO")
            echo -e "${BLUE}[${timestamp}] [INFO] ${message}${NC}"
            ;;
        "SUCCESS")
            echo -e "${GREEN}[${timestamp}] [SUCCESS] ${message}${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}[${timestamp}] [WARNING] ${message}${NC}"
            ;;
        "ERROR")
            echo -e "${RED}[${timestamp}] [ERROR] ${message}${NC}"
            ;;
        *)
            echo -e "[${timestamp}] [${level}] ${message}"
            ;;
    esac
}

# Function to check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_message "ERROR" "Docker is not installed or not in PATH"
        return 1
    fi

    if ! docker info &> /dev/null; then
        log_message "ERROR" "Docker daemon is not running"
        return 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_message "ERROR" "docker-compose is not installed"
        return 1
    fi

    log_message "INFO" "Docker environment verified"
    return 0
}

# Function to validate certificate files
validate_certificates() {
    log_message "INFO" "Validating SSL certificates..."

    # Check if certificate directory exists
    if [ ! -d "$CERT_DIR" ]; then
        log_message "ERROR" "Certificate directory $CERT_DIR does not exist"
        return 2
    fi

    # Check each required certificate file
    for cert in "${REQUIRED_CERTS[@]}"; do
        cert_path="$CERT_DIR/$cert"

        if [ ! -f "$cert_path" ]; then
            log_message "ERROR" "Required certificate file $cert_path does not exist"
            return 2
        fi

        if [ ! -r "$cert_path" ]; then
            log_message "ERROR" "Certificate file $cert_path is not readable"
            return 2
        fi

        # Check file size (should be > 0)
        file_size=$(stat -f%z "$cert_path" 2>/dev/null || stat -c%s "$cert_path" 2>/dev/null)
        if [ "$file_size" -eq 0 ]; then
            log_message "ERROR" "Certificate file $cert_path is empty"
            return 2
        fi

        log_message "INFO" "✓ Certificate $cert is valid and readable"
    done

    # Additional validation for PEM format
    for cert in "${REQUIRED_CERTS[@]}"; do
        cert_path="$CERT_DIR/$cert"
        if ! grep -q "BEGIN CERTIFICATE" "$cert_path" && ! grep -q "BEGIN RSA PRIVATE KEY" "$cert_path"; then
            log_message "WARNING" "Certificate $cert_path may not be in proper PEM format"
        fi
    done

    return 0
}

# Function to check if load balancer container is running
check_loadbalancer_running() {
    log_message "INFO" "Checking load balancer container status..."

    # Check if container is running using docker-compose
    if docker-compose -f "$DOCKER_COMPOSE_FILE" ps --services | grep -q "$LOADBALANCER_SERVICE"; then
        if docker-compose -f "$DOCKER_COMPOSE_FILE" ps "$LOADBALANCER_SERVICE" | grep "$LOADBALANCER_SERVICE" | grep -q "Up"; then
            log_message "INFO" "✓ Load balancer container is running"
            return 0
        else
            log_message "ERROR" "Load balancer container $LOADBALANCER_SERVICE is not running"
            return 3
        fi
    else
        log_message "ERROR" "Load balancer container $LOADBALANCER_SERVICE is not running"
        return 3
    fi
}

# Function to restart load balancer
restart_loadbalancer() {
    log_message "INFO" "Restarting load balancer container..."

    if [ "$DRY_RUN" = true ]; then
        log_message "INFO" "DRY-RUN: Would execute: docker-compose -f $DOCKER_COMPOSE_FILE restart $LOADBALANCER_SERVICE"
        log_message "SUCCESS" "DRY-RUN: Load balancer restart would be performed"
        return 0
    fi

    # Restart the container
    if docker-compose -f "$DOCKER_COMPOSE_FILE" restart "$LOADBALANCER_SERVICE"; then
        log_message "SUCCESS" "Load balancer container restarted successfully"

        # Verify the container is running after restart
        sleep 5
        if check_loadbalancer_running; then
            log_message "SUCCESS" "Load balancer is running after restart"
            return 0
        else
            log_message "ERROR" "Load balancer failed to start after restart"
            return 4
        fi
    else
        log_message "ERROR" "Failed to restart load balancer container"
        return 4
    fi
}

# Function to check certificate modification time
check_certificate_changes() {
    if [ "$FORCE_RESTART" = true ]; then
        log_message "INFO" "Force restart enabled - skipping certificate change detection"
        return 0
    fi

    # Get current modification time of haproxy.pem
    current_mtime=$(stat -c %Y "$CERT_DIR/haproxy.pem" 2>/dev/null || date -r "$CERT_DIR/haproxy.pem" +%s 2>/dev/null)

    # Check if we have a previous modification time stored
    if [ -f "/tmp/loadbalancer_cert_mtime.txt" ]; then
        previous_mtime=$(cat "/tmp/loadbalancer_cert_mtime.txt")

        if [ "$current_mtime" = "$previous_mtime" ]; then
            log_message "INFO" "Certificates have not changed since last restart"
            return 1
        fi
    fi

    # Store current modification time for next comparison
    echo "$current_mtime" > "/tmp/loadbalancer_cert_mtime.txt"
    log_message "INFO" "Certificate changes detected or first run"
    return 0
}

# Main execution
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_help
                exit 0
                ;;
            --dry-run|-n)
                DRY_RUN=true
                shift
                ;;
            --force|-f)
                FORCE_RESTART=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            *)
                log_message "ERROR" "Unknown option: $1"
                show_help
                exit 5
                ;;
        esac
    done

    log_message "INFO" "Starting load balancer restart procedure..."
    log_message "INFO" "Options: Dry-run=$DRY_RUN, Force=$FORCE_RESTART, Verbose=$VERBOSE"

    # Step 1: Check Docker environment
    if ! check_docker; then
        exit 1
    fi

    # Step 2: Validate certificates
    if ! validate_certificates; then
        exit 2
    fi

    # Step 3: Check if certificates have changed (unless forced)
    if ! check_certificate_changes; then
        log_message "INFO" "No certificate changes detected, restart not needed"
        exit 0
    fi

    # Step 4: Check if load balancer is running
    if ! check_loadbalancer_running; then
        exit 3
    fi

    # Step 5: Restart load balancer
    if ! restart_loadbalancer; then
        exit 4
    fi

    log_message "SUCCESS" "Load balancer restart procedure completed successfully"
    exit 0
}

# Execute main function
main "$@"