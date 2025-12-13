#!/bin/bash

# ==============================================
# Quick Dev Startup (minimal services)
# ==============================================

set -e

echo "🚀 Starting minimal dev environment..."

# Start just the essentials
docker compose up -d postgres redis consul minio

echo "⏳ Waiting for databases..."
sleep 5

# Check postgres is ready
until docker compose exec -T postgres pg_isready -U messaging > /dev/null 2>&1; do
    echo "  Waiting for PostgreSQL..."
    sleep 2
done

echo "✅ Databases ready!"

# Start the chat servers
docker compose up -d chat-server-1 chat-server-2 loadbalancer

echo ""
echo "✅ Dev environment started!"
echo ""
echo "Services:"
echo "  - API:      http://localhost/api/v1"
echo "  - WebSocket: ws://localhost/ws"
echo "  - HAProxy:  http://localhost:8404/stats"
echo "  - MinIO:    http://localhost:9001"
echo "  - Consul:   http://localhost:8500"
echo ""
echo "To start frontend:"
echo "  cd web-new && npm install && npm run dev"
echo ""
echo "To view logs:"
echo "  docker compose logs -f chat-server-1 chat-server-2"
echo ""

