package crypto

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)
	require.Len(t, key, KeySize, "Generated key should be 32 bytes")

	// Verify keys are unique
	key2, err := GenerateKey()
	require.NoError(t, err)
	require.NotEqual(t, key, key2, "Keys should be unique")
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce()
	require.NoError(t, err)
	require.Len(t, nonce, NonceSize, "Generated nonce should be 12 bytes")

	// Verify nonces are unique
	nonce2, err := GenerateNonce()
	require.NoError(t, err)
	require.NotEqual(t, nonce, nonce2, "Nonces should be unique")
}

func TestEncryptor_NewEncryptor(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
		wantErr bool
	}{
		{"valid key", KeySize, false},
		{"key too short", 16, true},
		{"key too long", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			encryptor, err := NewEncryptor(key)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
			}
		})
	}
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, KeySize)
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"small data", []byte("hello")},
		{"medium data", []byte("The quick brown fox jumps over the lazy dog")},
		{"large data", make([]byte, 1024*10)},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.Encrypt(tt.data)
			require.NoError(t, err)

			// Ciphertext should be different from plaintext
			assert.NotEqual(t, tt.data, ciphertext)

			// Ciphertext should be longer (nonce + tag + data)
			assert.GreaterOrEqual(t, len(ciphertext), len(tt.data)+NonceSize+TagSize)

			// Decrypt
			plaintext, err := encryptor.Decrypt(ciphertext)
			require.NoError(t, err)

			// Should recover original data
			// For empty input, plaintext will be nil, which is equivalent to empty byte slice
			if len(tt.data) == 0 {
				assert.Nil(t, plaintext)
			} else {
				assert.Equal(t, tt.data, plaintext)
			}
		})
	}
}

func TestEncryptor_EncryptString(t *testing.T) {
	key := make([]byte, KeySize)
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name string
		data string
	}{
		{"empty string", ""},
		{"simple text", "hello world"},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "Hello 世界 🌍"},
		{"long string", string(make([]byte, 1000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := encryptor.EncryptString(tt.data)
			require.NoError(t, err)

			// Should be valid base64
			_, err = base64.StdEncoding.DecodeString(ciphertext)
			assert.NoError(t, err)

			// Decrypt
			plaintext, err := encryptor.DecryptString(ciphertext)
			require.NoError(t, err)

			// Should recover original
			assert.Equal(t, tt.data, plaintext)
		})
	}
}

func TestEncryptor_Decrypt_Invalid(t *testing.T) {
	key := make([]byte, KeySize)
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{"too short", []byte{0x01, 0x02}},
		{"just nonce", make([]byte, NonceSize)},
		{"modified ciphertext", func() []byte {
			ct, _ := encryptor.Encrypt([]byte("test"))
			ct[0] ^= 0xFF // Modify first byte
			return ct
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestEncryptor_DecryptString_Invalid(t *testing.T) {
	key := make([]byte, KeySize)
	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name string
		data string
	}{
		{"invalid base64", "!!!invalid!!!"},
		{"empty", ""},
		{"random data", base64.StdEncoding.EncodeToString([]byte("too short"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.DecryptString(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestEnvelopeEncryption_NewEnvelopeEncryption(t *testing.T) {
	tests := []struct {
		name     string
		keySize  int
		wantErr  bool
	}{
		{"valid key", KeySize, false},
		{"key too short", 16, true},
		{"key too long", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			ee, err := NewEnvelopeEncryption(key)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ee)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ee)
			}
		})
	}
}

func TestEnvelopeEncryption_GenerateDEK(t *testing.T) {
	key := make([]byte, KeySize)
	ee, err := NewEnvelopeEncryption(key)
	require.NoError(t, err)

	dek, err := ee.GenerateDEK()
	require.NoError(t, err)
	require.Len(t, dek, KeySize, "DEK should be 32 bytes")

	// Verify uniqueness
	dek2, err := ee.GenerateDEK()
	require.NoError(t, err)
	require.NotEqual(t, dek, dek2, "DEKs should be unique")
}

func TestEnvelopeEncryption_EncryptDecryptDEK(t *testing.T) {
	key := make([]byte, KeySize)
	ee, err := NewEnvelopeEncryption(key)
	require.NoError(t, err)

	dek, err := ee.GenerateDEK()
	require.NoError(t, err)

	// Encrypt DEK
	encryptedDEK, err := ee.EncryptDEK(dek)
	require.NoError(t, err)
	require.NotEqual(t, dek, encryptedDEK)

	// Decrypt DEK
	decryptedDEK, err := ee.DecryptDEK(encryptedDEK)
	require.NoError(t, err)
	assert.Equal(t, dek, decryptedDEK)
}

func TestEnvelopeEncryption_EncryptDecrypt(t *testing.T) {
	key := make([]byte, KeySize)
	ee, err := NewEnvelopeEncryption(key)
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("test")},
		{"large", make([]byte, 1024*5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := ee.Encrypt(tt.data)
			require.NoError(t, err)
			require.NotNil(t, encrypted)
			assert.NotEmpty(t, encrypted.EncryptedDEK)
			assert.NotEmpty(t, encrypted.Nonce)
			assert.NotEmpty(t, encrypted.Ciphertext)

			// Decrypt
			decrypted, err := ee.Decrypt(encrypted)
			require.NoError(t, err)
			// For empty input, decrypted will be nil, which is equivalent to empty byte slice
			if len(tt.data) == 0 {
				assert.Nil(t, decrypted)
			} else {
				assert.Equal(t, tt.data, decrypted)
			}
		})
	}
}

func TestEncryptedData_MarshalUnmarshal(t *testing.T) {
	key := make([]byte, KeySize)
	ee, err := NewEnvelopeEncryption(key)
	require.NoError(t, err)

	data := []byte("test data for marshaling")

	// Encrypt
	encrypted, err := ee.Encrypt(data)
	require.NoError(t, err)

	// Marshal
	marshaled, err := encrypted.Marshal()
	require.NoError(t, err)
	assert.Contains(t, marshaled, ".") // Should have dots as separators

	// Unmarshal
	unmarshaled, err := Unmarshal(marshaled)
	require.NoError(t, err)
	assert.Equal(t, encrypted.EncryptedDEK, unmarshaled.EncryptedDEK)
	assert.Equal(t, encrypted.Nonce, unmarshaled.Nonce)
	assert.Equal(t, encrypted.Ciphertext, unmarshaled.Ciphertext)

	// Decrypt unmarshaled data
	decrypted, err := ee.Decrypt(unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}

func TestUnmarshal_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"empty", ""},
		{"missing parts", "only.one"},
		{"invalid base64", "invalid.base64.data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Unmarshal(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestKeyDerivation_DeriveKey(t *testing.T) {
	secret := []byte("test-secret")
	salt := []byte("test-salt")
	kd := NewKeyDerivation(secret, salt)

	tests := []struct {
		name   string
		info   []byte
		length int
	}{
		{"32 bytes", []byte("info-1"), 32},
		{"64 bytes", []byte("info-2"), 64},
		{"16 bytes", []byte("info-3"), 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := kd.DeriveKey(tt.info, tt.length)
			require.NoError(t, err)
			require.Len(t, key, tt.length)

			// Same input should produce same key
			key2, err := kd.DeriveKey(tt.info, tt.length)
			require.NoError(t, err)
			assert.Equal(t, key, key2)

			// Different info should produce different key
			if tt.info != nil {
				key3, err := kd.DeriveKey([]byte(string(tt.info)+"-different"), tt.length)
				require.NoError(t, err)
				assert.NotEqual(t, key, key3)
			}
		})
	}
}

func TestKeyDerivation_DeriveKey_Invalid(t *testing.T) {
	kd := NewKeyDerivation([]byte("secret"), []byte("salt"))

	_, err := kd.DeriveKey([]byte("info"), 0)
	assert.Error(t, err)

	_, err = kd.DeriveKey([]byte("info"), -1)
	assert.Error(t, err)
}

func TestHashPassword(t *testing.T) {
	password := "MySecurePassword123!"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.Contains(t, hash, "$") // Should have separator
	assert.NotEqual(t, password, hash)

	// Different hashes for same password (due to random salt)
	hash2, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}

func TestVerifyPassword(t *testing.T) {
	password := "MySecurePassword123!"

	hash, err := HashPassword(password)
	require.NoError(t, err)

	// Correct password should verify
	assert.True(t, VerifyPassword(password, hash))

	// Wrong password should not verify
	assert.False(t, VerifyPassword("WrongPassword", hash))

	// Invalid hash format
	assert.False(t, VerifyPassword(password, "invalid"))

	// Empty hash
	assert.False(t, VerifyPassword(password, ""))
}

func TestVerifyPassword_ConstantTime(t *testing.T) {
	password := "password123"
	hash, _ := HashPassword(password)

	// This test ensures verification is constant-time
	// by measuring multiple verifications
	durations := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		start := time.Now()
		VerifyPassword(password, hash)
		durations[i] = time.Since(start)
	}

	// Check that durations are relatively consistent
	// (within 10x of each other)
	min := durations[0]
	max := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	assert.Less(t, max, min*10, "Timing variance too high, may not be constant-time")
}

func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)
	require.NotNil(t, keyPair.PrivateKey)
	require.NotNil(t, keyPair.PublicKey)

	// Verify it's a P-256 key
	assert.Equal(t, keyPair.PrivateKey.Curve.Params().BitSize, 256)
}

func TestMarshalPrivateKey(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	pem, err := MarshalPrivateKey(keyPair.PrivateKey)
	require.NoError(t, err)
	assert.Contains(t, string(pem), "EC PRIVATE KEY")
	assert.Contains(t, string(pem), "-----BEGIN")
	assert.Contains(t, string(pem), "-----END")
}

func TestMarshalPublicKey(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	pem, err := MarshalPublicKey(keyPair.PublicKey)
	require.NoError(t, err)
	assert.Contains(t, string(pem), "PUBLIC KEY")
	assert.Contains(t, string(pem), "-----BEGIN")
	assert.Contains(t, string(pem), "-----END")
}

func TestParsePrivateKey(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	pem, err := MarshalPrivateKey(keyPair.PrivateKey)
	require.NoError(t, err)

	parsed, err := ParsePrivateKey(pem)
	require.NoError(t, err)
	assert.Equal(t, keyPair.PrivateKey.D, parsed.D)
}

func TestParsePublicKey(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	require.NoError(t, err)

	pem, err := MarshalPublicKey(keyPair.PublicKey)
	require.NoError(t, err)

	parsed, err := ParsePublicKey(pem)
	require.NoError(t, err)
	assert.Equal(t, keyPair.PublicKey.X, parsed.X)
	assert.Equal(t, keyPair.PublicKey.Y, parsed.Y)
}

func TestParsePrivateKey_Invalid(t *testing.T) {
	testKeyPair, _ := GenerateKeyPair()

	tests := []struct {
		name string
		pem  []byte
	}{
		{"empty", []byte{}},
		{"invalid", []byte("not a pem")},
		{"wrong type", func() []byte {
			pem, _ := MarshalPublicKey(&testKeyPair.PrivateKey.PublicKey)
			return pem
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePrivateKey(tt.pem)
			assert.Error(t, err)
		})
	}
}

func TestParsePublicKey_Invalid(t *testing.T) {
	testKeyPair, _ := GenerateKeyPair()

	tests := []struct {
		name string
		pem  []byte
	}{
		{"empty", []byte{}},
		{"invalid", []byte("not a pem")},
		{"wrong type", func() []byte {
			pem, _ := MarshalPrivateKey(testKeyPair.PrivateKey)
			return pem
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePublicKey(tt.pem)
			assert.Error(t, err)
		})
	}
}

func TestSecureRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"8 chars", 8},
		{"16 chars", 16},
		{"32 chars", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, err := SecureRandomString(tt.length)
			require.NoError(t, err)
			assert.Len(t, str, tt.length)

			// Should be unique
			str2, err := SecureRandomString(tt.length)
			require.NoError(t, err)
			assert.NotEqual(t, str, str2)
		})
	}
}

func TestRotateMasterKey(t *testing.T) {
	oldKey := make([]byte, KeySize)
	oldKey[0] = 0x01 // Make it different from new key
	newKey := make([]byte, KeySize)
	newKey[0] = 0x02 // Make it different from old key
	ee, err := NewEnvelopeEncryption(oldKey)
	require.NoError(t, err)

	// Create some encrypted DEKs
	var encryptedDEKs [][]byte
	for i := 0; i < 5; i++ {
		dek, _ := ee.GenerateDEK()
		encDEK, _ := ee.EncryptDEK(dek)
		encryptedDEKs = append(encryptedDEKs, encDEK)
	}

	// Rotate
	reencrypted, err := ee.RotateMasterKey(newKey, encryptedDEKs)
	require.NoError(t, err)
	assert.Len(t, reencrypted, len(encryptedDEKs))

	// Verify new DEKs can be decrypted with new key
	newEE, _ := NewEnvelopeEncryption(newKey)
	for _, reenc := range reencrypted {
		dek, err := newEE.DecryptDEK(reenc)
		assert.NoError(t, err)
		assert.Len(t, dek, KeySize)
	}

	// Old DEKs should not decrypt with new key
	for _, oldEnc := range encryptedDEKs {
		_, err := newEE.DecryptDEK(oldEnc)
		assert.Error(t, err)
	}
}
