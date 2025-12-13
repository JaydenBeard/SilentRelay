#!/bin/bash

# ==============================================
# Secure Messenger - One-Command Startup
# ==============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo -e "${BLUE}🔐 Secure Messenger - Startup Script${NC}"
echo "======================================"
echo ""

# Check prerequisites
echo -e "${YELLOW}[1/6]${NC} Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker not found. Install it first:${NC}"
    echo "   curl -fsSL https://get.docker.com | sh"
    echo "   sudo usermod -aG docker \$USER"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose not found. Install it:${NC}"
    echo "   sudo apt install docker-compose-plugin"
    exit 1
fi

echo -e "${GREEN}✓ Docker and Docker Compose found${NC}"

# Create required directories
echo -e "${YELLOW}[2/6]${NC} Creating directories..."
mkdir -p data/postgres data/redis data/minio data/consul
echo -e "${GREEN}✓ Directories created${NC}"

# Generate secrets if not exist
echo -e "${YELLOW}[3/6]${NC} Generating secrets..."

if [ ! -f .env ]; then
    JWT_SECRET=$(openssl rand -hex 32)
    SERVER_KEY=$(openssl rand -hex 32)
    
    cat > .env << EOF
# Auto-generated secrets - KEEP THESE SAFE!
JWT_SECRET=${JWT_SECRET}
SERVER_SIGNING_KEY=${SERVER_KEY}
POSTGRES_PASSWORD=$(openssl rand -hex 16)
REDIS_PASSWORD=$(openssl rand -hex 16)
MINIO_ROOT_PASSWORD=$(openssl rand -hex 16)

# Server config
SERVER_PORT=8080
ENVIRONMENT=development
EOF
    echo -e "${GREEN}✓ Generated .env file with secure secrets${NC}"
else
    echo -e "${GREEN}✓ Using existing .env file${NC}"
fi

# Build images
echo -e "${YELLOW}[4/6]${NC} Building Docker images (this may take a few minutes)..."
docker compose build --quiet
echo -e "${GREEN}✓ Images built${NC}"

# Start services
echo -e "${YELLOW}[5/6]${NC} Starting services..."
docker compose up -d
echo -e "${GREEN}✓ Services started${NC}"

# Wait for health checks
echo -e "${YELLOW}[6/6]${NC} Waiting for services to be healthy..."
sleep 10

# Check service health
echo ""
echo "Service Status:"
echo "---------------"

check_service() {
    local name=$1
    local url=$2
    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "200\|301\|302\|404"; then
        echo -e "  ${GREEN}✓${NC} $name"
    else
        echo -e "  ${YELLOW}⏳${NC} $name (starting...)"
    fi
}

check_service "HAProxy (Load Balancer)" "http://localhost:80/health"
check_service "Chat Server 1" "http://localhost:8081/health"
check_service "Chat Server 2" "http://localhost:8082/health"
check_service "Consul" "http://localhost:8500/v1/status/leader"
check_service "MinIO" "http://localhost:9000/minio/health/live"

echo ""
echo -e "${GREEN}🚀 Startup complete!${NC}"
echo ""
echo "======================================"
echo "Access Points:"
echo "======================================"
echo ""
echo -e "  ${BLUE}Frontend:${NC}        http://localhost"
echo -e "  ${BLUE}API:${NC}             http://localhost/api/v1"
echo -e "  ${BLUE}WebSocket:${NC}       ws://localhost/ws"
echo ""
echo -e "  ${YELLOW}HAProxy Stats:${NC}   http://localhost:8404/stats"
echo -e "  ${YELLOW}Consul UI:${NC}       http://localhost:8500"
echo -e "  ${YELLOW}MinIO Console:${NC}   http://localhost:9001"
echo -e "  ${YELLOW}Prometheus:${NC}      http://localhost:9090"
echo -e "  ${YELLOW}Grafana:${NC}         http://localhost:3001 (admin/admin)"
echo ""
echo "======================================"
echo ""
echo -e "${YELLOW}📝 Quick Test:${NC}"
echo "   curl http://localhost/health"
echo ""
echo -e "${YELLOW}📋 View Logs:${NC}"
echo "   docker compose logs -f"
echo ""
echo -e "${YELLOW}🛑 Stop Everything:${NC}"
echo "   docker compose down"
echo ""

