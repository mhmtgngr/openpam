package registry

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ServiceStatus represents the health status of a service
type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusUnknown   ServiceStatus = "unknown"
	StatusDegraded  ServiceStatus = "degraded"
)

// ServiceInfo holds metadata about a registered service
type ServiceInfo struct {
	Name      string            `json:"name"`
	BaseURL   string            `json:"base_url"`
	Port      int               `json:"port"`
	Status    ServiceStatus     `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	LastCheck time.Time         `json:"last_check"`
}

// Registry is a service registry that tracks available microservices and their health
type Registry struct {
	services map[string]*ServiceInfo
	mu       sync.RWMutex
	logger   zerolog.Logger
	client   *http.Client
	stopCh   chan struct{}
}

// New creates a new service registry
func New(logger zerolog.Logger) *Registry {
	return &Registry{
		services: make(map[string]*ServiceInfo),
		logger:   logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		stopCh: make(chan struct{}),
	}
}

// Register adds a service to the registry
func (r *Registry) Register(name, baseURL string, port int, metadata map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[name] = &ServiceInfo{
		Name:     name,
		BaseURL:  baseURL,
		Port:     port,
		Status:   StatusUnknown,
		Metadata: metadata,
	}

	r.logger.Info().
		Str("service", name).
		Str("url", baseURL).
		Int("port", port).
		Msg("Service registered")
}

// Deregister removes a service from the registry
func (r *Registry) Deregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, name)
}

// Resolve returns the base URL for a named service
func (r *Registry) Resolve(name string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, ok := r.services[name]
	if !ok {
		return "", fmt.Errorf("registry: service %q not found", name)
	}

	if svc.Status == StatusUnhealthy {
		return "", fmt.Errorf("registry: service %q is unhealthy", name)
	}

	return svc.BaseURL, nil
}

// GetService returns full service info
func (r *Registry) GetService(name string) (*ServiceInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, ok := r.services[name]
	if !ok {
		return nil, fmt.Errorf("registry: service %q not found", name)
	}

	return svc, nil
}

// ListServices returns all registered services
func (r *Registry) ListServices() map[string]*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*ServiceInfo, len(r.services))
	for k, v := range r.services {
		copy := *v
		result[k] = &copy
	}
	return result
}

// HealthSummary returns aggregated health of all services
func (r *Registry) HealthSummary() map[string]ServiceStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	summary := make(map[string]ServiceStatus, len(r.services))
	for name, svc := range r.services {
		summary[name] = svc.Status
	}
	return summary
}

// StartHealthChecks begins periodic health checking for all registered services
func (r *Registry) StartHealthChecks(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				r.checkAll(ctx)
			case <-r.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops health checking
func (r *Registry) Stop() {
	close(r.stopCh)
}

func (r *Registry) checkAll(ctx context.Context) {
	r.mu.RLock()
	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	r.mu.RUnlock()

	for _, name := range names {
		r.checkService(ctx, name)
	}
}

func (r *Registry) checkService(ctx context.Context, name string) {
	r.mu.RLock()
	svc, ok := r.services[name]
	if !ok {
		r.mu.RUnlock()
		return
	}
	healthURL := svc.BaseURL + "/health"
	r.mu.RUnlock()

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		r.updateStatus(name, StatusUnhealthy)
		return
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.updateStatus(name, StatusUnhealthy)
		r.logger.Warn().Str("service", name).Err(err).Msg("Health check failed")
		return
	}
	defer resp.Body.Close()

	status := StatusHealthy
	if resp.StatusCode >= 500 {
		status = StatusUnhealthy
	} else if resp.StatusCode >= 400 {
		status = StatusDegraded
	}

	r.updateStatus(name, status)
}

func (r *Registry) updateStatus(name string, status ServiceStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if svc, ok := r.services[name]; ok {
		svc.Status = status
		svc.LastCheck = time.Now()
	}
}
