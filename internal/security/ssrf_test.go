package security

import (
	"testing"
)

func TestValidateTargetHost_ForbiddenHosts(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()

	forbiddenHosts := []string{
		"169.254.169.254",
		"metadata.google.internal",
		"localhost",
		"::1",
	}

	for _, host := range forbiddenHosts {
		t.Run(host, func(t *testing.T) {
			err := ValidateTargetHostQuick(host, cfg)
			if err == nil {
				t.Errorf("Expected error for forbidden host %s", host)
			}
		})
	}
}

func TestValidateTargetHost_ForbiddenSuffixes(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()

	forbiddenSuffixes := []string{
		"server.local",
		"app.internal",
		"db.corp",
		"api.private",
	}

	for _, host := range forbiddenSuffixes {
		t.Run(host, func(t *testing.T) {
			err := ValidateTargetHostQuick(host, cfg)
			if err == nil {
				t.Errorf("Expected error for host with forbidden suffix %s", host)
			}
		})
	}
}

func TestValidateTargetHost_PrivateIPs(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()

	privateIPs := []string{
		"10.0.0.1",
		"10.255.255.255",
		"172.16.0.1",
		"172.31.255.255",
		"192.168.0.1",
		"192.168.255.255",
		"127.0.0.1",
		"127.0.0.2",
		"fc00::1",
		"fe80::1",
	}

	for _, ip := range privateIPs {
		t.Run(ip, func(t *testing.T) {
			err := ValidateTargetHostQuick(ip, cfg)
			if err == nil {
				t.Errorf("Expected error for private IP %s", ip)
			}
		})
	}
}

func TestValidateTargetHost_PublicIPs(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()

	// These should pass by default
	publicIPs := []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34", // example.com
		"2001:4860:4860::8888", // Google DNS IPv6
	}

	for _, ip := range publicIPs {
		t.Run(ip, func(t *testing.T) {
			err := ValidateTargetHostQuick(ip, cfg)
			if err != nil {
				t.Errorf("Expected no error for public IP %s, got: %v", ip, err)
			}
		})
	}
}

func TestValidateTargetHost_AllowPrivateNetworks(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()
	cfg.AllowPrivateNetworks = true

	privateIPs := []string{
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
	}

	for _, ip := range privateIPs {
		t.Run(ip, func(t *testing.T) {
			err := ValidateTargetHostQuick(ip, cfg)
			if err != nil {
				t.Errorf("Expected no error when AllowPrivateNetworks is true for %s", ip)
			}
		})
	}
}

func TestValidateTargetHost_AllowLoopback(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()
	cfg.AllowLoopback = true

	loopbackIPs := []string{
		"127.0.0.1",
		"127.0.0.2",
		"::1",
	}

	for _, ip := range loopbackIPs {
		t.Run(ip, func(t *testing.T) {
			err := ValidateTargetHostQuick(ip, cfg)
			if err != nil {
				t.Errorf("Expected no error when AllowLoopback is true for %s", ip)
			}
		})
	}
}

func TestValidateTargetHost_Empty(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()

	err := ValidateTargetHostQuick("", cfg)
	if err == nil {
		t.Error("Expected error for empty target")
	}
}

func TestValidateTargetHost_AllowedList(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()
	cfg.AllowedHosts = map[string]bool{
		"trusted.internal": true,
		"10.0.0.5":          true,
	}

	tests := []struct {
		name      string
		target    string
		wantError bool
	}{
		{"allowed host", "trusted.internal", false},
		{"allowed IP", "10.0.0.5", false},
		{"not allowed host", "other.internal", true},
		{"not allowed IP", "10.0.0.6", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetHostQuick(tt.target, cfg)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTargetHostQuick() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateTargetHost_BlockedList(t *testing.T) {
	cfg := DefaultSSRFValidatorConfig()
	cfg.BlockedHosts = map[string]bool{
		"malicious.example.com": true,
		"8.8.8.8":               true, // Block Google DNS as example
	}

	tests := []struct {
		name      string
		target    string
		wantError bool
	}{
		{"blocked host", "malicious.example.com", true},
		{"blocked IP", "8.8.8.8", true},
		{"not blocked host", "good.example.com", false},
		{"not blocked IP", "1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetHostQuick(tt.target, cfg)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTargetHostQuick() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSSRFValidationError(t *testing.T) {
	err := &SSRFValidationError{
		Target: "localhost",
		Reason: "host is explicitly forbidden",
	}

	expected := "SSRF validation failed for target 'localhost': host is explicitly forbidden"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}
