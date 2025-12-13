#!/bin/bash

# Script to regenerate a self-signed certificate with Subject Alternative Names (SAN)
# for HAProxy use. The certificate includes SAN for silentrelay.com.au, www.silentrelay.com.au, and localhost.
# Valid for 365 days.

# Step 1: Create a temporary OpenSSL configuration file with SAN extensions
cat > temp_openssl.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
subjectAltName = DNS:silentrelay.com.au,DNS:www.silentrelay.com.au,DNS:localhost
EOF

# Step 2: Generate a private key
openssl genrsa -out key.pem 2048

# Step 3: Generate the self-signed certificate using the temporary config
openssl req -new -x509 -key key.pem -out cert.pem -days 365 -config temp_openssl.conf -subj "/CN=localhost"

# Step 4: Concatenate the certificate and private key into a single PEM file for HAProxy
cat cert.pem key.pem > haproxy.pem

# Step 5: Clean up temporary files
rm temp_openssl.conf cert.pem key.pem

echo "Certificate regeneration complete. The haproxy.pem file is ready for use."