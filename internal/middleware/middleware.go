// Package middleware provides pluggable HTTP middleware for OpenPAM services.
//
// Middleware is organized into focused files by concern:
//   - auth.go:      Authentication, authorization, MFA verification
//   - tenant.go:    Multi-tenant isolation enforcement
//   - security.go:  CORS, security headers, body limits, content-type validation
//   - logging.go:   Structured logging, request IDs, panic recovery, error handling
//   - sanitize.go:  Sensitive field redaction for audit logging
//   - ratelimit.go: Pluggable rate limiting (per-IP, per-user, per-tenant, per-endpoint)
//   - audit.go:     Structured audit trail middleware
package middleware

// Config holds middleware configuration for CORS and request handling.
type Config struct {
	TrustProxy      bool
	AllowedOrigins  []string
	AllowedMethods  []string
	AllowedHeaders  []string
	ExposeHeaders   []string
	MaxRequestBody  int64
	EnableRequestID bool
}

// join concatenates strings with a separator.
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
