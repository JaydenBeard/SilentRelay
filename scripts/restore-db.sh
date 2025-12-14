#!/bin/bash
#
# PostgreSQL Restore Script for SilentRelay
#
# Usage: ./scripts/restore-db.sh <backup_file>
#
# ⚠️  WARNING: This will OVERWRITE the current database!
#
# Example:
#   ./scripts/restore-db.sh /backups/silentrelay_db_20251211_020000.sql.gz
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
DB_USER="${POSTGRES_USER:-messaging}"
DB_NAME="${POSTGRES_DB:-messaging}"

# Auto-detect PostgreSQL container name (exclude postgres-exporter)
DB_CONTAINER=$(docker ps --format '{{.Names}}' | grep -E 'postgres' | grep -v 'exporter' | head -1)
if [ -z "$DB_CONTAINER" ]; then
    echo -e "${RED}❌ Error: No PostgreSQL container found running${NC}"
    echo "Available containers:"
    docker ps --format '{{.Names}}'
    exit 1
fi

# Check arguments
if [ -z "$1" ]; then
    echo -e "${RED}❌ Error: No backup file specified${NC}"
    echo ""
    echo "Usage: $0 <backup_file>"
    echo ""
    echo "Available backups:"
    ls -lh /opt/silentrelay/backups/silentrelay_db_*.sql.gz 2>/dev/null || echo "  No backups found in /opt/silentrelay/backups/"
    exit 1
fi

BACKUP_FILE="$1"

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo -e "${RED}❌ Error: Backup file not found: $BACKUP_FILE${NC}"
    exit 1
fi

echo -e "${BLUE}🔄 PostgreSQL Restore Script${NC}"
echo -e "${BLUE}================================${NC}"
echo "Timestamp: $(date)"
echo "Backup File: $BACKUP_FILE"
echo ""

# Verify backup integrity
echo -e "${YELLOW}⏳ Verifying backup integrity...${NC}"
if gzip -t "$BACKUP_FILE" 2>/dev/null; then
    echo -e "${GREEN}✅ Backup file is valid${NC}"
else
    echo -e "${RED}❌ Error: Backup file is corrupted${NC}"
    exit 1
fi

# Warning prompt
echo ""
echo -e "${RED}⚠️  WARNING: This will OVERWRITE the current database!${NC}"
echo -e "${RED}   All current data will be LOST!${NC}"
echo ""
read -p "Are you sure you want to continue? (type 'yes' to confirm): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo -e "${YELLOW}Restore cancelled.${NC}"
    exit 0
fi

# Stop services that use the database
echo ""
echo -e "${YELLOW}⏳ Stopping dependent services...${NC}"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Project root is parent of scripts directory
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

docker compose stop chat-server-1 chat-server-2 group-service presence-service scheduler queue-worker 2>/dev/null || true
echo -e "${GREEN}✅ Services stopped${NC}"

# Drop and recreate database
echo ""
echo -e "${YELLOW}⏳ Preparing database for restore...${NC}"
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME WITH (FORCE);" 2>/dev/null || true
docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;"
echo -e "${GREEN}✅ Database prepared${NC}"

# Restore from backup
echo ""
echo -e "${YELLOW}⏳ Restoring database from backup...${NC}"
gunzip -c "$BACKUP_FILE" | docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1
echo -e "${GREEN}✅ Database restored${NC}"

# Verify restore
echo ""
echo -e "${YELLOW}⏳ Verifying restore...${NC}"
TABLE_COUNT=$(docker exec "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
echo -e "${GREEN}✅ Restore verified: $TABLE_COUNT tables found${NC}"

# Restart services
echo ""
echo -e "${YELLOW}⏳ Restarting services...${NC}"
docker compose up -d chat-server-1 chat-server-2 group-service presence-service scheduler queue-worker
echo -e "${GREEN}✅ Services restarted${NC}"

# Summary
echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}✅ Restore Complete!${NC}"
echo "Restored from: $BACKUP_FILE"
echo "Tables restored: $TABLE_COUNT"
echo ""
echo -e "${YELLOW}⚠️  Please verify the application is working correctly.${NC}"
