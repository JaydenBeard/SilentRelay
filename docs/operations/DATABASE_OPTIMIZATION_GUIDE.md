# Database Optimization Guide

**Version**: 2025.12.04
**Last Updated**: 2025-12-04
**Status**: Active
**Owner**: Performance Engineering Team

## Table of Contents

1. [Database Performance Monitoring](#database-performance-monitoring)
2. [Query Optimization](#query-optimization)
3. [Index Management](#index-management)
4. [Connection Pooling](#connection-pooling)
5. [Database Maintenance](#database-maintenance)
6. [Performance Alerts](#performance-alerts)
7. [Best Practices](#best-practices)
8. [Cross-References](#cross-references)

## Database Performance Monitoring

### Query Performance Analysis

1. **Slow Query Identification**
   ```sql
   -- Find queries taking longer than 100ms
   SELECT query, calls, total_time, mean_time, rows
   FROM pg_stat_statements
   WHERE mean_time > 100
   ORDER BY mean_time DESC
   LIMIT 20;
   ```

2. **Table Scan Analysis**
   ```sql
   -- Identify tables with excessive sequential scans
   SELECT schemaname, relname, seq_scan, seq_tup_read, idx_scan, idx_tup_fetch
   FROM pg_stat_user_tables
   WHERE seq_scan > idx_scan * 2
   ORDER BY seq_scan DESC;
   ```

3. **Index Usage Statistics**
   ```sql
   -- Check index effectiveness
   SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
   FROM pg_stat_user_indexes
   ORDER BY idx_scan DESC;
   ```

### Connection Monitoring

```sql
-- Monitor active connections
SELECT datname, usename, client_addr, state, query_start, state_change
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY query_start;

-- Connection count by state
SELECT state, count(*) as connections
FROM pg_stat_activity
GROUP BY state;
```

## Query Optimization

### Query Analysis Techniques

1. **EXPLAIN Plan Analysis**
   ```sql
   -- Analyze query execution plan
   EXPLAIN ANALYZE
   SELECT m.* FROM messages m
   WHERE m.user_id = $1 AND m.created_at > $2
   ORDER BY m.created_at DESC
   LIMIT 50;
   ```

2. **Query Rewriting**
   ```sql
   -- Original inefficient query
   SELECT * FROM messages WHERE user_id IN (
       SELECT user_id FROM users WHERE username LIKE 'john%'
   );

   -- Optimized version
   SELECT m.* FROM messages m
   JOIN users u ON m.user_id = u.id
   WHERE u.username LIKE 'john%';
   ```

### Common Query Patterns

**Message Retrieval Optimization**:
```sql
-- Use composite indexes for common query patterns
CREATE INDEX idx_messages_user_created ON messages(user_id, created_at DESC);
CREATE INDEX idx_messages_conversation_created ON messages(conversation_id, created_at DESC);
```

**User Search Optimization**:
```sql
-- Full-text search for usernames and display names
CREATE INDEX idx_users_search ON users USING gin(to_tsvector('english', username || ' ' || display_name));
```

## Index Management

### Index Creation Strategy

1. **Primary Key Indexes**
   - Automatically created on all tables
   - Used for foreign key relationships

2. **Foreign Key Indexes**
   ```sql
   -- Ensure foreign keys are indexed
   CREATE INDEX idx_messages_sender_id ON messages(sender_id);
   CREATE INDEX idx_messages_recipient_id ON messages(recipient_id);
   ```

3. **Composite Indexes**
   ```sql
   -- For complex query patterns
   CREATE INDEX idx_messages_conversation_time ON messages(conversation_id, created_at DESC, message_type);
   ```

### Index Maintenance

```sql
-- Rebuild fragmented indexes
REINDEX INDEX CONCURRENTLY idx_messages_user_created;

-- Analyze table statistics
ANALYZE messages;

-- Vacuum for space reclamation
VACUUM ANALYZE messages;
```

### Index Monitoring

```sql
-- Find unused indexes
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
AND schemaname NOT IN ('pg_catalog', 'information_schema');

-- Index bloat detection
SELECT schemaname, tablename, n_dead_tup, n_live_tup
FROM pg_stat_user_tables
WHERE n_dead_tup > n_live_tup * 0.2;
```

## Connection Pooling

### PostgreSQL Connection Configuration

```sql
-- Optimal connection settings for high concurrency
ALTER SYSTEM SET max_connections = '200';
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET pg_stat_statements.max = '10000';
ALTER SYSTEM SET pg_stat_statements.track = 'all';
```

### Application Connection Pooling

**Go pgx Configuration**:
```go
// Optimal connection pool settings
db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}

db.Config().MaxConns = 25
db.Config().MinConns = 5
db.Config().MaxConnLifetime = 30 * time.Minute
db.Config().MaxConnIdleTime = 5 * time.Minute
```

### Connection Pool Monitoring

```sql
-- Monitor connection pool usage
SELECT count(*) as active_connections
FROM pg_stat_activity
WHERE state = 'active';

-- Connection age distribution
SELECT
    CASE
        WHEN backend_start > now() - interval '1 hour' THEN 'new'
        WHEN backend_start > now() - interval '1 day' THEN 'recent'
        ELSE 'old'
    END as age_group,
    count(*) as connections
FROM pg_stat_activity
GROUP BY age_group;
```

## Database Maintenance

### Automated Maintenance Tasks

1. **Daily VACUUM**
   ```sql
   -- Remove dead tuples and reclaim space
   VACUUM ANALYZE;
   ```

2. **Weekly REINDEX**
   ```sql
   -- Rebuild indexes to remove fragmentation
   REINDEX TABLE CONCURRENTLY messages;
   REINDEX TABLE CONCURRENTLY users;
   ```

3. **Monthly ANALYZE**
   ```sql
   -- Update query planner statistics
   ANALYZE;
   ```

### Maintenance Scheduling

```bash
# Cron job for daily maintenance
0 2 * * * /usr/bin/vacuumdb --analyze --verbose --dbname=messenger

# Weekly reindex (Sunday 3 AM)
0 3 * * 0 /usr/bin/reindexdb --concurrently --dbname=messenger
```

## Performance Alerts

### Database Alert Rules

```yaml
# High query latency
- alert: HighDatabaseQueryLatency
  expr: histogram_quantile(0.95, rate(pg_stat_activity_max_duration_seconds[5m])) > 5
  for: 10m
  labels:
    severity: critical

# Connection pool exhaustion
- alert: DatabaseConnectionPoolExhausted
  expr: pg_stat_activity_count / pg_settings_max_connections > 0.8
  for: 5m
  labels:
    severity: warning

# Table bloat warning
- alert: HighTableBloat
  expr: pg_table_bloat_ratio > 0.3
  for: 1h
  labels:
    severity: warning
```

### Monitoring Dashboard

**Key Metrics to Monitor**:
- Query response time (P95, P99)
- Connection pool utilization
- Index hit ratio
- Table bloat percentage
- Dead tuple count
- Autovacuum activity

## Best Practices

### Query Optimization Guidelines

1. **Use Appropriate Data Types**
   ```sql
   -- Use UUID for primary keys
   CREATE TABLE messages (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       -- other columns
   );

   -- Use TIMESTAMPTZ for timestamps
   created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
   ```

2. **Implement Pagination**
   ```sql
   -- Use keyset pagination for large result sets
   SELECT * FROM messages
   WHERE conversation_id = $1
     AND (created_at, id) < ($2, $3)
   ORDER BY created_at DESC, id DESC
   LIMIT 50;
   ```

3. **Avoid SELECT ***
   ```sql
   -- Specify only needed columns
   SELECT id, sender_id, content, created_at
   FROM messages
   WHERE conversation_id = $1;
   ```

### Indexing Strategy

1. **Covering Indexes**
   ```sql
   -- Include all columns needed by query
   CREATE INDEX idx_messages_conversation_covering
   ON messages(conversation_id, created_at DESC)
   INCLUDE (sender_id, message_type, content_length);
   ```

2. **Partial Indexes**
   ```sql
   -- Index only relevant rows
   CREATE INDEX idx_messages_unread
   ON messages(user_id, created_at DESC)
   WHERE status != 'read';
   ```

### Connection Management

1. **Connection Pool Sizing**
   - Max connections: 4x CPU cores
   - Min connections: 25% of max
   - Connection lifetime: 30 minutes

2. **Prepared Statements**
   ```go
   // Use prepared statements for repeated queries
   stmt, err := db.Prepare("SELECT * FROM messages WHERE user_id = $1")
   if err != nil {
       log.Fatal(err)
   }
   defer stmt.Close()
   ```

## Cross-References

### Related Documentation

- **[Performance Monitoring Guide](PERFORMANCE_MONITORING_GUIDE.md)** - Overall performance monitoring
- **[System Administration Guide](SYSTEM_ADMINISTRATION_GUIDE.md)** - Database setup and configuration
- **[Backup Strategy Guide](BACKUP_STRATEGY_GUIDE.md)** - Database backup procedures

### Configuration Files

- [`internal/db/postgres.go`](../internal/db/postgres.go) - Database connection configuration
- [`infrastructure/prometheus/performance-alerts.yml`](../infrastructure/prometheus/performance-alerts.yml) - Database monitoring alerts

### Tools and Scripts

- **pgBadger**: PostgreSQL log analyzer
- **pg_stat_statements**: Query performance statistics
- **auto_explain**: Automatic EXPLAIN plan logging

---

*© 2025 SilentRelay. All rights reserved.*