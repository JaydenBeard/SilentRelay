-- E2E Encrypted Messaging App Database Schema
-- Note: All message content is encrypted client-side. Server only stores ciphertext.

-- Create postgres superuser role
CREATE ROLE postgres SUPERUSER LOGIN PASSWORD '$POSTGRES_PASSWORD';

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- USERS TABLE
-- ============================================
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    phone_hash TEXT NOT NULL,                         -- SHA-256 hash for privacy-preserving contact discovery
    username VARCHAR(50),
    display_name VARCHAR(100),
    avatar_url TEXT,
    public_identity_key TEXT NOT NULL,                -- Ed25519 public key for identity
    public_signed_prekey TEXT NOT NULL,               -- Current signed pre-key
    signed_prekey_signature TEXT NOT NULL,            -- Signature of signed pre-key
    signed_prekey_id INTEGER NOT NULL DEFAULT 1,      -- Current signed pre-key ID
    signed_prekey_updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    safety_number TEXT,                               -- Computed safety number for verification
    totp_secret TEXT,                                 -- AES-256-GCM encrypted TOTP secret
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_users_phone ON users(phone_number);
CREATE INDEX idx_users_phone_hash ON users(phone_hash);
CREATE UNIQUE INDEX idx_users_username ON users(LOWER(username)) WHERE username IS NOT NULL;

-- ============================================
-- USER PIN (for login security)
-- Stored as Argon2 hash - never plaintext!
-- ============================================
CREATE TABLE user_pins (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    pin_hash TEXT NOT NULL,                           -- Argon2id hash of PIN
    pin_length INTEGER NOT NULL CHECK (pin_length IN (4, 6)),
    failed_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- RECOVERY KEYS (24-word mnemonic for backup)
-- The recovery key encrypts the user's master key
-- If lost, account accessible but chats are GONE
-- ============================================
CREATE TABLE recovery_keys (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    recovery_key_hash TEXT NOT NULL,                  -- Hash of recovery key (for verification)
    encrypted_master_key TEXT NOT NULL,               -- Master key encrypted with recovery key
    key_version INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE,
    reminder_shown_at TIMESTAMP WITH TIME ZONE        -- Track when we reminded user
);

-- ============================================
-- ENCRYPTED BACKUPS (chat history)
-- Encrypted with key derived from recovery key
-- ============================================
CREATE TABLE message_backups (
    backup_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    encrypted_data BYTEA NOT NULL,                    -- Encrypted chat backup
    backup_size BIGINT NOT NULL,
    message_count INTEGER NOT NULL,
    backup_version INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_backups_user ON message_backups(user_id, created_at DESC);

-- ============================================
-- PRIVACY SETTINGS
-- ============================================
CREATE TABLE privacy_settings (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    show_read_receipts BOOLEAN DEFAULT true,
    show_online_status BOOLEAN DEFAULT true,           -- If false, always appear offline (Ghost Mode)
    show_last_seen BOOLEAN DEFAULT true,
    show_typing_indicator BOOLEAN DEFAULT true,
    who_can_see_profile VARCHAR(20) DEFAULT 'everyone' CHECK (who_can_see_profile IN ('everyone', 'contacts', 'nobody')),
    who_can_add_to_groups VARCHAR(20) DEFAULT 'everyone' CHECK (who_can_add_to_groups IN ('everyone', 'contacts', 'nobody')),
    disappearing_messages_default INTEGER,            -- Default expiry in seconds (NULL = disabled)
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- ONE-TIME PRE-KEYS (for X3DH key exchange)
-- ============================================
CREATE TABLE prekeys (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    prekey_id INTEGER NOT NULL,
    public_key TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    used_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, prekey_id)
);

CREATE INDEX idx_prekeys_user ON prekeys(user_id) WHERE used_at IS NULL;
CREATE INDEX idx_prekeys_count ON prekeys(user_id, used_at) WHERE used_at IS NULL;

-- ============================================
-- KEY ROTATION HISTORY
-- Track when keys were rotated for security auditing
-- ============================================
CREATE TABLE key_rotations (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('identity', 'signed_prekey', 'prekeys_batch')),
    old_key_hash TEXT,                                -- Hash of old key for audit
    new_key_hash TEXT NOT NULL,                       -- Hash of new key
    rotated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    reason VARCHAR(50) DEFAULT 'scheduled'            -- scheduled, manual, security
);

CREATE INDEX idx_key_rotations_user ON key_rotations(user_id, rotated_at DESC);

-- ============================================
-- GROUPS TABLE
-- ============================================
CREATE TABLE groups (
    group_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar_url TEXT,
    group_key_encrypted TEXT,
    disappearing_messages INTEGER,                    -- Seconds until messages expire (NULL = disabled)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID NOT NULL REFERENCES users(user_id)
);

-- ============================================
-- GROUP MEMBERS TABLE
-- ============================================
CREATE TABLE group_members (
    group_id UUID NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    encrypted_group_key TEXT NOT NULL,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_user ON group_members(user_id);

-- ============================================
-- MESSAGES TABLE
-- Server only stores encrypted ciphertext!
-- ============================================
CREATE TABLE messages (
    message_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id UUID NOT NULL REFERENCES users(user_id),
    receiver_id UUID REFERENCES users(user_id),
    group_id UUID REFERENCES groups(group_id),
    
    -- Reply support
    reply_to_id UUID REFERENCES messages(message_id),
    
    -- Encrypted content (Signal Protocol ciphertext)
    ciphertext BYTEA NOT NULL,
    message_type VARCHAR(20) NOT NULL,
    
    -- Media reference
    media_id UUID,
    media_type VARCHAR(20),
    
    -- Metadata
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    server_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP WITH TIME ZONE,               -- Track if message was edited
    
    -- Delivery status
    status VARCHAR(20) DEFAULT 'sent' CHECK (status IN ('sent', 'delivered', 'read')),
    delivered_at TIMESTAMP WITH TIME ZONE,
    read_at TIMESTAMP WITH TIME ZONE,
    
    -- Soft delete
    is_deleted BOOLEAN DEFAULT false,
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_for_everyone BOOLEAN DEFAULT false,       -- Delete for all participants
    
    -- Ephemeral messages
    expires_at TIMESTAMP WITH TIME ZONE,
    
    CHECK (receiver_id IS NOT NULL OR group_id IS NOT NULL)
);

CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_receiver ON messages(receiver_id) WHERE receiver_id IS NOT NULL;
CREATE INDEX idx_messages_group ON messages(group_id) WHERE group_id IS NOT NULL;
CREATE INDEX idx_messages_timestamp ON messages(timestamp DESC);
CREATE INDEX idx_messages_unread ON messages(receiver_id, status) WHERE status != 'read';
CREATE INDEX idx_messages_expiry ON messages(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_messages_reply ON messages(reply_to_id) WHERE reply_to_id IS NOT NULL;

-- ============================================
-- MESSAGE REACTIONS
-- ============================================
CREATE TABLE message_reactions (
    message_id UUID NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    emoji VARCHAR(10) NOT NULL,                       -- Unicode emoji
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message ON message_reactions(message_id);

-- ============================================
-- MESSAGE INBOX (for efficient message retrieval)
-- ============================================
CREATE TABLE message_inbox (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL,
    conversation_type VARCHAR(10) NOT NULL CHECK (conversation_type IN ('direct', 'group')),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    is_sender BOOLEAN NOT NULL,
    PRIMARY KEY (user_id, message_id)
);

CREATE INDEX idx_inbox_user_conv ON message_inbox(user_id, conversation_id, timestamp DESC);
CREATE INDEX idx_inbox_timestamp ON message_inbox(user_id, timestamp DESC);

-- NOTE: Conversation state is stored CLIENT-SIDE only for security.
-- Multi-device sync happens via encrypted device-to-device transfer.
-- Server never sees conversation metadata (who talks to whom).

-- ============================================
-- USER CONNECTION REGISTRY
-- ============================================
CREATE TABLE user_connections (
    connection_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    server_id VARCHAR(50) NOT NULL,
    device_id UUID NOT NULL,
    device_type VARCHAR(20) CHECK (device_type IN ('mobile', 'tablet', 'desktop', 'web')),
    device_name VARCHAR(100),
    push_token TEXT,
    last_active TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    connected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_connections_user ON user_connections(user_id) WHERE is_active = true;
CREATE INDEX idx_connections_server ON user_connections(server_id) WHERE is_active = true;

-- ============================================
-- DEVICES (for multi-device support)
-- ============================================
CREATE TABLE devices (
    device_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_name VARCHAR(100),
    device_type VARCHAR(20) CHECK (device_type IN ('mobile', 'tablet', 'desktop', 'web')),
    public_device_key TEXT NOT NULL,
    is_primary BOOLEAN DEFAULT false,                 -- Primary device for key provisioning
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_devices_user ON devices(user_id) WHERE is_active = true;

-- ============================================
-- DEVICE APPROVAL REQUESTS (for secure device linking)
-- When a new device tries to login, existing devices must approve
-- ============================================
CREATE TABLE device_approval_requests (
    request_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    new_device_id UUID NOT NULL,
    new_device_name VARCHAR(100),
    new_device_type VARCHAR(20),
    approval_code VARCHAR(6) NOT NULL,                -- 6-digit code shown on existing device
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    approved_by_device_id UUID REFERENCES devices(device_id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '5 minutes'),
    responded_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_device_approval_user ON device_approval_requests(user_id, status) WHERE status = 'pending';
CREATE INDEX idx_device_approval_code ON device_approval_requests(user_id, approval_code) WHERE status = 'pending';

-- ============================================
-- ENCRYPTED USER SETTINGS (synced across devices)
-- Settings are encrypted client-side with user's key
-- ============================================
CREATE TABLE user_settings_sync (
    user_id UUID PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    encrypted_settings TEXT NOT NULL,               -- AES-256-GCM encrypted JSON
    settings_hash TEXT NOT NULL,                    -- For detecting changes
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by_device_id UUID REFERENCES devices(device_id)
);

-- ============================================
-- BLOCKED USERS
-- ============================================
CREATE TABLE blocked_users (
    blocker_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    blocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (blocker_id, blocked_id)
);

CREATE INDEX idx_blocked_blocker ON blocked_users(blocker_id);
CREATE INDEX idx_blocked_blocked ON blocked_users(blocked_id);

-- ============================================
-- MEDIA METADATA
-- ============================================
CREATE TABLE media (
    media_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    uploader_id UUID NOT NULL REFERENCES users(user_id),
    blob_key TEXT NOT NULL,
    encrypted_key TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    thumbnail_blob_key TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_media_uploader ON media(uploader_id);
CREATE INDEX idx_media_expiry ON media(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================
-- VERIFICATION CODES (for phone auth)
-- ============================================
CREATE TABLE verification_codes (
    id SERIAL PRIMARY KEY,
    phone_number VARCHAR(20) NOT NULL,
    code VARCHAR(6) NOT NULL,
    purpose VARCHAR(20) DEFAULT 'login' CHECK (purpose IN ('login', 'register', 'pin_reset')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    attempts INTEGER DEFAULT 0,
    verified BOOLEAN DEFAULT false
);

CREATE INDEX idx_verification_phone ON verification_codes(phone_number, expires_at);

-- ============================================
-- SESSIONS (JWT session tracking)
-- ============================================
CREATE TABLE sessions (
    session_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_id UUID REFERENCES devices(device_id),
    token_hash TEXT NOT NULL,
    pin_verified BOOLEAN DEFAULT false,               -- Track if PIN was verified this session
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    last_used TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user ON sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_token ON sessions(token_hash);

-- ============================================
-- CONTACTS (for privacy-preserving discovery)
-- Users upload hashes, we match without seeing numbers
-- ============================================
CREATE TABLE user_contacts (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    contact_hash TEXT NOT NULL,                       -- SHA-256 of normalized phone number
    matched_user_id UUID REFERENCES users(user_id),   -- NULL if no match yet
    nickname VARCHAR(100),                            -- User's custom name for contact
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, contact_hash)
);

CREATE INDEX idx_contacts_hash ON user_contacts(contact_hash);
CREATE INDEX idx_contacts_matched ON user_contacts(user_id, matched_user_id) WHERE matched_user_id IS NOT NULL;

-- ============================================
-- RATE LIMITING (persistent tracking)
-- ============================================
CREATE TABLE rate_limits (
    key TEXT PRIMARY KEY,                             -- e.g., "sms:+1234567890" or "api:user_id"
    count INTEGER DEFAULT 1,
    window_start TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    window_seconds INTEGER NOT NULL
);

CREATE INDEX idx_rate_limits_cleanup ON rate_limits(window_start);

-- ============================================
-- SAFETY NUMBER VERIFICATIONS
-- Track which contacts user has verified
-- ============================================
CREATE TABLE safety_verifications (
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    verified_safety_number TEXT NOT NULL,             -- The safety number that was verified
    verified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    verified_via VARCHAR(20) DEFAULT 'manual' CHECK (verified_via IN ('manual', 'qr_code', 'in_person')),
    PRIMARY KEY (user_id, contact_id)
);

-- ============================================
-- AUDIT LOG (security events)
-- ============================================
CREATE TABLE security_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(user_id),
    session_id UUID,                                  -- Optional session reference
    device_id UUID REFERENCES devices(device_id),     -- Optional device reference
    event_type VARCHAR(50) NOT NULL,                  -- login, pin_failed, key_rotated, etc.
    severity VARCHAR(10) DEFAULT 'info' CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    result VARCHAR(20) DEFAULT 'unknown',             -- success, failure, unknown
    resource VARCHAR(100),                            -- Resource being accessed/modified
    resource_id VARCHAR(100),                         -- ID of the resource
    resource_type VARCHAR(50),                        -- Type of resource (user, message, key, etc.)
    action VARCHAR(50),                               -- Specific action taken
    event_data JSONB,
    description TEXT,                                 -- Human-readable description
    ip_address INET,
    user_agent TEXT,
    request_id TEXT,                                  -- Request correlation ID
    request_path TEXT,                                -- API endpoint path
    request_method VARCHAR(10),                       -- HTTP method
    country VARCHAR(2),                               -- ISO country code
    region VARCHAR(100),                              -- State/province
    city VARCHAR(100),                                -- City
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,                              -- Request duration
    compliance_flags TEXT[],                          -- Compliance tags (GDPR, HIPAA, etc.)
    data_category VARCHAR(50)                         -- Data sensitivity level
);

CREATE INDEX idx_audit_user ON security_audit_log(user_id, created_at DESC);
CREATE INDEX idx_audit_type ON security_audit_log(event_type, created_at DESC);
CREATE INDEX idx_audit_session ON security_audit_log(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_audit_timestamp ON security_audit_log(timestamp DESC);
CREATE INDEX idx_audit_severity ON security_audit_log(severity, created_at DESC) WHERE severity IN ('high', 'critical');
CREATE INDEX idx_audit_resource ON security_audit_log(resource_type, resource_id) WHERE resource_id IS NOT NULL;

-- ============================================
-- SEALED SENDER CERTIFICATES
-- ============================================
CREATE TABLE sealed_sender_certificates (
    certificate_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    public_key BYTEA NOT NULL,
    expiration TIMESTAMP WITH TIME ZONE NOT NULL,
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL,
    certificate_data BYTEA NOT NULL,
    signature BYTEA NOT NULL
);

CREATE INDEX idx_sealed_sender_user ON sealed_sender_certificates(user_id);
CREATE INDEX idx_sealed_sender_expiry ON sealed_sender_certificates(expiration);

-- ============================================
-- KEY TRANSPARENCY LOG
-- Immutable log of all key changes (like CT)
-- Allows clients to verify server isn't lying
-- ============================================
CREATE TABLE key_transparency_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(user_id),
    device_id UUID REFERENCES devices(device_id),
    key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('identity', 'signed_prekey', 'one_time_prekey')),
    public_key BYTEA NOT NULL,
    key_hash TEXT NOT NULL,
    previous_hash TEXT NOT NULL,
    signature BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_kt_user ON key_transparency_log(user_id, created_at DESC);
CREATE INDEX idx_kt_hash ON key_transparency_log(key_hash);

-- ============================================
-- IP REPUTATION / THREAT INTEL
-- ============================================
CREATE TABLE ip_reputation (
    ip_address INET PRIMARY KEY,
    threat_score INTEGER DEFAULT 0,
    failed_attempts INTEGER DEFAULT 0,
    blocked_until TIMESTAMP WITH TIME ZONE,
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    notes TEXT
);

CREATE INDEX idx_ip_blocked ON ip_reputation(ip_address) WHERE blocked_until IS NOT NULL;

-- ============================================
-- SECURITY INCIDENTS
-- ============================================
CREATE TABLE security_incidents (
    incident_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    incident_type VARCHAR(50) NOT NULL,
    severity VARCHAR(10) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    affected_user_id UUID REFERENCES users(user_id),
    source_ip INET,
    description TEXT,
    evidence JSONB,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'investigating', 'resolved', 'false_positive')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by TEXT
);

CREATE INDEX idx_incidents_status ON security_incidents(status, created_at DESC);
CREATE INDEX idx_incidents_user ON security_incidents(affected_user_id);

-- ============================================
-- Functions and Triggers
-- ============================================

-- Update last_seen on user activity
CREATE OR REPLACE FUNCTION update_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE user_id = NEW.user_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_last_seen
    AFTER INSERT OR UPDATE ON user_connections
    FOR EACH ROW
    EXECUTE FUNCTION update_last_seen();

-- Generate phone hash on insert/update
CREATE OR REPLACE FUNCTION generate_phone_hash()
RETURNS TRIGGER AS $$
BEGIN
    NEW.phone_hash = encode(digest(NEW.phone_number, 'sha256'), 'hex');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_phone_hash
    BEFORE INSERT OR UPDATE OF phone_number ON users
    FOR EACH ROW
    EXECUTE FUNCTION generate_phone_hash();

-- Compute safety number from identity keys
CREATE OR REPLACE FUNCTION compute_safety_number()
RETURNS TRIGGER AS $$
BEGIN
    -- Safety number is first 60 digits of SHA-256 of sorted identity keys
    NEW.safety_number = substring(
        encode(digest(NEW.public_identity_key || NEW.phone_number, 'sha256'), 'hex'),
        1, 60
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_safety_number
    BEFORE INSERT OR UPDATE OF public_identity_key ON users
    FOR EACH ROW
    EXECUTE FUNCTION compute_safety_number();

-- Cleanup expired verification codes
CREATE OR REPLACE FUNCTION cleanup_expired_codes()
RETURNS void AS $$
BEGIN
    DELETE FROM verification_codes WHERE expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Cleanup expired messages (ephemeral/disappearing)
CREATE OR REPLACE FUNCTION cleanup_expired_messages()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM messages 
        WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
        RETURNING message_id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Cleanup expired media
CREATE OR REPLACE FUNCTION cleanup_expired_media()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM media 
        WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
        RETURNING media_id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Cleanup old rate limit entries
CREATE OR REPLACE FUNCTION cleanup_rate_limits()
RETURNS void AS $$
BEGIN
    DELETE FROM rate_limits 
    WHERE window_start + (window_seconds || ' seconds')::interval < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Check and update rate limit (returns true if allowed)
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_key TEXT,
    p_limit INTEGER,
    p_window_seconds INTEGER
)
RETURNS BOOLEAN AS $$
DECLARE
    v_count INTEGER;
    v_window_start TIMESTAMP WITH TIME ZONE;
BEGIN
    SELECT count, window_start INTO v_count, v_window_start
    FROM rate_limits WHERE key = p_key;
    
    IF NOT FOUND THEN
        INSERT INTO rate_limits (key, count, window_seconds)
        VALUES (p_key, 1, p_window_seconds);
        RETURN true;
    END IF;
    
    -- Check if window has expired
    IF v_window_start + (p_window_seconds || ' seconds')::interval < CURRENT_TIMESTAMP THEN
        UPDATE rate_limits 
        SET count = 1, window_start = CURRENT_TIMESTAMP
        WHERE key = p_key;
        RETURN true;
    END IF;
    
    -- Check if limit exceeded
    IF v_count >= p_limit THEN
        RETURN false;
    END IF;
    
    -- Increment counter
    UPDATE rate_limits SET count = count + 1 WHERE key = p_key;
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Get remaining prekey count for a user
CREATE OR REPLACE FUNCTION get_prekey_count(p_user_id UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM prekeys WHERE user_id = p_user_id AND used_at IS NULL);
END;
$$ LANGUAGE plpgsql;

-- Lock PIN after too many failed attempts
CREATE OR REPLACE FUNCTION handle_pin_attempt(
    p_user_id UUID,
    p_success BOOLEAN
)
RETURNS void AS $$
BEGIN
    IF p_success THEN
        UPDATE user_pins 
        SET failed_attempts = 0, locked_until = NULL
        WHERE user_id = p_user_id;
    ELSE
        UPDATE user_pins 
        SET failed_attempts = failed_attempts + 1,
            locked_until = CASE 
                WHEN failed_attempts >= 4 THEN CURRENT_TIMESTAMP + interval '1 hour'
                WHEN failed_attempts >= 2 THEN CURRENT_TIMESTAMP + interval '5 minutes'
                ELSE NULL
            END
        WHERE user_id = p_user_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

