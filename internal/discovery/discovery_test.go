package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDiscoveredAsset_Struct(t *testing.T) {
	t.Run("asset with all fields", func(t *testing.T) {
		id := uuid.New()
		scanID := uuid.New()
		tenantID := uuid.New()
		now := time.Now()

		asset := &DiscoveredAsset{
			ID:            id,
			TenantID:      tenantID,
			ScanID:        scanID,
			Hostname:      "webserver-01",
			IP:            "192.168.1.50",
			MAC:           "00:11:22:33:44:55",
			OS:            "Ubuntu 22.04",
			OSVersion:     "22.04.3 LTS",
			OpenPorts:     []PortInfo{},
			Services:      []ServiceInfo{},
			AssetType:     "server",
			Sensitivity:   "high",
			Status:        "discovered",
			LastScannedAt: now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		assert.Equal(t, id, asset.ID)
		assert.Equal(t, tenantID, asset.TenantID)
		assert.Equal(t, scanID, asset.ScanID)
		assert.Equal(t, "webserver-01", asset.Hostname)
		assert.Equal(t, "192.168.1.50", asset.IP)
		assert.Equal(t, "00:11:22:33:44:55", asset.MAC)
		assert.Equal(t, "Ubuntu 22.04", asset.OS)
		assert.Equal(t, "server", asset.AssetType)
		assert.Equal(t, "high", asset.Sensitivity)
		assert.Equal(t, "discovered", asset.Status)
	})
}

func TestPortInfo_Struct(t *testing.T) {
	t.Run("port info with all fields", func(t *testing.T) {
		port := PortInfo{
			Port:     22,
			Protocol: "tcp",
			Service:  "ssh",
			Version:  "OpenSSH 8.9",
			Banner:   "SSH-2.0-OpenSSH_8.9p1 Ubuntu",
		}

		assert.Equal(t, 22, port.Port)
		assert.Equal(t, "tcp", port.Protocol)
		assert.Equal(t, "ssh", port.Service)
		assert.Equal(t, "OpenSSH 8.9", port.Version)
		assert.Equal(t, "SSH-2.0-OpenSSH_8.9p1 Ubuntu", port.Banner)
	})
}

func TestServiceInfo_Struct(t *testing.T) {
	t.Run("service info with all fields", func(t *testing.T) {
		service := ServiceInfo{
			Name:    "nginx",
			Port:    443,
			Product: "nginx web server",
			Version: "1.24.0",
		}

		assert.Equal(t, "nginx", service.Name)
		assert.Equal(t, 443, service.Port)
		assert.Equal(t, "nginx web server", service.Product)
		assert.Equal(t, "1.24.0", service.Version)
	})
}

func TestScan_Struct(t *testing.T) {
	t.Run("scan with all fields", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		now := time.Now()
		startedAt := now.Add(-1 * time.Hour)
		completedAt := now

		scan := &Scan{
			ID:               id,
			TenantID:         tenantID,
			Name:             "Weekly Network Scan",
			Subnets:          []string{"192.168.1.0/24", "10.0.0.0/8"},
			PortRange:        "1-1024",
			Status:           "completed",
			StartedAt:        &startedAt,
			CompletedAt:      &completedAt,
			TotalHosts:       254,
			DiscoveredAssets: 50,
			NewAssets:        5,
			ChangedAssets:    3,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		assert.Equal(t, id, scan.ID)
		assert.Equal(t, tenantID, scan.TenantID)
		assert.Equal(t, "Weekly Network Scan", scan.Name)
		assert.Len(t, scan.Subnets, 2)
		assert.Equal(t, "1-1024", scan.PortRange)
		assert.Equal(t, "completed", scan.Status)
		assert.NotNil(t, scan.StartedAt)
		assert.NotNil(t, scan.CompletedAt)
		assert.Equal(t, 254, scan.TotalHosts)
		assert.Equal(t, 50, scan.DiscoveredAssets)
		assert.Equal(t, 5, scan.NewAssets)
		assert.Equal(t, 3, scan.ChangedAssets)
	})

	t.Run("scan statuses", func(t *testing.T) {
		statuses := []string{"pending", "running", "completed", "failed", "cancelled"}

		for _, status := range statuses {
			scan := &Scan{Status: status}
			assert.Equal(t, status, scan.Status)
		}
	})
}

func TestAssetFilter_Struct(t *testing.T) {
	t.Run("asset filter with all fields", func(t *testing.T) {
		status := "discovered"
		assetType := "server"

		filter := AssetFilter{
			Status:    &status,
			AssetType: &assetType,
		}

		assert.NotNil(t, filter.Status)
		assert.NotNil(t, filter.AssetType)
		assert.Equal(t, "discovered", *filter.Status)
		assert.Equal(t, "server", *filter.AssetType)
	})

	t.Run("asset filter with nil values", func(t *testing.T) {
		filter := AssetFilter{}

		assert.Nil(t, filter.Status)
		assert.Nil(t, filter.AssetType)
	})
}

func TestNewRepository(t *testing.T) {
	t.Run("creates repository", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache // nil cache for testing
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)

		assert.NotNil(t, repo)
		assert.Equal(t, db, repo.db)
		assert.Equal(t, c, repo.cache)
		assert.Equal(t, logger, repo.logger)
	})
}

func TestNewService(t *testing.T) {
	t.Run("creates service", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)
		svc := NewService(repo, nil, c, logger)

		assert.NotNil(t, svc)
		assert.Equal(t, repo, svc.repo)
		assert.Nil(t, svc.targetSvc)
		assert.Equal(t, c, svc.cache)
		assert.NotNil(t, svc.scanners)
	})
}

func TestService_RegisterScanner(t *testing.T) {
	t.Run("register custom scanner", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)
		svc := NewService(repo, nil, c, logger)

		// Register default scanners should already exist
		assert.Contains(t, svc.scanners, "nmap")
		assert.Contains(t, svc.scanners, "port")
	})
}

func TestPortScanner_Name(t *testing.T) {
	t.Run("scanner name", func(t *testing.T) {
		scanner := NewPortScanner()
		assert.Equal(t, "port", scanner.Name())
	})
}

func TestPortScanner_ParsePorts(t *testing.T) {
	t.Run("parse ports returns defaults", func(t *testing.T) {
		scanner := NewPortScanner()
		ports, err := scanner.parsePorts("22,80,443")

		assert.NoError(t, err)
		// Default implementation returns predefined list
		assert.NotEmpty(t, ports)
	})
}

func TestPortScanner_ClassifyAsset(t *testing.T) {
	t.Run("classify SSH server", func(t *testing.T) {
		scanner := NewPortScanner()
		asset := &DiscoveredAsset{
			OpenPorts: []PortInfo{{Port: 22, Protocol: "tcp"}},
		}

		assetType := scanner.classifyAsset(asset)
		assert.Equal(t, "server", assetType)
	})

	t.Run("classify RDP workstation", func(t *testing.T) {
		scanner := NewPortScanner()
		asset := &DiscoveredAsset{
			OpenPorts: []PortInfo{{Port: 3389, Protocol: "tcp"}},
		}

		assetType := scanner.classifyAsset(asset)
		assert.Equal(t, "workstation", assetType)
	})

	t.Run("classify database", func(t *testing.T) {
		scanner := NewPortScanner()
		asset := &DiscoveredAsset{
			OpenPorts: []PortInfo{{Port: 5432, Protocol: "tcp"}},
		}

		assetType := scanner.classifyAsset(asset)
		assert.Equal(t, "database", assetType)
	})

	t.Run("classify unknown", func(t *testing.T) {
		scanner := NewPortScanner()
		asset := &DiscoveredAsset{
			OpenPorts: []PortInfo{{Port: 8080, Protocol: "tcp"}},
		}

		assetType := scanner.classifyAsset(asset)
		assert.Equal(t, "unknown", assetType)
	})
}

func TestNmapScanner_Name(t *testing.T) {
	t.Run("scanner name", func(t *testing.T) {
		scanner := NewNmapScanner()
		assert.Equal(t, "nmap", scanner.Name())
	})
}

func TestNmapScanner_Scan(t *testing.T) {
	t.Run("scan returns not implemented", func(t *testing.T) {
		scanner := NewNmapScanner()
		ctx := context.Background()
		results := make(chan *DiscoveredAsset)

		err := scanner.Scan(ctx, []string{"192.168.1.0/24"}, "1-1024", results)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not available")
	})
}

func TestService_ImportAsTarget(t *testing.T) {
	t.Run("import without DB panics", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)
		svc := NewService(repo, nil, c, logger)

		ctx := context.Background()
		assetID := uuid.New()

		// Will panic because db is empty
		assert.Panics(t, func() {
			_, _ = svc.ImportAsTarget(ctx, assetID)
		})
	})
}

func TestService_ListScans(t *testing.T) {
	t.Run("list scans without DB panics", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)
		svc := NewService(repo, nil, c, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		// Will panic because db is empty
		assert.Panics(t, func() {
			_, _, _ = svc.ListScans(ctx, tenantID, 10, 0)
		})
	})
}

func TestService_ListAssets(t *testing.T) {
	t.Run("list assets without DB panics", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)
		svc := NewService(repo, nil, c, logger)

		ctx := context.Background()
		tenantID := uuid.New()

		// Will panic because db is empty
		assert.Panics(t, func() {
			_, _, _ = svc.ListAssets(ctx, tenantID, AssetFilter{}, 10, 0)
		})
	})
}

func TestService_GetScan(t *testing.T) {
	t.Run("get scan without DB panics", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)
		svc := NewService(repo, nil, c, logger)

		ctx := context.Background()
		scanID := uuid.New()

		// Will panic because db is empty
		assert.Panics(t, func() {
			_, _ = svc.GetScan(ctx, scanID)
		})
	})
}

func TestRepository_CreateScan(t *testing.T) {
	t.Run("create scan without DB panics", func(t *testing.T) {
		db := &sqlx.DB{}
		var c *cache.Cache
		logger := zerolog.Nop()

		repo := NewRepository(db, c, logger)

		ctx := context.Background()
		scan := &Scan{
			Name:     "Test Scan",
			Subnets:  []string{"192.168.1.0/24"},
			TenantID: uuid.New(),
		}

		// Will panic because db is empty
		assert.Panics(t, func() {
			_ = repo.CreateScan(ctx, scan)
		})
	})
}

func TestDiscoveredAsset_StatusValues(t *testing.T) {
	t.Run("valid status values", func(t *testing.T) {
		validStatuses := []string{
			"discovered",
			"managed",
			"ignored",
			"false_positive",
		}

		for _, status := range validStatuses {
			asset := &DiscoveredAsset{Status: status}
			assert.Equal(t, status, asset.Status)
		}
	})
}

func TestDiscoveredAsset_SensitivityValues(t *testing.T) {
	t.Run("valid sensitivity values", func(t *testing.T) {
		validSensitivities := []string{
			"critical",
			"high",
			"medium",
			"low",
		}

		for _, sensitivity := range validSensitivities {
			asset := &DiscoveredAsset{Sensitivity: sensitivity}
			assert.Equal(t, sensitivity, asset.Sensitivity)
		}
	})
}

func TestDiscoveredAsset_AssetTypeValues(t *testing.T) {
	t.Run("valid asset type values", func(t *testing.T) {
		validTypes := []string{
			"server",
			"workstation",
			"network_device",
			"database",
			"unknown",
		}

		for _, assetType := range validTypes {
			asset := &DiscoveredAsset{AssetType: assetType}
			assert.Equal(t, assetType, asset.AssetType)
		}
	})
}
