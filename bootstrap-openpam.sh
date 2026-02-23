#!/bin/bash
# ═══════════════════════════════════════════════════════════════
# OpenPAM — Privileged Access Management Platform
# Bootstrap: creates full project structure + starts AI team
# ═══════════════════════════════════════════════════════════════

set -euo pipefail

PROJECT_DIR="${1:-$HOME/openpam}"
G='\033[0;32m'; Y='\033[1;33m'; NC='\033[0m'
log() { echo -e "${G}[+]${NC} $1"; }
warn() { echo -e "${Y}[!]${NC} $1"; }

log "Creating OpenPAM at: $PROJECT_DIR"
mkdir -p "$PROJECT_DIR"
cd "$PROJECT_DIR"

# ═══════════════════════════════════════
# Directory Structure
# ═══════════════════════════════════════

log "Creating directory structure..."

# Backend - Go services
mkdir -p cmd/{gateway,session-service,vault-service,credential-service,audit-service,admin-service,discovery-service}
mkdir -p internal/{gateway,session,vault,credential,audit,admin,discovery}
mkdir -p internal/{auth,middleware,config,database,cache,crypto,events,health,metrics,api}
mkdir -p internal/models
mkdir -p internal/pam/{checkout,recording,rotation,approval,policy,analytics}
mkdir -p pkg/{logger,errors,pagination,validator,httpclient}
mkdir -p migrations/{gateway,session,vault,credential,audit,admin}
mkdir -p scripts/{setup,migration,seed,tools}
mkdir -p configs/{development,staging,production}
mkdir -p deployments/docker
mkdir -p deployments/kubernetes/{base,overlays/{dev,staging,prod}}
mkdir -p api/openapi
mkdir -p docs/{architecture,runbooks,api}
mkdir -p test/{integration,e2e,fixtures,mocks}

# Frontend - React/TypeScript
mkdir -p frontend/src/{api,components/{common,layout,auth,sessions,vault,credentials,audit,admin,dashboard,pam},contexts,hooks,pages,types,utils,styles}
mkdir -p frontend/src/components/pam/{checkout,recording,approval,rotation}
mkdir -p frontend/e2e/{pages,fixtures}
mkdir -p frontend/public

log "✓ 80+ directories created"

# ═══════════════════════════════════════
# Go Module + Dependencies
# ═══════════════════════════════════════

log "Initializing Go module..."

cat > go.mod << 'EOF'
module github.com/openpam/openpam

go 1.22.0

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/redis/go-redis/v9 v9.4.0
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/rs/zerolog v1.31.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.8.4
	github.com/prometheus/client_golang v1.18.0
	golang.org/x/crypto v0.18.0
	github.com/gorilla/websocket v1.5.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/go-webauthn/webauthn v0.10.0
	github.com/hashicorp/vault/api v1.11.0
	github.com/aws/aws-sdk-go-v2 v1.24.0
	github.com/pion/webrtc/v3 v3.2.24
)
EOF

cat > go.sum << 'EOF'
// go mod tidy will populate this
EOF

# ═══════════════════════════════════════
# CLAUDE.md — AI Team Instructions
# ═══════════════════════════════════════

log "Creating CLAUDE.md..."

cat > CLAUDE.md << 'CLAUDEMD'
# OpenPAM — Privileged Access Management Platform

## Project Overview
OpenPAM is a Zero Trust Privileged Access Management (PAM) platform providing secure access to critical systems, credential vaulting, session recording, and compliance reporting.

## Architecture

### Services (Go/Gin, ports 8500-8506)
| Service | Port | Package | Responsibility |
|---------|------|---------|----------------|
| gateway | 8500 | cmd/gateway | API gateway, routing, auth, rate limiting |
| session-service | 8501 | cmd/session-service | Session management, recording, proxying |
| vault-service | 8502 | cmd/vault-service | Credential vault, encryption, secrets |
| credential-service | 8503 | cmd/credential-service | Credential checkout, rotation, discovery |
| audit-service | 8504 | cmd/audit-service | Audit logging, compliance, reporting |
| admin-service | 8505 | cmd/admin-service | Dashboard, config, tenant management |
| discovery-service | 8506 | cmd/discovery-service | Asset discovery, network scanning |

### Frontend (React 18 + TypeScript + Tailwind)
- Port: 3000 (dev) / 8580 (production nginx)
- State management: React Context + hooks
- Routing: react-router-dom v6

### Infrastructure
- PostgreSQL 16 — primary datastore (per-service schemas)
- Redis 7 — sessions, caching, rate limiting, pub/sub
- MinIO/S3 — session recordings storage
- Docker/Podman — containerization

## Code Standards

### Go
- Use interfaces for all service dependencies (testability)
- Dependency injection via constructors: `NewService(repo Repository, cache Cache) *Service`
- Errors: wrap with context `fmt.Errorf("service.Method: %w", err)`
- Logging: zerolog with structured fields (correlation_id, user_id, tenant_id)
- All handlers: validate input → check auth → business logic → structured response
- SQL: use sqlx with named queries, no raw string concatenation
- Tests: table-driven with testify, mock interfaces

### TypeScript/React
- Strict mode, no `any` type
- Functional components with hooks
- API client functions in src/api/
- Loading/error/empty states on every data component
- Tailwind CSS, responsive design

### API
- RESTful, JSON, versioned: /api/v1/
- Auth: Bearer JWT in Authorization header
- Pagination: cursor-based (after_id, limit)
- Errors: {"error": {"code": "NOT_FOUND", "message": "...", "details": {}}}
- Correlation: X-Request-ID on all requests/responses

### Security (CRITICAL — this is a PAM platform)
- All secrets encrypted at rest with AES-256-GCM
- Credential vault uses envelope encryption (master key → data encryption keys)
- Session recordings encrypted before storage
- All privileged actions require MFA verification
- Zero standing privileges — all access is just-in-time
- Audit every access, every action, no exceptions
- Rate limit everything, especially auth endpoints

### Database
- Migrations in migrations/{service}/ numbered sequentially
- Soft deletes (deleted_at timestamp)
- tenant_id on every table for multi-tenancy
- Indexes on all foreign keys and query columns
- JSONB for flexible metadata fields

### Docker
- Dockerfiles in deployments/docker/Dockerfile.{service}
- Multi-stage: golang:1.22-alpine → alpine:3.19
- Non-root user, minimal image
- Health check on /health endpoint
- docker-compose.yml with all services + PostgreSQL + Redis + MinIO
CLAUDEMD

# ═══════════════════════════════════════
# Service Entry Points
# ═══════════════════════════════════════

log "Creating service entry points..."

for svc in gateway session-service vault-service credential-service audit-service admin-service discovery-service; do
  pkg=$(echo "$svc" | tr '-' '_' | sed 's/_service//')
  port_map="gateway:8500 session:8501 vault:8502 credential:8503 audit:8504 admin:8505 discovery:8506"
  port=$(echo "$port_map" | tr ' ' '\n' | grep "^${pkg}:" | cut -d: -f2)
  [ -z "$port" ] && port="8500"

  cat > "cmd/${svc}/main.go" << GOEOF
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Router
	r := gin.New()
	r.Use(gin.Recovery())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up", "service": "${svc}"})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	// TODO: Wire routes, middleware, dependencies

	// Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "${port}"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Info().Str("service", "${svc}").Str("port", port).Msg("Starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Forced shutdown")
	}
	fmt.Println("Goodbye.")
}
GOEOF
done

# ═══════════════════════════════════════
# Core Internal Packages (Stubs)
# ═══════════════════════════════════════

log "Creating internal packages..."

# Models
cat > internal/models/models.go << 'EOF'
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a PAM platform user
type User struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	FirstName    string     `db:"first_name" json:"first_name"`
	LastName     string     `db:"last_name" json:"last_name"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Role         string     `db:"role" json:"role"` // super_admin, admin, operator, auditor, user
	Status       string     `db:"status" json:"status"` // active, suspended, locked
	MFAEnabled   bool       `db:"mfa_enabled" json:"mfa_enabled"`
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// Credential represents a stored privileged credential
type Credential struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	Name            string     `db:"name" json:"name"`
	Type            string     `db:"type" json:"type"` // ssh_key, password, api_key, certificate, database
	Host            string     `db:"host" json:"host"`
	Port            int        `db:"port" json:"port"`
	Username        string     `db:"username" json:"username"`
	EncryptedSecret []byte     `db:"encrypted_secret" json:"-"`
	RotationPolicy  string     `db:"rotation_policy" json:"rotation_policy"` // manual, daily, weekly, on_checkin
	LastRotatedAt   *time.Time `db:"last_rotated_at" json:"last_rotated_at"`
	FolderID        *uuid.UUID `db:"folder_id" json:"folder_id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	CreatedBy       uuid.UUID  `db:"created_by" json:"created_by"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// Session represents a privileged session (SSH, RDP, DB, etc.)
type Session struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	UserID       uuid.UUID  `db:"user_id" json:"user_id"`
	CredentialID uuid.UUID  `db:"credential_id" json:"credential_id"`
	Type         string     `db:"type" json:"type"` // ssh, rdp, database, kubernetes, web
	Status       string     `db:"status" json:"status"` // active, ended, terminated, failed
	TargetHost   string     `db:"target_host" json:"target_host"`
	TargetPort   int        `db:"target_port" json:"target_port"`
	RecordingURL string     `db:"recording_url" json:"recording_url"`
	StartedAt    time.Time  `db:"started_at" json:"started_at"`
	EndedAt      *time.Time `db:"ended_at" json:"ended_at"`
	TerminatedBy *uuid.UUID `db:"terminated_by" json:"terminated_by"`
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	Metadata     []byte     `db:"metadata" json:"metadata"`
}

// CheckoutRequest represents a credential checkout request
type CheckoutRequest struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	UserID       uuid.UUID  `db:"user_id" json:"user_id"`
	CredentialID uuid.UUID  `db:"credential_id" json:"credential_id"`
	Justification string   `db:"justification" json:"justification"`
	Duration     int        `db:"duration_minutes" json:"duration_minutes"`
	Status       string     `db:"status" json:"status"` // pending, approved, denied, checked_out, checked_in, expired
	ApprovedBy   *uuid.UUID `db:"approved_by" json:"approved_by"`
	CheckedOutAt *time.Time `db:"checked_out_at" json:"checked_out_at"`
	ExpiresAt    *time.Time `db:"expires_at" json:"expires_at"`
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// AuditEvent represents an immutable audit log entry
type AuditEvent struct {
	ID            uuid.UUID `db:"id" json:"id"`
	TenantID      uuid.UUID `db:"tenant_id" json:"tenant_id"`
	ActorID       uuid.UUID `db:"actor_id" json:"actor_id"`
	ActorType     string    `db:"actor_type" json:"actor_type"` // user, system, api
	Action        string    `db:"action" json:"action"`
	ResourceType  string    `db:"resource_type" json:"resource_type"`
	ResourceID    string    `db:"resource_id" json:"resource_id"`
	Outcome       string    `db:"outcome" json:"outcome"` // success, failure, denied
	IP            string    `db:"ip" json:"ip"`
	UserAgent     string    `db:"user_agent" json:"user_agent"`
	CorrelationID string    `db:"correlation_id" json:"correlation_id"`
	Details       []byte    `db:"details" json:"details"`
	PreviousHash  string    `db:"previous_hash" json:"previous_hash"`
	Hash          string    `db:"hash" json:"hash"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// DiscoveredAsset represents an auto-discovered network asset
type DiscoveredAsset struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Hostname     string     `db:"hostname" json:"hostname"`
	IP           string     `db:"ip" json:"ip"`
	OS           string     `db:"os" json:"os"`
	OpenPorts    []byte     `db:"open_ports" json:"open_ports"`
	Services     []byte     `db:"services" json:"services"`
	Status       string     `db:"status" json:"status"` // discovered, managed, ignored
	LastScannedAt time.Time `db:"last_scanned_at" json:"last_scanned_at"`
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Domain    string    `db:"domain" json:"domain"`
	Plan      string    `db:"plan" json:"plan"` // free, pro, enterprise
	Config    []byte    `db:"config" json:"config"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
EOF

# Auth package
cat > internal/auth/auth.go << 'EOF'
package auth

// TODO: Implement by AI team
// - JWT token service (RS256, access + refresh tokens)
// - Password hashing (argon2id)
// - RBAC middleware (super_admin, admin, operator, auditor, user)
// - MFA verification middleware
// - Session management (Redis-backed)
EOF

# Crypto package
cat > internal/crypto/crypto.go << 'EOF'
package crypto

// TODO: Implement by AI team
// - AES-256-GCM encryption/decryption
// - Envelope encryption (master key → DEK → data)
// - Key derivation (HKDF)
// - Secure random generation
// - Certificate management
EOF

# Database package
cat > internal/database/database.go << 'EOF'
package database

// TODO: Implement by AI team
// - PostgreSQL connection pool (sqlx)
// - Migration runner
// - Health check
// - Multi-tenant query helpers (always filter by tenant_id)
EOF

# Cache package
cat > internal/cache/cache.go << 'EOF'
package cache

// TODO: Implement by AI team
// - Redis client wrapper
// - Session store
// - Rate limiter store
// - Pub/sub for real-time events
EOF

# Events package
cat > internal/events/events.go << 'EOF'
package events

// TODO: Implement by AI team
// - Event bus (Redis pub/sub)
// - Event types: session.started, credential.checked_out, alert.triggered
// - Webhook delivery with retry
// - WebSocket streaming
EOF

# PAM core packages
cat > internal/pam/checkout/checkout.go << 'EOF'
package checkout

// TODO: Implement by AI team
// - Credential checkout flow (request → approve → checkout → use → checkin)
// - Time-bound access with auto-expiry
// - Concurrent checkout prevention
// - Emergency break-glass access
EOF

cat > internal/pam/recording/recording.go << 'EOF'
package recording

// TODO: Implement by AI team
// - Session recording (SSH keystroke capture, RDP video)
// - Recording storage (MinIO/S3 with encryption)
// - Playback API
// - Recording search and indexing
EOF

cat > internal/pam/rotation/rotation.go << 'EOF'
package rotation

// TODO: Implement by AI team
// - Credential rotation engine
// - Rotation strategies: on-checkin, scheduled (daily/weekly/monthly), on-demand
// - Platform connectors: Linux (passwd), Windows (AD), databases, cloud APIs
// - Rotation verification (test new credential works)
// - Rollback on failure
EOF

cat > internal/pam/approval/approval.go << 'EOF'
package approval

// TODO: Implement by AI team
// - Approval workflow engine
// - Multi-level approval chains
// - Auto-approve policies (low-risk, known patterns)
// - Approval timeout and escalation
// - Notification (email, webhook, in-app)
EOF

cat > internal/pam/policy/policy.go << 'EOF'
package policy

// TODO: Implement by AI team
// - Access policies: who can access which credentials, when, how long
// - Time-based restrictions (business hours only)
// - IP/network restrictions
// - MFA requirements per credential
// - Maximum checkout duration per credential/role
EOF

cat > internal/pam/analytics/analytics.go << 'EOF'
package analytics

// TODO: Implement by AI team
// - Privileged access analytics
// - Usage patterns and anomaly detection
// - Risk scoring per session/user
// - Compliance dashboards
// - Unused credential detection
EOF

# Middleware
cat > internal/middleware/middleware.go << 'EOF'
package middleware

// TODO: Implement by AI team
// - Rate limiting (Redis sliding window)
// - CORS configuration
// - Request ID (X-Request-ID)
// - Structured logging (zerolog)
// - Panic recovery
// - Tenant extraction from JWT
EOF

# ═══════════════════════════════════════
# Database Migrations
# ═══════════════════════════════════════

log "Creating initial migrations..."

cat > migrations/gateway/001_initial.up.sql << 'SQL'
-- Users
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_secret TEXT,
    tenant_id UUID NOT NULL,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(email, tenant_id)
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_status ON users(status);

-- Tenants
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

cat > migrations/vault/001_initial.up.sql << 'SQL'
-- Credential Vault
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- ssh_key, password, api_key, certificate, database
    host VARCHAR(255),
    port INTEGER,
    username VARCHAR(255),
    encrypted_secret BYTEA NOT NULL,
    encryption_key_id UUID NOT NULL,
    rotation_policy VARCHAR(50) DEFAULT 'manual',
    last_rotated_at TIMESTAMPTZ,
    folder_id UUID,
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    tenant_id UUID NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_creds_tenant ON credentials(tenant_id);
CREATE INDEX idx_creds_type ON credentials(type);
CREATE INDEX idx_creds_folder ON credentials(folder_id);

-- Encryption keys (envelope encryption)
CREATE TABLE IF NOT EXISTS encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    encrypted_dek BYTEA NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMPTZ
);

-- Credential folders
CREATE TABLE IF NOT EXISTS credential_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES credential_folders(id),
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

cat > migrations/session/001_initial.up.sql << 'SQL'
-- Privileged Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    credential_id UUID NOT NULL,
    checkout_id UUID,
    type VARCHAR(50) NOT NULL, -- ssh, rdp, database, kubernetes, web
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    target_host VARCHAR(255) NOT NULL,
    target_port INTEGER NOT NULL,
    recording_url TEXT,
    recording_size_bytes BIGINT,
    commands_count INTEGER DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    terminated_by UUID,
    termination_reason TEXT,
    tenant_id UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_started ON sessions(started_at);

-- Session recordings metadata
CREATE TABLE IF NOT EXISTS session_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id),
    storage_path TEXT NOT NULL,
    encryption_key_id UUID NOT NULL,
    size_bytes BIGINT,
    duration_seconds INTEGER,
    format VARCHAR(50) NOT NULL, -- asciicast, mp4, json
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

cat > migrations/credential/001_initial.up.sql << 'SQL'
-- Credential Checkout / Check-in
CREATE TABLE IF NOT EXISTS checkout_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    credential_id UUID NOT NULL,
    justification TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 60,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    denied_reason TEXT,
    checked_out_at TIMESTAMPTZ,
    checked_in_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    auto_rotated BOOLEAN DEFAULT false,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkout_user ON checkout_requests(user_id);
CREATE INDEX idx_checkout_cred ON checkout_requests(credential_id);
CREATE INDEX idx_checkout_status ON checkout_requests(status);
CREATE INDEX idx_checkout_expires ON checkout_requests(expires_at);

-- Credential rotation history
CREATE TABLE IF NOT EXISTS rotation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL,
    trigger VARCHAR(50) NOT NULL, -- scheduled, on_checkin, manual, emergency
    status VARCHAR(50) NOT NULL, -- success, failed, rolled_back
    old_version_hash TEXT,
    new_version_hash TEXT,
    error_message TEXT,
    tenant_id UUID NOT NULL,
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

cat > migrations/audit/001_initial.up.sql << 'SQL'
-- Audit Events (append-only, tamper-evident)
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    actor_id UUID NOT NULL,
    actor_type VARCHAR(50) NOT NULL DEFAULT 'user',
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    outcome VARCHAR(50) NOT NULL,
    ip VARCHAR(45),
    user_agent TEXT,
    correlation_id VARCHAR(100),
    details JSONB DEFAULT '{}',
    previous_hash VARCHAR(64),
    hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

CREATE INDEX idx_audit_tenant ON audit_events(tenant_id);
CREATE INDEX idx_audit_actor ON audit_events(actor_id);
CREATE INDEX idx_audit_action ON audit_events(action);
CREATE INDEX idx_audit_created ON audit_events(created_at);
CREATE INDEX idx_audit_correlation ON audit_events(correlation_id);
CREATE INDEX idx_audit_resource ON audit_events(resource_type, resource_id);

-- Create monthly partitions
CREATE TABLE audit_events_default PARTITION OF audit_events DEFAULT;

-- Access policies
CREATE TABLE IF NOT EXISTS access_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    tenant_id UUID NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
SQL

cat > migrations/admin/001_initial.up.sql << 'SQL'
-- System configuration
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    tenant_id UUID NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Discovered assets
CREATE TABLE IF NOT EXISTS discovered_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname VARCHAR(255),
    ip VARCHAR(45) NOT NULL,
    os VARCHAR(100),
    open_ports JSONB DEFAULT '[]',
    services JSONB DEFAULT '[]',
    status VARCHAR(50) NOT NULL DEFAULT 'discovered',
    risk_score INTEGER DEFAULT 0,
    last_scanned_at TIMESTAMPTZ,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assets_tenant ON discovered_assets(tenant_id);
CREATE INDEX idx_assets_status ON discovered_assets(status);
CREATE INDEX idx_assets_ip ON discovered_assets(ip);
SQL

# ═══════════════════════════════════════
# Frontend Setup
# ═══════════════════════════════════════

log "Creating frontend..."

cat > frontend/package.json << 'EOF'
{
  "name": "openpam-frontend",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest",
    "e2e": "playwright test",
    "lint": "eslint src/",
    "typecheck": "tsc --noEmit"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.21.0",
    "axios": "^1.6.0",
    "@tanstack/react-query": "^5.17.0",
    "date-fns": "^3.2.0",
    "lucide-react": "^0.303.0",
    "react-hot-toast": "^2.4.1",
    "xterm": "^5.3.0",
    "xterm-addon-fit": "^0.8.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "@vitejs/plugin-react": "^4.2.0",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0",
    "tailwindcss": "^3.4.0",
    "typescript": "^5.3.0",
    "vite": "^5.0.0",
    "vitest": "^1.2.0",
    "@playwright/test": "^1.41.0",
    "eslint": "^8.56.0"
  }
}
EOF

cat > frontend/tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] }
  },
  "include": ["src"]
}
EOF

cat > frontend/src/types/index.ts << 'EOF'
export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: 'super_admin' | 'admin' | 'operator' | 'auditor' | 'user';
  status: 'active' | 'suspended' | 'locked';
  mfa_enabled: boolean;
  tenant_id: string;
  last_login_at: string | null;
  created_at: string;
}

export interface Credential {
  id: string;
  name: string;
  type: 'ssh_key' | 'password' | 'api_key' | 'certificate' | 'database';
  host: string;
  port: number;
  username: string;
  rotation_policy: 'manual' | 'daily' | 'weekly' | 'on_checkin';
  last_rotated_at: string | null;
  folder_id: string | null;
  tenant_id: string;
  created_at: string;
}

export interface Session {
  id: string;
  user_id: string;
  credential_id: string;
  type: 'ssh' | 'rdp' | 'database' | 'kubernetes' | 'web';
  status: 'active' | 'ended' | 'terminated' | 'failed';
  target_host: string;
  target_port: number;
  recording_url: string;
  started_at: string;
  ended_at: string | null;
  terminated_by: string | null;
}

export interface CheckoutRequest {
  id: string;
  user_id: string;
  credential_id: string;
  justification: string;
  duration_minutes: number;
  status: 'pending' | 'approved' | 'denied' | 'checked_out' | 'checked_in' | 'expired';
  approved_by: string | null;
  expires_at: string | null;
  created_at: string;
}

export interface AuditEvent {
  id: string;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  outcome: 'success' | 'failure' | 'denied';
  ip: string;
  details: Record<string, unknown>;
  created_at: string;
}

export interface DiscoveredAsset {
  id: string;
  hostname: string;
  ip: string;
  os: string;
  open_ports: number[];
  services: string[];
  status: 'discovered' | 'managed' | 'ignored';
  risk_score: number;
  last_scanned_at: string;
}
EOF

cat > frontend/src/api/client.ts << 'EOF'
import axios from 'axios';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8500/api/v1';

const client = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

client.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('access_token');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export default client;
EOF

# ═══════════════════════════════════════
# Config Files
# ═══════════════════════════════════════

log "Creating config files..."

cat > configs/development/config.yaml << 'EOF'
server:
  port: 8500
  mode: debug

database:
  host: localhost
  port: 5432
  name: openpam
  user: openpam
  password: openpam_dev
  sslmode: disable
  max_open_conns: 25
  max_idle_conns: 5

redis:
  addr: localhost:6379
  db: 0

jwt:
  signing_method: RS256
  access_token_ttl: 15m
  refresh_token_ttl: 7d

vault:
  master_key_source: env  # env, file, kms
  encryption_algorithm: AES-256-GCM

session_recording:
  storage: local  # local, s3, minio
  path: /var/openpam/recordings
  encryption: true

discovery:
  scan_interval: 24h
  port_range: "22,80,443,3306,5432,3389,8080,8443"
EOF

cat > .gitignore << 'EOF'
# Go
/bin/
*.exe
*.test
*.out
vendor/

# Frontend
frontend/node_modules/
frontend/dist/
frontend/.vite/

# IDE
.idea/
.vscode/
*.swp

# Environment
.env
.env.local
*.pem
*.key

# Team state
.team/

# OS
.DS_Store
Thumbs.db

# Recordings
recordings/
EOF

cat > .env.example << 'EOF'
# Database
DATABASE_URL=postgres://openpam:openpam_dev@localhost:5432/openpam?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# JWT
JWT_PRIVATE_KEY_PATH=./configs/jwt.key
JWT_PUBLIC_KEY_PATH=./configs/jwt.pub

# Vault Master Key (CHANGE IN PRODUCTION)
VAULT_MASTER_KEY=dev-master-key-change-me-in-production

# Session Recording
RECORDING_STORAGE_PATH=/var/openpam/recordings
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Service Ports
GATEWAY_PORT=8500
SESSION_PORT=8501
VAULT_PORT=8502
CREDENTIAL_PORT=8503
AUDIT_PORT=8504
ADMIN_PORT=8505
DISCOVERY_PORT=8506
EOF

# ═══════════════════════════════════════
# Docker
# ═══════════════════════════════════════

log "Creating Docker setup..."

cat > deployments/docker/docker-compose.yml << 'EOF'
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: openpam
      POSTGRES_PASSWORD: openpam_dev
      POSTGRES_DB: openpam
    ports: ["5432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U openpam"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes: [minio_data:/data]

  gateway:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.gateway }
    ports: ["8500:8500"]
    environment: &common_env
      DATABASE_URL: postgres://openpam:openpam_dev@postgres:5432/openpam?sslmode=disable
      REDIS_URL: redis://redis:6379/0
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

  session-service:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.session-service }
    ports: ["8501:8501"]
    environment: *common_env
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

  vault-service:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.vault-service }
    ports: ["8502:8502"]
    environment: *common_env
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

  credential-service:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.credential-service }
    ports: ["8503:8503"]
    environment: *common_env
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

  audit-service:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.audit-service }
    ports: ["8504:8504"]
    environment: *common_env
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

  admin-service:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.admin-service }
    ports: ["8505:8505"]
    environment: *common_env
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

  discovery-service:
    build: { context: ../.. , dockerfile: deployments/docker/Dockerfile.discovery-service }
    ports: ["8506:8506"]
    environment: *common_env
    depends_on: { postgres: { condition: service_healthy }, redis: { condition: service_healthy } }

volumes:
  pgdata:
  minio_data:

networks:
  default:
    name: openpam-net
EOF

# Dockerfile template for each service
for svc in gateway session-service vault-service credential-service audit-service admin-service discovery-service; do
  port_map="gateway:8500 session-service:8501 vault-service:8502 credential-service:8503 audit-service:8504 admin-service:8505 discovery-service:8506"
  port=$(echo "$port_map" | tr ' ' '\n' | grep "^${svc}:" | cut -d: -f2)

  cat > "deployments/docker/Dockerfile.${svc}" << DOCKERFILE
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./cmd/${svc}

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 appuser
COPY --from=builder /app /usr/local/bin/app
USER appuser
EXPOSE ${port}
HEALTHCHECK --interval=10s --timeout=3s CMD wget -qO- http://localhost:${port}/health || exit 1
CMD ["app"]
DOCKERFILE
done

# ═══════════════════════════════════════
# Git Init
# ═══════════════════════════════════════

log "Initializing git..."

if [ ! -d ".git" ]; then
  git init
  git add -A
  git commit -m "Initial project structure — OpenPAM Privileged Access Management"
fi

# ═══════════════════════════════════════
# Summary
# ═══════════════════════════════════════

echo ""
log "═══════════════════════════════════════════════════"
log "  ✅ OpenPAM project created at: $PROJECT_DIR"
log "═══════════════════════════════════════════════════"
echo ""

# Count files
total_files=$(find . -type f | grep -v ".git/" | wc -l)
total_dirs=$(find . -type d | grep -v ".git/" | wc -l)
log "  📁 Directories: $total_dirs"
log "  📄 Files:       $total_files"
echo ""
log "  Services:"
log "    gateway           :8500  — API gateway, auth, routing"
log "    session-service   :8501  — Session management, recording"
log "    vault-service     :8502  — Credential vault, encryption"
log "    credential-service:8503  — Checkout, rotation, discovery"
log "    audit-service     :8504  — Audit logging, compliance"
log "    admin-service     :8505  — Dashboard, config, tenants"
log "    discovery-service :8506  — Asset discovery, scanning"
echo ""
log "  Next steps:"
log "    1. Copy team.sh: cp ~/team.sh $PROJECT_DIR/team.sh"
log "    2. Start AI team:"
echo ""
echo "    cd $PROJECT_DIR"
echo '    ./team.sh --project "Build complete PAM platform: credential vault with AES-256-GCM envelope encryption, checkout/checkin workflow with approval chains, SSH/RDP session recording with playback, automatic credential rotation on checkin, asset discovery with port scanning, real-time session monitoring with admin termination, compliance reports (SOC2/ISO27001), emergency break-glass access with post-incident review"'
echo ""
