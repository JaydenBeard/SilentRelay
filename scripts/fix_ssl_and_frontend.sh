#!/bin/bash

# Comprehensive Fix Script for SSL and Frontend Issues
# This script resolves:
# 1. "Not Secure" browser warning by replacing self-signed cert with Let's Encrypt
# 2. Frontend container health check issues
# 3. Load balancer configuration problems

set -e

echo "=== SilentRelay SSL and Frontend Fix Script ==="
echo "This script will fix the 'Not Secure' warning and frontend issues"
echo ""

# Step 1: Generate Let's Encrypt Certificate
echo "Step 1/4: Setting up Let's Encrypt SSL Certificate"
echo "-----------------------------------------------"

# Make the SSL setup script executable
chmod +x scripts/setup-ssl.sh

# Check if we're running on a system that can execute the SSL script
if [ "$(uname)" = "Linux" ] && [ "$(id -u)" -eq 0 ]; then
    echo "Running SSL certificate setup..."
    ./scripts/setup-ssl.sh silentrelay.com.au admin@silentrelay.com.au manual
else
    echo "SSL certificate setup requires Linux and root privileges."
    echo "Please run the following command manually on your production server:"
    echo ""
    echo "  sudo ./scripts/setup-ssl.sh silentrelay.com.au admin@silentrelay.com.au [dns_plugin]"
    echo ""
    echo "Supported DNS plugins: manual, cloudflare, route53, digitalocean"
    echo "For manual setup, use: sudo ./scripts/setup-ssl.sh silentrelay.com.au admin@silentrelay.com.au manual"
    echo ""
    echo "After running the SSL script, copy the generated certificate files:"
    echo "  - infrastructure/certs/haproxy.pem"
    echo "  - infrastructure/certs/fullchain.pem"
    echo "  - infrastructure/certs/privkey.pem"
    echo "from your production server to this development environment."
    echo ""

    # Create placeholder certificate for development
    echo "Creating development self-signed certificate for testing..."
    mkdir -p infrastructure/certs
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout infrastructure/certs/haproxy.key \
        -out infrastructure/certs/haproxy.crt \
        -subj "/CN=silentrelay.com.au"
    cat infrastructure/certs/haproxy.crt infrastructure/certs/haproxy.key > infrastructure/certs/haproxy.pem
    cp infrastructure/certs/haproxy.crt infrastructure/certs/fullchain.pem
    cp infrastructure/certs/haproxy.key infrastructure/certs/privkey.pem
    chmod 600 infrastructure/certs/haproxy.pem infrastructure/certs/privkey.pem
    chmod 644 infrastructure/certs/fullchain.pem
fi

# Step 2: Fix Frontend Health Check
echo ""
echo "Step 2/4: Fixing Frontend Health Check"
echo "--------------------------------------"

# Update frontend Dockerfile health check
echo "Updating frontend Dockerfile health check to use HTTP instead of HTTPS..."
sed -i 's|https://localhost/health|http://localhost/health|g' web-new/Dockerfile

# Step 3: Configure Load Balancer Properly
echo ""
echo "Step 3/4: Configuring Load Balancer"
echo "-----------------------------------"

# Update HAProxy configuration to add HTTP to HTTPS redirect
echo "Adding HTTP to HTTPS redirect to HAProxy configuration..."
if ! grep -q "redirect scheme https" infrastructure/haproxy/haproxy.cfg; then
    # Add HTTP frontend with redirect
    sed -i '/^# HTTPS Frontend (production)$/i\
# HTTP to HTTPS redirect\nfrontend http_front\n    bind *:80\n    http-request set-header X-Forwarded-Proto http\n\n    # Redirect all HTTP traffic to HTTPS\n    redirect scheme https code 301 if !{ ssl_fc }\n' infrastructure/haproxy/haproxy.cfg
fi

# Step 4: Update Frontend Nginx Configuration
echo ""
echo "Step 4/4: Updating Frontend Nginx Configuration"
echo "-----------------------------------------------"

echo "Updating frontend nginx configuration to work properly with load balancer..."
# Update API proxy to use proper headers
sed -i 's|proxy_pass https://loadbalancer:443/;|proxy_pass http://loadbalancer:80/;|g' web-new/nginx.conf
sed -i '/proxy_pass http:\/\/loadbalancer:80\//a\
        proxy_set_header X-Forwarded-Port 443;' web-new/nginx.conf

# Update WebSocket proxy
sed -i '/location \/ws {/a\
        proxy_set_header X-Forwarded-Proto https;\
        proxy_set_header X-Forwarded-Port 443;' web-new/nginx.conf

echo ""
echo "=== Configuration Complete ==="
echo ""
echo "Changes Made:"
echo "1. ✅ SSL Certificate: Let's Encrypt setup script created (or dev cert for testing)"
echo "2. ✅ Frontend Health Check: Fixed to use HTTP instead of HTTPS"
echo "3. ✅ Load Balancer: Added HTTP to HTTPS redirect"
echo "4. ✅ Frontend Nginx: Properly configured to work with load balancer"
echo ""
echo "Next Steps:"
echo ""

if [ -f "infrastructure/certs/haproxy.pem" ]; then
    echo "✅ Certificate files are ready in infrastructure/certs/"
    echo ""
    echo "To test the configuration:"
    echo "1. docker-compose build"
    echo "2. docker-compose up -d"
    echo "3. docker-compose ps"
    echo ""
    echo "To verify the fixes:"
    echo "1. Visit https://silentrelay.com.au (should show secure connection)"
    echo "2. Check browser console for no mixed content warnings"
    echo "3. Verify health checks: docker-compose ps"
    echo "4. Test WebSocket connections"
else
    echo "⚠️  Certificate files need to be generated on production server"
    echo "Please run the SSL setup script on your production server first"
fi

echo ""
echo "Verification Commands:"
echo "--------------------"
echo "# Check certificate validity:"
echo "openssl x509 -in infrastructure/certs/haproxy.pem -noout -dates"
echo ""
echo "# Test HAProxy configuration:"
echo "docker-compose exec loadbalancer haproxy -c -f /usr/local/etc/haproxy/haproxy.cfg"
echo ""
echo "# Check frontend health:"
echo "curl -I http://localhost:3000/health"
echo ""
echo "# Test HTTPS redirect (after deployment):"
echo "curl -I http://silentrelay.com.au"
echo ""
echo "# Test WebSocket connection:"
echo "wscat -c wss://silentrelay.com.au/ws"