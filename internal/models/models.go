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
