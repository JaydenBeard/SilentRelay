# Quick Start Guide

Get your secure messenger running in 5 minutes!

## Prerequisites

- Linux server (Ubuntu 20.04+ recommended)
- Docker & Docker Compose
- At least 4GB RAM, 20GB disk

## One-Liner Install

```bash
# Clone the repo
git clone https://github.com/yourusername/messaging-app.git
cd messaging-app

# Make scripts executable
chmod +x scripts/*.sh

# Start everything
./scripts/startup.sh
```

That's it! 

## Manual Setup

### 1. Install Docker (if needed)

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Add yourself to docker group (logout/login after)
sudo usermod -aG docker $USER

# Install Docker Compose plugin
sudo apt update
sudo apt install docker-compose-plugin
```

### 2. Clone & Configure

```bash
# Clone repo
git clone https://github.com/yourusername/messaging-app.git
cd messaging-app

# Generate secrets (optional - startup.sh does this too)
cat > .env << EOF
JWT_SECRET=$(openssl rand -hex 32)
SERVER_SIGNING_KEY=$(openssl rand -hex 32)
EOF
```

### 3. Build & Start

```bash
# Build all images
docker compose build

# Start everything
docker compose up -d

# Watch the logs
docker compose logs -f
```

### 4. Access the App

| Service | URL |
|---------|-----|
| **Frontend** | http://your-server |
| **API** | http://your-server/api/v1 |
| HAProxy Stats | http://your-server:8404/stats |
| Consul | http://your-server:8500 |
| MinIO Console | http://your-server:9001 |
| Prometheus | http://your-server:9090 |
| Grafana | http://your-server:3001 |

## Test It's Working

```bash
# Health check
curl http://localhost/health

# Should return: OK

# Check all containers are running
docker compose ps

# View logs if something's wrong
docker compose logs chat-server-1
```

## Common Issues

### Port 80 already in use
```bash
# Check what's using it
sudo lsof -i :80

# Stop the service (e.g., Apache)
sudo systemctl stop apache2
```

### Containers keep restarting
```bash
# Check the logs
docker compose logs --tail=50 [service-name]

# Common fix: wait for dependencies
docker compose down
docker compose up -d postgres redis
sleep 10
docker compose up -d
```

### Out of disk space
```bash
# Clean up Docker
docker system prune -a
```

## Useful Commands

```bash
# Stop everything
docker compose down

# Stop and remove all data (fresh start)
docker compose down -v

# Rebuild a specific service
docker compose build chat-server-1
docker compose up -d chat-server-1

# Scale workers
docker compose up -d --scale queue-worker=4

# View resource usage
docker stats
```

## Firewall Setup

If using UFW:

```bash
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS (if using SSL)
```

## Next Steps

1. Access the app at http://your-server
2. Register with your phone number
3. Set up your PIN (4 or 6 digits)
4. **SAVE YOUR RECOVERY KEY** (24 words)
5. Start chatting securely!

## Production Hardening

Before going live:

1. **Set up HTTPS** - Get certs from Let's Encrypt
2. **Change default passwords** - Edit `.env`
3. **Configure firewall** - Only expose ports 80/443
4. **Set up backups** - Backup the volumes
5. **Enable monitoring alerts** - Configure Grafana

See [docs/SECURITY.md](docs/SECURITY.md) for the full security checklist.

---

Questions? Check the full [README.md](README.md) or open an issue!

