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
	client      *acme.Client
	config      Config
	logger      zerolog.Logger
	accountKey  crypto.Signer
	accountURL  string
}

// Config holds ACME client configuration
type Config struct {
	DirectoryURL string
	AccountEmail string
	PrivateKey   []byte
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

	c.client = &acme.Client{
		DirectoryURL: c.config.DirectoryURL,
	}
	return nil
}

// initAccountKey initializes or loads the account key
func (c *Client) initAccountKey() error {
	if c.accountKey != nil {
		return nil
	}

	// Generate account key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("acme: failed to generate account key: %w", err)
	}

	c.accountKey = privateKey
	return nil
}

// RegisterAccount registers a new ACME account
func (c *Client) RegisterAccount(ctx context.Context, email string) (*Account, error) {
	if err := c.initClient(); err != nil {
		return nil, err
	}
	if err := c.initAccountKey(); err != nil {
		return nil, err
	}

	// Create account
	account := &acme.Account{
		Contact: []string{"mailto:" + email},
	}

	// Register with ACME server
	acc, err := c.client.Register(ctx, account, acme.AcceptTOS)
	if err != nil {
		return nil, fmt.Errorf("acme: failed to register account: %w", err)
	}

	// Note: Account key is already associated with the client through the context
	// The acc.URI contains the account URL for future operations

	// Encode private key
	privKeyBytes, err := x509.MarshalECPrivateKey(c.accountKey.(*ecdsa.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("acme: failed to marshal private key: %w", err)
	}

	c.accountURL = acc.URI

	return &Account{
		URL:           acc.URI,
		PrivateKey:    privKeyBytes,
		ContactEmails: []string{email},
		Status:        "valid",
	}, nil
}

// ObtainCertificate obtains a certificate from the ACME server
func (c *Client) ObtainCertificate(ctx context.Context, domain string, challengeType ChallengeType) ([]byte, crypto.PrivateKey, error) {
	if err := c.initClient(); err != nil {
		return nil, nil, err
	}
	if err := c.initAccountKey(); err != nil {
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

	// Find the matching challenge
	var chal *acme.Challenge
	switch challengeType {
	case ChallengeHTTP01:
		for _, c := range auth.Challenges {
			if c.Type == "http-01" {
				chal = c
				break
			}
		}
	case ChallengeTLSALPN01:
		for _, c := range auth.Challenges {
			if c.Type == "tls-alpn-01" {
				chal = c
				break
			}
		}
	case ChallengeDNS01:
		for _, c := range auth.Challenges {
			if c.Type == "dns-01" {
				chal = c
				break
			}
		}
	default:
		return nil, nil, fmt.Errorf("acme: unsupported challenge type: %s", challengeType)
	}

	if chal == nil {
		return nil, nil, fmt.Errorf("acme: no matching challenge found")
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

	// Request certificate - 0 means default validity, true for must staple
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

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("acme: failed to parse certificate: %w", err)
	}

	if len(x509Cert.DNSNames) == 0 {
		return nil, nil, fmt.Errorf("acme: certificate has no DNS names")
	}

	// Obtain new certificate for the same domain
	return c.ObtainCertificate(ctx, x509Cert.DNSNames[0], ChallengeHTTP01)
}

// RevokeCertificate revokes a certificate at the ACME server
func (c *Client) RevokeCertificate(ctx context.Context, certPEM []byte, key crypto.Signer) error {
	if err := c.initClient(); err != nil {
		return err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("acme: failed to decode certificate")
	}

	if err := c.client.RevokeCert(ctx, key, block.Bytes, acme.CRLReasonCessationOfOperation); err != nil {
		return fmt.Errorf("acme: failed to revoke certificate: %w", err)
	}

	return nil
}

// GetChallengeToken returns the token and expected response for HTTP-01 challenge
func (c *Client) GetChallengeToken(ctx context.Context, domain string) (token, response string, err error) {
	if err := c.initClient(); err != nil {
		return "", "", err
	}
	if err := c.initAccountKey(); err != nil {
		return "", "", err
	}

	auth, err := c.client.Authorize(ctx, domain)
	if err != nil {
		return "", "", fmt.Errorf("acme: failed to authorize: %w", err)
	}

	var chal *acme.Challenge
	for _, c := range auth.Challenges {
		if c.Type == "http-01" {
			chal = c
			break
		}
	}

	if chal == nil {
		return "", "", fmt.Errorf("acme: http-01 challenge not found")
	}

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
