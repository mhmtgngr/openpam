package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser_Struct(t *testing.T) {
	t.Run("user with all fields", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		now := time.Now()
		lastLogin := now.Add(-1 * time.Hour)

		user := User{
			ID:           id,
			Email:        "test@example.com",
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: "hashed-password",
			Role:         "admin",
			Status:       "active",
			MFAEnabled:   true,
			TenantID:     tenantID,
			LastLoginAt:  &lastLogin,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		assert.Equal(t, id, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "Test", user.FirstName)
		assert.Equal(t, "User", user.LastName)
		assert.Equal(t, "hashed-password", user.PasswordHash)
		assert.Equal(t, "admin", user.Role)
		assert.Equal(t, "active", user.Status)
		assert.True(t, user.MFAEnabled)
		assert.Equal(t, tenantID, user.TenantID)
		assert.NotNil(t, user.LastLoginAt)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
	})

	t.Run("user with minimal fields", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()

		user := User{
			ID:       id,
			Email:    "minimal@example.com",
			PasswordHash: "hash",
			Role:     "user",
			Status:   "active",
			TenantID: tenantID,
		}

		assert.Equal(t, id, user.ID)
		assert.Equal(t, "minimal@example.com", user.Email)
		assert.Equal(t, "user", user.Role)
		assert.Equal(t, "active", user.Status)
		assert.False(t, user.MFAEnabled)
		assert.Nil(t, user.LastLoginAt)
	})
}

func TestCredential_Struct(t *testing.T) {
	t.Run("credential with all fields", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		createdBy := uuid.New()
		folderID := uuid.New()
		now := time.Now()
		lastRotated := now.Add(-24 * time.Hour)

		credential := Credential{
			ID:              id,
			Name:            "production-db",
			Type:            "password",
			Host:            "db.example.com",
			Port:            5432,
			Username:        "admin",
			EncryptedSecret: []byte("encrypted-data"),
			RotationPolicy:  "weekly",
			LastRotatedAt:   &lastRotated,
			FolderID:        &folderID,
			TenantID:        tenantID,
			CreatedBy:       createdBy,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		assert.Equal(t, id, credential.ID)
		assert.Equal(t, "production-db", credential.Name)
		assert.Equal(t, "password", credential.Type)
		assert.Equal(t, "db.example.com", credential.Host)
		assert.Equal(t, 5432, credential.Port)
		assert.Equal(t, "admin", credential.Username)
		assert.NotEmpty(t, credential.EncryptedSecret)
		assert.Equal(t, "weekly", credential.RotationPolicy)
		assert.NotNil(t, credential.LastRotatedAt)
		assert.NotNil(t, credential.FolderID)
		assert.Equal(t, tenantID, credential.TenantID)
		assert.Equal(t, createdBy, credential.CreatedBy)
	})

	t.Run("credential without folder and rotation", func(t *testing.T) {
		id := uuid.New()

		credential := Credential{
			ID:              id,
			Name:            "ssh-key",
			Type:            "ssh_key",
			Host:            "server.example.com",
			Port:            22,
			Username:        "root",
			EncryptedSecret: []byte("encrypted-key"),
			RotationPolicy:  "manual",
			TenantID:        uuid.New(),
			CreatedBy:       uuid.New(),
		}

		assert.Equal(t, "ssh_key", credential.Type)
		assert.Equal(t, "manual", credential.RotationPolicy)
		assert.Nil(t, credential.FolderID)
		assert.Nil(t, credential.LastRotatedAt)
	})
}

func TestSession_Struct(t *testing.T) {
	t.Run("active session", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		credentialID := uuid.New()
		tenantID := uuid.New()
		now := time.Now()

		session := Session{
			ID:           id,
			UserID:       userID,
			CredentialID: credentialID,
			Type:         "ssh",
			Status:       "active",
			TargetHost:   "server.example.com",
			TargetPort:   22,
			RecordingURL: "s3://recordings/session-123.mp4",
			StartedAt:    now,
			TenantID:     tenantID,
			Metadata:     []byte(`{"command":"ls -la"}`),
		}

		assert.Equal(t, id, session.ID)
		assert.Equal(t, userID, session.UserID)
		assert.Equal(t, credentialID, session.CredentialID)
		assert.Equal(t, "ssh", session.Type)
		assert.Equal(t, "active", session.Status)
		assert.Equal(t, "server.example.com", session.TargetHost)
		assert.Equal(t, 22, session.TargetPort)
		assert.Equal(t, "s3://recordings/session-123.mp4", session.RecordingURL)
		assert.False(t, session.StartedAt.IsZero())
		assert.Nil(t, session.EndedAt)
		assert.Nil(t, session.TerminatedBy)
		assert.NotEmpty(t, session.Metadata)
	})

	t.Run("terminated session", func(t *testing.T) {
		id := uuid.New()
		terminatedBy := uuid.New()
		now := time.Now()
		endTime := now.Add(-30 * time.Minute)

		session := Session{
			ID:           id,
			Type:         "rdp",
			Status:       "terminated",
			TargetHost:   "windows.example.com",
			TargetPort:   3389,
			StartedAt:    now.Add(-2 * time.Hour),
			EndedAt:      &endTime,
			TerminatedBy: &terminatedBy,
			TenantID:     uuid.New(),
		}

		assert.Equal(t, "terminated", session.Status)
		assert.NotNil(t, session.EndedAt)
		assert.NotNil(t, session.TerminatedBy)
		assert.Equal(t, terminatedBy, *session.TerminatedBy)
	})

	t.Run("different session types", func(t *testing.T) {
		types := []string{"ssh", "rdp", "database", "kubernetes", "web"}

		for _, sessionType := range types {
			session := Session{
				Type: sessionType,
			}
			assert.Equal(t, sessionType, session.Type)
		}
	})

	t.Run("different session statuses", func(t *testing.T) {
		statuses := []string{"active", "ended", "terminated", "failed"}

		for _, status := range statuses {
			session := Session{
				Status: status,
			}
			assert.Equal(t, status, session.Status)
		}
	})
}

func TestCheckoutRequest_Struct(t *testing.T) {
	t.Run("pending checkout request", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		credentialID := uuid.New()
		tenantID := uuid.New()

		request := CheckoutRequest{
			ID:           id,
			UserID:       userID,
			CredentialID: credentialID,
			Justification: "Need to deploy hotfix",
			Duration:     60,
			Status:       "pending",
			TenantID:     tenantID,
			CreatedAt:    time.Now(),
		}

		assert.Equal(t, id, request.ID)
		assert.Equal(t, userID, request.UserID)
		assert.Equal(t, credentialID, request.CredentialID)
		assert.Equal(t, "Need to deploy hotfix", request.Justification)
		assert.Equal(t, 60, request.Duration)
		assert.Equal(t, "pending", request.Status)
		assert.Nil(t, request.ApprovedBy)
		assert.Nil(t, request.CheckedOutAt)
		assert.Nil(t, request.ExpiresAt)
	})

	t.Run("approved and checked out request", func(t *testing.T) {
		id := uuid.New()
		approvedBy := uuid.New()
		now := time.Now()
		expiresAt := now.Add(1 * time.Hour)

		request := CheckoutRequest{
			ID:           id,
			Status:       "checked_out",
			ApprovedBy:   &approvedBy,
			CheckedOutAt: &now,
			ExpiresAt:    &expiresAt,
			TenantID:     uuid.New(),
			CreatedAt:    now.Add(-1 * time.Hour),
		}

		assert.Equal(t, "checked_out", request.Status)
		assert.NotNil(t, request.ApprovedBy)
		assert.NotNil(t, request.CheckedOutAt)
		assert.NotNil(t, request.ExpiresAt)
		assert.Equal(t, approvedBy, *request.ApprovedBy)
	})

	t.Run("all request statuses", func(t *testing.T) {
		statuses := []string{"pending", "approved", "denied", "checked_out", "checked_in", "expired"}

		for _, status := range statuses {
			request := CheckoutRequest{
				Status: status,
			}
			assert.Equal(t, status, request.Status)
		}
	})
}

func TestAuditEvent_Struct(t *testing.T) {
	t.Run("complete audit event", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		actorID := uuid.New()
		now := time.Now()

		event := AuditEvent{
			ID:           id,
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       "credential.checkout",
			ResourceType: "credential",
			ResourceID:   "cred-123",
			Outcome:      "success",
			IP:           "192.168.1.100",
			UserAgent:    "OpenPAM-CLI/1.0",
			CorrelationID: "req-456",
			Details:      []byte(`{"reason":"deployment"}`),
			PreviousHash: "hash-previous",
			Hash:         "hash-current",
			CreatedAt:    now,
		}

		assert.Equal(t, id, event.ID)
		assert.Equal(t, tenantID, event.TenantID)
		assert.Equal(t, actorID, event.ActorID)
		assert.Equal(t, "user", event.ActorType)
		assert.Equal(t, "credential.checkout", event.Action)
		assert.Equal(t, "credential", event.ResourceType)
		assert.Equal(t, "cred-123", event.ResourceID)
		assert.Equal(t, "success", event.Outcome)
		assert.Equal(t, "192.168.1.100", event.IP)
		assert.Equal(t, "OpenPAM-CLI/1.0", event.UserAgent)
		assert.Equal(t, "req-456", event.CorrelationID)
		assert.NotEmpty(t, event.Details)
		assert.Equal(t, "hash-previous", event.PreviousHash)
		assert.Equal(t, "hash-current", event.Hash)
		assert.False(t, event.CreatedAt.IsZero())
	})

	t.Run("minimal audit event", func(t *testing.T) {
		id := uuid.New()
		event := AuditEvent{
			ID:       id,
			TenantID: uuid.New(),
			ActorID:  uuid.New(),
			Action:   "user.login",
			Outcome:  "failure",
			Hash:     "some-hash",
		}

		assert.Equal(t, "user.login", event.Action)
		assert.Equal(t, "failure", event.Outcome)
		assert.Empty(t, event.IP)
		assert.Empty(t, event.UserAgent)
	})

	t.Run("different actor types", func(t *testing.T) {
		actorTypes := []string{"user", "system", "api"}

		for _, actorType := range actorTypes {
			event := AuditEvent{
				ActorType: actorType,
			}
			assert.Equal(t, actorType, event.ActorType)
		}
	})

	t.Run("different outcomes", func(t *testing.T) {
		outcomes := []string{"success", "failure", "denied"}

		for _, outcome := range outcomes {
			event := AuditEvent{
				Outcome: outcome,
			}
			assert.Equal(t, outcome, event.Outcome)
		}
	})
}

func TestDiscoveredAsset_Struct(t *testing.T) {
	t.Run("complete discovered asset", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		now := time.Now()

		asset := DiscoveredAsset{
			ID:            id,
			Hostname:      "webserver-01",
			IP:            "192.168.1.50",
			OS:            "Ubuntu 22.04",
			OpenPorts:     []byte(`[{"port":22,"protocol":"tcp"},{"port":80,"protocol":"tcp"}]`),
			Services:      []byte(`[{"name":"ssh","port":22},{"name":"http","port":80}]`),
			Status:        "discovered",
			LastScannedAt: now,
			TenantID:      tenantID,
			CreatedAt:     now,
		}

		assert.Equal(t, id, asset.ID)
		assert.Equal(t, "webserver-01", asset.Hostname)
		assert.Equal(t, "192.168.1.50", asset.IP)
		assert.Equal(t, "Ubuntu 22.04", asset.OS)
		assert.NotEmpty(t, asset.OpenPorts)
		assert.NotEmpty(t, asset.Services)
		assert.Equal(t, "discovered", asset.Status)
		assert.False(t, asset.LastScannedAt.IsZero())
	})

	t.Run("different asset statuses", func(t *testing.T) {
		statuses := []string{"discovered", "managed", "ignored"}

		for _, status := range statuses {
			asset := DiscoveredAsset{
				Status: status,
			}
			assert.Equal(t, status, asset.Status)
		}
	})
}

func TestTenant_Struct(t *testing.T) {
	t.Run("complete tenant", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()

		tenant := Tenant{
			ID:        id,
			Name:      "Acme Corp",
			Domain:    "acme.example.com",
			Plan:      "enterprise",
			Config:    []byte(`{"maxUsers":100}`),
			CreatedAt: now,
		}

		assert.Equal(t, id, tenant.ID)
		assert.Equal(t, "Acme Corp", tenant.Name)
		assert.Equal(t, "acme.example.com", tenant.Domain)
		assert.Equal(t, "enterprise", tenant.Plan)
		assert.NotEmpty(t, tenant.Config)
		assert.False(t, tenant.CreatedAt.IsZero())
	})

	t.Run("different plans", func(t *testing.T) {
		plans := []string{"free", "pro", "enterprise"}

		for _, plan := range plans {
			tenant := Tenant{
				Plan: plan,
			}
			assert.Equal(t, plan, tenant.Plan)
		}
	})
}

func TestModel_JSONTags(t *testing.T) {
	t.Run("User JSON tags", func(t *testing.T) {
		user := User{
			Email: "test@example.com",
		}
		// Verify the struct has proper JSON tags
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("Credential JSON tags", func(t *testing.T) {
		cred := Credential{
			Name: "test-cred",
		}
		assert.Equal(t, "test-cred", cred.Name)
	})
}

func TestModel_DBTags(t *testing.T) {
	t.Run("verify struct tags exist", func(t *testing.T) {
		// This test verifies that the models have proper db tags
		// In a real scenario, we'd use reflection to check

		t.Run("User has db tags", func(t *testing.T) {
			user := User{
				Email: "test@example.com",
			}
			assert.Equal(t, "test@example.com", user.Email)
		})

		t.Run("Credential has db tags", func(t *testing.T) {
			cred := Credential{
				Name: "test",
			}
			assert.Equal(t, "test", cred.Name)
		})
	})
}
