# SilentRelay Operations Runbook

## Deployment

### Automatic (CI/CD)
Push to `main` branch → GitHub Actions automatically:
1. Runs tests
2. Builds Docker images
3. SSHs into server
4. Runs `scripts/deploy.sh`

### Manual Deployment
```bash
cd /opt/end2endsecure.com
git pull origin main
bash scripts/deploy.sh              # Full deploy (backend + frontend)
bash scripts/deploy.sh --skip-frontend  # Backend only
```

---

## 📊 Monitoring & Dashboards

### Grafana (Dashboards)
- **URL**: http://your-server:3001
- **Login**: admin / (check `GRAFANA_ADMIN_PASSWORD` in .env)

**Available Dashboards:**
| Dashboard | What it shows |
|-----------|---------------|
| Node Exporter Full | Server CPU, RAM, disk, network |
| PostgreSQL Database | DB connections, queries, size |
| Redis Dashboard | Memory, commands, connections |

**To import more dashboards:**
1. Dashboards → New → Import
2. Enter ID from [grafana.com/dashboards](https://grafana.com/grafana/dashboards)
3. Select datasource → Import

### Loki (Log Search)
- **Access**: Grafana → Explore → Select "Loki" datasource

**Common queries:**
```logql
# All chat server logs
{compose_service=~"chat-server.*"}

# Errors only
{compose_service=~"chat-server.*"} |= "error"

# Rate limiting events
{compose_service=~"chat-server.*"} |= "RATE LIMIT"

# Specific user ID
{compose_service=~".*"} |= "user_abc123"
```

### Prometheus (Metrics)
- **URL**: http://your-server:9090
- Scrapes metrics from all services every 15s

---

## 🐛 Error Tracking (Sentry)

- **Dashboard**: https://sentry.io
- **DSN**: Check `web-new/src/main.tsx`

**What Sentry captures:**
- [x] JavaScript errors
- [x] React component crashes
- [x] Unhandled promise rejections
- [x] Performance data (10% sample)

**Test Sentry:**
Open browser console on production site:
```javascript
throw new Error("Test error");
```

---

## 💾 Database Backups

### Manual Backup
```bash
bash scripts/backup-db.sh              # Local backup
bash scripts/backup-db.sh --upload-s3  # Backup + upload to S3/MinIO
```

### Automatic Backups (Cron)
Add to crontab (`crontab -e`):
```cron
# Daily backup at 2 AM
0 2 * * * /opt/end2endsecure.com/scripts/backup-db.sh >> /var/log/silentrelay-backup.log 2>&1
```

### Restore from Backup
```bash
bash scripts/restore-db.sh /opt/end2endsecure.com/backups/silentrelay_db_YYYYMMDD_HHMMSS.sql.gz
```

### Backup Location
- **Local**: `/opt/end2endsecure.com/backups/`
- **Retention**: 30 days (auto-cleanup)

---

## Common Commands

### Service Management
```bash
# View all running services
docker compose ps

# View logs (real-time)
docker compose logs -f chat-server-1

# Restart a service
docker compose restart chat-server-1

# Rebuild and restart
docker compose up -d --build chat-server-1
```

### Health Checks
```bash
# Check all services are healthy
docker compose ps

# Check specific service
curl http://localhost:8081/health  # chat-server-1
curl http://localhost:8082/health  # chat-server-2
```

### Database Access
```bash
# Connect to PostgreSQL
docker exec -it $(docker ps -qf "name=postgres") psql -U messaging -d messaging

# Redis CLI
docker exec -it $(docker ps -qf "name=redis") redis-cli
```

---

## 🚨 Incident Response

### Service Down
1. Check `docker compose ps` for unhealthy services
2. Check logs: `docker compose logs --tail 100 <service-name>`
3. Restart: `docker compose restart <service-name>`
4. If persists: `docker compose up -d --build <service-name>`

### High CPU/Memory
1. Check Grafana Node Exporter dashboard
2. Find culprit: `docker stats`
3. Check logs of high-usage container
4. Consider scaling or restarting

### Database Issues
1. Check Grafana PostgreSQL dashboard
2. Check connections: `docker exec postgres psql -U messaging -c "SELECT count(*) FROM pg_stat_activity;"`
3. If corrupted: Restore from backup

### Can't SSH into Server
1. Try from different network
2. Check server provider dashboard (OVH, etc.)
3. Use console access if available

---

## 📍 Key URLs

| Service | URL |
|---------|-----|
| **App** | https://end2endsecure.com |
| **Grafana** | http://server-ip:3001 |
| **Prometheus** | http://server-ip:9090 |
| **Consul** | http://server-ip:8500 |
| **MinIO Console** | http://server-ip:9001 |
| **Sentry** | https://sentry.io |

---

## Secrets Location

| Secret | Location |
|--------|----------|
| Database password | `.env` → `POSTGRES_PASSWORD` |
| JWT secret | `.env` → `JWT_SECRET` |
| ClickSend API | `.env` → `CLICKSEND_*` |
| Grafana password | `.env` → `GRAFANA_ADMIN_PASSWORD` |
| MinIO credentials | `.env` → `MINIO_ROOT_*` |
| TURN secret | `.env` → `TURN_SECRET` |
| SSH deploy key | Server: `~/.ssh/deploy_key` |

---

## Useful Grafana Dashboard IDs

| Dashboard | ID | Description |
|-----------|----|-------------|
| Node Exporter Full | 1860 | System metrics |
| Docker Container | 11600 | Container stats |
| PostgreSQL | 9628 | Database metrics |
| Redis | 11835 | Redis stats |
| Loki Logs | 13639 | Log dashboard |
| Go Application | 10826 | Go runtime metrics |

Import: Dashboards → New → Import → Enter ID
