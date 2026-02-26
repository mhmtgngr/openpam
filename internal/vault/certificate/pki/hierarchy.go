package pki

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/vault/cert"
	"github.com/openpam/openpam/internal/vault/repository"
	"github.com/rs/zerolog"
)

// Hierarchy manages the PKI certificate hierarchy
type Hierarchy struct {
	repo   *repository.CertificateRepository
	config Config
	logger zerolog.Logger
}

// Config holds PKI hierarchy configuration
type Config struct {
	RootCADuration         time.Duration
	IntermediateCADuration time.Duration
	LeafCertDuration       time.Duration
	KeySize                int
	KeyType                string // rsa, ecdsa
}

// KeyType represents supported key types
type KeyType string

const (
	KeyTypeRSA   KeyType = "rsa"
	KeyTypeECDSA KeyType = "ecdsa"
)

// NewHierarchy creates a new PKI hierarchy manager
func NewHierarchy(cfg Config, repo *repository.CertificateRepository, logger zerolog.Logger) (*Hierarchy, error) {
	if cfg.RootCADuration == 0 {
		cfg.RootCADuration = 10 * 365 * 24 * time.Hour // 10 years
	}
	if cfg.IntermediateCADuration == 0 {
		cfg.IntermediateCADuration = 5 * 365 * 24 * time.Hour // 5 years
	}
	if cfg.LeafCertDuration == 0 {
		cfg.LeafCertDuration = 365 * 24 * time.Hour // 1 year
	}
	if cfg.KeySize == 0 {
		cfg.KeySize = 2048
	}
	if cfg.KeyType == "" {
		cfg.KeyType = string(KeyTypeRSA)
	}

	return &Hierarchy{
		repo:   repo,
		config: cfg,
		logger: logger,
	}, nil
}

// InitializeRootCA initializes the root CA for a tenant
func (h *Hierarchy) InitializeRootCA(ctx context.Context, tenantID uuid.UUID, subject cert.CertificateSubject) (*cert.Certificate, error) {
	// Check if root CA already exists
	existing, err := h.repo.GetRootCA(ctx, tenantID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Generate key pair
	privateKey, err := h.generateKey()
	if err != nil {
		return nil, fmt.Errorf("pki: failed to generate key: %w", err)
	}

	// Create certificate template
	serialNumber, err := h.generateSerialNumber()
	if err != nil {
		return nil, fmt.Errorf("pki: failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         subject.CommonName,
			Organization:       []string{subject.Organization},
			OrganizationalUnit: []string{subject.OrganizationalUnit},
			Country:            []string{subject.Country},
			Locality:           []string{subject.Locality},
			Province:           []string{subject.Province},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(h.config.RootCADuration),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	// Self-sign the root CA certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, h.publicKey(privateKey), privateKey)
	if err != nil {
		return nil, fmt.Errorf("pki: failed to create certificate: %w", err)
	}

	// Encode certificate and private key to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM, err := h.encodePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("pki: failed to encode private key: %w", err)
	}

	// Create certificate record
	cert := &cert.Certificate{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "Root CA",
		Type:           cert.TypeRootCA,
		Status:         cert.StatusActive,
		PEMCertificate: certPEM,
		PEMPrivateKey:  keyPEM,
		SerialNumber:   serialNumber.String(),
		Subject:        subject.CommonName,
		NotBefore:      template.NotBefore,
		NotAfter:       template.NotAfter,
		KeyUsage:       []string{"cert_sign", "crl_sign"},
		ExtKeyUsage:    []string{"server_auth", "client_auth"},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.repo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("pki: failed to save root CA: %w", err)
	}

	h.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("certificate_id", cert.ID.String()).
		Msg("Root CA initialized")

	return cert, nil
}

// IssueIntermediateCA issues an intermediate CA certificate
func (h *Hierarchy) IssueIntermediateCA(ctx context.Context, crt *cert.Certificate, subject cert.CertificateSubject, issuerID *uuid.UUID) error {
	var issuerCert *cert.Certificate
	var err error

	if issuerID == nil {
		// Get root CA
		issuerCert, err = h.repo.GetRootCA(ctx, crt.TenantID)
		if err != nil {
			return fmt.Errorf("pki: failed to get root CA: %w", err)
		}
	} else {
		issuerCert, err = h.repo.GetByID(ctx, *issuerID)
		if err != nil {
			return fmt.Errorf("pki: failed to get issuer: %w", err)
		}
	}

	// Parse issuer certificate and key
	issuerCertParsed, issuerPrivateKey, err := h.parseCertificateAndKey(issuerCert)
	if err != nil {
		return fmt.Errorf("pki: failed to parse issuer: %w", err)
	}

	// Generate key pair for intermediate CA
	privateKey, err := h.generateKey()
	if err != nil {
		return fmt.Errorf("pki: failed to generate key: %w", err)
	}

	// Create certificate template
	serialNumber, err := h.generateSerialNumber()
	if err != nil {
		return fmt.Errorf("pki: failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         subject.CommonName,
			Organization:       []string{subject.Organization},
			OrganizationalUnit: []string{subject.OrganizationalUnit},
			Country:            []string{subject.Country},
			Locality:           []string{subject.Locality},
			Province:           []string{subject.Province},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(h.config.IntermediateCADuration),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
		DNSNames:              []string{subject.CommonName},
	}

	// Sign with issuer
	certBytes, err := x509.CreateCertificate(rand.Reader, template, issuerCertParsed, h.publicKey(privateKey), issuerPrivateKey)
	if err != nil {
		return fmt.Errorf("pki: failed to create certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM, err := h.encodePrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("pki: failed to encode private key: %w", err)
	}

	// Update certificate record
	crt.Type = cert.TypeIntermediateCA
	crt.Status = cert.StatusActive
	crt.PEMCertificate = certPEM
	crt.PEMPrivateKey = keyPEM
	crt.SerialNumber = serialNumber.String()
	crt.Subject = subject.CommonName
	crt.NotBefore = template.NotBefore
	crt.NotAfter = template.NotAfter
	crt.IssuerID = &issuerCert.ID
	crt.KeyUsage = []string{"cert_sign", "crl_sign"}
	crt.ExtKeyUsage = []string{"server_auth", "client_auth"}

	return nil
}

// IssueLeaf issues a leaf certificate
func (h *Hierarchy) IssueLeaf(ctx context.Context, leafCert *cert.Certificate, subject cert.CertificateSubject, issuerID uuid.UUID) error {
	// Get issuer certificate
	issuerCert, err := h.repo.GetByID(ctx, issuerID)
	if err != nil {
		return fmt.Errorf("pki: failed to get issuer: %w", err)
	}

	// Parse issuer certificate and key
	issuerCertParsed, issuerPrivateKey, err := h.parseCertificateAndKey(issuerCert)
	if err != nil {
		return fmt.Errorf("pki: failed to parse issuer: %w", err)
	}

	// Generate key pair for leaf certificate
	privateKey, err := h.generateKey()
	if err != nil {
		return fmt.Errorf("pki: failed to generate key: %w", err)
	}

	// Create certificate template
	serialNumber, err := h.generateSerialNumber()
	if err != nil {
		return fmt.Errorf("pki: failed to generate serial number: %w", err)
	}

	// Parse key usage and extended key usage
	keyUsage := h.parseKeyUsage(leafCert.KeyUsage)
	extKeyUsage := h.parseExtKeyUsage(leafCert.ExtKeyUsage)

	// Build subject name with slices
	var org, ou, country []string
	if subject.Organization != "" {
		org = []string{subject.Organization}
	}
	if subject.OrganizationalUnit != "" {
		ou = []string{subject.OrganizationalUnit}
	}
	if subject.Country != "" {
		country = []string{subject.Country}
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         subject.CommonName,
			Organization:       org,
			OrganizationalUnit: ou,
			Country:            country,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(h.config.LeafCertDuration),
		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              leafCert.DNSNames,
		IPAddresses:           h.parseIPAddresses(leafCert.IPAddresses),
	}

	// Sign with issuer
	certBytes, err := x509.CreateCertificate(rand.Reader, template, issuerCertParsed, h.publicKey(privateKey), issuerPrivateKey)
	if err != nil {
		return fmt.Errorf("pki: failed to create certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM, err := h.encodePrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("pki: failed to encode private key: %w", err)
	}

	// Update certificate record
	leafCert.Type = cert.TypeLeaf
	leafCert.Status = cert.StatusActive
	leafCert.PEMCertificate = certPEM
	leafCert.PEMPrivateKey = keyPEM
	leafCert.SerialNumber = serialNumber.String()
	leafCert.Subject = subject.CommonName
	leafCert.NotBefore = template.NotBefore
	leafCert.NotAfter = template.NotAfter
	leafCert.IssuerID = &issuerCert.ID

	return nil
}

// AddToCRL adds a revoked certificate to the CRL
func (h *Hierarchy) AddToCRL(ctx context.Context, cert *cert.Certificate) error {
	// In production, this would:
	// 1. Parse the certificate
	// 2. Add it to the CRL maintained by the issuing CA
	// 3. Sign the CRL
	// 4. Store the CRL in a location accessible by clients

	h.logger.Info().
		Str("certificate_id", cert.ID.String()).
		Str("serial_number", cert.SerialNumber).
		Msg("Certificate added to CRL")

	return nil
}

// GetCRL returns the CRL for a CA certificate
func (h *Hierarchy) GetCRL(ctx context.Context, caCertID uuid.UUID) ([]byte, error) {
	// In production, this would retrieve and return the CRL
	// For now, return a placeholder
	return nil, fmt.Errorf("pki: CRL not implemented")
}

// ValidateChain validates a certificate chain
func (h *Hierarchy) ValidateChain(ctx context.Context, chain []*cert.Certificate) error {
	if len(chain) == 0 {
		return fmt.Errorf("pki: empty certificate chain")
	}

	// Validate each certificate in the chain
	for i := 0; i < len(chain); i++ {
		cert := chain[i]

		// Parse certificate
		block, _ := pem.Decode(cert.PEMCertificate)
		if block == nil {
			return fmt.Errorf("pki: failed to decode certificate %d", i)
		}

		parsedCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("pki: failed to parse certificate %d: %w", i, err)
		}

		// Check expiration
		now := time.Now()
		if now.Before(parsedCert.NotBefore) {
			return fmt.Errorf("pki: certificate %d not yet valid", i)
		}
		if now.After(parsedCert.NotAfter) {
			return fmt.Errorf("pki: certificate %d expired", i)
		}

		// Verify signature with issuer (except for root)
		if i < len(chain)-1 {
			issuer := chain[i+1]
			issuerBlock, _ := pem.Decode(issuer.PEMCertificate)
			if issuerBlock == nil {
				return fmt.Errorf("pki: failed to decode issuer certificate for %d", i)
			}

			issuerCert, err := x509.ParseCertificate(issuerBlock.Bytes)
			if err != nil {
				return fmt.Errorf("pki: failed to parse issuer certificate for %d: %w", i, err)
			}

			if err := parsedCert.CheckSignatureFrom(issuerCert); err != nil {
				return fmt.Errorf("pki: certificate %d signature verification failed: %w", i, err)
			}
		}
	}

	return nil
}

// generateKey generates a new key pair
func (h *Hierarchy) generateKey() (crypto.PrivateKey, error) {
	switch KeyType(h.config.KeyType) {
	case KeyTypeRSA:
		// Implementation would use crypto/rsa
		return nil, fmt.Errorf("pki: RSA key generation not implemented")
	case KeyTypeECDSA:
		return ecdsaGenerateKey()
	default:
		return nil, fmt.Errorf("pki: unsupported key type: %s", h.config.KeyType)
	}
}

// publicKey extracts the public key from a private key
func (h *Hierarchy) publicKey(privateKey crypto.PrivateKey) crypto.PublicKey {
	switch k := privateKey.(type) {
	case interface {
		Public() crypto.PublicKey
	}:
		return k.Public()
	default:
		return nil
	}
}

// encodePrivateKey encodes a private key to PEM format
func (h *Hierarchy) encodePrivateKey(privateKey crypto.PrivateKey) ([]byte, error) {
	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		bytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: bytes}), nil
	default:
		return nil, fmt.Errorf("pki: unsupported private key type")
	}
}

// parseCertificateAndKey parses a certificate and its private key
func (h *Hierarchy) parseCertificateAndKey(cert *cert.Certificate) (*x509.Certificate, crypto.PrivateKey, error) {
	// Parse certificate
	certBlock, _ := pem.Decode(cert.PEMCertificate)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("pki: failed to decode certificate")
	}

	parsedCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("pki: failed to parse certificate: %w", err)
	}

	// Parse private key
	keyBlock, _ := pem.Decode(cert.PEMPrivateKey)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("pki: failed to decode private key")
	}

	parsedKey, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("pki: failed to parse private key: %w", err)
	}

	return parsedCert, parsedKey, nil
}

// generateSerialNumber generates a random serial number
func (h *Hierarchy) generateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}

// parseKeyUsage parses string key usage to x509.KeyUsage
func (h *Hierarchy) parseKeyUsage(usages []string) x509.KeyUsage {
	var usage x509.KeyUsage
	for _, u := range usages {
		switch u {
		case "digital_signature":
			usage |= x509.KeyUsageDigitalSignature
		case "content_commitment":
			usage |= x509.KeyUsageContentCommitment
		case "key_encipherment":
			usage |= x509.KeyUsageKeyEncipherment
		case "data_encipherment":
			usage |= x509.KeyUsageDataEncipherment
		case "key_agreement":
			usage |= x509.KeyUsageKeyAgreement
		case "cert_sign":
			usage |= x509.KeyUsageCertSign
		case "crl_sign":
			usage |= x509.KeyUsageCRLSign
		case "encipher_only":
			usage |= x509.KeyUsageEncipherOnly
		case "decipher_only":
			usage |= x509.KeyUsageDecipherOnly
		}
	}
	return usage
}

// parseExtKeyUsage parses string extended key usage to x509.ExtKeyUsage
func (h *Hierarchy) parseExtKeyUsage(usages []string) []x509.ExtKeyUsage {
	var extUsage []x509.ExtKeyUsage
	for _, u := range usages {
		switch u {
		case "server_auth":
			extUsage = append(extUsage, x509.ExtKeyUsageServerAuth)
		case "client_auth":
			extUsage = append(extUsage, x509.ExtKeyUsageClientAuth)
		case "code_signing":
			extUsage = append(extUsage, x509.ExtKeyUsageCodeSigning)
		case "email_protection":
			extUsage = append(extUsage, x509.ExtKeyUsageEmailProtection)
		case "ipsec_end_system":
			extUsage = append(extUsage, x509.ExtKeyUsageIPSECEndSystem)
		case "ipsec_tunnel":
			extUsage = append(extUsage, x509.ExtKeyUsageIPSECTunnel)
		case "ipsec_user":
			extUsage = append(extUsage, x509.ExtKeyUsageIPSECUser)
		case "timestamping":
			extUsage = append(extUsage, x509.ExtKeyUsageTimeStamping)
		case "ocsp_signing":
			extUsage = append(extUsage, x509.ExtKeyUsageOCSPSigning)
		}
	}
	return extUsage
}

// parseIPAddresses parses IP addresses from strings
func (h *Hierarchy) parseIPAddresses(ips []string) []net.IP {
	var result []net.IP
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			result = append(result, ip)
		}
	}
	return result
}

// ecdsaGenerateKey generates an ECDSA key pair
func ecdsaGenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
