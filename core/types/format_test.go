package types

import "testing"

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		format Format
		valid  bool
	}{
		{FormatURI, true},
		{FormatHostname, true},
		{FormatIPv4, true},
		{FormatIPv6, true},
		{FormatEmail, true},
		{FormatCIDR, true},
		{FormatSemver, true},
		{FormatDuration, true},
		{Format("invalid"), false},
		{Format(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got := IsValidFormat(tt.format)
			if got != tt.valid {
				t.Errorf("IsValidFormat(%q) = %v, want %v", tt.format, got, tt.valid)
			}
		})
	}
}

func TestIsOpalFormat(t *testing.T) {
	tests := []struct {
		format   Format
		opalOnly bool
	}{
		// Standard JSON Schema formats
		{FormatURI, false},
		{FormatHostname, false},
		{FormatIPv4, false},
		{FormatIPv6, false},
		{FormatEmail, false},

		// Opal-specific formats
		{FormatCIDR, true},
		{FormatSemver, true},
		{FormatDuration, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got := IsOpalFormat(tt.format)
			if got != tt.opalOnly {
				t.Errorf("IsOpalFormat(%q) = %v, want %v", tt.format, got, tt.opalOnly)
			}
		})
	}
}

func TestFormatConstants(t *testing.T) {
	// Verify format constants have expected string values
	tests := []struct {
		format Format
		want   string
	}{
		{FormatURI, "uri"},
		{FormatHostname, "hostname"},
		{FormatIPv4, "ipv4"},
		{FormatIPv6, "ipv6"},
		{FormatEmail, "email"},
		{FormatCIDR, "cidr"},
		{FormatSemver, "semver"},
		{FormatDuration, "duration"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.format) != tt.want {
				t.Errorf("Format constant = %q, want %q", tt.format, tt.want)
			}
		})
	}
}
