package types

// Format represents a typed string format for validation
type Format string

const (
	// Standard JSON Schema formats
	FormatURI      Format = "uri"
	FormatHostname Format = "hostname"
	FormatIPv4     Format = "ipv4"
	FormatIPv6     Format = "ipv6"
	FormatEmail    Format = "email"

	// Opal-specific formats (x-opal-format)
	FormatCIDR     Format = "cidr"     // IP CIDR notation (e.g., "10.0.0.0/8")
	FormatSemver   Format = "semver"   // Semantic version (e.g., "1.2.3")
	FormatDuration Format = "duration" // Opal duration (e.g., "1h30m")
)

// IsValidFormat checks if a format is recognized
func IsValidFormat(f Format) bool {
	switch f {
	case FormatURI, FormatHostname, FormatIPv4, FormatIPv6, FormatEmail,
		FormatCIDR, FormatSemver, FormatDuration:
		return true
	default:
		return false
	}
}

// IsOpalFormat returns true if this is an Opal-specific format (not standard JSON Schema)
func IsOpalFormat(f Format) bool {
	switch f {
	case FormatCIDR, FormatSemver, FormatDuration:
		return true
	default:
		return false
	}
}
