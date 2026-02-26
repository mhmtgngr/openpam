package cert

import (
	"time"

	"github.com/google/uuid"
)

// CertificateType represents the type of certificate
type CertificateType string

const (
	TypeRootCA         CertificateType = "root_ca"
	TypeIntermediateCA CertificateType = "intermediate_ca"
	TypeLeaf           CertificateType = "leaf"
	TypeExternal       CertificateType = "external"
	TypeACME           CertificateType = "acme"
)

// CertificateStatus represents the status of a certificate
type CertificateStatus string

const (
	StatusActive    CertificateStatus = "active"
	StatusExpired   CertificateStatus = "expired"
	StatusRevoked   CertificateStatus = "revoked"
	StatusPending   CertificateStatus = "pending"
	StatusRenewing  CertificateStatus = "renewing"
)

// CertificateSubject represents X.509 subject information
type CertificateSubject struct {
	CommonName         string `json:"common_name"`
	Organization       string `json:"organization,omitempty"`
	OrganizationalUnit string `json:"organizational_unit,omitempty"`
	Country            string `json:"country,omitempty"`
	Locality           string `json:"locality,omitempty"`
	Province           string `json:"province,omitempty"`
}

// CertificateRequest represents a certificate signing request
type CertificateRequest struct {
	TenantID       uuid.UUID           `json:"tenant_id"`
	Name           string              `json:"name"`
	Type           CertificateType     `json:"type"`
	Subject        CertificateSubject   `json:"subject"`
	DNSNames       []string            `json:"dns_names"`
	IPAddresses    []string            `json:"ip_addresses"`
	KeyUsage       []string            `json:"key_usage"`
	ExtKeyUsage    []string            `json:"ext_key_usage"`
	Duration       time.Duration       `json:"duration"`
	IssuerID       *uuid.UUID          `json:"issuer_id,omitempty"`
	ACMEChallenge  string              `json:"acme_challenge,omitempty"`
}

// Certificate represents a certificate in the system
type Certificate struct {
	ID              uuid.UUID           `db:"id" json:"id"`
	TenantID        uuid.UUID           `db:"tenant_id" json:"tenant_id"`
	Name            string              `db:"name" json:"name"`
	Type            CertificateType     `db:"type" json:"type"`
	Status          CertificateStatus   `db:"status" json:"status"`

	// Certificate data
	PEMCertificate  []byte              `db:"pem_certificate" json:"-"`
	PEMPrivateKey   []byte              `db:"pem_private_key" json:"-"`
	SerialNumber    string              `db:"serial_number" json:"serial_number"`
	Subject         string              `db:"subject" json:"subject"`
	IssuerID        *uuid.UUID          `db:"issuer_id" json:"issuer_id,omitempty"`

	// Validity
	NotBefore       time.Time           `db:"not_before" json:"not_before"`
	NotAfter        time.Time           `db:"not_after" json:"not_after"`

	// Usage
	KeyUsage        []string            `db:"key_usage" json:"key_usage"`
	ExtKeyUsage     []string            `db:"ext_key_usage" json:"ext_key_usage"`
	DNSNames        []string            `db:"dns_names" json:"dns_names"`
	IPAddresses     []string            `db:"ip_addresses" json:"ip_addresses"`

	// ACME specific
	ACMEAccountID   *string             `db:"acme_account_id" json:"acme_account_id,omitempty"`
	ACMEOrderURL    *string             `db:"acme_order_url" json:"acme_order_url,omitempty"`

	// Metadata
	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at" json:"updated_at"`
	RevokedAt       *time.Time          `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedBy       *uuid.UUID          `db:"revoked_by" json:"revoked_by,omitempty"`
	RevocationReason *string            `db:"revocation_reason" json:"revocation_reason,omitempty"`
}

// CertificateFilter filters certificate queries
type CertificateFilter struct {
	Type     *CertificateType
	Status   *CertificateStatus
	IssuerID *uuid.UUID
}
