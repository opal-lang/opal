package types

// ValidationConfig controls validation behavior and security
type ValidationConfig struct {
	// Security: Schema size/depth limits
	MaxSchemaSize  int // Max schema size in bytes (default: 1MB)
	MaxSchemaDepth int // Max schema nesting depth (default: 10)

	// Security: $ref resolution
	AllowRemoteRef bool     // Allow remote $ref (default: false)
	AllowedSchemes []string // Allowed URL schemes (default: ["file"])

	// Performance: Caching
	EnableCache  bool // Enable validator caching (default: true)
	MaxCacheSize int  // Max cached validators (default: 1000)

	// Validation behavior
	AssertFormat  bool // Enable format assertions (default: true)
	AssertContent bool // Enable content assertions (default: false)
}

// DefaultValidationConfig returns secure defaults
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxSchemaSize:  1024 * 1024, // 1MB
		MaxSchemaDepth: 10,          // Reasonable nesting limit (industry standard)
		AllowRemoteRef: false,
		AllowedSchemes: []string{"file"},
		EnableCache:    true,
		MaxCacheSize:   1000,
		AssertFormat:   true,
		AssertContent:  false,
	}
}
