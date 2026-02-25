// Package security provides SSRF (Server-Side Request Forgery) protection
// for validating user-supplied hostnames and IPs before making network connections.
package security

import (
	"net"
	"strings"
)

// ForbiddenHosts contains hostnames that should never be allowed as targets
// This includes metadata endpoints and localhost variants
// IPv6 loopback (::1) is handled via IP validation since it parses as an IP address
var ForbiddenHosts = map[string]bool{
	// Cloud metadata endpoints (AWS/GCP/Azure all use 169.254.169.254)
	"169.254.169.254":        true,
	"metadata.google.internal": true,
	"metadata":               true,
	// localhost variants (hostname variants, not IPs)
	"localhost": true,
	"localhost.localdomain": true,
}

// ForbiddenSuffixes contains domain suffixes that should never be allowed
var ForbiddenSuffixes = []string{
	".local",
	".internal",
	".corp",
	".private",
}

// PrivateCIDRs contains CIDR blocks for private IP ranges that should be blocked
// unless explicitly allowed by configuration
var PrivateCIDRs = []*net.IPNet{
	// IPv4 private ranges
	parseCIDR("10.0.0.0/8"),
	parseCIDR("172.16.0.0/12"),
	parseCIDR("192.168.0.0/16"),
	parseCIDR("127.0.0.0/8"),    // Loopback
	parseCIDR("169.254.0.0/16"), // Link-local
	// IPv6 private ranges
	parseCIDR("fc00::/7"),      // Unique local
	parseCIDR("fe80::/10"),     // Link-local
	parseCIDR("::1/128"),       // Loopback
}

// SSRFValidatorConfig configures SSRF validation behavior
type SSRFValidatorConfig struct {
	// AllowPrivateNetworks permits connections to private IP ranges
	// WARNING: Only enable this if you understand the SSRF risks
	AllowPrivateNetworks bool

	// AllowLinkLocal permits link-local addresses (169.254.0.0/16)
	AllowLinkLocal bool

	// AllowLoopback permits loopback addresses (127.0.0.0/8, ::1)
	AllowLoopback bool

	// AllowedHosts is a whitelist of explicitly permitted hosts
	AllowedHosts map[string]bool

	// BlockedHosts is a blacklist of explicitly blocked hosts
	// This is in addition to the default ForbiddenHosts
	BlockedHosts map[string]bool
}

// DefaultSSRFValidatorConfig returns the default secure configuration
// This configuration blocks all private networks, link-local, and loopback
func DefaultSSRFValidatorConfig() SSRFValidatorConfig {
	return SSRFValidatorConfig{
		AllowPrivateNetworks: false,
		AllowLinkLocal:       false,
		AllowLoopback:        false,
		AllowedHosts:         make(map[string]bool),
		BlockedHosts:         make(map[string]bool),
	}
}

// ValidateTargetHost validates a target hostname or IP to prevent SSRF attacks
// Returns an error if the target is forbidden or potentially dangerous
func ValidateTargetHost(target string, cfg SSRFValidatorConfig) error {
	if target == "" {
		return &SSRFValidationError{Target: target, Reason: "target cannot be empty"}
	}

	// Trim whitespace
	target = strings.TrimSpace(target)

	// Check against explicitly forbidden hosts
	if ForbiddenHosts[target] || cfg.BlockedHosts[target] {
		return &SSRFValidationError{Target: target, Reason: "host is explicitly forbidden"}
	}

	// Check against forbidden suffixes
	for _, suffix := range ForbiddenSuffixes {
		if strings.HasSuffix(target, suffix) {
			return &SSRFValidationError{Target: target, Reason: "host has forbidden suffix: " + suffix}
		}
	}

	// Parse as IP address first
	if ip := net.ParseIP(target); ip != nil {
		return validateIP(ip, cfg)
	}

	// If it contains a colon, it might be host:port format
	if strings.Contains(target, ":") {
		host, _, err := net.SplitHostPort(target)
		if err == nil {
			// Recursively validate just the host portion
			return ValidateTargetHost(host, cfg)
		}
	}

	// Resolve hostname to check if it resolves to a forbidden IP
	// WARNING: DNS resolution can be slow and is subject to DNS rebinding attacks
	// In production, consider maintaining a cache of resolved IPs
	ips, err := net.LookupIP(target)
	if err != nil {
		// If DNS resolution fails, we conservatively reject the target
		// This prevents attackers from causing DNS queries to internal networks
		return &SSRFValidationError{Target: target, Reason: "DNS resolution failed"}
	}

	// Check all resolved IPs
	for _, ip := range ips {
		if err := validateIP(ip, cfg); err != nil {
			return &SSRFValidationError{Target: target, Reason: "resolves to forbidden IP: " + err.Error()}
		}
	}

	// Check against allowed hosts whitelist (if configured)
	if len(cfg.AllowedHosts) > 0 && !cfg.AllowedHosts[target] {
		return &SSRFValidationError{Target: target, Reason: "host not in allowed list"}
	}

	return nil
}

// validateIP validates an IP address against SSRF rules
func validateIP(ip net.IP, cfg SSRFValidatorConfig) error {
	// SPECIAL CASE: Check for loopback IPs FIRST before checking CIDR ranges
	// This handles both IPv4 (127.0.0.0/8) and IPv6 (::1) loopback addresses
	if ip.IsLoopback() {
		if cfg.AllowLoopback {
			return nil
		}
		return &SSRFValidationError{Target: ip.String(), Reason: "IP is a loopback address"}
	}

	// Check if IP is in any of the private CIDR ranges
	for _, cidr := range PrivateCIDRs {
		if cidr != nil && cidr.Contains(ip) {
			// Check if this specific range is allowed
			if isLinkLocal(cidr) && cfg.AllowLinkLocal {
				continue
			}
			if isPrivateNetwork(cidr) && cfg.AllowPrivateNetworks {
				continue
			}
			return &SSRFValidationError{Target: ip.String(), Reason: "IP is in forbidden range: " + cidr.String()}
		}
	}
	return nil
}

// isLoopback checks if a CIDR is a loopback range
func isLoopback(cidr *net.IPNet) bool {
	ones, _ := cidr.Mask.Size()
	if cidr.IP.To4() != nil {
		// IPv4 loopback: 127.0.0.0/8
		return cidr.IP.Equal(net.ParseIP("127.0.0.0")) && ones == 8
	}
	// IPv6 loopback: ::1/128 (or ::/128 with ::1 being the specific address)
	// Check if this is the loopback CIDR or if the IP itself is ::1
	if cidr.IP.Equal(net.ParseIP("::1")) {
		return true
	}
	if cidr.IP.IsLoopback() {
		return true
	}
	return false
}

// isLinkLocal checks if a CIDR is a link-local range
func isLinkLocal(cidr *net.IPNet) bool {
	if cidr.IP.To4() != nil {
		// IPv4 link-local: 169.254.0.0/16
		return cidr.IP.Equal(net.ParseIP("169.254.0.0")) && cidr.Mask.String() == "ffff0000"
	}
	// IPv6 link-local: fe80::/10
	ones, _ := cidr.Mask.Size()
	return strings.HasPrefix(cidr.IP.String(), "fe") && ones == 10
}

// isPrivateNetwork checks if a CIDR is a private network range
func isPrivateNetwork(cidr *net.IPNet) bool {
	if cidr.IP.To4() != nil {
		// IPv4 private: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
		if cidr.IP.Equal(net.ParseIP("10.0.0.0")) && cidr.Mask.String() == "ff000000" {
			return true
		}
		if cidr.IP.Equal(net.ParseIP("172.16.0.0")) && cidr.Mask.String() == "fff00000" {
			return true
		}
		if cidr.IP.Equal(net.ParseIP("192.168.0.0")) && cidr.Mask.String() == "ffff0000" {
			return true
		}
	}
	// IPv6 unique local: fc00::/7
	ones, _ := cidr.Mask.Size()
	return (strings.HasPrefix(cidr.IP.String(), "fc") || strings.HasPrefix(cidr.IP.String(), "fd")) && ones == 7
}

// parseCIDR is a helper that safely parses CIDR notation
func parseCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	return network
}

// SSRFValidationError is returned when a target fails SSRF validation
type SSRFValidationError struct {
	Target string
	Reason string
}

func (e *SSRFValidationError) Error() string {
	return "SSRF validation failed for target '" + e.Target + "': " + e.Reason
}

// ValidateTargetHostQuick performs a quick validation without DNS resolution
// This is faster but may miss some SSRF vectors via DNS rebinding
func ValidateTargetHostQuick(target string, cfg SSRFValidatorConfig) error {
	if target == "" {
		return &SSRFValidationError{Target: target, Reason: "target cannot be empty"}
	}

	target = strings.TrimSpace(target)

	// SECURITY: Check allowlist FIRST - allowlisted hosts bypass all other checks
	// This is important for environments where internal access is legitimate
	if len(cfg.AllowedHosts) > 0 && cfg.AllowedHosts[target] {
		return nil
	}

	// Check against explicitly forbidden hosts (unless in allowlist)
	// Note: ForbiddenHosts includes localhost, but we handle ::1 separately below
	if ForbiddenHosts[target] || cfg.BlockedHosts[target] {
		return &SSRFValidationError{Target: target, Reason: "host is explicitly forbidden"}
	}

	// Check against forbidden suffixes (unless in allowlist)
	for _, suffix := range ForbiddenSuffixes {
		if strings.HasSuffix(target, suffix) {
			return &SSRFValidationError{Target: target, Reason: "host has forbidden suffix: " + suffix}
		}
	}

	// Parse as IP address first
	if ip := net.ParseIP(target); ip != nil {
		return validateIP(ip, cfg)
	}

	// If it contains a colon, it might be host:port format (but not IPv6 address)
	// IPv6 addresses are already handled by net.ParseIP above
	if strings.Contains(target, ":") && !strings.Contains(target, "]") {
		host, _, err := net.SplitHostPort(target)
		if err == nil {
			// Check if host portion is allowlisted
			if len(cfg.AllowedHosts) > 0 && cfg.AllowedHosts[host] {
				return nil
			}
			// Quick check just the host portion
			if ForbiddenHosts[host] || cfg.BlockedHosts[host] {
				return &SSRFValidationError{Target: target, Reason: "host is explicitly forbidden"}
			}
		}
	}

	// Check against allowed hosts whitelist (if configured)
	// If allowlist is configured, target must be in it
	if len(cfg.AllowedHosts) > 0 && !cfg.AllowedHosts[target] {
		return &SSRFValidationError{Target: target, Reason: "host not in allowed list"}
	}

	return nil
}
