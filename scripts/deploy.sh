#!/bin/bash
#
# Zero-Downtime Deployment Script for SilentRelay
# 
# Usage: ./scripts/deploy.sh [--skip-build] [--skip-frontend] [--skip-supporting]
#
# This script performs a rolling update of all services,
# ensuring the service remains available throughout the deployment.
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
HEALTH_CHECK_RETRIES=30
HEALTH_CHECK_INTERVAL=2
DRAIN_WAIT=5

# Parse arguments
SKIP_BUILD=false
SKIP_FRONTEND=false
SKIP_SUPPORTING=false
for arg in "$@"; do
    case $arg in
        --skip-build)
            SKIP_BUILD=true
            ;;
        --skip-frontend)
            SKIP_FRONTEND=true
            ;;
        --skip-supporting)
            SKIP_SUPPORTING=true
            ;;
    esac
done

echo -e "${BLUE}🚀 Starting Zero-Downtime Deployment${NC}"
echo -e "${BLUE}================================================${NC}"
date

# Function to check if a service is healthy
check_health() {
    local port=$1
    local name=$2
    local retries=$HEALTH_CHECK_RETRIES
    
    echo -e "${YELLOW}⏳ Waiting for $name to be healthy...${NC}"
    
    while [ $retries -gt 0 ]; do
        if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ $name is healthy${NC}"
            return 0
        fi
        retries=$((retries - 1))
        sleep $HEALTH_CHECK_INTERVAL
    done
    
    echo -e "${RED}❌ $name failed health check after $HEALTH_CHECK_RETRIES attempts${NC}"
    return 1
}

# Step 1: Pull latest code
echo ""
echo -e "${BLUE}📥 Step 1: Pulling latest code from git...${NC}"
git pull origin main

# Step 2: Build new images (if not skipped)
if [ "$SKIP_BUILD" = false ]; then
    echo ""
    echo -e "${BLUE}📦 Step 2: Building Docker images...${NC}"
    
    # Build chat servers
    echo "Building chat-server-1 and chat-server-2..."
    docker compose build chat-server-1 chat-server-2
    
    # Build supporting services (if not skipped)
    if [ "$SKIP_SUPPORTING" = false ]; then
        echo "Building supporting services..."
        docker compose build group-service presence-service notification-service scheduler queue-worker
    fi
    
    # Build frontend (if not skipped)
    if [ "$SKIP_FRONTEND" = false ]; then
        echo "Building frontend..."
        docker compose build frontend
    fi
else
    echo ""
    echo -e "${YELLOW}⏭️  Step 2: Skipping build (--skip-build flag set)${NC}"
fi

# Step 3: Deploy chat-server-1
echo ""
echo -e "${BLUE}🔄 Step 3: Deploying chat-server-1...${NC}"

# Send SIGTERM to allow graceful shutdown
echo "Sending graceful shutdown signal to chat-server-1..."
docker compose kill -s SIGTERM chat-server-1 2>/dev/null || true

# Wait for connections to drain
echo "Waiting ${DRAIN_WAIT}s for connections to drain..."
sleep $DRAIN_WAIT

# Stop and start with new image
docker compose stop chat-server-1
docker compose up -d chat-server-1

# Wait for health
if ! check_health 8081 "chat-server-1"; then
    echo -e "${RED}❌ Deployment failed! chat-server-1 is not healthy.${NC}"
    echo -e "${YELLOW}⚠️  chat-server-2 is still running with the old version.${NC}"
    exit 1
fi

# Step 4: Deploy chat-server-2
echo ""
echo -e "${BLUE}🔄 Step 4: Deploying chat-server-2...${NC}"

# Send SIGTERM to allow graceful shutdown
echo "Sending graceful shutdown signal to chat-server-2..."
docker compose kill -s SIGTERM chat-server-2 2>/dev/null || true

# Wait for connections to drain
echo "Waiting ${DRAIN_WAIT}s for connections to drain..."
sleep $DRAIN_WAIT

# Stop and start with new image
docker compose stop chat-server-2
docker compose up -d chat-server-2

# Wait for health
if ! check_health 8082 "chat-server-2"; then
    echo -e "${RED}❌ Deployment partially failed! chat-server-2 is not healthy.${NC}"
    echo -e "${YELLOW}⚠️  chat-server-1 is running with the new version.${NC}"
    exit 1
fi

# Step 5: Deploy supporting services (if not skipped)
if [ "$SKIP_SUPPORTING" = false ]; then
    echo ""
    echo -e "${BLUE}🔄 Step 5: Deploying supporting services...${NC}"
    docker compose up -d group-service presence-service notification-service scheduler queue-worker turn
    echo -e "${GREEN}✅ Supporting services deployed${NC}"
else
    echo ""
    echo -e "${YELLOW}⏭️  Step 5: Skipping supporting services (--skip-supporting flag set)${NC}"
fi

# Step 6: Deploy frontend (if not skipped)
if [ "$SKIP_FRONTEND" = false ]; then
    echo ""
    echo -e "${BLUE}🔄 Step 6: Deploying frontend...${NC}"
    docker compose stop frontend
    docker compose up -d frontend
    
    # Wait for frontend to be ready
    sleep 5
    if curl -sf "http://localhost:3000" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Frontend is running${NC}"
    else
        echo -e "${YELLOW}⚠️  Frontend may still be starting up${NC}"
    fi
else
    echo ""
    echo -e "${YELLOW}⏭️  Step 6: Skipping frontend deployment (--skip-frontend flag set)${NC}"
fi

# Step 7: Reload HAProxy and restart Prometheus to pick up config changes
echo ""
echo -e "${BLUE}🔄 Step 7: Reloading HAProxy and Prometheus...${NC}"
docker compose up -d --force-recreate loadbalancer
docker compose restart prometheus
echo -e "${GREEN}✅ HAProxy and Prometheus reloaded${NC}"

# Final status
echo ""
echo -e "${BLUE}================================================${NC}"
echo -e "${GREEN}🎉 Deployment Complete!${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""
echo "Service Status:"
docker compose ps
echo ""
date
echo ""
echo -e "${GREEN}All services are running the new version.${NC}"
