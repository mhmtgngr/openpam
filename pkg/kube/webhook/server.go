package kube

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/pkg/kube/rbac"
	"github.com/rs/zerolog"
	"k8s.io/api/admission/v1"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// WebhookServer handles Kubernetes admission webhooks
type WebhookServer struct {
	server       *http.Server
	rbacHandlers *rbac.HandlerManager
	config       WebhookConfig
	logger       zerolog.Logger
	serializer   runtime.Serializer
	codecFactory serializer.CodecFactory
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	Port           int
	CertFile       string
	KeyFile        string
	CABundleFile   string
	TLSCertFile    string
	TLSKeyFile     string
	EnableTLS      bool
}

// DefaultWebhookConfig returns default webhook configuration
func DefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{
		Port:      9443,
		EnableTLS: true,
	}
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(port int, rbacHandlers *rbac.HandlerManager, logger zerolog.Logger) (*WebhookServer, error) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)

	return &WebhookServer{
		rbacHandlers: rbacHandlers,
		config: WebhookConfig{
			Port:      port,
			EnableTLS: false, // Default to HTTP for local dev
		},
		logger:       logger,
		serializer:   codecs.LegacyCodec(),
		codecFactory: codecs,
	}, nil
}

// Start starts the webhook server
func (w *WebhookServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", w.serveValidate)
	mux.HandleFunc("/mutate", w.serveMutate)
	mux.HandleFunc("/ready", w.serveReady)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", w.config.Port),
		Handler: mux,
	}

	if w.config.EnableTLS {
		// Configure TLS
		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	w.server = server

	w.logger.Info().
		Int("port", w.config.Port).
		Bool("tls", w.config.EnableTLS).
		Msg("Webhook server starting")

	// Start server in background
	go func() {
		if w.config.EnableTLS {
			if err := server.ListenAndServeTLS(w.config.TLSCertFile, w.config.TLSKeyFile); err != nil {
				w.logger.Error().Err(err).Msg("Webhook server TLS failed")
			}
		} else {
			if err := server.ListenAndServe(); err != nil {
				w.logger.Error().Err(err).Msg("Webhook server failed")
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	return nil
}

// serveReady handles readiness probes
func (w *WebhookServer) serveReady(wr http.ResponseWriter, r *http.Request) {
	wr.WriteHeader(http.StatusOK)
	wr.Write([]byte("OK"))
}

// serveValidate handles validation webhook requests
func (w *WebhookServer) serveValidate(wr http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.logger.Error().Err(err).Msg("Failed to read request body")
			http.Error(wr, "failed to read request body", http.StatusBadRequest)
			return
		}
		body = data
	}

	// Parse AdmissionReview request
	ar := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &ar); err != nil {
		w.logger.Error().Err(err).Msg("Failed to parse AdmissionReview")
		http.Error(wr, "failed to parse AdmissionReview", http.StatusBadRequest)
		return
	}

	// Create response
	response := v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &v1.AdmissionResponse{
			UID: ar.Request.UID,
		},
	}

	// Validate the request
	allowed, reason := w.validateAdmission(r, ar.Request)

	response.Response.Allowed = allowed
	if !allowed {
		response.Response.Result = &metav1.Status{
			Message: reason,
		}
	}

	// Marshal response
	respBytes, err := json.Marshal(response)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to marshal response")
		http.Error(wr, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(http.StatusOK)
	wr.Write(respBytes)

	w.logger.Debug().
		Bool("allowed", allowed).
		Str("uid", string(ar.Request.UID)).
		Msg("Validation webhook processed")
}

// serveMutate handles mutation webhook requests
func (w *WebhookServer) serveMutate(wr http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.logger.Error().Err(err).Msg("Failed to read request body")
			http.Error(wr, "failed to read request body", http.StatusBadRequest)
			return
		}
		body = data
	}

	// Parse AdmissionReview request
	ar := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &ar); err != nil {
		w.logger.Error().Err(err).Msg("Failed to parse AdmissionReview")
		http.Error(wr, "failed to parse AdmissionReview", http.StatusBadRequest)
		return
	}

	// Create response
	response := v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &v1.AdmissionResponse{
			UID: ar.Request.UID,
		},
	}

	// Mutate the request if needed
	patch := w.mutateAdmission(r, ar.Request)

	if patch != nil {
		response.Response.Allowed = true
		response.Response.Patch = patch
		patchType := v1.PatchTypeJSONPatch
		response.Response.PatchType = &patchType
	} else {
		response.Response.Allowed = true
	}

	// Marshal response
	respBytes, err := json.Marshal(response)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to marshal response")
		http.Error(wr, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(http.StatusOK)
	wr.Write(respBytes)
}

// validateAdmission validates an admission request
func (w *WebhookServer) validateAdmission(r *http.Request, req *v1.AdmissionRequest) (bool, string) {
	// Extract user info from request headers (set by API server)
	user := r.Header.Get("X-Remote-User")
	groups := r.Header.Values("X-Remote-Group")

	// Check if this is an OpenPAM-managed resource
	if req.Kind.Kind == "Pod" {
		return w.validatePodCreation(user, groups, req)
	}

	// Default to allowing the request
	return true, ""
}

// validatePodCreation validates pod creation requests
func (w *WebhookServer) validatePodCreation(user string, groups []string, req *v1.AdmissionRequest) (bool, string) {
	// Parse the pod object
	var pod struct {
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string           `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			ServiceAccountName string `json:"serviceAccountName"`
			Containers []struct {
				Name  string `json:"name"`
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
	}

	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		w.logger.Error().Err(err).Msg("Failed to parse pod object")
		return false, "failed to parse pod"
	}

	// Check if pod has JIT access label
	if _, hasJITLabel := pod.Metadata.Labels["openpam-jit-access"]; hasJITLabel {
		// Verify that a valid JIT grant exists
		// This would check against our database
		if !w.verifyJITAccess(user, pod.Metadata.Namespace) {
			return false, "no valid JIT access grant found for user/namespace"
		}
	}

	// Check if service account is the JIT service account
	if pod.Spec.ServiceAccountName == "openpam-jit" {
		if !w.verifyJITAccess(user, pod.Metadata.Namespace) {
			return false, "no valid JIT access grant found for JIT service account"
		}
	}

	return true, ""
}

// mutateAdmission mutates an admission request
func (w *WebhookServer) mutateAdmission(r *http.Request, req *v1.AdmissionRequest) []byte {
	// Add annotations for tracking
	annotations := map[string]string{
		"openpam.org/request-id": uuid.New().String(),
		"openpam.org/user":       r.Header.Get("X-Remote-User"),
		"openpam.org/timestamp":  time.Now().Format(time.RFC3339),
	}

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": annotations,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to marshal patch")
		return nil
	}

	return patchBytes
}

// verifyJITAccess verifies that a user has valid JIT access
func (w *WebhookServer) verifyJITAccess(user, namespace string) bool {
	// This would check against the database
	// For now, return false to require explicit grants
	return false
}

// GenerateMutatingWebhookConfig generates the MutatingWebhookConfiguration manifest
func (w *WebhookServer) GenerateMutatingWebhookConfig(caBundle []byte) string {
	return fmt.Sprintf(`apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: openpam-jit-webhook
  namespace: openpam-system
webhooks:
- name: jit-access.mutating.openpam.org
  rules:
  - operations: ["CREATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
    scope: "Namespaced"
  clientConfig:
    service:
      namespace: openpam-system
      name: openpam-webhook
      path: /mutate
    caBundle: %s
  admissionReviewVersions: ["v1"]
  sideEffects: None
  timeoutSeconds: 5
`, caBundle)
}

// GenerateValidatingWebhookConfig generates the ValidatingWebhookConfiguration manifest
func (w *WebhookServer) GenerateValidatingWebhookConfig(caBundle []byte) string {
	return fmt.Sprintf(`apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: openpam-jit-validation
  namespace: openpam-system
webhooks:
- name: jit-access.validating.openpam.org
  rules:
  - operations: ["CREATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
    scope: "Namespaced"
  clientConfig:
    service:
      namespace: openpam-system
      name: openpam-webhook
      path: /validate
    caBundle: %s
  admissionReviewVersions: ["v1"]
  sideEffects: None
  timeoutSeconds: 5
  failurePolicy: Fail
`, caBundle)
}
