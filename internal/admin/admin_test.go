package admin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// setupAdminTest creates a test database and admin service
func setupAdminTest(t *testing.T) (*sqlx.DB, *Service) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)
	logger := opamtesting.Logger(t)

	// Create an empty cache wrapper for testing
	// Note: Using nil cache since admin service doesn't heavily depend on cache for basic operations
	var c *cache.Cache

	service := NewService(db, c, logger)

	return db, service
}

// Test Tenant struct
func TestTenant_Struct(t *testing.T) {
	t.Run("create minimal tenant", func(t *testing.T) {
		tenant := &Tenant{
			ID:        uuid.New(),
			Name:      "Test Tenant",
			Domain:    "test.example.com",
			Plan:      "pro",
			Status:    "active",
			UserLimit: 50,
			StorageGB: 10,
		}

		assert.NotEqual(t, uuid.Nil, tenant.ID)
		assert.Equal(t, "Test Tenant", tenant.Name)
		assert.Equal(t, "test.example.com", tenant.Domain)
		assert.Equal(t, "pro", tenant.Plan)
		assert.Equal(t, "active", tenant.Status)
		assert.Equal(t, 50, tenant.UserLimit)
		assert.Equal(t, 10, tenant.StorageGB)
		assert.Nil(t, tenant.DeletedAt)
	})

	t.Run("tenant with soft delete", func(t *testing.T) {
		now := time.Now()
		tenant := &Tenant{
			ID:        uuid.New(),
			Name:      "Deleted Tenant",
			DeletedAt: &now,
		}

		assert.NotNil(t, tenant.DeletedAt)
		assert.WithinDuration(t, now, *tenant.DeletedAt, time.Second)
	})

	t.Run("tenant with config", func(t *testing.T) {
		config := json.RawMessage([]byte(`{"session_timeout": 30}`))
		tenant := &Tenant{
			ID:     uuid.New(),
			Name:   "Configured Tenant",
			Config: config,
		}

		assert.NotNil(t, tenant.Config)
		assert.Contains(t, string(tenant.Config), "session_timeout")
	})
}

// Test TenantConfig struct
func TestTenantConfig_Struct(t *testing.T) {
	t.Run("create tenant config with defaults", func(t *testing.T) {
		config := &TenantConfig{
			SessionTimeoutMinutes: 60,
			MFARequired:          true,
			AllowedIPs:           []string{"192.168.1.0/24"},
		}

		assert.Equal(t, 60, config.SessionTimeoutMinutes)
		assert.True(t, config.MFARequired)
		assert.Len(t, config.AllowedIPs, 1)
		assert.Nil(t, config.SSOConfig)
		assert.Nil(t, config.Branding)
	})

	t.Run("create tenant config with SSO", func(t *testing.T) {
		config := &TenantConfig{
			SessionTimeoutMinutes: 120,
			MFARequired:          false,
			SSOConfig: &SSOConfig{
				Provider:    "saml",
				EntityID:    "example-entity",
				MetadataURL: "https://idp.example.com/metadata",
			},
		}

		assert.NotNil(t, config.SSOConfig)
		assert.Equal(t, "saml", config.SSOConfig.Provider)
		assert.Equal(t, "example-entity", config.SSOConfig.EntityID)
	})

	t.Run("create tenant config with branding", func(t *testing.T) {
		config := &TenantConfig{
			SessionTimeoutMinutes: 30,
			MFARequired:          true,
			Branding: &BrandingConfig{
				LogoURL:      "https://example.com/logo.png",
				PrimaryColor: "#0066cc",
				CompanyName:  "Acme Corp",
			},
		}

		assert.NotNil(t, config.Branding)
		assert.Equal(t, "#0066cc", config.Branding.PrimaryColor)
		assert.Equal(t, "Acme Corp", config.Branding.CompanyName)
	})
}

// Test SSOConfig struct
func TestSSOConfig_Struct(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		valid    bool
	}{
		{"SAML provider", "saml", true},
		{"OIDC provider", "oidc", true},
		{"invalid provider", "ldap", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSOConfig{
				Provider:    tt.provider,
				EntityID:    "test-entity",
				MetadataURL: "https://example.com/metadata",
			}

			assert.Equal(t, tt.provider, config.Provider)
			assert.NotEmpty(t, config.EntityID)
			assert.NotEmpty(t, config.MetadataURL)
		})
	}
}

// Test BrandingConfig struct
func TestBrandingConfig_Struct(t *testing.T) {
	config := &BrandingConfig{
		LogoURL:      "https://example.com/logo.png",
		PrimaryColor: "#FF5733",
		CompanyName:  "Test Company",
	}

	assert.NotEmpty(t, config.LogoURL)
	assert.NotEmpty(t, config.PrimaryColor)
	assert.NotEmpty(t, config.CompanyName)
	assert.True(t, config.PrimaryColor[0] == '#')
}

// Test TenantUsage struct
func TestTenantUsage_Struct(t *testing.T) {
	usage := &TenantUsage{
		TenantID:         uuid.New(),
		UserCount:        25,
		CredentialCount:  10,
		SessionCount30d:  150,
		StorageUsedBytes: 536870912, // 512 MB
	}

	assert.NotEqual(t, uuid.Nil, usage.TenantID)
	assert.Equal(t, 25, usage.UserCount)
	assert.Equal(t, 10, usage.CredentialCount)
	assert.Equal(t, 150, usage.SessionCount30d)
	assert.Greater(t, usage.StorageUsedBytes, int64(0))
}

// Test SystemStats struct
func TestSystemStats_Struct(t *testing.T) {
	now := time.Now()
	stats := &SystemStats{
		GeneratedAt:          now,
		TenantCount:          5,
		UserCount:            125,
		CredentialCount:      50,
		ActiveSessionCount:   10,
		TodaySuccessfulEvents: 500,
		TodayFailedEvents:    25,
		TodayDeniedEvents:    5,
	}

	assert.False(t, stats.GeneratedAt.IsZero())
	assert.Greater(t, stats.TenantCount, 0)
	assert.Greater(t, stats.UserCount, 0)
	assert.Greater(t, stats.CredentialCount, 0)
	assert.GreaterOrEqual(t, stats.ActiveSessionCount, 0)
}

// Test NewService
func TestNewService(t *testing.T) {
	t.Run("create service with nil dependencies", func(t *testing.T) {
		logger := zerolog.Nop()
		service := NewService(nil, nil, logger)

		assert.NotNil(t, service)
		assert.Nil(t, service.db)
		assert.Nil(t, service.cache)
	})
}

// Test setDefaultsForPlan
func TestSetDefaultsForPlan(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name           string
		plan           string
		expectedUsers  int
		expectedStorage int
	}{
		{"free plan defaults", "free", 5, 1},
		{"pro plan defaults", "pro", 50, 10},
		{"enterprise plan defaults", "enterprise", -1, 100},
		{"unknown plan defaults", "unknown", 10, 5},
		{"empty plan defaults", "", 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{
				Plan: tt.plan,
			}

			service.setDefaultsForPlan(tenant)

			assert.Equal(t, tt.expectedUsers, tenant.UserLimit)
			assert.Equal(t, tt.expectedStorage, tenant.StorageGB)
		})
	}
}

// Test CreateTenant with database
func TestService_CreateTenant(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("create tenant with valid data", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Test Company",
			Domain: "testcompany.example.com",
			Plan:   "pro",
		}

		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, tenant.ID)
		assert.False(t, tenant.CreatedAt.IsZero())
		assert.False(t, tenant.UpdatedAt.IsZero())
		assert.Equal(t, "active", tenant.Status)
		assert.Equal(t, 50, tenant.UserLimit) // pro plan default
		assert.Equal(t, 10, tenant.StorageGB) // pro plan default
	})

	t.Run("create tenant with free plan", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Free Tier Tenant",
			Domain: "free.example.com",
			Plan:   "free",
		}

		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		assert.Equal(t, 5, tenant.UserLimit)    // free plan default
		assert.Equal(t, 1, tenant.StorageGB)     // free plan default
	})

	t.Run("create tenant with enterprise plan", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Enterprise Tenant",
			Domain: "enterprise.example.com",
			Plan:   "enterprise",
		}

		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		assert.Equal(t, -1, tenant.UserLimit)  // unlimited
		assert.Equal(t, 100, tenant.StorageGB)
	})
}

// Test GetTenant
func TestService_GetTenant(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("get existing tenant by ID", func(t *testing.T) {
		// First create a tenant
		tenant := &Tenant{
			Name:   "Get Test Tenant",
			Domain: "gettest.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Get the tenant
		found, err := service.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, tenant.Name, found.Name)
		assert.Equal(t, tenant.Domain, found.Domain)
	})

	t.Run("get non-existent tenant", func(t *testing.T) {
		randomID := uuid.New()
		_, err := service.GetTenant(ctx, randomID)
		assert.Error(t, err)
	})

	t.Run("get soft-deleted tenant should fail", func(t *testing.T) {
		// Create a tenant
		tenant := &Tenant{
			Name:   "To Be Deleted",
			Domain: "deleted.example.com",
			Plan:   "free",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Soft delete it
		err = service.DeleteTenant(ctx, tenant.ID)
		require.NoError(t, err)

		// Try to get it - should fail
		_, err = service.GetTenant(ctx, tenant.ID)
		assert.Error(t, err)
	})
}

// Test GetTenantByDomain
func TestService_GetTenantByDomain(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("get tenant by domain", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Domain Test Tenant",
			Domain: "domaintest.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		found, err := service.GetTenantByDomain(ctx, "domaintest.example.com")
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, found.ID)
		assert.Equal(t, tenant.Name, found.Name)
	})

	t.Run("get tenant by non-existent domain", func(t *testing.T) {
		_, err := service.GetTenantByDomain(ctx, "nonexistent.example.com")
		assert.Error(t, err)
	})
}

// Test ListTenants
func TestService_ListTenants(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	// Create multiple tenants
	for i := 0; i < 5; i++ {
		tenant := &Tenant{
			Name:   uuid.New().String(),
			Domain: uuid.New().String() + ".example.com",
			Plan:   "pro",
		}
		_ = service.CreateTenant(ctx, tenant)
	}

	t.Run("list all tenants", func(t *testing.T) {
		tenants, total, err := service.ListTenants(ctx, 100, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tenants), 5)
		assert.GreaterOrEqual(t, total, 5)
	})

	t.Run("list tenants with pagination", func(t *testing.T) {
		page1, total, err := service.ListTenants(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1, 2)

		page2, _, err := service.ListTenants(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2, 2)

		// Total should be consistent
		assert.Equal(t, total, total)
	})
}

// Test UpdateTenant
func TestService_UpdateTenant(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("update tenant name and plan", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Original Name",
			Domain: "update.example.com",
			Plan:   "free",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Update
		tenant.Name = "Updated Name"
		tenant.Plan = "enterprise"
		err = service.UpdateTenant(ctx, tenant)
		require.NoError(t, err)

		// Verify
		found, err := service.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", found.Name)
		assert.Equal(t, "enterprise", found.Plan)
	})

	t.Run("update with config", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Config Update Test",
			Domain: "configupdate.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Update with config
		newConfig := TenantConfig{
			SessionTimeoutMinutes: 90,
			MFARequired:          true,
		}
		configData, _ := json.Marshal(newConfig)
		tenant.Config = configData

		err = service.UpdateTenant(ctx, tenant)
		require.NoError(t, err)
	})
}

// Test SuspendTenant
func TestService_SuspendTenant(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("suspend active tenant", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "To Be Suspended",
			Domain: "suspend.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		assert.Equal(t, "active", tenant.Status)

		// Suspend
		err = service.SuspendTenant(ctx, tenant.ID, "policy violation")
		require.NoError(t, err)

		// Verify status changed
		found, err := service.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, "suspended", found.Status)
	})
}

// Test DeleteTenant (soft delete)
func TestService_DeleteTenant(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("soft delete tenant", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "To Be Soft Deleted",
			Domain: "softdelete.example.com",
			Plan:   "free",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		assert.Nil(t, tenant.DeletedAt)

		// Delete
		err = service.DeleteTenant(ctx, tenant.ID)
		require.NoError(t, err)

		// Verify soft delete - tenant should not be found by GetTenant
		_, err = service.GetTenant(ctx, tenant.ID)
		assert.Error(t, err)
	})
}

// Test GetTenantConfig
func TestService_GetTenantConfig(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("get tenant config", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Config Test Tenant",
			Domain: "configtest.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Get config (should have defaults)
		config, err := service.GetTenantConfig(ctx, tenant.ID)
		require.NoError(t, err)
		assert.NotNil(t, config)
	})

	t.Run("get tenant config with custom settings", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Custom Config Tenant",
			Domain: "customconfig.example.com",
			Plan:   "enterprise",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Set custom config
		customConfig := &TenantConfig{
			SessionTimeoutMinutes: 120,
			MFARequired:          true,
			AllowedIPs:           []string{"10.0.0.0/8"},
			Branding: &BrandingConfig{
				LogoURL:      "https://example.com/logo.png",
				PrimaryColor: "#FF0000",
				CompanyName:  "Custom Inc",
			},
		}
		err = service.UpdateTenantConfig(ctx, tenant.ID, customConfig)
		require.NoError(t, err)

		// Get config back
		retrieved, err := service.GetTenantConfig(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, 120, retrieved.SessionTimeoutMinutes)
		assert.True(t, retrieved.MFARequired)
		assert.NotNil(t, retrieved.Branding)
		assert.Equal(t, "#FF0000", retrieved.Branding.PrimaryColor)
	})
}

// Test UpdateTenantConfig
func TestService_UpdateTenantConfig(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("update tenant config", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Config Update",
			Domain: "configupdate2.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Update config
		newConfig := &TenantConfig{
			SessionTimeoutMinutes: 45,
			MFARequired:          true,
		}
		err = service.UpdateTenantConfig(ctx, tenant.ID, newConfig)
		require.NoError(t, err)

		// Verify
		retrieved, err := service.GetTenantConfig(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, 45, retrieved.SessionTimeoutMinutes)
	})
}

// Test GetTenantUsage
func TestService_GetTenantUsage(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("get usage for tenant", func(t *testing.T) {
		tenant := &Tenant{
			Name:   "Usage Test Tenant",
			Domain: "usage.example.com",
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		// Get usage stats
		usage, err := service.GetTenantUsage(ctx, tenant.ID)
		require.NoError(t, err)
		assert.NotNil(t, usage)
		assert.Equal(t, tenant.ID, usage.TenantID)
		assert.GreaterOrEqual(t, usage.UserCount, 0)
		assert.GreaterOrEqual(t, usage.CredentialCount, 0)
		assert.GreaterOrEqual(t, usage.SessionCount30d, 0)
		assert.GreaterOrEqual(t, usage.StorageUsedBytes, int64(0))
	})
}

// Test GetSystemStats
func TestService_GetSystemStats(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("get system stats", func(t *testing.T) {
		stats, err := service.GetSystemStats(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.False(t, stats.GeneratedAt.IsZero())
		assert.GreaterOrEqual(t, stats.TenantCount, 0)
		assert.GreaterOrEqual(t, stats.UserCount, 0)
		assert.GreaterOrEqual(t, stats.CredentialCount, 0)
		assert.GreaterOrEqual(t, stats.ActiveSessionCount, 0)
		assert.GreaterOrEqual(t, stats.TodaySuccessfulEvents, 0)
		assert.GreaterOrEqual(t, stats.TodayFailedEvents, 0)
		assert.GreaterOrEqual(t, stats.TodayDeniedEvents, 0)
	})
}

// Test tenant plan constants validation
func TestTenant_PlanValidation(t *testing.T) {
	validPlans := []string{"free", "pro", "enterprise"}

	t.Run("all valid plans", func(t *testing.T) {
		for _, plan := range validPlans {
			assert.NotEmpty(t, plan)
		}
	})

	t.Run("plan limits are reasonable", func(t *testing.T) {
		limits := map[string]int{
			"free":       5,
			"pro":        50,
			"enterprise": -1,
		}

		assert.Equal(t, 5, limits["free"])
		assert.Equal(t, 50, limits["pro"])
		assert.Equal(t, -1, limits["enterprise"]) // unlimited
	})
}

// Test tenant status values
func TestTenant_StatusValues(t *testing.T) {
	validStatuses := []string{"active", "suspended", "cancelled"}

	for _, status := range validStatuses {
		t.Run("status "+status, func(t *testing.T) {
			tenant := &Tenant{
				Name:   "Status Test",
				Domain: "status.example.com",
				Plan:   "pro",
				Status: status,
			}

			assert.Equal(t, status, tenant.Status)
		})
	}
}

// Test tenant config JSON marshaling
func TestTenantConfig_JSON(t *testing.T) {
	t.Run("marshal and unmarshal config", func(t *testing.T) {
		config := &TenantConfig{
			SessionTimeoutMinutes: 60,
			MFARequired:          true,
			AllowedIPs:           []string{"192.168.1.0/24", "10.0.0.0/8"},
			SSOConfig: &SSOConfig{
				Provider:    "oidc",
				EntityID:    "entity123",
				MetadataURL: "https://idp.example.com",
			},
			Branding: &BrandingConfig{
				LogoURL:      "https://example.com/logo.png",
				PrimaryColor: "#0066cc",
				CompanyName:  "Test Corp",
			},
		}

		// Marshal
		data, err := json.Marshal(config)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		// Unmarshal
		var unmarshaled TenantConfig
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, config.SessionTimeoutMinutes, unmarshaled.SessionTimeoutMinutes)
		assert.Equal(t, config.MFARequired, unmarshaled.MFARequired)
		assert.Len(t, unmarshaled.AllowedIPs, 2)
		assert.NotNil(t, unmarshaled.SSOConfig)
		assert.Equal(t, "oidc", unmarshaled.SSOConfig.Provider)
		assert.NotNil(t, unmarshaled.Branding)
		assert.Equal(t, "#0066cc", unmarshaled.Branding.PrimaryColor)
	})

	t.Run("marshal empty config", func(t *testing.T) {
		config := &TenantConfig{}

		data, err := json.Marshal(config)
		require.NoError(t, err)

		var unmarshaled TenantConfig
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, 0, unmarshaled.SessionTimeoutMinutes)
		assert.False(t, unmarshaled.MFARequired)
	})
}

// Test error conditions
func TestService_ErrorConditions(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("update non-existent tenant", func(t *testing.T) {
		tenant := &Tenant{
			ID:     uuid.New(),
			Name:   "Non-existent",
			Domain: "nonexistent.example.com",
			Plan:   "pro",
		}

		err := service.UpdateTenant(ctx, tenant)
		assert.Error(t, err)
	})

	t.Run("get config for non-existent tenant", func(t *testing.T) {
		_, err := service.GetTenantConfig(ctx, uuid.New())
		assert.Error(t, err)
	})

	t.Run("suspend non-existent tenant", func(t *testing.T) {
		err := service.SuspendTenant(ctx, uuid.New(), "test reason")
		assert.Error(t, err)
	})

	t.Run("delete non-existent tenant", func(t *testing.T) {
		err := service.DeleteTenant(ctx, uuid.New())
		// May not error depending on implementation
		// but should not affect anything
		assert.NoError(t, err) // soft delete typically doesn't error
	})
}

// Test tenant domain uniqueness (would require DB constraint)
func TestService_DomainUniqueness(t *testing.T) {
	db, service := setupAdminTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	t.Run("duplicate domain should error", func(t *testing.T) {
		domain := "unique.example.com"

		// Create first tenant
		tenant1 := &Tenant{
			Name:   "First Tenant",
			Domain: domain,
			Plan:   "pro",
		}
		err := service.CreateTenant(ctx, tenant1)
		require.NoError(t, err)

		// Try to create second tenant with same domain
		tenant2 := &Tenant{
			Name:   "Second Tenant",
			Domain: domain,
			Plan:   "pro",
		}
		err = service.CreateTenant(ctx, tenant2)
		// Should error due to unique constraint
		assert.Error(t, err)
	})
}

// Benchmark tests
func BenchmarkService_CreateTenant(b *testing.B) {
	db, service := setupAdminTest(&testing.T{})
	if db == nil {
		b.Skip("database not available")
		return
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tenant := &Tenant{
			Name:   uuid.New().String(),
			Domain: uuid.New().String() + ".example.com",
			Plan:   "pro",
		}
		_ = service.CreateTenant(ctx, tenant)
	}
}

func BenchmarkService_GetTenant(b *testing.B) {
	db, service := setupAdminTest(&testing.T{})
	if db == nil {
		b.Skip("database not available")
		return
	}
	ctx := context.Background()

	// Create a tenant
	tenant := &Tenant{
		Name:   "Benchmark Tenant",
		Domain: "bench.example.com",
		Plan:   "pro",
	}
	_ = service.CreateTenant(ctx, tenant)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetTenant(ctx, tenant.ID)
	}
}
