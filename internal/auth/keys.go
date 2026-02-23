package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
)

// LoadOrGenerateRSAKeys loads RSA keys from files or generates new ones
func LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath string, logger zerolog.Logger) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Try to load existing keys
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err == nil {
		publicKey, err := loadPublicKey(publicKeyPath)
		if err == nil {
			logger.Info().Str("private_key", privateKeyPath).Msg("Loaded existing RSA keys")
			return privateKey, publicKey, nil
		}
		logger.Warn().Msg("Private key found but public key missing, regenerating")
	}

	// Generate new keys
	logger.Info().Msg("Generating new RSA key pair")
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("auth.GenerateKey: %w", err)
	}

	// Save keys to files
	if err := savePrivateKey(privateKeyPath, privateKey); err != nil {
		logger.Warn().Err(err).Msg("Failed to save private key, using in-memory key")
	}

	if err := savePublicKey(publicKeyPath, &privateKey.PublicKey); err != nil {
		logger.Warn().Err(err).Msg("Failed to save public key, using in-memory key")
	}

	logger.Info().Str("private_key", privateKeyPath).Str("public_key", publicKeyPath).Msg("Generated new RSA keys")

	return privateKey, &privateKey.PublicKey, nil
}

// loadPrivateKey loads an RSA private key from a PEM file
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("auth.ReadFile: %w", err)
	}

	return crypto.ParseRSAPrivateKey(data)
}

// loadPublicKey loads an RSA public key from a PEM file
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("auth.ReadFile: %w", err)
	}

	return crypto.ParseRSAPublicKey(data)
}

// savePrivateKey saves an RSA private key to a PEM file
func savePrivateKey(path string, key *rsa.PrivateKey) error {
	data, err := crypto.MarshalRSAPrivateKey(key)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepathDir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("auth.MkdirAll: %w", err)
		}
	}

	return os.WriteFile(path, data, 0600)
}

// savePublicKey saves an RSA public key to a PEM file
func savePublicKey(path string, key *rsa.PublicKey) error {
	data, err := crypto.MarshalRSAPublicKey(key)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepathDir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("auth.MkdirAll: %w", err)
		}
	}

	return os.WriteFile(path, data, 0644)
}

func filepathDir(path string) string {
	i := len(path) - 1
	for i > 0 && path[i] != '/' {
		i--
	}
	return path[:i]
}
