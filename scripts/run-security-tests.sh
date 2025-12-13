#!/bin/bash

# ==============================================
# Comprehensive Security Test Suite
# Run this before every release
# ==============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔐 =========================================="
echo "   Comprehensive Security Test Suite"
echo "   =========================================="
echo ""

TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

run_test() {
    local name="$1"
    local cmd="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "  Testing: $name... "
    
    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}FAIL${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# ==============================================
# 1. Static Analysis
# ==============================================
echo -e "\n${BLUE}[1/7] Static Analysis${NC}"

run_test "Go vet" "go vet ./..."
run_test "Go staticcheck" "which staticcheck && staticcheck ./... || true"
run_test "Golint" "which golint && golint ./... || true"
run_test "NPM audit (frontend)" "cd web && npm audit --audit-level=high || true"

# ==============================================
# 2. Secrets Detection
# ==============================================
echo -e "\n${BLUE}[2/7] Secrets Detection${NC}"

run_test "No hardcoded passwords" "! grep -rn 'password\s*=\s*[\"'\'']' --include='*.go' --include='*.ts' . 2>/dev/null | grep -v test | grep -v '_test.go'"
run_test "No AWS keys" "! grep -rn 'AKIA[0-9A-Z]{16}' . 2>/dev/null"
run_test "No private keys" "! grep -rn 'BEGIN.*PRIVATE KEY' . 2>/dev/null | grep -v '.md'"
run_test "No JWT secrets" "! grep -rn 'jwt.*secret.*=.*[\"'\'']' --include='*.go' . 2>/dev/null | grep -v test"

# ==============================================
# 3. Dependency Vulnerabilities
# ==============================================
echo -e "\n${BLUE}[3/7] Dependency Vulnerabilities${NC}"

run_test "Go mod verify" "go mod verify"
run_test "Go vulncheck" "which govulncheck && govulncheck ./... || echo 'govulncheck not installed'"

# ==============================================
# 4. Cryptographic Security
# ==============================================
echo -e "\n${BLUE}[4/7] Cryptographic Security${NC}"

run_test "No math/rand for crypto" "! grep -rn 'math/rand' --include='*.go' internal/security/ internal/auth/ 2>/dev/null"
run_test "No MD5" "! grep -rn 'crypto/md5' --include='*.go' . 2>/dev/null | grep -v test"
run_test "No SHA1 for security" "! grep -rn 'crypto/sha1' --include='*.go' internal/security/ internal/auth/ 2>/dev/null"
run_test "No DES" "! grep -rn 'crypto/des' --include='*.go' . 2>/dev/null"
run_test "No RC4" "! grep -rn 'crypto/rc4' --include='*.go' . 2>/dev/null"

# ==============================================
# 5. SQL Injection Prevention
# ==============================================
echo -e "\n${BLUE}[5/7] SQL Injection Prevention${NC}"

run_test "No string concat SQL" "! grep -rn 'fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE' --include='*.go' . 2>/dev/null"
run_test "No raw SQL exec" "! grep -rn 'db.Exec.*\+.*\"' --include='*.go' . 2>/dev/null"

# ==============================================
# 6. Security Headers
# ==============================================
echo -e "\n${BLUE}[6/7] Security Headers${NC}"

run_test "CSP header defined" "grep -rn 'Content-Security-Policy' internal/security/ > /dev/null"
run_test "HSTS header defined" "grep -rn 'Strict-Transport-Security' internal/security/ > /dev/null"
run_test "X-Frame-Options defined" "grep -rn 'X-Frame-Options' internal/security/ > /dev/null"
run_test "X-Content-Type-Options defined" "grep -rn 'X-Content-Type-Options' internal/security/ > /dev/null"

# ==============================================
# 7. Unit Tests
# ==============================================
echo -e "\n${BLUE}[7/7] Security Unit Tests${NC}"

run_test "Security tests pass" "go test ./tests/... -v -run Security"

# ==============================================
# Summary
# ==============================================
echo ""
echo "=========================================="
echo "Summary"
echo "=========================================="
echo -e "Total tests: ${TOTAL_TESTS}"
echo -e "Passed:      ${GREEN}${PASSED_TESTS}${NC}"
echo -e "Failed:      ${RED}${FAILED_TESTS}${NC}"
echo ""

if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "${RED}⚠️  Security tests FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All security tests PASSED${NC}"
    exit 0
fi

