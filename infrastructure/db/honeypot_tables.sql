-- ============================================
-- HONEYPOT & DECEPTION TABLES
-- ============================================

-- Track honeypot triggers
CREATE TABLE IF NOT EXISTS honeypot_hits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL,              -- endpoint, header, credential, file, database
    name VARCHAR(100) NOT NULL,             -- Which honeypot was triggered
    source_ip INET NOT NULL,
    user_agent TEXT,
    method VARCHAR(10),
    path TEXT,
    headers JSONB,
    body TEXT,
    extra JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_honeypot_ip ON honeypot_hits(source_ip);
CREATE INDEX idx_honeypot_type ON honeypot_hits(type, created_at DESC);
CREATE INDEX idx_honeypot_time ON honeypot_hits(created_at DESC);

-- Canary tokens - unique tracking tokens
CREATE TABLE IF NOT EXISTS canary_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token VARCHAR(64) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL,              -- url, email, file, db_record, document
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    triggered_at TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER DEFAULT 0
);

CREATE INDEX idx_canary_token ON canary_tokens(token);
CREATE INDEX idx_canary_triggered ON canary_tokens(triggered_at) WHERE triggered_at IS NOT NULL;

-- Mark users table to support honeypot users
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_honeypot BOOLEAN DEFAULT false;
CREATE INDEX IF NOT EXISTS idx_users_honeypot ON users(is_honeypot) WHERE is_honeypot = true;

-- ============================================
-- CHAOS ENGINEERING TABLES
-- ============================================

-- Track chaos experiments
CREATE TABLE IF NOT EXISTS chaos_experiments (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    duration_seconds INTEGER NOT NULL,
    probability DECIMAL(3,2) NOT NULL,
    active BOOLEAN DEFAULT false,
    started_at TIMESTAMP WITH TIME ZONE,
    ends_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Game day execution log
CREATE TABLE IF NOT EXISTS game_day_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(50) NOT NULL,
    scenario_name VARCHAR(100) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'running' CHECK (status IN ('running', 'completed', 'aborted', 'failed')),
    results JSONB,
    lessons_learned TEXT
);

CREATE INDEX idx_game_day_status ON game_day_runs(status, started_at DESC);

-- ============================================
-- ZERO TRUST TABLES
-- ============================================

-- Risk scores for sessions
CREATE TABLE IF NOT EXISTS session_risk_scores (
    session_id UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    score INTEGER NOT NULL CHECK (score >= 0 AND score <= 100),
    factors JSONB NOT NULL,                 -- What contributed to the score
    PRIMARY KEY (session_id, calculated_at)
);

CREATE INDEX idx_risk_session ON session_risk_scores(session_id, calculated_at DESC);
CREATE INDEX idx_risk_high ON session_risk_scores(score) WHERE score > 50;

-- Step-up authentication records
CREATE TABLE IF NOT EXISTS step_up_challenges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
    challenge_type VARCHAR(20) NOT NULL,    -- pin, biometric, security_key
    reason VARCHAR(100) NOT NULL,           -- Why step-up was required
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    success BOOLEAN
);

CREATE INDEX idx_stepup_session ON step_up_challenges(session_id, created_at DESC);

-- ============================================
-- SUPPLY CHAIN SECURITY TABLES
-- ============================================

-- Dependency inventory (SBOM)
CREATE TABLE IF NOT EXISTS dependency_inventory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    package_name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    package_type VARCHAR(20) NOT NULL,      -- go, npm, docker
    license VARCHAR(100),
    hash VARCHAR(128),
    purl VARCHAR(500),                      -- Package URL
    cpe VARCHAR(500),                       -- Common Platform Enumeration
    last_verified TIMESTAMP WITH TIME ZONE,
    UNIQUE(package_name, version, package_type)
);

CREATE INDEX idx_dep_package ON dependency_inventory(package_name);
CREATE INDEX idx_dep_type ON dependency_inventory(package_type);

-- Known vulnerabilities in dependencies
CREATE TABLE IF NOT EXISTS dependency_vulnerabilities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dependency_id UUID NOT NULL REFERENCES dependency_inventory(id) ON DELETE CASCADE,
    vuln_id VARCHAR(50) NOT NULL,           -- CVE-XXXX-XXXXX
    severity VARCHAR(10) NOT NULL,          -- critical, high, medium, low
    description TEXT,
    fixed_in_version VARCHAR(50),
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(dependency_id, vuln_id)
);

CREATE INDEX idx_vuln_severity ON dependency_vulnerabilities(severity, discovered_at DESC);
CREATE INDEX idx_vuln_unresolved ON dependency_vulnerabilities(dependency_id) WHERE resolved_at IS NULL;

-- Build attestations
CREATE TABLE IF NOT EXISTS build_attestations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    artifact_hash VARCHAR(128) NOT NULL,
    build_type VARCHAR(50) NOT NULL,
    builder_id VARCHAR(255) NOT NULL,
    slsa_level INTEGER NOT NULL CHECK (slsa_level >= 0 AND slsa_level <= 4),
    provenance JSONB NOT NULL,
    signature BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attestation_hash ON build_attestations(artifact_hash);
CREATE INDEX idx_attestation_slsa ON build_attestations(slsa_level);

-- ============================================
-- HELPER FUNCTIONS
-- ============================================

-- Get high-risk sessions
CREATE OR REPLACE FUNCTION get_high_risk_sessions(min_score INTEGER DEFAULT 50)
RETURNS TABLE (
    session_id UUID,
    user_id UUID,
    score INTEGER,
    factors JSONB,
    calculated_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT ON (srs.session_id)
        srs.session_id,
        s.user_id,
        srs.score,
        srs.factors,
        srs.calculated_at
    FROM session_risk_scores srs
    JOIN sessions s ON s.session_id = srs.session_id
    WHERE srs.score >= min_score
    AND s.is_revoked = false
    ORDER BY srs.session_id, srs.calculated_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Get honeypot activity summary
CREATE OR REPLACE FUNCTION get_honeypot_summary(hours INTEGER DEFAULT 24)
RETURNS TABLE (
    type VARCHAR,
    hit_count BIGINT,
    unique_ips BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        h.type,
        COUNT(*) as hit_count,
        COUNT(DISTINCT h.source_ip) as unique_ips
    FROM honeypot_hits h
    WHERE h.created_at > NOW() - (hours || ' hours')::interval
    GROUP BY h.type
    ORDER BY hit_count DESC;
END;
$$ LANGUAGE plpgsql;

-- Auto-block repeat honeypot offenders
CREATE OR REPLACE FUNCTION auto_block_honeypot_offenders()
RETURNS INTEGER AS $$
DECLARE
    blocked_count INTEGER := 0;
BEGIN
    -- Block IPs with 5+ honeypot hits in last hour
    INSERT INTO ip_reputation (ip_address, blocked_until, threat_score, notes)
    SELECT 
        source_ip,
        NOW() + INTERVAL '24 hours',
        100,
        'Auto-blocked: Repeated honeypot triggers'
    FROM honeypot_hits
    WHERE created_at > NOW() - INTERVAL '1 hour'
    GROUP BY source_ip
    HAVING COUNT(*) >= 5
    ON CONFLICT (ip_address) DO UPDATE SET
        blocked_until = GREATEST(ip_reputation.blocked_until, EXCLUDED.blocked_until),
        threat_score = LEAST(100, ip_reputation.threat_score + 25);
    
    GET DIAGNOSTICS blocked_count = ROW_COUNT;
    RETURN blocked_count;
END;
$$ LANGUAGE plpgsql;

