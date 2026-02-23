package credential

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_validateSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  *vault.Secret
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid secret",
			secret: &vault.Secret{
				Name:     "test-credential",
				Type:     vault.SecretTypePassword,
				TenantID: uuid.New(),
				Host:     "example.com",
				Port:     22,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			secret: &vault.Secret{
				Type:     vault.SecretTypePassword,
				TenantID: uuid.New(),
				Host:     "example.com",
				Port:     22,
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing type",
			secret: &vault.Secret{
				Name:     "test-credential",
				TenantID: uuid.New(),
				Host:     "example.com",
				Port:     22,
			},
			wantErr: true,
			errMsg:  "type is required",
		},
		{
			name: "missing tenant ID",
			secret: &vault.Secret{
				Name: "test-credential",
				Type: vault.SecretTypePassword,
				Host: "example.com",
				Port: 22,
			},
			wantErr: true,
			errMsg:  "tenant ID is required",
		},
		{
			name: "missing host",
			secret: &vault.Secret{
				Name:     "test-credential",
				Type:     vault.SecretTypePassword,
				TenantID: uuid.New(),
				Port:     22,
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "invalid port - zero",
			secret: &vault.Secret{
				Name:     "test-credential",
				Type:     vault.SecretTypePassword,
				TenantID: uuid.New(),
				Host:     "example.com",
				Port:     0,
			},
			wantErr: true,
			errMsg:  "port is required",
		},
		{
			name: "invalid port - negative",
			secret: &vault.Secret{
				Name:     "test-credential",
				Type:     vault.SecretTypePassword,
				TenantID: uuid.New(),
				Host:     "example.com",
				Port:     -1,
			},
			wantErr: true,
			errMsg:  "port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{logger: zerolog.Nop()}
			err := svc.validateSecret(tt.secret)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_TestCredential(t *testing.T) {
	t.Run("test credential panics with nil vault", func(t *testing.T) {
		svc := &Service{logger: zerolog.Nop()}
		ctx := context.Background()
		id := uuid.New()

		// Will panic because vault is nil
		assert.Panics(t, func() {
			_ = svc.TestCredential(ctx, id)
		})
	})
}

func TestCredentialHistoryEntry(t *testing.T) {
	t.Run("CredentialHistoryEntry struct", func(t *testing.T) {
		id := uuid.New()
		credentialID := uuid.New()
		rotatedBy := uuid.New()
		now := time.Now()

		entry := CredentialHistoryEntry{
			ID:           id,
			CredentialID: credentialID,
			RotatedAt:    now,
			RotatedBy:    rotatedBy,
			Method:       "manual",
			Success:      true,
		}

		assert.Equal(t, id, entry.ID)
		assert.Equal(t, credentialID, entry.CredentialID)
		assert.Equal(t, now, entry.RotatedAt)
		assert.Equal(t, rotatedBy, entry.RotatedBy)
		assert.Equal(t, "manual", entry.Method)
		assert.True(t, entry.Success)
		assert.Empty(t, entry.ErrorMsg)
	})

	t.Run("CredentialHistoryEntry with error", func(t *testing.T) {
		entry := CredentialHistoryEntry{
			Method:   "scheduled",
			Success:  false,
			ErrorMsg: "connection timeout",
		}

		assert.False(t, entry.Success)
		assert.Equal(t, "connection timeout", entry.ErrorMsg)
	})
}

func TestCredentialStats(t *testing.T) {
	t.Run("CredentialStats struct", func(t *testing.T) {
		id := uuid.New()
		userID1 := uuid.New()
		userID2 := uuid.New()
		now := time.Now()

		stats := CredentialStats{
			CredentialID:     id,
			CheckoutCount30d: 15,
			SessionCount30d:  42,
			LastUsedAt:       &now,
			TopUsers: []struct {
				UserID uuid.UUID `json:"user_id"`
				Count  int       `json:"count"`
			}{
				{UserID: userID1, Count: 10},
				{UserID: userID2, Count: 5},
			},
		}

		assert.Equal(t, id, stats.CredentialID)
		assert.Equal(t, 15, stats.CheckoutCount30d)
		assert.Equal(t, 42, stats.SessionCount30d)
		assert.NotNil(t, stats.LastUsedAt)
		assert.Len(t, stats.TopUsers, 2)
	})
}

func TestService_SyncCredential(t *testing.T) {
	svc := &Service{logger: zerolog.Nop()}
	ctx := context.Background()
	credentialID := uuid.New()
	targetID := uuid.New()

	t.Run("sync credential panics with nil vault", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = svc.SyncCredential(ctx, credentialID, targetID)
		})
	})
}

func TestRotationPolicies(t *testing.T) {
	t.Run("rotation policy constants", func(t *testing.T) {
		policies := []vault.RotationPolicy{
			vault.RotationManual,
			vault.RotationDaily,
			vault.RotationWeekly,
			vault.RotationMonthly,
		}

		for _, p := range policies {
			assert.NotEmpty(t, string(p), "policy should have string value")
		}
	})
}

func TestSecretTypes(t *testing.T) {
	t.Run("secret type constants", func(t *testing.T) {
		types := []vault.SecretType{
			vault.SecretTypePassword,
			vault.SecretTypeSSHKey,
			vault.SecretTypeAPIToken,
			vault.SecretTypeDatabase,
			vault.SecretTypeCertificate,
		}

		for _, st := range types {
			assert.NotEmpty(t, string(st), "type should have string value")
		}
	})
}

func TestCreateCredentialValidation(t *testing.T) {
	svc := &Service{
		vault:    nil, // Will be set to nil for validation testing
		rotation: nil,
		logger:   zerolog.Nop(),
	}

	ctx := context.Background()
	tenantID := uuid.New()

	validSecret := &vault.Secret{
		ID:        uuid.New(),
		Name:      "test-credential",
		Type:      vault.SecretTypePassword,
		TenantID:  tenantID,
		Host:      "example.com",
		Port:      22,
		Username:  "testuser",
	}

	plaintext := vault.SecretData{
		Password: "test-password-123",
	}

	t.Run("validation failure with nil vault", func(t *testing.T) {
		invalidSecret := &vault.Secret{
			Name: "",
		}

		err := svc.CreateCredential(ctx, invalidSecret, plaintext)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("vault store failure with nil vault", func(t *testing.T) {
		secret := *validSecret
		secret.ID = uuid.New()

		err := svc.CreateCredential(ctx, &secret, plaintext)
		require.Error(t, err)
	})
}

func TestUpdateCredentialValidation(t *testing.T) {
	svc := &Service{
		vault:  nil,
		logger: zerolog.Nop(),
	}

	ctx := context.Background()
	id := uuid.New()
	plaintext := vault.SecretData{Password: "new-password"}

	t.Run("update with nil vault panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = svc.UpdateCredential(ctx, id, plaintext)
		})
	})
}

func TestDeleteCredentialValidation(t *testing.T) {
	svc := &Service{
		vault:  nil,
		logger: zerolog.Nop(),
	}

	ctx := context.Background()
	id := uuid.New()

	t.Run("delete with nil vault panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = svc.DeleteCredential(ctx, id)
		})
	})
}

func TestRotateCredentialValidation(t *testing.T) {
	svc := &Service{
		vault:  nil,
		logger: zerolog.Nop(),
	}

	ctx := context.Background()
	id := uuid.New()
	newPlaintext := vault.SecretData{Password: "rotated-password"}

	t.Run("rotate with nil vault panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = svc.RotateCredential(ctx, id, newPlaintext)
		})
	})
}
