package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	// KeySize is the AES-256 key size in bytes
	KeySize = 32
	// NonceSize is the GCM nonce size
	NonceSize = 12
	// TagSize is the GCM tag size
	TagSize = 16
)

// Encryptor handles AES-256-GCM encryption
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("crypto: invalid key size %d, expected %d", len(key), KeySize)
	}
	return &Encryptor{key: key}, nil
}

// GenerateKey generates a random AES-256 key
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("crypto.GenerateKey: %w", err)
	}
	return key, nil
}

// GenerateNonce generates a random nonce for GCM
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto.GenerateNonce: %w", err)
	}
	return nonce, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
// Format: nonce(12) + ciphertext + tag(16)
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewGCM: %w", err)
	}

	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Seal appends the tag to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize+TagSize {
		return nil, fmt.Errorf("crypto: ciphertext too short")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewGCM: %w", err)
	}

	nonce := ciphertext[:NonceSize]
	ct := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto.GCMOpen: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded result
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a base64-encoded string
func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("crypto.Base64Decode: %w", err)
	}
	plaintext, err := e.Decrypt(data)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EnvelopeEncryption implements envelope encryption (master key → DEK → data)
type EnvelopeEncryption struct {
	masterKey []byte
}

// NewEnvelopeEncryption creates a new envelope encryption handler
func NewEnvelopeEncryption(masterKey []byte) (*EnvelopeEncryption, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("crypto: invalid master key size %d, expected %d", len(masterKey), KeySize)
	}
	return &EnvelopeEncryption{masterKey: masterKey}, nil
}

// EncryptedData represents envelope-encrypted data
type EncryptedData struct {
	EncryptedDEK []byte // DEK encrypted with master key
	Nonce        []byte // GCM nonce
	Ciphertext   []byte // Data encrypted with DEK
}

// GenerateDEK generates a new Data Encryption Key
func (ee *EnvelopeEncryption) GenerateDEK() ([]byte, error) {
	return GenerateKey()
}

// EncryptDEK encrypts a DEK with the master key
func (ee *EnvelopeEncryption) EncryptDEK(dek []byte) ([]byte, error) {
	encryptor, err := NewEncryptor(ee.masterKey)
	if err != nil {
		return nil, err
	}
	return encryptor.Encrypt(dek)
}

// DecryptDEK decrypts a DEK with the master key
func (ee *EnvelopeEncryption) DecryptDEK(encryptedDEK []byte) ([]byte, error) {
	encryptor, err := NewEncryptor(ee.masterKey)
	if err != nil {
		return nil, err
	}
	return encryptor.Decrypt(encryptedDEK)
}

// Encrypt encrypts data using envelope encryption
func (ee *EnvelopeEncryption) Encrypt(plaintext []byte) (*EncryptedData, error) {
	// Generate DEK
	dek, err := ee.GenerateDEK()
	if err != nil {
		return nil, err
	}

	// Encrypt DEK with master key
	encryptedDEK, err := ee.EncryptDEK(dek)
	if err != nil {
		return nil, err
	}

	// Encrypt data with DEK
	dataEncryptor, err := NewEncryptor(dek)
	if err != nil {
		return nil, err
	}

	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewGCM: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return &EncryptedData{
		EncryptedDEK: encryptedDEK,
		Nonce:        nonce,
		Ciphertext:   ciphertext,
	}, nil
}

// Decrypt decrypts envelope-encrypted data
func (ee *EnvelopeEncryption) Decrypt(data *EncryptedData) ([]byte, error) {
	// Decrypt DEK
	dek, err := ee.DecryptDEK(data.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("envelope.DecryptDEK: %w", err)
	}

	// Decrypt data with DEK
	dataEncryptor, err := NewEncryptor(dek)
	if err != nil {
		return nil, err
	}

	return dataEncryptor.Decrypt(data.Ciphertext)
}

// Marshal serializes EncryptedData to base64
func (ed *EncryptedData) Marshal() (string, error) {
	// Format: base64(encryptedDEK) + "." + base64(nonce) + "." + base64(ciphertext)
	encDEK := base64.StdEncoding.EncodeToString(ed.EncryptedDEK)
	nonce := base64.StdEncoding.EncodeToString(ed.Nonce)
	ct := base64.StdEncoding.EncodeToString(ed.Ciphertext)
	return fmt.Sprintf("%s.%s.%s", encDEK, nonce, ct), nil
}

// Unmarshal parses base64 string into EncryptedData
func Unmarshal(data string) (*EncryptedData, error) {
	parts := split(data, '.', 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("crypto: invalid envelope format")
	}

	encDEK, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("crypto.DecodeDEK: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("crypto.DecodeNonce: %w", err)
	}

	ct, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("crypto.DecodeCT: %w", err)
	}

	return &EncryptedData{
		EncryptedDEK: encDEK,
		Nonce:        nonce,
		Ciphertext:   ct,
	}, nil
}

func split(s string, sep rune, n int) []string {
	parts := make([]string, 0, n)
	current := ""
	count := 0
	for _, c := range s {
		if c == sep && count < n-1 {
			parts = append(parts, current)
			current = ""
			count++
		} else {
			current += string(c)
		}
	}
	parts = append(parts, current)
	return parts
}

// KeyDerivation implements HKDF key derivation
type KeyDerivation struct {
	secret []byte
	salt   []byte
}

// NewKeyDerivation creates a new KDF instance
func NewKeyDerivation(secret, salt []byte) *KeyDerivation {
	return &KeyDerivation{secret: secret, salt: salt}
}

// DeriveKey derives a key using HKDF-SHA256
func (kd *KeyDerivation) DeriveKey(info []byte, length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("crypto: invalid key length")
	}

	hkdf := hkdf.New(sha256.New, kd.secret, kd.salt, info)
	key := make([]byte, length)
	if _, err := io.ReadFull(hkdf, key); err != nil {
		return nil, fmt.Errorf("crypto.HKDF: %w", err)
	}
	return key, nil
}

// PasswordHash implements Argon2id password hashing
import (
	"golang.org/x/crypto/argon2"
)

// HashPassword hashes a password using Argon2id
func HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("crypto.GenerateSalt: %w", err)
	}

	// Argon2id parameters
	time := 3
	memory := 64 * 1024
	threads := 4
	keyLen := 32

	hash := argon2.IDKey([]byte(password), salt, uint32(time), uint32(memory), uint8(threads), uint32(keyLen))

	// Encode as: salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("%s$%s", b64Salt, b64Hash), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	parts := split(hash, '$', 2)
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	time := uint32(3)
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	testHash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	return subtle.ConstantTimeCompare(decodedHash, testHash) == 1
}

// KeyPair generates an ECDSA key pair
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// GenerateKeyPair generates a P-256 ECDSA key pair
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("crypto.GenerateKeyPair: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// MarshalPrivateKey marshals a private key to PEM
func MarshalPrivateKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.MarshalECPrivateKey: %w", err)
	}

	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}

	return pem.EncodeToMemory(block), nil
}

// MarshalPublicKey marshals a public key to PEM
func MarshalPublicKey(key *ecdsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.MarshalPKIXPublicKey: %w", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}

	return pem.EncodeToMemory(block), nil
}

// ParsePrivateKey parses a PEM-encoded private key
func ParsePrivateKey(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("crypto: failed to decode PEM block")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto.ParseECPrivateKey: %w", err)
	}

	return key, nil
}

// ParsePublicKey parses a PEM-encoded public key
func ParsePublicKey(pemData []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("crypto: failed to decode PEM block")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto.ParsePKIXPublicKey: %w", err)
	}

	ecKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("crypto: not an ECDSA public key")
	}

	return ecKey, nil
}

// SecureRandomString generates a cryptographically secure random string
func SecureRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("crypto.SecureRandom: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}

// RotateMasterKey re-encrypts all DEKs with a new master key
func (ee *EnvelopeEncryption) RotateMasterKey(newMasterKey []byte, encryptedDEKs [][]byte) ([][]byte, error) {
	newEE, err := NewEnvelopeEncryption(newMasterKey)
	if err != nil {
		return nil, err
	}

	reencrypted := make([][]byte, len(encryptedDEKs))
	for i, encDEK := range encryptedDEKs {
		// Decrypt DEK with old master key
		dek, err := ee.DecryptDEK(encDEK)
		if err != nil {
			return nil, fmt.Errorf("rotate.DecryptDEK[%d]: %w", i, err)
		}

		// Re-encrypt with new master key
		reencrypted[i], err = newEE.EncryptDEK(dek)
		if err != nil {
			return nil, fmt.Errorf("rotate.EncryptDEK[%d]: %w", i, err)
		}
	}

	return reencrypted, nil
}
