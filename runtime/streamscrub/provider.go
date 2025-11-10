package streamscrub

import (
	"bytes"
	"sort"
)

// SecretProvider processes chunks to handle secrets.
//
// Implementations can use any strategy (Vault, config, regex, etc.) and
// control behavior: replace secrets, error if found, log, or custom actions.
//
// # Security Model
//
// The provider NEVER reveals secret patterns to the scrubber.
// Instead, the provider processes the chunk internally and returns the result.
// The scrubber never sees what the secrets are - only the sanitized output.
//
// This ensures:
//   - Maximum security: Scrubber never sees secret patterns
//   - Provider controls behavior: replace, error, log, etc.
//   - Minimal exposure (defense in depth)
//
// # Behavior Flexibility
//
// Providers can implement different behaviors:
//
//   - Replace mode: Replace secrets with placeholders (default)
//   - Fail-fast mode: Return error if secrets detected
//   - Audit mode: Log secrets then replace
//   - Custom mode: Any combination of above
//
// # Performance Considerations
//
// For optimal performance with Aho-Corasick or similar algorithms:
//   - Provider can build automaton from all known secrets
//   - HandleChunk() runs automaton on chunk (O(n) scan)
//   - Handles longest-match for overlapping secrets internally
//
// Current implementations use simple linear search, which is sufficient
// for typical use cases (10-100 secrets). Future optimization can use
// Aho-Corasick without changing this interface.
//
// # Example Implementations
//
// Replace mode (default):
//
//	type VaultProvider struct {
//	    mu          sync.Mutex
//	    expressions map[string]*Expression
//	}
//
//	func (v *VaultProvider) HandleChunk(chunk []byte) ([]byte, error) {
//	    v.mu.Lock()
//	    defer v.mu.Unlock()
//
//	    result := chunk
//
//	    // Sort secrets by length (longest first)
//	    secrets := v.sortedSecrets()
//
//	    // Replace all secrets
//	    for _, secret := range secrets {
//	        result = bytes.ReplaceAll(result, secret.value, secret.placeholder)
//	    }
//
//	    return result, nil
//	}
//
// Fail-fast mode:
//
//	type StrictProvider struct {
//	    inner SecretProvider
//	}
//
//	func (s *StrictProvider) HandleChunk(chunk []byte) ([]byte, error) {
//	    if s.hasSecret(chunk) {
//	        return nil, errors.New("secret detected in output")
//	    }
//	    return chunk, nil
//	}
type SecretProvider interface {
	// HandleChunk processes chunk and returns modified version.
	//
	// Provider controls behavior:
	//   - Replace secrets with placeholders (most common)
	//   - Return error if secrets found (fail-fast mode)
	//   - Log secrets then replace (audit mode)
	//   - Custom behavior
	//
	// Returns:
	//   - processed: Modified chunk (may be same as input if no secrets)
	//   - err: Error if provider rejects chunk (e.g., secret found in strict mode)
	//
	// # Implementation Requirements
	//
	// 1. Longest-match (greedy): When multiple secrets overlap, replace longest.
	//    This prevents partial leakage.
	//    Example: If chunk contains "SECRET_EXTENDED" and you know both
	//    "SECRET" and "SECRET_EXTENDED", replace "SECRET_EXTENDED".
	//
	// 2. Handle all secrets: Process entire chunk in one call.
	//    Replace all secrets, not just the first one.
	//
	// 3. Thread-safety: Must be safe for concurrent calls.
	//
	// 4. Idempotent: Calling multiple times on same chunk should be safe.
	HandleChunk(chunk []byte) (processed []byte, err error)

	// MaxSecretLength returns the length of the longest secret in bytes.
	//
	// The scrubber uses this to maintain a carry buffer for chunk-boundary safety.
	// If a secret could span chunk boundaries, the scrubber holds back
	// (maxLen - 1) bytes from each chunk to ensure complete secret detection.
	//
	// Example: If longest secret is "SECRET_TOKEN" (12 bytes), scrubber holds
	// back 11 bytes between chunks to catch secrets split across boundaries.
	//
	// Return 0 if no secrets are registered or if all secrets are guaranteed
	// to be within single chunks (not recommended - unsafe for streaming).
	//
	// Thread-safety: Must be safe for concurrent calls.
	MaxSecretLength() int
}

// Pattern represents a secret to find and replace.
type Pattern struct {
	Value       []byte // Secret bytes to find
	Placeholder []byte // Replacement bytes
}

// PatternSource provides patterns dynamically.
// This function is called each time HandleChunk is invoked,
// allowing the pattern list to change over time.
type PatternSource func() []Pattern

// NewPatternProvider creates a SecretProvider from a pattern source.
//
// This is a helper for the common case where you have a list of
// patterns (secrets) to find and replace. The provider handles:
//   - Longest-first matching (prevents partial leakage)
//   - Efficient replacement (optimized internally)
//   - Thread-safety (if your source function is thread-safe)
//
// The source function is called on each HandleChunk invocation,
// so patterns can change dynamically.
//
// Example:
//
//	// Define a function that returns current secrets
//	getSecrets := func() []streamscrub.Pattern {
//	    return []streamscrub.Pattern{
//	        {Value: []byte("secret1"), Placeholder: []byte("REDACTED-1")},
//	        {Value: []byte("secret2"), Placeholder: []byte("REDACTED-2")},
//	    }
//	}
//
//	// Create provider using helper
//	provider := streamscrub.NewPatternProvider(getSecrets)
//	scrubber := streamscrub.New(output, streamscrub.WithSecretProvider(provider))
//
// Performance: Currently uses simple linear replacement. Future versions
// may use Aho-Corasick algorithm for better performance with many patterns.
func NewPatternProvider(source PatternSource) SecretProvider {
	return &patternProvider{
		getPatterns: source,
	}
}

// patternProvider implements SecretProvider using a pattern source.
type patternProvider struct {
	getPatterns PatternSource
}

// HandleChunk implements SecretProvider interface.
func (p *patternProvider) HandleChunk(chunk []byte) ([]byte, error) {
	// Get current patterns from source
	patterns := p.getPatterns()

	if len(patterns) == 0 {
		return chunk, nil
	}

	// Sort by descending length (longest first)
	// This ensures overlapping secrets use longest match
	sort.Slice(patterns, func(i, j int) bool {
		return len(patterns[i].Value) > len(patterns[j].Value)
	})

	// Replace all patterns (longest first)
	result := chunk
	for _, pattern := range patterns {
		if len(pattern.Value) > 0 {
			result = bytes.ReplaceAll(result, pattern.Value, pattern.Placeholder)
		}
	}

	return result, nil
}

// MaxSecretLength implements SecretProvider interface.
func (p *patternProvider) MaxSecretLength() int {
	patterns := p.getPatterns()

	maxLen := 0
	for _, pattern := range patterns {
		if len(pattern.Value) > maxLen {
			maxLen = len(pattern.Value)
		}
	}

	return maxLen
}

// NewPatternProviderWithVariants creates a SecretProvider that automatically
// generates encoding variants for defense-in-depth.
//
// For each secret, it generates variants in common encodings:
//   - Hex (lowercase and uppercase)
//   - Base64 (standard, raw, URL)
//   - Percent encoding (lowercase and uppercase)
//   - Separator-inserted variants (-, _, :, ., space)
//
// This provides additional security if secrets are accidentally encoded
// somewhere in the pipeline. The tradeoff is more patterns to match.
//
// Example:
//
//	getSecrets := func() []streamscrub.Pattern {
//	    return []streamscrub.Pattern{
//	        {Value: []byte("secret"), Placeholder: []byte("REDACTED")},
//	    }
//	}
//	provider := streamscrub.NewPatternProviderWithVariants(getSecrets)
//	// Will also match: "736563726574" (hex), "c2VjcmV0" (base64), etc.
func NewPatternProviderWithVariants(source PatternSource) SecretProvider {
	expandedSource := func() []Pattern {
		base := source()
		var expanded []Pattern

		for _, pattern := range base {
			// Add original pattern
			expanded = append(expanded, pattern)

			// Add encoding variants
			expanded = append(expanded, generateVariants(pattern)...)
		}

		return expanded
	}

	return &patternProvider{
		getPatterns: expandedSource,
	}
}

// generateVariants creates encoding variants of a pattern for defense-in-depth.
func generateVariants(pattern Pattern) []Pattern {
	var variants []Pattern
	secret := pattern.Value
	placeholder := pattern.Placeholder

	// Hex: lowercase and uppercase
	hexLower := toHex(secret)
	hexUpper := toUpperHex(hexLower)
	variants = append(variants, Pattern{Value: []byte(hexLower), Placeholder: placeholder})
	variants = append(variants, Pattern{Value: []byte(hexUpper), Placeholder: placeholder})

	// Base64: standard, raw, and URL encodings
	b64Std := toBase64(secret)
	b64Raw := toBase64Raw(secret)
	b64URL := toBase64URL(secret)
	variants = append(variants, Pattern{Value: []byte(b64Std), Placeholder: placeholder})
	variants = append(variants, Pattern{Value: []byte(b64Raw), Placeholder: placeholder})
	variants = append(variants, Pattern{Value: []byte(b64URL), Placeholder: placeholder})

	// Percent encoding: lowercase and uppercase
	percentLower := toPercentEncoding(secret, false)
	percentUpper := toPercentEncoding(secret, true)
	variants = append(variants, Pattern{Value: []byte(percentLower), Placeholder: placeholder})
	variants = append(variants, Pattern{Value: []byte(percentUpper), Placeholder: placeholder})

	// Separator-inserted variants (common in formatted output)
	separators := []string{"-", "_", ":", ".", " "}
	for _, sep := range separators {
		variant := insertSeparators(secret, sep)
		variants = append(variants, Pattern{Value: []byte(variant), Placeholder: placeholder})
	}

	return variants
}
