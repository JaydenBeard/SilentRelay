-- ==============================================
-- Database Reset SQL
-- Run with: psql -U messaging -d messagingdb -f reset-database.sql
-- Or via Docker: docker compose exec -T postgres psql -U messaging messagingdb < scripts/reset-database.sql
-- ==============================================

-- WARNING: This will delete ALL data!

-- Disable triggers temporarily for faster truncation
SET session_replication_role = replica;

-- Truncate all data tables (preserving schema)
-- Order matters due to foreign key constraints, CASCADE handles it
TRUNCATE TABLE 
    security_incidents,
    ip_reputation,
    key_transparency_log,
    sealed_sender_certificates,
    security_audit_log,
    safety_verifications,
    rate_limits,
    user_contacts,
    blocked_users,
    sessions,
    verification_codes,
    media,
    user_settings_sync,
    device_approval_requests,
    devices,
    user_connections,
    message_reactions,
    message_inbox,
    messages,
    group_members,
    groups,
    key_rotations,
    prekeys,
    privacy_settings,
    message_backups,
    recovery_keys,
    user_pins,
    users
CASCADE;

-- Re-enable triggers
SET session_replication_role = DEFAULT;

-- Optional: Reset auto-increment sequences
ALTER SEQUENCE IF EXISTS verification_codes_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS prekeys_id_seq RESTART WITH 1;
ALTER SEQUENCE IF EXISTS key_rotations_id_seq RESTART WITH 1;

SELECT 'Database reset complete - all tables empty, schema preserved' as status;