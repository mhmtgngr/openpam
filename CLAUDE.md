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
