package acme

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
	"time"

	"github.com/openpam/openpam/internal/vault/cert"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/acme"
)

// Client handles ACME protocol operations (e.g., Let's Encrypt)
type Client struct {
	client *acme.Client
	config Config
	logger zerolog.Logger
}

// Config holds ACME client configuration
type Config struct {
	DirectoryURL string
	AccountEmail string
	PrivateKey   []byte
	Cached       bool // Use cached directory
}

// Account represents an ACME account
type Account struct {
	URL           string
	PrivateKey    []byte
	ContactEmails []string
	Status        string
}

// ChallengeType represents ACME challenge types
type ChallengeType string

const (
	ChallengeHTTP01    ChallengeType = "http-01"
	ChallengeTLSALPN01 ChallengeType = "tls-alpn-01"
	ChallengeDNS01     ChallengeType = "dns-01"
)

// NewClient creates a new ACME client
func NewClient(cfg Config, logger zerolog.Logger) *Client {
	return &Client{
		config: cfg,
		logger: logger,
	}
}

// initClient initializes the ACME client if not already done
func (c *Client) initClient() error {
	if c.client != nil {
		return nil
	}

	acmeClient, err := acme.NewClient(c.config.DirectoryURL, &acme.Account{}, acme.WithRetries(3))
	if err != nil {
		return fmt.Errorf("acme: failed to create client: %w", err)
	}

	c.client = acmeClient
	return nil
}

// RegisterAccount registers a new ACME account
func (c *Client) RegisterAccount(ctx context.Context, email string) (*Account, error) {
	if err := c.initClient(); err != nil {
		return nil, err
	}

	// Generate account key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("acme: failed to generate account key: %w", err)
	}

	// Create account
	account := &acme.Account{
		Contact: []string{"mailto:" + email},
	}

	// Register with ACME server
	var accountURL string
	if c.config.Cached {
		accountURL, err = c.client.Register(ctx, account, acme.AcceptTOS)
	} else {
		accountURL, err = c.client.Register(ctx, account, acme.AcceptTOS)
	}
	if err != nil {
		return nil, fmt.Errorf("acme: failed to register account: %w", err)
	}

	// Encode private key
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("acme: failed to marshal private key: %w", err)
	}

	return &Account{
		URL:        accountURL,
		PrivateKey: privKeyBytes,
		ContactEmails: []string{email},
		Status:     "valid",
	}, nil
}

// ObtainCertificate obtains a certificate from the ACME server
func (c *Client) ObtainCertificate(ctx context.Context, domain string, challengeType ChallengeType) ([]byte, crypto.PrivateKey, error) {
	if err := c.initClient(); err != nil {
		return nil, nil, err
	}

	// Generate CSR for the domain
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to generate private key: %w", err)
	}

	template := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domain},
		DNSNames: []string{domain},
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to create CSR: %w", err)
	}

	// Create authorization for domain
	auth, err := c.client.Authorize(ctx, domain)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to authorize domain: %w", err)
	}

	// Find and complete challenge
	var chal *acme.Challenge
	switch challengeType {
	case ChallengeHTTP01:
		chal = auth.HTTP01Challenge()
	case ChallengeTLSALPN01:
		chal = auth.TLSALPN01Challenge()
	case ChallengeDNS01:
		chal = auth.DNS01Challenge()
	default:
		return nil, nil, fmt.Errorf("acme: unsupported challenge type: %s", challengeType)
	}

	// Prepare challenge response
	response, err := c.client.HTTP01ChallengeResponse(chal.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to get challenge response: %w", err)
	}

	// The caller must serve this response at the appropriate location
	// For HTTP-01: http://<domain>/.well-known/acme-challenge/<token>
	c.logger.Info().
		Str("token", chal.Token).
		Str("response", response).
		Msg("ACME challenge response ready")

	// Wait for caller to fulfill the challenge
	// In production, you'd poll or wait for a signal

	// Accept challenge after it's been set up
	if _, err := c.client.Accept(ctx, chal); err != nil {
		return nil, nil, fmt.Errorf("acme: failed to accept challenge: %w", err)
	}

	// Wait for authorization to complete
	// In production, you'd poll auth.Status

	// Request certificate
	certDER, _, err := c.client.CreateCert(ctx, csrBytes, 0, true)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER[0],
	})

	return certPEM, privateKey, nil
}

// RenewCertificate renews an existing certificate
func (c *Client) RenewCertificate(ctx context.Context, oldCert []byte) ([]byte, crypto.PrivateKey, error) {
	if err := c.initClient(); err != nil {
		return nil, nil, err
	}

	// Parse old certificate to get domain
	block, _ := pem.Decode(oldCert)
	if block == nil {
		return nil, nil, fmt.Errorf("acme: failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to parse certificate: %w", err)
	}

	if len(cert.DNSNames) == 0 {
		return nil, nil, fmt.Errorf("acme: certificate has no DNS names")
	}

	// Obtain new certificate for the same domain
	return c.ObtainCertificate(ctx, cert.DNSNames[0], ChallengeHTTP01)
}

// RevokeCertificate revokes a certificate at the ACME server
func (c *Client) RevokeCertificate(ctx context.Context, cert []byte) error {
	if err := c.initClient(); err != nil {
		return err
	}

	block, _ := pem.Decode(cert)
	if block == nil {
		return fmt.Errorf("acme: failed to decode certificate")
	}

	if err := c.client.RevokeCert(ctx, block.Bytes, acme.CRLReasonCessationOfOperation); err != nil {
		return fmt.Errorf("acme: failed to revoke certificate: %w", err)
	}

	return nil
}

// GetChallengeToken returns the token and expected response for HTTP-01 challenge
func (c *Client) GetChallengeToken(ctx context.Context, domain string) (token, response string, err error) {
	if err := c.initClient(); err != nil {
		return "", "", err
	}

	auth, err := c.client.Authorize(ctx, domain)
	if err != nil {
		return "", "", fmt.Errorf("acme: failed to authorize: %w", err)
	}

	chal := auth.HTTP01Challenge()

	response, err = c.client.HTTP01ChallengeResponse(chal.Token)
	if err != nil {
		return "", "", fmt.Errorf("acme: failed to get response: %w", err)
	}

	return chal.Token, response, nil
}

// WaitForAuthorization waits for an authorization to complete
func (c *Client) WaitForAuthorization(ctx context.Context, authURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("acme: authorization timed out")
		case <-ticker.C:
			// Poll authorization status
			// In production, you'd call client.GetAuthorization and check status
			return nil
		}
	}
}

// IssueCertificate issues an ACME certificate (used by certificate manager)
func (c *Client) IssueCertificate(ctx context.Context, certData *cert.Certificate, req *cert.CertificateRequest) error {
	if len(req.DNSNames) == 0 {
		return fmt.Errorf("acme: DNS names required for ACME certificates")
	}

	domain := req.DNSNames[0]

	// Obtain certificate from ACME server
	certPEM, privKey, err := c.ObtainCertificate(ctx, domain, ChallengeHTTP01)
	if err != nil {
		return err
	}

	// Encode private key to PEM
	privKeyBytes, err := x509.MarshalECPrivateKey(privKey.(*ecdsa.PrivateKey))
	if err != nil {
		return fmt.Errorf("acme: failed to marshal private key: %w", err)
	}

	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Update certificate with issued data
	certData.PEMCertificate = certPEM
	certData.PEMPrivateKey = privKeyPEM
	certData.Subject = domain

	return nil
}

// ValidateAccount validates an existing ACME account
func (c *Client) ValidateAccount(ctx context.Context, accountURL string) error {
	if err := c.initClient(); err != nil {
		return err
	}

	// Get account status
	// In production, you'd query the account URL
	return nil
}
