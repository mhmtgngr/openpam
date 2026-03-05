#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== OpenPAM Docker Setup ==="

# Generate JWT keys
JWT_DIR="jwt-keys"
if [ ! -f "$JWT_DIR/private.pem" ]; then
    echo "Generating JWT RSA keys..."
    mkdir -p "$JWT_DIR"
    openssl genrsa -out "$JWT_DIR/private.pem" 4096
    openssl rsa -in "$JWT_DIR/private.pem" -pubout -out "$JWT_DIR/public.pem"
    chmod 600 "$JWT_DIR/private.pem"
    chmod 644 "$JWT_DIR/public.pem"
    echo "JWT keys generated in $JWT_DIR/"
else
    echo "JWT keys already exist, skipping"
fi

# Generate Postgres TLS certs if server.key is missing
PG_TLS_DIR="postgres-tls"
if [ ! -f "$PG_TLS_DIR/server.key" ]; then
    echo "Generating PostgreSQL TLS certificates..."
    mkdir -p "$PG_TLS_DIR"

    # Generate CA key and cert
    openssl req -new -x509 -days 3650 -nodes \
        -out "$PG_TLS_DIR/ca.crt" \
        -keyout "$PG_TLS_DIR/ca.key" \
        -subj "/CN=OpenPAM-CA/O=OpenPAM/C=US"

    # Generate server key and CSR
    openssl req -new -nodes \
        -out "$PG_TLS_DIR/server.csr" \
        -keyout "$PG_TLS_DIR/server.key" \
        -subj "/CN=postgres/O=OpenPAM/C=US"

    # Create SAN extension file
    cat > "$PG_TLS_DIR/san.cnf" <<EOF
[v3_req]
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = postgres
DNS.3 = *.openpam-net
IP.1 = 127.0.0.1
IP.2 = 10.89.0.2
EOF

    # Sign the server cert with the CA
    openssl x509 -req -days 3650 \
        -in "$PG_TLS_DIR/server.csr" \
        -CA "$PG_TLS_DIR/ca.crt" \
        -CAkey "$PG_TLS_DIR/ca.key" \
        -CAcreateserial \
        -out "$PG_TLS_DIR/server.crt" \
        -extfile "$PG_TLS_DIR/san.cnf" \
        -extensions v3_req

    # Clean up temp files
    rm -f "$PG_TLS_DIR/server.csr" "$PG_TLS_DIR/san.cnf" "$PG_TLS_DIR/ca.key"

    chmod 600 "$PG_TLS_DIR/server.key"
    chmod 644 "$PG_TLS_DIR/server.crt" "$PG_TLS_DIR/ca.crt"
    echo "PostgreSQL TLS certificates generated in $PG_TLS_DIR/"
else
    echo "PostgreSQL TLS certs already exist, skipping"
fi

# Generate self-signed SSL certs for nginx (production profile)
SSL_DIR="ssl"
if [ ! -f "$SSL_DIR/server.crt" ]; then
    echo "Generating self-signed SSL certificates for nginx..."
    mkdir -p "$SSL_DIR"
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -out "$SSL_DIR/server.crt" \
        -keyout "$SSL_DIR/server.key" \
        -subj "/CN=localhost/O=OpenPAM/C=US"
    chmod 600 "$SSL_DIR/server.key"
    chmod 644 "$SSL_DIR/server.crt"
    echo "Nginx SSL certificates generated in $SSL_DIR/"
else
    echo "Nginx SSL certs already exist, skipping"
fi

# Create .env from example if not present
if [ ! -f .env ]; then
    if [ -f ../../.env.example ]; then
        cp ../../.env.example .env
        echo "Created .env from .env.example"
    fi
fi

echo ""
echo "=== Setup complete ==="
echo "Run 'make up' or 'docker compose up -d' to start OpenPAM"
