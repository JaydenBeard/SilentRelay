# SSL Certificates

This directory contains SSL certificates for HTTPS. Certificates are generated using Let's Encrypt with DNS validation.

## Production Setup (Let's Encrypt with DNS Validation)

Use the automated script for certificate generation:

```bash
# Run the SSL setup script with DNS validation
sudo ./scripts/setup-ssl.sh silentrelay.com.au your@email.com [dns_plugin]

# Supported DNS plugins:
# - manual (default) - Manual DNS record addition
# - cloudflare - Cloudflare DNS API
# - route53 - AWS Route 53
# - digitalocean - DigitalOcean DNS API

# Example with Cloudflare:
sudo ./scripts/setup-ssl.sh silentrelay.com.au your@email.com cloudflare
```

### DNS Plugin Setup

For automated DNS validation, create `/etc/letsencrypt/dns-credentials.ini`:

**Cloudflare:**
```
dns_cloudflare_email = your-email@example.com
dns_cloudflare_api_key = your-api-key
```

**AWS Route 53:**
```
aws_access_key_id = YOUR_ACCESS_KEY_ID
aws_secret_access_key = YOUR_SECRET_ACCESS_KEY
```

**DigitalOcean:**
```
dns_digitalocean_token = your-token
```

## Required Files

- `haproxy.pem` - Combined certificate + private key for HAProxy
- `fullchain.pem` - Certificate chain for Nginx
- `privkey.pem` - Private key for Nginx

## Certificate Renewal

Certificates auto-renew when they have 30 days remaining. The renewal script automatically:

1. Renews certificates via Let's Encrypt
2. Updates certificate files in this directory
3. Restarts HAProxy and Nginx services

### Manual Renewal

```bash
# Check certificate status
sudo certbot certificates

# Renew certificates
sudo certbot renew

# Test renewal (dry run)
sudo certbot renew --dry-run
```

### Renewal Logs

Check renewal activity:
```bash
tail -f /var/log/letsencrypt-renewal.log
```

## Development (Self-signed)

For local development, generate a self-signed certificate:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout haproxy.key \
  -out haproxy.crt \
  -subj "/CN=localhost"

cat haproxy.crt haproxy.key > haproxy.pem
rm haproxy.crt haproxy.key
```

## Security Notes

- Certificates are stored with 600 permissions (owner read/write only)
- DNS validation is more secure than HTTP validation as it doesn't require port 80 access
- Auto-renewal ensures certificates never expire
- All certificate operations require root privileges

