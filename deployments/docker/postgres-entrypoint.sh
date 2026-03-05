#!/bin/bash
set -e

# OpenPAM PostgreSQL entrypoint
# Wraps the official postgres entrypoint to enable SSL/TLS

# If TLS certs are provided, configure SSL
if [ -f /tls/server.key ] && [ -f /tls/server.crt ]; then
    # Copy certs to a location postgres can read
    mkdir -p /var/lib/postgresql/tls
    cp /tls/server.key /var/lib/postgresql/tls/server.key
    cp /tls/server.crt /var/lib/postgresql/tls/server.crt
    if [ -f /tls/ca.crt ]; then
        cp /tls/ca.crt /var/lib/postgresql/tls/ca.crt
    fi
    chown -R postgres:postgres /var/lib/postgresql/tls
    chmod 600 /var/lib/postgresql/tls/server.key
    chmod 644 /var/lib/postgresql/tls/server.crt /var/lib/postgresql/tls/ca.crt 2>/dev/null || true

    echo "TLS certificates configured for PostgreSQL"

    # Delegate to the official postgres entrypoint with SSL args
    exec docker-entrypoint.sh postgres \
        -c ssl=on \
        -c ssl_cert_file=/var/lib/postgresql/tls/server.crt \
        -c ssl_key_file=/var/lib/postgresql/tls/server.key \
        "$@"
else
    echo "No TLS certificates found, starting PostgreSQL without SSL"
    exec docker-entrypoint.sh postgres "$@"
fi
