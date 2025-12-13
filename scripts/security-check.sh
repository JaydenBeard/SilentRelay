#!/bin/bash

# ==============================================
# Security Audit Script
# Run before every deployment
# ==============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔐 Running Security Audit..."
echo "================================"

FAILED=0

# 1. Check for secrets in code
echo -e "\n${YELLOW}[1/8]${NC} Checking for hardcoded secrets..."
if grep -rn "password\s*=\s*['\"]" --include="*.go" --include="*.ts" --include="*.tsx" . 2>/dev/null | grep -v "test" | grep -v "_test.go"; then
    echo -e "${RED}✗ Found potential hardcoded passwords${NC}"
    FAILED=1
else
    echo -e "${GREEN}✓ No hardcoded passwords found${NC}"
fi

# 2. Check for TODO/FIXME security items
echo -e "\n${YELLOW}[2/8]${NC} Checking for security TODOs..."
if grep -rn "TODO.*security\|FIXME.*security\|XXX.*security" --include="*.go" --include="*.ts" . 2>/dev/null; then
    echo -e "${YELLOW}⚠ Found security-related TODOs${NC}"
else
    echo -e "${GREEN}✓ No security TODOs found${NC}"
fi

# 3. Check for unsafe crypto
echo -e "\n${YELLOW}[3/8]${NC} Checking for unsafe crypto usage..."
if grep -rn "math/rand\|md5\|sha1\|des\|rc4" --include="*.go" . 2>/dev/null | grep -v "_test.go" | grep -v "vendor"; then
    echo -e "${RED}✗ Found potentially unsafe crypto${NC}"
    FAILED=1
else
    echo -e "${GREEN}✓ No unsafe crypto found${NC}"
fi

# 4. Check for SQL injection vectors
echo -e "\n${YELLOW}[4/8]${NC} Checking for SQL injection vectors..."
if grep -rn 'fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE' --include="*.go" . 2>/dev/null; then
    echo -e "${RED}✗ Found potential SQL injection vectors${NC}"
    FAILED=1
else
    echo -e "${GREEN}✓ No SQL injection vectors found${NC}"
fi

# 5. Check Go dependencies for vulnerabilities
echo -e "\n${YELLOW}[5/8]${NC} Checking Go dependencies..."
if command -v govulncheck &> /dev/null; then
    if govulncheck ./... 2>/dev/null; then
        echo -e "${GREEN}✓ No known vulnerabilities in Go deps${NC}"
    else
        echo -e "${RED}✗ Vulnerabilities found in Go deps${NC}"
        FAILED=1
    fi
else
    echo -e "${YELLOW}⚠ govulncheck not installed, skipping${NC}"
fi

# 6. Check npm dependencies for vulnerabilities
echo -e "\n${YELLOW}[6/8]${NC} Checking npm dependencies..."
if [ -d "web" ]; then
    cd web
    if npm audit 2>/dev/null | grep -q "found 0 vulnerabilities"; then
        echo -e "${GREEN}✓ No known vulnerabilities in npm deps${NC}"
    else
        echo -e "${YELLOW}⚠ Check npm audit output for details${NC}"
    fi
    cd ..
fi

# 7. Check for exposed .env files
echo -e "\n${YELLOW}[7/8]${NC} Checking for exposed secrets files..."
if [ -f ".env" ] && ! grep -q "^\.env$" .gitignore 2>/dev/null; then
    echo -e "${RED}✗ .env file may not be gitignored${NC}"
    FAILED=1
else
    echo -e "${GREEN}✓ Secret files properly ignored${NC}"
fi

# 8. Check docker security
echo -e "\n${YELLOW}[8/8]${NC} Checking Docker security..."
if grep -q "USER" Dockerfile 2>/dev/null; then
    echo -e "${GREEN}✓ Dockerfile uses non-root user${NC}"
else
    echo -e "${YELLOW}⚠ Consider using non-root user in Dockerfile${NC}"
fi

# Summary
echo ""
echo "================================"
if [ $FAILED -eq 1 ]; then
    echo -e "${RED}✗ Security audit FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Security audit PASSED${NC}"
fi

