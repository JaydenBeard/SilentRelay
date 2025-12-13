#!/bin/bash

# ==============================================
# Database Reset Script
# Wipes all data while preserving schema structure
# ==============================================

set -e

echo "⚠️  WARNING: This will permanently delete ALL data from the database!"
echo "   - All user accounts"
echo "   - All messages and chats"
echo "   - All encryption keys"
echo "   - All sessions and devices"
echo ""

read -p "Are you sure you want to continue? (type 'yes' to confirm): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 1
fi

echo ""
echo "🗑️  Resetting database..."

# Connect to the database and truncate all tables
# Uses CASCADE to handle foreign key dependencies
docker compose exec -T postgres psql -U messaging messagingdb << 'EOF'
-- Disable triggers temporarily for faster truncation
SET session_replication_role = replica;

-- Truncate all data tables (preserving schema)
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

-- Reset sequences (optional, for cleaner IDs)
-- ALTER SEQUENCE verification_codes_id_seq RESTART WITH 1;
-- ALTER SEQUENCE prekeys_id_seq RESTART WITH 1;
-- ALTER SEQUENCE security_audit_log_id_seq RESTART WITH 1;
-- ALTER SEQUENCE key_rotations_id_seq RESTART WITH 1;

SELECT 'Database reset complete!' as status;
EOF

echo ""
echo "✅ Database has been reset!"
echo "   All tables are now empty but schema is preserved."
echo ""
echo "To restart the application:"
echo "   docker compose restart chat-server-1 chat-server-2"
echo ""