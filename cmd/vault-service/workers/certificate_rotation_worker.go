package workers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/vault/cert"
	"github.com/openpam/openpam/internal/vault/certificate"
	"github.com/openpam/openpam/internal/vault/repository"
	"github.com/rs/zerolog"
)

// CertificateRotationWorker handles automatic certificate rotation
type CertificateRotationWorker struct {
	db          *sqlx.DB
	cache       *cache.Cache
	certManager *certificate.Manager
	eventBus    *events.EventBus
	repo        *repository.CertificateRepository
	logger      zerolog.Logger

	// Configuration
	checkInterval time.Duration
	renewBefore   time.Duration
	batchSize     int

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// Config holds the worker configuration
type Config struct {
	CheckInterval time.Duration // How often to check for expiring certificates
	RenewBefore   time.Duration // How long before expiration to renew certificates
	BatchSize     int           // Number of certificates to process per batch
}

// NewCertificateRotationWorker creates a new certificate rotation worker
func NewCertificateRotationWorker(
	db *sqlx.DB,
	cache *cache.Cache,
	certManager *certificate.Manager,
	eventBus *events.EventBus,
	cfg Config,
	logger zerolog.Logger,
) *CertificateRotationWorker {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 1 * time.Hour
	}
	if cfg.RenewBefore == 0 {
		cfg.RenewBefore = 30 * 24 * time.Hour // 30 days
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10
	}

	return &CertificateRotationWorker{
		db:            db,
		cache:         cache,
		certManager:   certManager,
		eventBus:      eventBus,
		repo:          repository.NewCertificateRepository(db, cache, logger),
		logger:        logger,
		checkInterval: cfg.CheckInterval,
		renewBefore:   cfg.RenewBefore,
		batchSize:     cfg.BatchSize,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the rotation worker
func (w *CertificateRotationWorker) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("certificate_rotation_worker: already running")
	}

	w.running = true

	// Run initial check
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run()
	}()

	w.logger.Info().
		Dur("check_interval", w.checkInterval).
		Dur("renew_before", w.renewBefore).
		Msg("Certificate rotation worker started")

	return nil
}

// Stop stops the rotation worker gracefully
func (w *CertificateRotationWorker) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()

	w.logger.Info().Msg("Certificate rotation worker stopped")
	return nil
}

// run is the main worker loop
func (w *CertificateRotationWorker) run() {
	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.checkAndRotate()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.checkAndRotate()
		}
	}
}

// checkAndRotate checks for certificates needing rotation
func (w *CertificateRotationWorker) checkAndRotate() {
	w.logger.Debug().Msg("Checking for certificates to rotate")

	ctx := context.Background()

	// Get certificates expiring within the renewal window
	expiring, err := w.repo.GetExpiring(ctx, time.Now().Add(w.renewBefore))
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to get expiring certificates")
		return
	}

	if len(expiring) == 0 {
		w.logger.Debug().Msg("No certificates need rotation")
		return
	}

	w.logger.Info().
		Int("count", len(expiring)).
		Msg("Found certificates requiring rotation")

	// Process certificates in batches
	for i := 0; i < len(expiring); i += w.batchSize {
		end := i + w.batchSize
		if end > len(expiring) {
			end = len(expiring)
		}

		batch := expiring[i:end]
		w.processBatch(ctx, batch)
	}
}

// processBatch processes a batch of certificates
func (w *CertificateRotationWorker) processBatch(ctx context.Context, certs []*cert.Certificate) {
	for _, cert := range certs {
		if err := w.rotateCertificate(ctx, cert); err != nil {
			w.logger.Error().
				Err(err).
				Str("certificate_id", cert.ID.String()).
				Str("name", cert.Name).
				Msg("Failed to rotate certificate")
		}
	}
}

// rotateCertificate rotates a single certificate
func (w *CertificateRotationWorker) rotateCertificate(ctx context.Context, certificate *cert.Certificate) error {
	w.logger.Info().
		Str("certificate_id", certificate.ID.String()).
		Str("name", certificate.Name).
		Str("type", string(certificate.Type)).
		Time("expires", certificate.NotAfter).
		Msg("Rotating certificate")

	// Mark as renewing
	certificate.Status = cert.StatusRenewing
	certificate.UpdatedAt = time.Now()
	if err := w.repo.Update(ctx, certificate); err != nil {
		return fmt.Errorf("failed to update certificate status: %w", err)
	}

	// Renew the certificate
	newCert, err := w.certManager.RenewCertificate(ctx, certificate.ID)
	if err != nil {
		// Revert status on error
		certificate.Status = cert.StatusActive
		_ = w.repo.Update(ctx, certificate)
		return fmt.Errorf("failed to renew certificate: %w", err)
	}

	w.logger.Info().
		Str("old_cert_id", certificate.ID.String()).
		Str("new_cert_id", newCert.ID.String()).
		Time("new_expires", newCert.NotAfter).
		Msg("Certificate rotated successfully")

	// Publish event
	if w.eventBus != nil {
		_ = w.eventBus.Publish(ctx, events.Event{
			Type:     "certificate.rotated",
			TenantID: certificate.TenantID.String(),
			ActorID:  "system",
			Action:   "rotate",
			Resource: "certificate",
			Data: map[string]interface{}{
				"old_certificate_id": certificate.ID.String(),
				"new_certificate_id": newCert.ID.String(),
				"type":              string(certificate.Type),
			},
		})
	}

	return nil
}

// CheckCertificate checks if a certificate needs rotation
func (w *CertificateRotationWorker) CheckCertificate(ctx context.Context, certID string) (bool, time.Duration, error) {
	id, err := uuid.Parse(certID)
	if err != nil {
		return false, 0, fmt.Errorf("invalid certificate ID: %w", err)
	}
	certificate, err := w.repo.GetByID(ctx, id)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get certificate: %w", err)
	}

	if certificate.Status != cert.StatusActive {
		return false, 0, fmt.Errorf("certificate not active")
	}

	now := time.Now()
	timeUntilExpiry := certificate.NotAfter.Sub(now)

	needsRotation := timeUntilExpiry <= w.renewBefore
	return needsRotation, timeUntilExpiry, nil
}

// RotateCertificateNow forces immediate rotation of a certificate
func (w *CertificateRotationWorker) RotateCertificateNow(ctx context.Context, certID string) error {
	id, err := uuid.Parse(certID)
	if err != nil {
		return fmt.Errorf("invalid certificate ID: %w", err)
	}
	certificate, err := w.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	return w.rotateCertificate(ctx, certificate)
}

// GetRotationStatus returns the status of certificate rotation
func (w *CertificateRotationWorker) GetRotationStatus(ctx context.Context) (*RotationStatus, error) {
	expiring, err := w.repo.GetExpiring(ctx, time.Now().Add(w.renewBefore))
	if err != nil {
		return nil, err
	}

	expired, err := w.repo.GetExpired(ctx)
	if err != nil {
		return nil, err
	}

	return &RotationStatus{
		Running:        w.running,
		CheckInterval:  w.checkInterval,
		RenewBefore:    w.renewBefore,
		ExpiringCount:  len(expiring),
		ExpiredCount:   len(expired),
		NextCheckTime:  time.Now().Add(w.checkInterval),
	}, nil
}

// RotationStatus represents the status of certificate rotation
type RotationStatus struct {
	Running       bool          `json:"running"`
	CheckInterval time.Duration `json:"check_interval"`
	RenewBefore   time.Duration `json:"renew_before"`
	ExpiringCount int           `json:"expiring_count"`
	ExpiredCount  int           `json:"expired_count"`
	NextCheckTime time.Time     `json:"next_check_time"`
}

// RotationWorker is a singleton instance
var rotationWorker *CertificateRotationWorker

// InitRotationWorker initializes the global rotation worker
func InitRotationWorker(
	db *sqlx.DB,
	cache *cache.Cache,
	certManager *certificate.Manager,
	eventBus *events.EventBus,
	cfg Config,
	logger zerolog.Logger,
) {
	rotationWorker = NewCertificateRotationWorker(db, cache, certManager, eventBus, cfg, logger)
}

// GetRotationWorker returns the global rotation worker
func GetRotationWorker() *CertificateRotationWorker {
	return rotationWorker
}
