#!/bin/bash

# ==============================================
# Production Deployment Script
# Deploys to silentrelay.com.au server
# ==============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SERVER="silentrelay.com.au"
REMOTE_PATH="/root/SilentRelay"

echo ""
echo -e "${BLUE}🚀 Deploying to Production: ${SERVER}${NC}"
echo "======================================"
echo ""

# Check if we have the latest changes
echo -e "${YELLOW}[1/6]${NC} Checking local git status..."
if [[ -n $(git status --porcelain) ]]; then
    echo -e "${RED}❌ You have uncommitted changes. Please commit or stash them first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Working directory clean${NC}"

# Get current commit
CURRENT_COMMIT=$(git rev-parse --short HEAD)
echo -e "${YELLOW}[2/6]${NC} Deploying commit: ${CURRENT_COMMIT}"

# Test SSH connection
echo -e "${YELLOW}[3/6]${NC} Testing SSH connection..."
if ! ssh -o ConnectTimeout=10 -o BatchMode=yes root@${SERVER} "echo 'SSH connection successful'" &>/dev/null; then
    echo -e "${RED}❌ Cannot connect to ${SERVER}. Check SSH key and network.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ SSH connection established${NC}"

# Deploy to server
echo -e "${YELLOW}[4/6]${NC} Deploying to server..."

ssh root@${SERVER} << EOF
    set -e

    echo "Updating code on server..."
    cd ${REMOTE_PATH}

    # Pull latest changes
    git fetch origin
    git checkout main
    git pull origin main

    # Verify we're on the right commit
    CURRENT_COMMIT_SERVER=\$(git rev-parse --short HEAD)
    if [ "\${CURRENT_COMMIT_SERVER}" != "${CURRENT_COMMIT}" ]; then
        echo "❌ Commit mismatch! Expected ${CURRENT_COMMIT}, got \${CURRENT_COMMIT_SERVER}"
        exit 1
    fi

    echo "✓ Code updated to commit ${CURRENT_COMMIT}"

    # Load production secrets
    echo "Loading production secrets..."
    if [ -f .env.production ]; then
        source .env.production
        echo "✓ Production secrets loaded from .env.production"
    else
        echo "⚠️  Warning: .env.production not found. Ensure secrets are set in environment."
    fi

    # Build and start services
    echo "Building and starting services..."
    cd web-new && npm install && npm run build && cd ..

    # Start/restart services
    docker compose down || true
    docker compose build --no-cache
    docker compose up -d

    echo "✓ Services deployed"
EOF

echo -e "${GREEN}✓ Deployment commands sent to server${NC}"

# Wait for services to start
echo -e "${YELLOW}[5/6]${NC} Waiting for services to start..."
sleep 15

# Test deployment
echo -e "${YELLOW}[6/6]${NC} Testing deployment..."

# Test health endpoint (using HTTP for now since SSL may not be configured)
if curl -s -o /dev/null -w "%{http_code}" "http://${SERVER}/health" | grep -q "200"; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${YELLOW}⚠️  Health check failed - services may still be starting${NC}"
fi

# Test frontend
if curl -s -o /dev/null -w "%{http_code}" "http://${SERVER}" | grep -q "200"; then
    echo -e "${GREEN}✓ Frontend responding${NC}"
else
    echo -e "${YELLOW}⚠️  Frontend not responding${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Deployment Complete!${NC}"
echo ""
echo "======================================"
echo "Production URLs:"
echo "======================================"
echo ""
echo -e "  ${BLUE}Frontend:${NC}        http://${SERVER}"
echo -e "  ${BLUE}API:${NC}             http://${SERVER}/api/v1"
echo -e "  ${BLUE}WebSocket:${NC}       ws://${SERVER}/ws"
echo ""
echo -e "  ${YELLOW}Health Check:${NC}    http://${SERVER}/health"
echo ""
echo -e "  ${YELLOW}Note:${NC} HTTPS will be enabled once SSL certificates are configured"
echo ""
echo "======================================"
echo ""
echo -e "${YELLOW}📋 Check server logs:${NC}"
echo "   ssh root@${SERVER} 'cd ${REMOTE_PATH} && docker compose logs -f'"
echo ""
echo -e "${YELLOW}🛑 Emergency rollback:${NC}"
echo "   ssh root@${SERVER} 'cd ${REMOTE_PATH} && git checkout HEAD~1 && docker compose build --no-cache && docker compose up -d'"
echo ""

