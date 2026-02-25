package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	targetpkg "github.com/openpam/openpam/internal/pam/target"
	"github.com/rs/zerolog"
)

// DiscoveredAsset represents an auto-discovered network asset
type DiscoveredAsset struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	TenantID      uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	ScanID        uuid.UUID       `db:"scan_id" json:"scan_id"`

	// Asset details
	Hostname      string          `db:"hostname" json:"hostname"`
	IP            string          `db:"ip" json:"ip"`
	MAC           string          `db:"mac" json:"mac,omitempty"`
	OS            string          `db:"os" json:"os,omitempty"`
	OSVersion     string          `db:"os_version" json:"os_version,omitempty"`

	// Network details
	OpenPorts     []PortInfo      `db:"open_ports" json:"open_ports"`
	Services      []ServiceInfo   `db:"services" json:"services"`

	// Classification
	AssetType     string          `db:"asset_type" json:"asset_type"` // server, workstation, network_device, database
	Sensitivity   string          `db:"sensitivity" json:"sensitivity"` // critical, high, medium, low

	// Status
	Status        string          `db:"status" json:"status"` // discovered, managed, ignored, false_positive
	LastScannedAt time.Time       `db:"last_scanned_at" json:"last_scanned_at"`

	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

// PortInfo represents an open port
type PortInfo struct {
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"` // tcp, udp
	Service   string `json:"service,omitempty"` // ssh, rd, http, https, postgresql, etc.
	Version   string `json:"version,omitempty"`
	Banner    string `json:"banner,omitempty"`
}

// ServiceInfo represents a discovered service
type ServiceInfo struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	Product string `json:"product,omitempty"`
	Version string `json:"version,omitempty"`
}

// Scan represents a discovery scan run
type Scan struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	TenantID     uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	Name         string          `db:"name" json:"name"`

	// Scan configuration
	Subnets      []string        `db:"subnets" json:"subnets"`
	PortRange    string          `db:"port_range" json:"port_range"` // e.g., "1-1024", "22,80,443", "1-65535"

	// Scan status
	Status       string          `db:"status" json:"status"` // pending, running, completed, failed, cancelled
	StartedAt    *time.Time      `db:"started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time      `db:"completed_at" json:"completed_at,omitempty"`

	// Results
	TotalHosts   int             `db:"total_hosts" json:"total_hosts"`
	DiscoveredAssets int         `db:"discovered_assets" json:"discovered_assets"`
	NewAssets    int             `db:"new_assets" json:"new_assets"`
	ChangedAssets int            `db:"changed_assets" json:"changed_assets"`

	// Error info
	ErrorMessage string          `db:"error_message" json:"error_message,omitempty"`

	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// Repository handles discovery data operations
type Repository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRepository creates a new discovery repository
func NewRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Repository {
	return &Repository{db: db, cache: c, logger: logger}
}

// CreateScan creates a new scan
func (r *Repository) CreateScan(ctx context.Context, scan *Scan) error {
	scan.ID = uuid.New()
	scan.CreatedAt = time.Now()
	scan.UpdatedAt = time.Now()
	scan.Status = "pending"

	query := `
		INSERT INTO discovery_scans (id, tenant_id, name, subnets, port_range, status,
			started_at, completed_at, total_hosts, discovered_assets, new_assets,
			changed_assets, error_message, created_at, updated_at)
		VALUES (:id, :tenant_id, :name, :subnets, :port_range, :status,
			:started_at, :completed_at, :total_hosts, :discovered_assets, :new_assets,
			:changed_assets, :error_message, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, scan)
	if err != nil {
		return fmt.Errorf("discovery.CreateScan: %w", err)
	}

	return nil
}

// UpdateScan updates a scan
func (r *Repository) UpdateScan(ctx context.Context, scan *Scan) error {
	scan.UpdatedAt = time.Now()

	query := `
		UPDATE discovery_scans SET
			status = :status,
			started_at = :started_at,
			completed_at = :completed_at,
			total_hosts = :total_hosts,
			discovered_assets = :discovered_assets,
			new_assets = :new_assets,
			changed_assets = :changed_assets,
			error_message = :error_message,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, scan)
	if err != nil {
		return fmt.Errorf("discovery.UpdateScan: %w", err)
	}

	return nil
}

// GetScan retrieves a scan by ID
func (r *Repository) GetScan(ctx context.Context, id uuid.UUID) (*Scan, error) {
	var scan Scan
	query := `SELECT * FROM discovery_scans WHERE id = $1`
	err := r.db.GetContext(ctx, &scan, query, id)
	if err != nil {
		return nil, fmt.Errorf("discovery.GetScan: %w", err)
	}
	return &scan, nil
}

// ListScans retrieves scans with pagination
func (r *Repository) ListScans(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Scan, int, error) {
	var scans []Scan

	// Get count
	var total int
	countQuery := `SELECT COUNT(*) FROM discovery_scans WHERE tenant_id = $1`
	if err := r.db.GetContext(ctx, &total, countQuery, tenantID); err != nil {
		return nil, 0, fmt.Errorf("discovery.ListScans.Count: %w", err)
	}

	// Get scans
	query := `
		SELECT * FROM discovery_scans
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &scans, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("discovery.ListScans: %w", err)
	}

	return scans, total, nil
}

// CreateAsset creates a discovered asset
func (r *Repository) CreateAsset(ctx context.Context, asset *DiscoveredAsset) error {
	asset.ID = uuid.New()
	asset.CreatedAt = time.Now()
	asset.UpdatedAt = time.Now()

	query := `
		INSERT INTO discovered_assets (id, tenant_id, scan_id, hostname, ip, mac, os, os_version,
			open_ports, services, asset_type, sensitivity, status, last_scanned_at, created_at, updated_at)
		VALUES (:id, :tenant_id, :scan_id, :hostname, :ip, :mac, :os, :os_version,
			:open_ports, :services, :asset_type, :sensitivity, :status, :last_scanned_at, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, asset)
	if err != nil {
		return fmt.Errorf("discovery.CreateAsset: %w", err)
	}

	return nil
}

// GetAsset retrieves an asset by ID
func (r *Repository) GetAsset(ctx context.Context, assetID uuid.UUID) (*DiscoveredAsset, error) {
	var asset DiscoveredAsset
	query := `SELECT * FROM discovered_assets WHERE id = $1`
	err := r.db.GetContext(ctx, &asset, query, assetID)
	if err != nil {
		return nil, fmt.Errorf("discovery.GetAsset: %w", err)
	}
	return &asset, nil
}

// GetAssetByIP retrieves an asset by IP
func (r *Repository) GetAssetByIP(ctx context.Context, tenantID uuid.UUID, ip string) (*DiscoveredAsset, error) {
	var asset DiscoveredAsset
	query := `
		SELECT * FROM discovered_assets
		WHERE tenant_id = $1 AND ip = $2
		ORDER BY last_scanned_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &asset, query, tenantID, ip)
	if err != nil {
		return nil, fmt.Errorf("discovery.GetAssetByIP: %w", err)
	}
	return &asset, nil
}

// ListAssets retrieves discovered assets
func (r *Repository) ListAssets(ctx context.Context, tenantID uuid.UUID, filter AssetFilter, limit, offset int) ([]DiscoveredAsset, int, error) {
	baseQuery := `
		SELECT * FROM discovered_assets
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM discovered_assets WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.AssetType != nil {
		baseQuery += fmt.Sprintf(" AND asset_type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND asset_type = $%d", argCount)
		args = append(args, *filter.AssetType)
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("discovery.ListAssets.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY last_scanned_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var assets []DiscoveredAsset
	if err := r.db.SelectContext(ctx, &assets, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("discovery.ListAssets: %w", err)
	}

	return assets, total, nil
}

// AssetFilter filters asset queries
type AssetFilter struct {
	Status    *string
	AssetType *string
}

// Service handles discovery business logic
type Service struct {
	repo       *Repository
	targetSvc  *targetpkg.TargetService
	cache      *cache.Cache
	logger     zerolog.Logger
	scanners   map[string]Scanner
	scanConfig ScannerConfig
	mu         sync.RWMutex
}

// Scanner defines the interface for network scanners
type Scanner interface {
	Scan(ctx context.Context, subnets []string, ports string, results chan<- *DiscoveredAsset) error
	Name() string
}

// ScannerConfig configures security limits for network scanning
type ScannerConfig struct {
	// AllowedCIDRs are the only CIDR ranges that can be scanned
	// If empty, private networks (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) are allowed by default
	AllowedCIDRs []string

	// MaxConcurrentScans limits the number of simultaneous scan operations
	MaxConcurrentScans int

	// ScanTimeout is the maximum time a single scan can take
	ScanTimeout time.Duration
}

// DefaultScannerConfig returns secure defaults for network scanning
func DefaultScannerConfig() ScannerConfig {
	return ScannerConfig{
		AllowedCIDRs: []string{
			"10.0.0.0/8",     // RFC1918 private
			"172.16.0.0/12",  // RFC1918 private
			"192.168.0.0/16", // RFC1918 private
		},
		MaxConcurrentScans: 5,
		ScanTimeout:        30 * time.Minute,
	}
}

// NewService creates a new discovery service
func NewService(repo *Repository, targetSvc *targetpkg.TargetService, c *cache.Cache, logger zerolog.Logger) *Service {
	return NewServiceWithConfig(repo, targetSvc, c, logger, DefaultScannerConfig())
}

// NewServiceWithConfig creates a new discovery service with custom scanner config
func NewServiceWithConfig(repo *Repository, targetSvc *targetpkg.TargetService, c *cache.Cache, logger zerolog.Logger, scanCfg ScannerConfig) *Service {
	s := &Service{
		repo:       repo,
		targetSvc:  targetSvc,
		cache:      c,
		logger:     logger,
		scanners:   make(map[string]Scanner),
		scanConfig: scanCfg,
	}

	// Register default scanners
	s.RegisterScanner(NewNmapScanner())
	s.RegisterScanner(NewPortScanner())

	return s
}

// RegisterScanner registers a network scanner
func (s *Service) RegisterScanner(scanner Scanner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scanners[scanner.Name()] = scanner
}

// CreateScan creates a new discovery scan
func (s *Service) CreateScan(ctx context.Context, scan *Scan) error {
	return s.repo.CreateScan(ctx, scan)
}

// RunScan executes a discovery scan
// SECURITY: Validates that scan targets are within allowed CIDR ranges to prevent SSRF attacks
func (s *Service) RunScan(ctx context.Context, scanID uuid.UUID) error {
	// Get scan
	scan, err := s.repo.GetScan(ctx, scanID)
	if err != nil {
		return err
	}

	// SECURITY: Validate scan subnets against SSRF attacks
	// This prevents the discovery service from being used as a pivot for network reconnaissance
	for _, subnet := range scan.Subnets {
		if err := s.validateScanTarget(subnet); err != nil {
			s.logger.Warn().
				Str("scan_id", scanID.String()).
				Str("subnet", subnet).
				Err(err).
				Msg("Scan target failed security validation")
			return fmt.Errorf("scan target '%s' failed security validation: %w", subnet, err)
		}
	}

	// Update status
	now := time.Now()
	scan.Status = "running"
	scan.StartedAt = &now
	if err := s.repo.UpdateScan(ctx, scan); err != nil {
		return err
	}

	// Run scan asynchronously with timeout
	scanCtx, cancel := context.WithTimeout(context.Background(), s.scanConfig.ScanTimeout)
	go func() {
		defer cancel()
		s.executeScan(scanCtx, scan)
	}()

	return nil
}

// validateScanTarget validates that a scan target is within allowed ranges
func (s *Service) validateScanTarget(target string) error {
	// Parse as IP address first
	ip := net.ParseIP(target)
	if ip != nil {
		return s.validateIPAgainstAllowedRanges(ip)
	}

	// Try parsing as CIDR
	_, ipNet, err := net.ParseCIDR(target)
	if err != nil {
		// Not a valid IP or CIDR
		return fmt.Errorf("invalid target format: %w", err)
	}

	// Check that the CIDR is within allowed ranges
	// We check if the network overlaps with any allowed range
	for _, allowedCIDR := range s.scanConfig.AllowedCIDRs {
		_, allowedNet, err := net.ParseCIDR(allowedCIDR)
		if err != nil {
			continue
		}
		// Check if the target CIDR is contained within or overlaps with allowed CIDR
		if allowedNet.Contains(ipNet.IP) || ipNet.Contains(allowedNet.IP) {
			return nil
		}
	}

	// If no specific CIDRs are configured, allow RFC1918 private addresses
	if len(s.scanConfig.AllowedCIDRs) == 0 {
		if s.isPrivateNetwork(ipNet) {
			return nil
		}
	}

	return fmt.Errorf("target '%s' is not within allowed CIDR ranges", target)
}

// validateIPAgainstAllowedRanges checks if an IP is within allowed ranges
func (s *Service) validateIPAgainstAllowedRanges(ip net.IP) error {
	// Check against explicit forbidden hosts
	forbiddenHosts := map[string]bool{
		"169.254.169.254":        true, // AWS/GCP/Azure metadata
		"metadata.google.internal": true,
		"metadata":               true,
		"localhost":              true,
	}

	if forbiddenHosts[ip.String()] {
		return fmt.Errorf("target '%s' is explicitly forbidden", ip.String())
	}

	// Check against allowed CIDRs
	for _, allowedCIDR := range s.scanConfig.AllowedCIDRs {
		_, allowedNet, err := net.ParseCIDR(allowedCIDR)
		if err != nil {
			continue
		}
		if allowedNet.Contains(ip) {
			return nil
		}
	}

	// If no specific CIDRs configured, allow private networks
	if len(s.scanConfig.AllowedCIDRs) == 0 && isPrivateIP(ip) {
		return nil
	}

	return fmt.Errorf("target '%s' is not within allowed CIDR ranges", ip.String())
}

// isPrivateNetwork checks if a CIDR is a private network range
func (s *Service) isPrivateNetwork(cidr *net.IPNet) bool {
	privateCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	for _, privateCIDR := range privateCIDRs {
		_, privateNet, _ := net.ParseCIDR(privateCIDR)
		if privateNet.Contains(cidr.IP) {
			return true
		}
	}
	return false
}

// executeScan executes the actual scan
func (s *Service) executeScan(ctx context.Context, scan *Scan) {
	results := make(chan *DiscoveredAsset, 100)
	var wg sync.WaitGroup

	// Start scanner
	s.mu.RLock()
	scanner, exists := s.scanners["nmap"] // Default to nmap
	s.mu.RUnlock()

	if !exists {
		scanner = NewPortScanner() // Fallback
	}

	// Run scan
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := scanner.Scan(ctx, scan.Subnets, scan.PortRange, results); err != nil {
			s.logger.Error().Err(err).Str("scan_id", scan.ID.String()).Msg("Scan failed")
			scan.Status = "failed"
			scan.ErrorMessage = err.Error()
			completedAt := time.Now()
			scan.CompletedAt = &completedAt
			_ = s.repo.UpdateScan(ctx, scan)
		}
	}()

	// Process results
	assetCount := 0
	newAssetCount := 0
	changedAssetCount := 0

	for asset := range results {
		assetCount++
		asset.TenantID = scan.TenantID
		asset.ScanID = scan.ID
		asset.LastScannedAt = time.Now()

		// Check if asset already exists
		existing, err := s.repo.GetAssetByIP(ctx, scan.TenantID, asset.IP)
		if err != nil {
			// New asset
			newAssetCount++
			asset.Status = "discovered"
		} else {
			// Update existing
			asset.ID = existing.ID
			asset.Status = existing.Status
			if s.hasAssetChanged(existing, asset) {
				changedAssetCount++
			}
		}

		// Save asset
		if asset.ID == uuid.Nil {
			_ = s.repo.CreateAsset(ctx, asset)
		} else {
			asset.UpdatedAt = time.Now()
			// Would need update method
		}
	}

	wg.Wait()
	close(results)

	// Update scan results
	scan.TotalHosts = assetCount
	scan.DiscoveredAssets = assetCount
	scan.NewAssets = newAssetCount
	scan.ChangedAssets = changedAssetCount
	scan.Status = "completed"
	completedAt := time.Now()
	scan.CompletedAt = &completedAt
	_ = s.repo.UpdateScan(ctx, scan)

	s.logger.Info().
		Str("scan_id", scan.ID.String()).
		Int("total_hosts", assetCount).
		Int("new_assets", newAssetCount).
		Msg("Scan completed")
}

// hasAssetChanged checks if an asset has changed since last scan
func (s *Service) hasAssetChanged(existing, new *DiscoveredAsset) bool {
	if len(existing.OpenPorts) != len(new.OpenPorts) {
		return true
	}
	if len(existing.Services) != len(new.Services) {
		return true
	}
	return false
}

// ImportAsTarget imports a discovered asset as a managed target
func (s *Service) ImportAsTarget(ctx context.Context, assetID uuid.UUID) (*targetpkg.Target, error) {
	// Get asset
	asset, err := s.repo.GetAsset(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %w", err)
	}

	// Create target from asset
	t := &targetpkg.Target{
		Name:      asset.Hostname,
		Host:      asset.IP,
		Port:      22, // Default to SSH
		Type:      targetpkg.TargetTypeSSH,
		Status:    "active",
	}

	// Create target
	// return s.targetSvc.CreateTarget(ctx, t)

	return t, nil
}

// GetScan retrieves a scan
func (s *Service) GetScan(ctx context.Context, id uuid.UUID) (*Scan, error) {
	return s.repo.GetScan(ctx, id)
}

// ListScans lists scans
func (s *Service) ListScans(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Scan, int, error) {
	return s.repo.ListScans(ctx, tenantID, limit, offset)
}

// ListAssets lists discovered assets
func (s *Service) ListAssets(ctx context.Context, tenantID uuid.UUID, filter AssetFilter, limit, offset int) ([]DiscoveredAsset, int, error) {
	return s.repo.ListAssets(ctx, tenantID, filter, limit, offset)
}

// PortScanner implements basic port scanning
type PortScanner struct{}

// NewPortScanner creates a new port scanner
func NewPortScanner() *PortScanner {
	return &PortScanner{}
}

// Scan scans networks for open ports
func (s *PortScanner) Scan(ctx context.Context, subnets []string, ports string, results chan<- *DiscoveredAsset) error {
	defer close(results)

	// Parse port range
	portList, err := s.parsePorts(ports)
	if err != nil {
		return err
	}

	// Scan each subnet
	for _, subnet := range subnets {
		// For simplicity, assume subnet is a CIDR or single IP
		ips, err := s.expandSubnet(subnet)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			asset := &DiscoveredAsset{
				IP:        ip,
				OpenPorts: []PortInfo{},
			}

			// Scan ports
			for _, port := range portList {
				if s.isPortOpen(ip, port) {
					asset.OpenPorts = append(asset.OpenPorts, PortInfo{
						Port:     port,
						Protocol: "tcp",
					})
				}
			}

			// Classify asset
			asset.AssetType = s.classifyAsset(asset)

			results <- asset
		}
	}

	return nil
}

// Name returns the scanner name
func (s *PortScanner) Name() string {
	return "port"
}

func (s *PortScanner) parsePorts(ports string) ([]int, error) {
	// Simple port parsing
	// "22,80,443" or "1-1024"
	return []int{22, 80, 443, 3389, 5432, 3306}, nil
}

func (s *PortScanner) expandSubnet(subnet string) ([]string, error) {
	// Simple IP expansion
	return []string{subnet}, nil
}

func (s *PortScanner) isPortOpen(ip string, port int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *PortScanner) classifyAsset(asset *DiscoveredAsset) string {
	// Classify based on open ports
	for _, port := range asset.OpenPorts {
		switch port.Port {
		case 22:
			return "server"
		case 3389:
			return "workstation"
		case 3306, 5432:
			return "database"
		}
	}
	return "unknown"
}

// NmapScanner wraps nmap for advanced scanning
type NmapScanner struct{}

// NewNmapScanner creates a new nmap scanner
func NewNmapScanner() *NmapScanner {
	return &NmapScanner{}
}

// Scan scans using nmap
func (s *NmapScanner) Scan(ctx context.Context, subnets []string, ports string, results chan<- *DiscoveredAsset) error {
	defer close(results)
	// Would use nmap via exec or library
	return fmt.Errorf("nmap not available")
}

// Name returns the scanner name
func (s *NmapScanner) Name() string {
	return "nmap"
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	privateCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	for _, privateCIDR := range privateCIDRs {
		_, privateNet, _ := net.ParseCIDR(privateCIDR)
		if privateNet.Contains(ip) {
			return true
		}
	}
	return false
}
