package certificate

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/vault/certificate/acme"
	"github.com/openpam/openpam/internal/vault/certificate/pki"
	"github.com/openpam/openpam/internal/vault/repository"
	"github.com/rs/zerolog"
)

// Manager handles certificate lifecycle management
type Manager struct {
	db           *sqlx.DB
	cache        *cache.Cache
	repo         *repository.CertificateRepository
	pkiHierarchy *pki.Hierarchy
	acmeClient   *acme.Client
	publisher    *events.Publisher
	logger       zerolog.Logger
}

// Config holds certificate manager configuration
type Config struct {
	// PKI Configuration
	RootCADuration        time.Duration
	IntermediateCADuration time.Duration
	LeafCertDuration      time.Duration
	KeySize               int
	KeyType               string // rsa, ecdsa

	// ACME Configuration
	ACMEDirectory  string
	ACMEAccountURL string

	// Auto-rotation
	AutoRotateBefore time.Duration
	CheckInterval    time.Duration
}

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

// CertificateSubject represents X.509 subject information
type CertificateSubject struct {
	CommonName         string `json:"common_name"`
	Organization       string `json:"organization,omitempty"`
	OrganizationalUnit string `json:"organizational_unit,omitempty"`
	Country            string `json:"country,omitempty"`
	Locality           string `json:"locality,omitempty"`
	Province           string `json:"province,omitempty"`
}

// NewManager creates a new certificate manager
func NewManager(
	db *sqlx.DB,
	cache *cache.Cache,
	publisher *events.Publisher,
	cfg Config,
	logger zerolog.Logger,
) (*Manager, error) {
	// Initialize repository
	repo := repository.NewCertificateRepository(db, cache, logger)

	// Initialize PKI hierarchy
	pkiHierarchy, err := pki.NewHierarchy(pki.Config{
		RootCADuration:         cfg.RootCADuration,
		IntermediateCADuration: cfg.IntermediateCADuration,
		LeafCertDuration:       cfg.LeafCertDuration,
		KeySize:                cfg.KeySize,
		KeyType:                cfg.KeyType,
	}, repo, logger)
	if err != nil {
		return nil, fmt.Errorf("certificate: failed to initialize PKI hierarchy: %w", err)
	}

	// Initialize ACME client
	var acmeClient *acme.Client
	if cfg.ACMEDirectory != "" {
		acmeClient = acme.NewClient(acme.Config{
			DirectoryURL: cfg.ACMEDirectory,
		}, logger)
	}

	return &Manager{
		db:           db,
		cache:        cache,
		repo:         repo,
		pkiHierarchy: pkiHierarchy,
		acmeClient:   acmeClient,
		publisher:    publisher,
		logger:       logger,
	}, nil
}

// IssueCertificate issues a new certificate
func (m *Manager) IssueCertificate(ctx context.Context, req *CertificateRequest) (*Certificate, error) {
	cert := &Certificate{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		Name:      req.Name,
		Type:      req.Type,
		Status:    StatusActive,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(req.Duration),
		DNSNames:  req.DNSNames,
		IPAddresses: func() []string {
			ips := make([]string, len(req.IPAddresses))
			copy(ips, req.IPAddresses)
			return ips
		}(),
		KeyUsage:   req.KeyUsage,
		ExtKeyUsage: req.ExtKeyUsage,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	switch req.Type {
	case TypeLeaf:
		if req.IssuerID == nil {
			return nil, fmt.Errorf("certificate: issuer_id required for leaf certificates")
		}
		if err := m.pkiHierarchy.IssueLeaf(ctx, cert, req.Subject, *req.IssuerID); err != nil {
			return nil, fmt.Errorf("certificate: failed to issue leaf: %w", err)
		}

	case TypeIntermediateCA:
		if err := m.pkiHierarchy.IssueIntermediateCA(ctx, cert, req.Subject, req.IssuerID); err != nil {
			return nil, fmt.Errorf("certificate: failed to issue intermediate CA: %w", err)
		}

	case TypeACME:
		if m.acmeClient == nil {
			return nil, fmt.Errorf("certificate: ACME client not configured")
		}
		if err := m.acmeClient.IssueCertificate(ctx, cert, req); err != nil {
			return nil, fmt.Errorf("certificate: failed to issue ACME certificate: %w", err)
		}

	default:
		return nil, fmt.Errorf("certificate: unsupported certificate type: %s", req.Type)
	}

	// Save to database
	if err := m.repo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("certificate: failed to save certificate: %w", err)
	}

	// Publish event
	if m.publisher != nil {
		_ = m.publisher.Publish(ctx, events.Event{
			Type:     "certificate.issued",
			TenantID: cert.TenantID.String(),
			ActorID:  "system",
			Action:   "issue",
			Resource: "certificate",
			Data: map[string]interface{}{
				"certificate_id": cert.ID.String(),
				"type":          string(cert.Type),
				"subject":       cert.Subject,
			},
		})
	}

	m.logger.Info().
		Str("certificate_id", cert.ID.String()).
		Str("type", string(cert.Type)).
		Str("subject", cert.Subject).
		Msg("Certificate issued")

	return cert, nil
}

// RevokeCertificate revokes a certificate
func (m *Manager) RevokeCertificate(ctx context.Context, certID uuid.UUID, reason string, revokedBy uuid.UUID) error {
	cert, err := m.repo.GetByID(ctx, certID)
	if err != nil {
		return fmt.Errorf("certificate: failed to get certificate: %w", err)
	}

	if cert.Status == StatusRevoked {
		return fmt.Errorf("certificate: already revoked")
	}

	now := time.Now()
	cert.Status = StatusRevoked
	cert.RevokedAt = &now
	cert.RevokedBy = &revokedBy
	cert.RevocationReason = &reason
	cert.UpdatedAt = now

	if err := m.repo.Update(ctx, cert); err != nil {
		return fmt.Errorf("certificate: failed to update certificate: %w", err)
	}

	// Add to CRL if this is a CA certificate
	if cert.Type == TypeRootCA || cert.Type == TypeIntermediateCA {
		if err := m.pkiHierarchy.AddToCRL(ctx, cert); err != nil {
			m.logger.Error().Err(err).Msg("Failed to add revoked certificate to CRL")
		}
	}

	// Publish event
	if m.publisher != nil {
		_ = m.publisher.Publish(ctx, events.Event{
			Type:     "certificate.revoked",
			TenantID: cert.TenantID.String(),
			ActorID:  revokedBy.String(),
			Action:   "revoke",
			Resource: "certificate",
			Data: map[string]interface{}{
				"certificate_id": cert.ID.String(),
				"reason":        reason,
			},
		})
	}

	m.logger.Info().
		Str("certificate_id", cert.ID.String()).
		Str("reason", reason).
		Msg("Certificate revoked")

	return nil
}

// RenewCertificate renews an existing certificate
func (m *Manager) RenewCertificate(ctx context.Context, certID uuid.UUID) (*Certificate, error) {
	cert, err := m.repo.GetByID(ctx, certID)
	if err != nil {
		return nil, fmt.Errorf("certificate: failed to get certificate: %w", err)
	}

	// Create renewal request
	renewalReq := &CertificateRequest{
		TenantID:    cert.TenantID,
		Name:        cert.Name + "-renewed",
		Type:        cert.Type,
		DNSNames:    cert.DNSNames,
		IPAddresses: cert.IPAddresses,
		KeyUsage:    cert.KeyUsage,
		ExtKeyUsage: cert.ExtKeyUsage,
		Duration:    cert.NotAfter.Sub(cert.NotBefore),
		IssuerID:    cert.IssuerID,
	}

	// Parse existing subject for renewal
	// In production, you'd parse the PEM certificate properly
	subject := CertificateSubject{
		CommonName: cert.Subject,
	}
	renewalReq.Subject = subject

	// Issue new certificate
	newCert, err := m.IssueCertificate(ctx, renewalReq)
	if err != nil {
		return nil, fmt.Errorf("certificate: failed to issue renewed certificate: %w", err)
	}

	// Mark old certificate as being replaced
	cert.Status = StatusExpired
	cert.UpdatedAt = time.Now()
	if err := m.repo.Update(ctx, cert); err != nil {
		m.logger.Error().Err(err).Msg("Failed to update old certificate status")
	}

	m.logger.Info().
		Str("old_cert_id", cert.ID.String()).
		Str("new_cert_id", newCert.ID.String()).
		Msg("Certificate renewed")

	return newCert, nil
}

// GetCertificate retrieves a certificate by ID
func (m *Manager) GetCertificate(ctx context.Context, certID uuid.UUID) (*Certificate, error) {
	return m.repo.GetByID(ctx, certID)
}

// ListCertificates lists certificates with filtering
func (m *Manager) ListCertificates(ctx context.Context, tenantID uuid.UUID, filter *CertificateFilter) ([]*Certificate, error) {
	return m.repo.List(ctx, tenantID, filter)
}

// GetCertificateChain returns the full certificate chain for a leaf certificate
func (m *Manager) GetCertificateChain(ctx context.Context, certID uuid.UUID) ([]*Certificate, error) {
	cert, err := m.repo.GetByID(ctx, certID)
	if err != nil {
		return nil, err
	}

	chain := []*Certificate{cert}

	for cert.IssuerID != nil {
		cert, err = m.repo.GetByID(ctx, *cert.IssuerID)
		if err != nil {
			break
		}
		chain = append(chain, cert)
	}

	return chain, nil
}

// ValidateCertificate validates a certificate's signature and expiration
func (m *Manager) ValidateCertificate(ctx context.Context, certPEM []byte) (*ValidationResult, error) {
	// Parse PEM block
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("certificate: failed to decode PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("certificate: failed to parse certificate: %w", err)
	}

	result := &ValidationResult{
		IsValid:    true,
		Subject:    cert.Subject.CommonName,
		Issuer:     cert.Issuer.CommonName,
		NotBefore:  cert.NotBefore,
		NotAfter:   cert.NotAfter,
		SerialNumber: cert.SerialNumber.String(),
	}

	// Check expiration
	now := time.Now()
	if now.Before(cert.NotBefore) {
		result.IsValid = false
		result.Reasons = append(result.Reasons, "certificate not yet valid")
	}
	if now.After(cert.NotAfter) {
		result.IsValid = false
		result.Reasons = append(result.Reasons, "certificate expired")
	}

	// Verify signature chain if issuer is known
	if cert.Issuer.CommonName != cert.Subject.CommonName {
		// This is not a self-signed cert, verify against issuer
		if err := m.verifyChain(ctx, cert); err != nil {
			result.IsValid = false
			result.Reasons = append(result.Reasons, "signature verification failed")
		}
	}

	return result, nil
}

// verifyChain verifies the certificate chain
func (m *Manager) verifyChain(ctx context.Context, cert *x509.Certificate) error {
	// Find issuer certificate
	issuer, err := m.repo.GetBySerialNumber(ctx, cert.Issuer.SerialNumber.String())
	if err != nil {
		return fmt.Errorf("issuer not found")
	}

	// Parse issuer certificate
	issuerBlock, _ := pem.Decode(issuer.PEMCertificate)
	if issuerBlock == nil {
		return fmt.Errorf("failed to decode issuer PEM")
	}

	issuerCert, err := x509.ParseCertificate(issuerBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse issuer certificate: %w", err)
	}

	// Verify signature
	if err := cert.CheckSignatureFrom(issuerCert); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// CheckExpiringCertificates finds certificates expiring soon
func (m *Manager) CheckExpiringCertificates(ctx context.Context, within time.Duration) ([]*Certificate, error) {
	return m.repo.GetExpiring(ctx, time.Now().Add(within))
}

// GenerateKeyPair generates a new key pair
func (m *Manager) GenerateKeyPair(keyType string, bits int) (crypto.PrivateKey, interface{}, error) {
	switch keyType {
	case "rsa":
		return generateRSAKey(bits)
	case "ecdsa":
		return generateECDSAKey(bits)
	default:
		return nil, nil, fmt.Errorf("certificate: unsupported key type: %s", keyType)
	}
}

// generateRSAKey generates an RSA key pair
func generateRSAKey(bits int) (crypto.PrivateKey, interface{}, error) {
	// Implementation would use crypto/rsa
	return nil, nil, fmt.Errorf("not implemented")
}

// generateECDSAKey generates an ECDSA key pair
func generateECDSAKey(bits int) (crypto.PrivateKey, interface{}, error) {
	// Implementation would use crypto/ecdsa
	return nil, nil, fmt.Errorf("not implemented")
}

// CertificateFilter filters certificate queries
type CertificateFilter struct {
	Type   *CertificateType
	Status *CertificateStatus
	IssuerID *uuid.UUID
}

// ValidationResult represents the result of certificate validation
type ValidationResult struct {
	IsValid     bool      `json:"is_valid"`
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	SerialNumber string   `json:"serial_number"`
	Reasons     []string  `json:"reasons,omitempty"`
}

// generateSerialNumber generates a random serial number
func generateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}

// createCertificateTemplate creates an X.509 certificate template
func createCertificateTemplate(subject pkix.Name, validity time.Duration, dnsNames []string, keyUsages x509.KeyUsage, extKeyUsage []x509.ExtKeyUsage) *x509.Certificate {
	serialNumber, err := generateSerialNumber()
	if err != nil {
		serialNumber = big.NewInt(1)
	}

	return &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              keyUsages,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
	}
}
