#!/bin/sh
# Generate TLS certificates for PostgreSQL local development
# These are self-signed certificates for Docker Compose local development only

set -e

TLS_DIR="$(dirname "$0")/postgres-tls"
mkdir -p "$TLS_DIR"

cd "$TLS_DIR"

# Generate CA private key
openssl genrsa -out ca.key 2048

# Generate CA certificate
openssl req -new -x509 -days 365 -key ca.key -out ca.crt \
  -subj "/CN=PostgreSQL-Development-CA/O=OpenPAM/C=US"

# Generate server private key
openssl genrsa -out server.key 2048

# Generate server certificate signing request
openssl req -new -key server.key -out server.csr \
  -subj "/CN=localhost/O=OpenPAM/C=US"

# Create SAN extension file for PostgreSQL
cat > server.ext <<-EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = postgres
DNS.3 = *.openpam-net
IP.1 = 127.0.0.1
EOF

# Sign server certificate with CA
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 365 -extfile server.ext

# Set proper permissions
chmod 600 server.key ca.key
chmod 644 server.crt ca.crt

# Clean up CSR and extension file
rm -f server.csr server.ext

echo "TLS certificates generated successfully in $TLS_DIR"
echo "Files created:"
echo "  - ca.crt  (CA certificate - trusted by clients)"
echo "  - ca.key  (CA private key)"
echo "  - server.crt  (Server certificate)"
echo "  - server.key  (Server private key)"
