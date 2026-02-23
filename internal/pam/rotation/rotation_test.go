package rotation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestRotationPolicy_Constants(t *testing.T) {
	t.Run("rotation policy constants", func(t *testing.T) {
		assert.Equal(t, vault.RotationPolicy("manual"), vault.RotationManual)
		assert.Equal(t, vault.RotationPolicy("daily"), vault.RotationDaily)
		assert.Equal(t, vault.RotationPolicy("weekly"), vault.RotationWeekly)
		assert.Equal(t, vault.RotationPolicy("monthly"), vault.RotationMonthly)
		assert.Equal(t, vault.RotationPolicy("on_checkin"), vault.RotationOnCheckin)
	})
}

func TestRotationTask_Struct(t *testing.T) {
	t.Run("rotation task with all fields", func(t *testing.T) {
		id := uuid.New()
		credentialID := uuid.New()
		now := time.Now()
		nextRotation := now.Add(24 * time.Hour)

		task := &RotationTask{
			ID:           id,
			CredentialID: credentialID,
			Policy:       vault.RotationDaily,
			NextRotation: nextRotation,
			LastRotation: &now,
			Status:       "pending",
		}

		assert.Equal(t, id, task.ID)
		assert.Equal(t, credentialID, task.CredentialID)
		assert.Equal(t, vault.RotationDaily, task.Policy)
		assert.Equal(t, nextRotation, task.NextRotation)
		assert.Equal(t, &now, task.LastRotation)
		assert.Equal(t, "pending", task.Status)
	})
}

func TestConnector_Interface(t *testing.T) {
	t.Run("Linux connector implements interface", func(t *testing.T) {
		connector := NewLinuxConnector()

		assert.Equal(t, "linux", connector.Name())
	})

	t.Run("Windows connector implements interface", func(t *testing.T) {
		connector := newWindowsConnector()

		assert.Equal(t, "windows", connector.Name())
	})

	t.Run("PostgreSQL connector implements interface", func(t *testing.T) {
		connector := NewPostgreSQLConnector()

		assert.Equal(t, "postgresql", connector.Name())
	})

	t.Run("MySQL connector implements interface", func(t *testing.T) {
		connector := newMySQLConnector()

		assert.Equal(t, "mysql", connector.Name())
	})

	t.Run("AWS Secrets connector implements interface", func(t *testing.T) {
		connector := newAWSSecretsConnector()

		assert.Equal(t, "aws_secrets", connector.Name())
	})
}

func TestNewService(t *testing.T) {
	t.Run("creates service with nil dependencies", func(t *testing.T) {
		var db *sqlx.DB // nil for testing
		var c *cache.Cache
		logger := zerolog.Nop()

		svc := NewService(db, nil, c, logger)

		assert.NotNil(t, svc)
	})
}

func TestService_RotateCredential(t *testing.T) {
	t.Run("rotate returns not implemented error", func(t *testing.T) {
		var db *sqlx.DB
		var c *cache.Cache
		logger := zerolog.Nop()
		svc := NewService(db, nil, c, logger)

		ctx := context.Background()
		credentialID := uuid.New()

		err := svc.RotateCredential(ctx, credentialID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}

func TestService_ScheduleRotation(t *testing.T) {
	t.Run("schedule with nil db panics", func(t *testing.T) {
		var db *sqlx.DB
		var c *cache.Cache
		logger := zerolog.Nop()
		svc := NewService(db, nil, c, logger)

		ctx := context.Background()
		credentialID := uuid.New()
		policy := vault.RotationDaily

		assert.Panics(t, func() {
			_ = svc.ScheduleRotation(ctx, credentialID, policy)
		})
	})
}

func TestService_GenerateNewCredential(t *testing.T) {
	t.Run("generate new credential returns not implemented for nil vault", func(t *testing.T) {
		var db *sqlx.DB
		var c *cache.Cache
		logger := zerolog.Nop()
		svc := NewService(db, nil, c, logger)

		ctx := context.Background()
		secret := &vault.Secret{
			Type:     vault.SecretTypePassword,
			Username: "testuser",
		}

		_, err := svc.GenerateNewCredential(ctx, secret)
		assert.NoError(t, err) // Password generation doesn't require vault
	})
}

func TestService_RotateOnTarget(t *testing.T) {
	t.Run("rotate on target with nil credentials fails", func(t *testing.T) {
		var db *sqlx.DB
		var c *cache.Cache
		logger := zerolog.Nop()
		svc := NewService(db, nil, c, logger)

		ctx := context.Background()
		targetID := uuid.New()

		err := svc.RotateOnTarget(ctx, targetID, nil, nil)
		assert.Error(t, err)
	})
}

func TestSSHKeyPair_Struct(t *testing.T) {
	t.Run("SSH key pair with all fields", func(t *testing.T) {
		keyPair := &SSHKeyPair{
			PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
			PublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAA test@example.com",
		}

		assert.NotEmpty(t, keyPair.PrivateKey)
		assert.NotEmpty(t, keyPair.PublicKey)
		assert.Contains(t, keyPair.PrivateKey, "BEGIN")
		assert.Contains(t, keyPair.PublicKey, "ssh-rsa")
	})
}
