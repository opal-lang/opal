package cli

// ================================================================================================
// MINIMAL CLI TYPES - Just what's needed for argument parsing
// ================================================================================================

// Args represents decorator arguments (for potential future use)
type Args map[string]interface{}

// GetString returns a string argument with fallback
func (a Args) GetString(key, fallback string) string {
	if val, exists := a[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return fallback
}

// GetBool returns a bool argument with fallback
func (a Args) GetBool(key string, fallback bool) bool {
	if val, exists := a[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return fallback
}

// Note: All execution logic removed - CLI now delegates to runtime
