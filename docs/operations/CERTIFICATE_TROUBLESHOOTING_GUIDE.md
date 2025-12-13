# HAProxy SSL Certificate Troubleshooting Guide

## 1. Expected Certificate File Paths

HAProxy expects SSL certificates at these specific locations:

### Primary Configuration (`haproxy.cfg`)
- **Certificate Path**: `/etc/ssl/certs/haproxy.pem`
- **Usage**: Combined certificate + private key file
- **Line Reference**: Line 32 in `infrastructure/haproxy/haproxy.cfg`

### Enhanced SSL Configuration (`haproxy-ssl.cfg`)
- **Certificate Path**: `/etc/ssl/certs/server.pem`
- **DH Parameters**: `/etc/ssl/certs/dh-param.pem`
- **Usage**: Enhanced security configuration with OCSP stapling
- **Line References**: Line 34 (certificate), Line 11 (DH params)

## 2. Troubleshooting Commands

### Check Certificate Files
```bash
# Check if certificate files exist
ls -la /etc/ssl/certs/haproxy.pem
ls -la /etc/ssl/certs/server.pem
ls -la /etc/ssl/certs/dh-param.pem

# Check certificate validity and details
openssl x509 -in /etc/ssl/certs/haproxy.pem -text -noout
openssl x509 -in /etc/ssl/certs/server.pem -text -noout

# Check private key
openssl rsa -in /etc/ssl/certs/haproxy.pem -check -noout
openssl rsa -in /etc/ssl/certs/server.pem -check -noout

# Verify certificate chain
openssl verify -CAfile /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/haproxy.pem
```

### Check HAProxy Configuration
```bash
# Test HAProxy configuration syntax
haproxy -c -f /etc/haproxy/haproxy.cfg

# Check HAProxy logs for SSL errors
journalctl -u haproxy -n 50 --no-pager
tail -f /var/log/haproxy.log

# Check if HAProxy can read certificate files
sudo -u haproxy cat /etc/ssl/certs/haproxy.pem
```

### Check Certificate Permissions
```bash
# Check file permissions
ls -la /etc/ssl/certs/haproxy.pem
ls -la /etc/ssl/certs/server.pem

# Fix permissions if needed
sudo chmod 600 /etc/ssl/certs/haproxy.pem
sudo chmod 600 /etc/ssl/certs/server.pem
sudo chown haproxy:haproxy /etc/ssl/certs/haproxy.pem
sudo chown haproxy:haproxy /etc/ssl/certs/server.pem
```

## 3. Step-by-Step Fix Instructions

### Step 1: Identify the Issue
```bash
# Check which configuration is active
ps aux | grep haproxy
cat /etc/haproxy/haproxy.cfg | grep "bind.*443"

# Check certificate expiration
openssl x509 -enddate -noout -in /etc/ssl/certs/haproxy.pem
```

### Step 2: Standardize Certificate Paths
```bash
# Create symlink to standardize paths (choose one approach)
sudo ln -s /etc/ssl/certs/haproxy.pem /etc/ssl/certs/server.pem
# OR
sudo ln -s /etc/ssl/certs/server.pem /etc/ssl/certs/haproxy.pem
```

### Step 3: Generate Missing Certificates
```bash
# For development (self-signed)
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/ssl/certs/haproxy.key \
  -out /etc/ssl/certs/haproxy.crt \
  -subj "/CN=yourdomain.com"

sudo cat /etc/ssl/certs/haproxy.crt /etc/ssl/certs/haproxy.key > /etc/ssl/certs/haproxy.pem
sudo rm /etc/ssl/certs/haproxy.crt /etc/ssl/certs/haproxy.key

# For production (Let's Encrypt)
sudo ./scripts/setup-ssl.sh yourdomain.com your@email.com cloudflare
```

### Step 4: Generate DH Parameters
```bash
# Generate 2048-bit DH parameters
sudo openssl dhparam -out /etc/ssl/certs/dh-param.pem 2048
```

### Step 5: Update HAProxy Configuration
```bash
# Edit the configuration to use consistent paths
sudo sed -i 's|/etc/ssl/certs/server.pem|/etc/ssl/certs/haproxy.pem|g' /etc/haproxy/haproxy-ssl.cfg
```

### Step 6: Test and Reload
```bash
# Test configuration
sudo haproxy -c -f /etc/haproxy/haproxy.cfg

# Reload HAProxy
sudo systemctl reload haproxy
```

## 4. Verification Commands

### Verify Certificate Installation
```bash
# Check certificate is loaded
echo | openssl s_client -connect localhost:443 -servername yourdomain.com 2>/dev/null | openssl x509 -noout -dates

# Check TLS version and cipher
echo | openssl s_client -connect localhost:443 -servername yourdomain.com -tls1_3 2>/dev/null | grep -i "protocol"

# Check certificate chain
echo | openssl s_client -connect localhost:443 -servername yourdomain.com -showcerts 2>/dev/null
```

### Verify HAProxy Functionality
```bash
# Check HAProxy stats
curl -s http://localhost:8404/stats | grep -i "status"

# Check backend health
curl -s https://localhost/health

# Check WebSocket connectivity
wscat -c "wss://localhost/ws" -x '{"test":"websocket"}'
```

### Verify Security Headers
```bash
# Check security headers
curl -I https://localhost 2>/dev/null | grep -i "strict-transport"

# Check TLS configuration
curl -v https://localhost 2>&1 | grep -i "tls"
```

## 5. Common Issues and Solutions

### Issue: Certificate file not found
**Solution**: Ensure certificates exist at expected paths and have correct permissions

### Issue: Permission denied errors
**Solution**: Run `sudo chown haproxy:haproxy /etc/ssl/certs/haproxy.pem` and `sudo chmod 600 /etc/ssl/certs/haproxy.pem`

### Issue: Certificate expired
**Solution**: Renew with `sudo certbot renew` or generate new self-signed certificate

### Issue: Weak cipher or protocol
**Solution**: Update HAProxy configuration to enforce TLSv1.3 only

### Issue: Mixed certificate paths
**Solution**: Standardize paths by creating symlinks or updating configuration

## 6. Final Verification Checklist

- [ ] Certificate files exist at `/etc/ssl/certs/haproxy.pem`
- [ ] Certificate has correct permissions (600) and ownership (haproxy:haproxy)
- [ ] Certificate is valid and not expired
- [ ] DH parameters file exists at `/etc/ssl/certs/dh-param.pem`
- [ ] HAProxy configuration syntax is valid
- [ ] HAProxy service is running without SSL errors
- [ ] HTTPS connections work on port 443
- [ ] Security headers are properly set
- [ ] WebSocket connections work over wss://
- [ ] All backend services are healthy