#!/bin/sh
set -e

# Fix TLS key ownership for PostgreSQL
if [ -f /tls/server.key ]; then
    # Copy key to data directory with correct permissions
    cp /tls/server.key /var/lib/postgresql/data/server.key
    chown postgres:postgres /var/lib/postgresql/data/server.key
    chmod 600 /var/lib/postgresql/data/server.key
    cp /tls/server.crt /var/lib/postgresql/data/server.crt
    chown postgres:postgres /var/lib/postgresql/data/server.crt
    cp /tls/ca.crt /var/lib/postgresql/data/ca.crt
    chown postgres:postgres /var/lib/postgresql/data/ca.crt
fi

# Drop privileges and run postgres with SSL
exec su-exec postgres postgres \
    -c ssl=on \
    -c ssl_cert_file=/var/lib/postgresql/data/server.crt \
    -c ssl_key_file=/var/lib/postgresql/data/server.key \
    -c ssl_ca_file=/var/lib/postgresql/data/ca.crt \
    "$@"
