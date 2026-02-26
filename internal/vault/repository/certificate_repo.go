package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/vault/cert"
	"github.com/rs/zerolog"
)

// CertificateRepository handles certificate data persistence
type CertificateRepository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewCertificateRepository creates a new certificate repository
func NewCertificateRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *CertificateRepository {
	return &CertificateRepository{
		db:     db,
		cache:  c,
		logger: logger,
	}
}

// Create saves a new certificate
func (r *CertificateRepository) Create(ctx context.Context, cert *cert.Certificate) error {
	query := `
		INSERT INTO certificates (
			id, tenant_id, name, type, status, pem_certificate, pem_private_key,
			serial_number, subject, issuer_id, not_before, not_after,
			key_usage, ext_key_usage, dns_names, ip_addresses,
			acme_account_id, acme_order_url, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :type, :status, :pem_certificate, :pem_private_key,
			:serial_number, :subject, :issuer_id, :not_before, :not_after,
			:key_usage, :ext_key_usage, :dns_names, :ip_addresses,
			:acme_account_id, :acme_order_url, :created_at, :updated_at
		)
	`

	keyUsageJSON, _ := json.Marshal(cert.KeyUsage)
	extKeyUsageJSON, _ := json.Marshal(cert.ExtKeyUsage)
	dnsNamesJSON, _ := json.Marshal(cert.DNSNames)
	ipAddressesJSON, _ := json.Marshal(cert.IPAddresses)

	args := map[string]interface{}{
		"id":               cert.ID,
		"tenant_id":        cert.TenantID,
		"name":             cert.Name,
		"type":             cert.Type,
		"status":           cert.Status,
		"pem_certificate":  cert.PEMCertificate,
		"pem_private_key":  cert.PEMPrivateKey,
		"serial_number":    cert.SerialNumber,
		"subject":          cert.Subject,
		"issuer_id":        cert.IssuerID,
		"not_before":       cert.NotBefore,
		"not_after":        cert.NotAfter,
		"key_usage":        keyUsageJSON,
		"ext_key_usage":    extKeyUsageJSON,
		"dns_names":        dnsNamesJSON,
		"ip_addresses":     ipAddressesJSON,
		"acme_account_id":  cert.ACMEAccountID,
		"acme_order_url":   cert.ACMEOrderURL,
		"created_at":       cert.CreatedAt,
		"updated_at":       cert.UpdatedAt,
	}

	_, err := r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("certificate_repo.Create: %w", err)
	}

	// Invalidate cache
	_ = r.cache.DeleteByPattern(ctx, fmt.Sprintf("cert:*:%s", cert.TenantID))

	return nil
}

// GetByID retrieves a certificate by ID
func (r *CertificateRepository) GetByID(ctx context.Context, id uuid.UUID) (*cert.Certificate, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("cert:id:%s", id)
	var cached cert.Certificate
	if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	query := `SELECT * FROM certificates WHERE id = $1`
	var cert cert.Certificate
	if err := r.db.GetContext(ctx, &cert, query, id); err != nil {
		return nil, fmt.Errorf("certificate_repo.GetByID: %w", err)
	}

	// Cache for 5 minutes
	_ = r.cache.Set(ctx, cacheKey, &cert, 5*time.Minute)

	return &cert, nil
}

// GetBySerialNumber retrieves a certificate by serial number
func (r *CertificateRepository) GetBySerialNumber(ctx context.Context, serialNumber string) (*cert.Certificate, error) {
	query := `SELECT * FROM certificates WHERE serial_number = $1`
	var cert cert.Certificate
	if err := r.db.GetContext(ctx, &cert, query, serialNumber); err != nil {
		return nil, fmt.Errorf("certificate_repo.GetBySerialNumber: %w", err)
	}
	return &cert, nil
}

// GetRootCA retrieves the root CA for a tenant
func (r *CertificateRepository) GetRootCA(ctx context.Context, tenantID uuid.UUID) (*cert.Certificate, error) {
	query := `
		SELECT * FROM certificates
		WHERE tenant_id = $1 AND type = 'root_ca' AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`
	var cert cert.Certificate
	if err := r.db.GetContext(ctx, &cert, query, tenantID); err != nil {
		return nil, fmt.Errorf("certificate_repo.GetRootCA: %w", err)
	}
	return &cert, nil
}

// List retrieves certificates with filtering
func (r *CertificateRepository) List(ctx context.Context, tenantID uuid.UUID, filter *cert.CertificateFilter) ([]*cert.Certificate, error) {
	baseQuery := `
		SELECT * FROM certificates
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argCount := 2

	if filter != nil {
		if filter.Type != nil {
			baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
			args = append(args, *filter.Type)
			argCount++
		}
		if filter.Status != nil {
			baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
			args = append(args, *filter.Status)
			argCount++
		}
		if filter.IssuerID != nil {
			baseQuery += fmt.Sprintf(" AND issuer_id = $%d", argCount)
			args = append(args, *filter.IssuerID)
			argCount++
		}
	}

	baseQuery += " ORDER BY created_at DESC"

	var certs []*cert.Certificate
	if err := r.db.SelectContext(ctx, &certs, baseQuery, args...); err != nil {
		return nil, fmt.Errorf("certificate_repo.List: %w", err)
	}

	return certs, nil
}

// Update updates a certificate
func (r *CertificateRepository) Update(ctx context.Context, cert *cert.Certificate) error {
	query := `
		UPDATE certificates SET
			name = :name,
			status = :status,
			serial_number = :serial_number,
			subject = :subject,
			issuer_id = :issuer_id,
			not_before = :not_before,
			not_after = :not_after,
			key_usage = :key_usage,
			ext_key_usage = :ext_key_usage,
			dns_names = :dns_names,
			ip_addresses = :ip_addresses,
			revoked_at = :revoked_at,
			revoked_by = :revoked_by,
			revocation_reason = :revocation_reason,
			updated_at = :updated_at
		WHERE id = :id
	`

	keyUsageJSON, _ := json.Marshal(cert.KeyUsage)
	extKeyUsageJSON, _ := json.Marshal(cert.ExtKeyUsage)
	dnsNamesJSON, _ := json.Marshal(cert.DNSNames)
	ipAddressesJSON, _ := json.Marshal(cert.IPAddresses)

	args := map[string]interface{}{
		"id":                cert.ID,
		"name":              cert.Name,
		"status":            cert.Status,
		"serial_number":     cert.SerialNumber,
		"subject":           cert.Subject,
		"issuer_id":         cert.IssuerID,
		"not_before":        cert.NotBefore,
		"not_after":         cert.NotAfter,
		"key_usage":         keyUsageJSON,
		"ext_key_usage":     extKeyUsageJSON,
		"dns_names":         dnsNamesJSON,
		"ip_addresses":      ipAddressesJSON,
		"revoked_at":        cert.RevokedAt,
		"revoked_by":        cert.RevokedBy,
		"revocation_reason": cert.RevocationReason,
		"updated_at":        cert.UpdatedAt,
	}

	_, err := r.db.NamedExecContext(ctx, query, args)
	if err != nil {
		return fmt.Errorf("certificate_repo.Update: %w", err)
	}

	// Invalidate cache
	_ = r.cache.Delete(ctx, fmt.Sprintf("cert:id:%s", cert.ID))

	return nil
}

// Delete soft-deletes a certificate
func (r *CertificateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE certificates SET status = 'revoked', updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("certificate_repo.Delete: %w", err)
	}

	// Invalidate cache
	_ = r.cache.Delete(ctx, fmt.Sprintf("cert:id:%s", id))

	return nil
}

// GetExpiring retrieves certificates expiring before the given time
func (r *CertificateRepository) GetExpiring(ctx context.Context, before time.Time) ([]*cert.Certificate, error) {
	query := `
		SELECT * FROM certificates
		WHERE status = 'active'
			AND not_after <= $1
			AND not_after > NOW()
		ORDER BY not_after ASC
	`
	var certs []*cert.Certificate
	if err := r.db.SelectContext(ctx, &certs, query, before); err != nil {
		return nil, fmt.Errorf("certificate_repo.GetExpiring: %w", err)
	}
	return certs, nil
}

// GetExpired retrieves certificates that have expired
func (r *CertificateRepository) GetExpired(ctx context.Context) ([]*cert.Certificate, error) {
	query := `
		SELECT * FROM certificates
		WHERE status = 'active'
			AND not_after <= NOW()
		ORDER BY not_after ASC
	`
	var certs []*cert.Certificate
	if err := r.db.SelectContext(ctx, &certs, query); err != nil {
		return nil, fmt.Errorf("certificate_repo.GetExpired: %w", err)
	}
	return certs, nil
}

// ListByIssuer retrieves all certificates issued by a given issuer
func (r *CertificateRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*cert.Certificate, error) {
	query := `
		SELECT * FROM certificates
		WHERE issuer_id = $1
		ORDER BY created_at DESC
	`
	var certs []*cert.Certificate
	if err := r.db.SelectContext(ctx, &certs, query, issuerID); err != nil {
		return nil, fmt.Errorf("certificate_repo.ListByIssuer: %w", err)
	}
	return certs, nil
}

// GetChainIssuers returns the issuer chain for a certificate
func (r *CertificateRepository) GetChainIssuers(ctx context.Context, certID uuid.UUID) ([]*cert.Certificate, error) {
	cert, err := r.GetByID(ctx, certID)
	if err != nil {
		return nil, err
	}

	var chain []*cert.Certificate

	for cert.IssuerID != nil {
		issuer, err := r.GetByID(ctx, *cert.IssuerID)
		if err != nil {
			break
		}
		chain = append(chain, issuer)
		cert = issuer
	}

	return chain, nil
}
