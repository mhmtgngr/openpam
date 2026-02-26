package operator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/openpam/openpam/pkg/kube/rbac"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Manager manages the Kubernetes operator for OpenPAM
type Manager struct {
	config       *rest.Config
	clientSet    *kubernetes.Clientset
	dynamicClient dynamic.Interface
	rbacManager  *rbac.HandlerManager
	logger       zerolog.Logger

	// Resource tracking
	watchedResources map[schema.GroupVersionResource]*ResourceWatcher
	mu              sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Config holds the operator configuration
type Config struct {
	KubeconfigPath      string
	Namespace           string
	WebhookPort         int
	MetricsPort         int
	LeaderElection      bool
	LeaderElectionID    string
	ResyncInterval      time.Duration
	SessionTimeout      time.Duration
	AuditLoggingEnabled bool
}

// ResourceWatcher watches a Kubernetes resource type
type ResourceWatcher struct {
	gvr        schema.GroupVersionResource
	controller Controller
}

// Controller handles events for a resource type
type Controller interface {
	Start(ctx context.Context) error
	Stop() error
}

// NewManager creates a new Kubernetes operator manager
func NewManager(cfg Config, logger zerolog.Logger) (*Manager, error) {
	// Create Kubernetes config
	var restConfig *rest.Config
	var err error

	if cfg.KubeconfigPath != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	} else {
		restConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("operator: failed to create kubernetes config: %w", err)
	}

	// Create clientset
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("operator: failed to create kubernetes clientset: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("operator: failed to create dynamic client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize RBAC manager
	rbacManager, err := rbac.NewHandlerManager(clientSet, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("operator: failed to create rbac manager: %w", err)
	}

	return &Manager{
		config:        restConfig,
		clientSet:     clientSet,
		dynamicClient: dynamicClient,
		rbacManager:   rbacManager,
		logger:        logger,
		watchedResources: make(map[schema.GroupVersionResource]*ResourceWatcher),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start starts the operator
func (m *Manager) Start() error {
	m.logger.Info().Msg("Starting Kubernetes operator")

	// Start all resource watchers
	m.mu.RLock()
	for gvr, watcher := range m.watchedResources {
		gvr := gvr // capture loop variable
		watcher := watcher
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.logger.Info().
				Stringer("gvr", gvr).
				Msg("Starting resource watcher")
			if err := watcher.controller.Start(m.ctx); err != nil {
				m.logger.Error().
					Err(err).
					Stringer("gvr", gvr).
					Msg("Resource watcher error")
			}
		}()
	}
	m.mu.RUnlock()

	// Start all resource watchers
	m.mu.RLock()
	for gvr, watcher := range m.watchedResources {
		gvr := gvr // capture loop variable
		watcher := watcher
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.logger.Info().
				Stringer("gvr", gvr).
				Msg("Starting resource watcher")
			if err := watcher.controller.Start(m.ctx); err != nil {
				m.logger.Error().
					Err(err).
					Stringer("gvr", gvr).
					Msg("Resource watcher error")
			}
		}()
	}
	m.mu.RUnlock()

	m.logger.Info().Msg("Kubernetes operator started")
	return nil
}

// Stop stops the operator gracefully
func (m *Manager) Stop() error {
	m.logger.Info().Msg("Stopping Kubernetes operator")

	// Cancel context to signal all goroutines
	m.cancel()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info().Msg("Kubernetes operator stopped")
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("operator: timeout waiting for goroutines to finish")
	}
}

// WatchResource starts watching a Kubernetes resource type
func (m *Manager) WatchResource(gvr schema.GroupVersionResource, controller Controller) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.watchedResources[gvr] = &ResourceWatcher{
		gvr:        gvr,
		controller: controller,
	}

	m.logger.Info().
		Stringer("gvr", gvr).
		Msg("Registered resource watcher")
}

// GetClientSet returns the Kubernetes clientset
func (m *Manager) GetClientSet() *kubernetes.Clientset {
	return m.clientSet
}

// GetDynamicClient returns the dynamic client
func (m *Manager) GetDynamicClient() dynamic.Interface {
	return m.dynamicClient
}

// GetRBACManager returns the RBAC manager
func (m *Manager) GetRBACManager() *rbac.HandlerManager {
	return m.rbacManager
}

// GetScheme returns the runtime scheme for API types
func (m *Manager) GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	return scheme
}

// Health checks the health of the operator
func (m *Manager) Health(ctx context.Context) error {
	// Check Kubernetes connection
	if _, err := m.clientSet.ServerVersion(); err != nil {
		return fmt.Errorf("operator: kubernetes health check failed: %w", err)
	}
	return nil
}

// CreateAccessRequest creates a Kubernetes access request (JIT role binding)
func (m *Manager) CreateAccessRequest(ctx context.Context, req *AccessRequest) (*AccessResponse, error) {
	// TODO: Implement JIT access request using rbac.HandlerManager
	return nil, fmt.Errorf("operator: CreateAccessRequest not implemented")
}

// RevokeAccess revokes a Kubernetes access grant
func (m *Manager) RevokeAccess(ctx context.Context, requestID string) error {
	// TODO: Implement access revocation using rbac.HandlerManager
	return fmt.Errorf("operator: RevokeAccess not implemented")
}

// ListActiveAccess returns all active access grants
func (m *Manager) ListActiveAccess(ctx context.Context) ([]*AccessGrant, error) {
	// TODO: Implement access listing using rbac.HandlerManager
	return nil, fmt.Errorf("operator: ListActiveAccess not implemented")
}

// AccessRequest represents a JIT Kubernetes access request
type AccessRequest struct {
	RequestID   string            `json:"request_id"`
	Username    string            `json:"username"`
	Groups      []string          `json:"groups"`
	Namespace   string            `json:"namespace"`
	Role        string            `json:"role"`
	Kind        string            `json:"kind"`        // Role, ClusterRole
	Resource    string            `json:"resource"`    // RoleBinding, ClusterRoleBinding
	Duration    time.Duration     `json:"duration"`
	Reason      string            `json:"reason"`
	Metadata    map[string]string `json:"metadata"`
}

// AccessResponse represents the response to an access request
type AccessResponse struct {
	RequestID    string    `json:"request_id"`
	BindingName  string    `json:"binding_name"`
	Namespace    string    `json:"namespace"`
	GrantedAt    time.Time `json:"granted_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	KubeConfig  string    `json:"kubeconfig,omitempty"`
}

// AccessGrant represents an active access grant
type AccessGrant struct {
	GrantID     string    `json:"grant_id"`
	RequestID   string    `json:"request_id"`
	Username    string    `json:"username"`
	Namespace   string    `json:"namespace"`
	Role        string    `json:"role"`
	BindingName string    `json:"binding_name"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}
