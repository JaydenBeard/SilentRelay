#!/bin/bash
#
# PostgreSQL Backup Script for SilentRelay
#
# Usage: ./scripts/backup-db.sh [--upload-s3]
#
# This script creates a compressed backup of the PostgreSQL database.
# Backups are stored in /backups with automatic cleanup of old backups.
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/opt/end2endsecure.com/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DB_USER="${POSTGRES_USER:-messaging}"
DB_NAME="${POSTGRES_DB:-messaging}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="silentrelay_db_${TIMESTAMP}.sql.gz"

# Auto-detect PostgreSQL container name (exclude postgres-exporter)
DB_CONTAINER=$(docker ps --format '{{.Names}}' | grep -E 'postgres' | grep -v 'exporter' | head -1)
if [ -z "$DB_CONTAINER" ]; then
    echo -e "${RED}❌ Error: No PostgreSQL container found running${NC}"
    echo "Available containers:"
    docker ps --format '{{.Names}}'
    exit 1
fi

# Parse arguments
UPLOAD_S3=false
for arg in "$@"; do
    case $arg in
        --upload-s3)
            UPLOAD_S3=true
            ;;
    esac
done

echo -e "${BLUE}📦 PostgreSQL Backup Script${NC}"
echo -e "${BLUE}================================${NC}"
echo "Timestamp: $(date)"
echo "Backup Dir: $BACKUP_DIR"
echo "Container: $DB_CONTAINER"
echo ""

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Create backup
echo -e "${YELLOW}⏳ Creating backup...${NC}"
docker exec "$DB_CONTAINER" pg_dump -U "$DB_USER" "$DB_NAME" | gzip > "$BACKUP_DIR/$BACKUP_FILE"

# Verify backup was created
if [ -f "$BACKUP_DIR/$BACKUP_FILE" ]; then
    BACKUP_SIZE=$(du -h "$BACKUP_DIR/$BACKUP_FILE" | cut -f1)
    echo -e "${GREEN}✅ Backup created: $BACKUP_FILE ($BACKUP_SIZE)${NC}"
else
    echo -e "${RED}❌ Error: Backup file was not created${NC}"
    exit 1
fi

# Verify backup integrity (check it's valid gzip)
if gzip -t "$BACKUP_DIR/$BACKUP_FILE" 2>/dev/null; then
    echo -e "${GREEN}✅ Backup integrity verified${NC}"
else
    echo -e "${RED}❌ Error: Backup file is corrupted${NC}"
    exit 1
fi

# Upload to S3/MinIO if requested
if [ "$UPLOAD_S3" = true ]; then
    echo -e "${YELLOW}⏳ Uploading to object storage...${NC}"
    
    # Use MinIO client if available
    if command -v mc &> /dev/null; then
        mc cp "$BACKUP_DIR/$BACKUP_FILE" minio/backups/database/
        echo -e "${GREEN}✅ Uploaded to MinIO${NC}"
    elif command -v aws &> /dev/null; then
        aws s3 cp "$BACKUP_DIR/$BACKUP_FILE" "s3://${S3_BUCKET:-silentrelay-backups}/database/"
        echo -e "${GREEN}✅ Uploaded to S3${NC}"
    else
        echo -e "${YELLOW}⚠️  No S3/MinIO client found, skipping upload${NC}"
    fi
fi

# Cleanup old backups
echo -e "${YELLOW}🧹 Cleaning up backups older than $RETENTION_DAYS days...${NC}"
DELETED_COUNT=$(find "$BACKUP_DIR" -name "silentrelay_db_*.sql.gz" -mtime +$RETENTION_DAYS -delete -print | wc -l)
echo -e "${GREEN}✅ Deleted $DELETED_COUNT old backup(s)${NC}"

# Show current backups
echo ""
echo -e "${BLUE}📋 Current Backups:${NC}"
ls -lh "$BACKUP_DIR"/silentrelay_db_*.sql.gz 2>/dev/null | tail -10 || echo "No backups found"

# Summary
echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${GREEN}✅ Backup Complete!${NC}"
echo "File: $BACKUP_DIR/$BACKUP_FILE"
echo "Size: $BACKUP_SIZE"
echo "Retention: $RETENTION_DAYS days"
