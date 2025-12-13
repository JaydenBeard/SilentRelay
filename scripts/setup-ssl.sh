#!/bin/bash

# Let's Encrypt SSL Certificate Setup Script for SilentRelay
# This script generates and configures SSL certificates for silentrelay.com.au

set -e

# Check if running as root
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root" >&2
    exit 1
fi

# Check dependencies
if ! command -v certbot &> /dev/null; then
    echo "Certbot not found. Installing..."
    apt-get update && apt-get install -y certbot python3-certbot-nginx
fi

if ! command -v openssl &> /dev/null; then
    echo "OpenSSL not found. Installing..."
    apt-get update && apt-get install -y openssl
fi

# Parse arguments
DOMAIN=$1
EMAIL=$2
DNS_PLUGIN=${3:-manual}

if [ -z "$DOMAIN" ] || [ -z "$EMAIL" ]; then
    echo "Usage: $0 domain email [dns_plugin]"
    echo "Example: $0 silentrelay.com.au admin@silentrelay.com.au cloudflare"
    echo "Supported DNS plugins: manual, cloudflare, route53, digitalocean"
    exit 1
fi

# Create certificate directory if it doesn't exist
mkdir -p /etc/letsencrypt
mkdir -p infrastructure/certs

# Check if certificate already exists
if [ -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ] && [ -f "/etc/letsencrypt/live/$DOMAIN/privkey.pem" ]; then
    echo "Certificate already exists for $DOMAIN"
    echo "Copying existing certificate to infrastructure/certs/"
    cat "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "/etc/letsencrypt/live/$DOMAIN/privkey.pem" > infrastructure/certs/haproxy.pem
    cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" infrastructure/certs/fullchain.pem
    cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" infrastructure/certs/privkey.pem
    exit 0
fi

# Install DNS plugin if specified
if [ "$DNS_PLUGIN" != "manual" ]; then
    echo "Installing DNS plugin: $DNS_PLUGIN"
    apt-get update && apt-get install -y "python3-certbot-dns-$DNS_PLUGIN"
fi

# Obtain certificate using DNS validation
echo "Obtaining certificate for $DOMAIN using $DNS_PLUGIN DNS validation..."
if [ "$DNS_PLUGIN" = "manual" ]; then
    # Manual DNS validation
    certbot certonly --manual --preferred-challenges dns -d "$DOMAIN" -d "*.$DOMAIN" --email "$EMAIL" --agree-tos --manual-public-ip-logging-ok

    echo "Please add the following DNS TXT record for validation:"
    echo "After adding the record, press Enter to continue..."
    read -p "Press Enter after adding DNS record..."
else
    # Automated DNS validation
    certbot certonly --dns-$DNS_PLUGIN --dns-$DNS_PLUGIN-credentials /etc/letsencrypt/dns-credentials.ini -d "$DOMAIN" -d "*.$DOMAIN" --email "$EMAIL" --agree-tos --non-interactive
fi

# Combine certificate and private key for HAProxy
echo "Combining certificate and private key for HAProxy..."
cat "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "/etc/letsencrypt/live/$DOMAIN/privkey.pem" > infrastructure/certs/haproxy.pem

# Copy individual files for Nginx
echo "Copying certificate files for Nginx..."
cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" infrastructure/certs/fullchain.pem
cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" infrastructure/certs/privkey.pem

# Set proper permissions
echo "Setting secure permissions..."
chmod 600 infrastructure/certs/haproxy.pem
chmod 600 infrastructure/certs/privkey.pem
chmod 644 infrastructure/certs/fullchain.pem

# Create renewal hook
echo "Creating certificate renewal hook..."
mkdir -p /etc/letsencrypt/renewal-hooks/deploy
cat > /etc/letsencrypt/renewal-hooks/deploy/reload-haproxy.sh << 'EOF'
#!/bin/bash

# Combine renewed certificate and private key
cat "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "/etc/letsencrypt/live/$DOMAIN/privkey.pem" > infrastructure/certs/haproxy.pem
cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" infrastructure/certs/fullchain.pem
cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" infrastructure/certs/privkey.pem

# Set permissions
chmod 600 infrastructure/certs/haproxy.pem
chmod 600 infrastructure/certs/privkey.pem
chmod 644 infrastructure/certs/fullchain.pem

# Restart services to pick up new certificate
echo "Restarting services to apply new certificate..."
docker-compose restart loadbalancer frontend

echo "Certificate renewal complete!"
EOF

chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-haproxy.sh

# Test renewal
echo "Testing certificate renewal..."
certbot renew --dry-run

echo "SSL certificate setup complete!"
echo "Certificate files created in infrastructure/certs/"
echo "HAProxy will use: haproxy.pem"
echo "Nginx will use: fullchain.pem and privkey.pem"