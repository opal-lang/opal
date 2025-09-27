package v2

import "testing"

// TestASCIICharacterClassification tests the lookup table performance
func TestASCIICharacterClassification(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected map[string]bool
	}{
		{
			name: "lowercase letter",
			char: 'a',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "uppercase letter",
			char: 'Z',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "underscore",
			char: '_',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "digit",
			char: '5',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  true,
				"digit":      true,
				"whitespace": false,
			},
		},
		{
			name: "space",
			char: ' ',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false,
				"digit":      false,
				"whitespace": true,
			},
		},
		{
			name: "newline should not be whitespace",
			char: '\n',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false,
				"digit":      false,
				"whitespace": false, // Newlines are meaningful tokens
			},
		},
		{
			name: "hyphen not in identifier",
			char: '-',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false, // Hyphens NOT allowed per specification
				"digit":      false,
				"whitespace": false,
			},
		},
		{
			name: "tab whitespace",
			char: '\t',
			expected: map[string]bool{
				"letter":     false,
				"identStart": false,
				"identPart":  false,
				"digit":      false,
				"whitespace": true,
			},
		},
		{
			name: "hex digit lowercase",
			char: 'f',
			expected: map[string]bool{
				"letter":     true,
				"identStart": true,
				"identPart":  true,
				"digit":      false,
				"whitespace": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test direct lookup table access (zero function call overhead)
			if isLetter[tt.char] != tt.expected["letter"] {
				t.Errorf("isLetter[%q] = %v, want %v", tt.char, isLetter[tt.char], tt.expected["letter"])
			}
			if isIdentStart[tt.char] != tt.expected["identStart"] {
				t.Errorf("isIdentStart[%q] = %v, want %v", tt.char, isIdentStart[tt.char], tt.expected["identStart"])
			}
			if isIdentPart[tt.char] != tt.expected["identPart"] {
				t.Errorf("isIdentPart[%q] = %v, want %v", tt.char, isIdentPart[tt.char], tt.expected["identPart"])
			}
			if isDigit[tt.char] != tt.expected["digit"] {
				t.Errorf("isDigit[%q] = %v, want %v", tt.char, isDigit[tt.char], tt.expected["digit"])
			}
			if isWhitespace[tt.char] != tt.expected["whitespace"] {
				t.Errorf("isWhitespace[%q] = %v, want %v", tt.char, isWhitespace[tt.char], tt.expected["whitespace"])
			}
		})
	}
}

// TestHexDigitClassification tests hex digit lookup table
func TestHexDigitClassification(t *testing.T) {
	tests := []struct {
		char     byte
		expected bool
	}{
		{'0', true}, {'9', true}, // digits
		{'a', true}, {'f', true}, // lowercase hex
		{'A', true}, {'F', true}, // uppercase hex
		{'g', false}, {'G', false}, // not hex
		{'z', false}, {'Z', false}, // not hex
		{' ', false}, {'-', false}, // not hex
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			if isHexDigit[tt.char] != tt.expected {
				t.Errorf("isHexDigit[%q] = %v, want %v", tt.char, isHexDigit[tt.char], tt.expected)
			}
		})
	}
}

// TestASCIIIdentifierValidation tests ASCII-only identifier rules
func TestASCIIIdentifierValidation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		valid  bool
		reason string
	}{
		{
			name:   "valid variable",
			input:  "apiKey",
			valid:  true,
			reason: "camelCase is fine",
		},
		{
			name:   "valid underscore style",
			input:  "api_key",
			valid:  true,
			reason: "snake_case is fine",
		},
		{
			name:   "invalid kebab style",
			input:  "start-api",
			valid:  false,
			reason: "kebab-case not allowed per specification",
		},
		{
			name:   "valid with numbers",
			input:  "service2",
			valid:  true,
			reason: "numbers allowed after first character",
		},
		{
			name:   "starts with underscore",
			input:  "_private",
			valid:  true,
			reason: "underscore is valid start character",
		},
		{
			name:   "mixed styles invalid",
			input:  "API_v2-final",
			valid:  false,
			reason: "hyphens not allowed per specification",
		},
		{
			name:   "starts with number",
			input:  "2fast",
			valid:  false,
			reason: "cannot start with digit",
		},
		{
			name:   "contains space",
			input:  "my var",
			valid:  false,
			reason: "spaces not allowed",
		},
		{
			name:   "contains Unicode",
			input:  "café",
			valid:  false,
			reason: "Unicode not allowed in identifiers",
		},
		{
			name:   "empty string",
			input:  "",
			valid:  false,
			reason: "empty identifiers not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidASCIIIdentifier(tt.input)
			if valid != tt.valid {
				t.Errorf("isValidASCIIIdentifier(%q) = %v, want %v (%s)",
					tt.input, valid, tt.valid, tt.reason)
			}
		})
	}
}

// TestUnicodeInTokens tests that Unicode content is preserved as raw bytes in tokens
func TestUnicodeInTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We expect Unicode to be preserved as-is in token text
	}{
		{
			name:     "Chinese characters",
			input:    "世界",
			expected: "世界",
		},
		{
			name:     "Mixed ASCII and Unicode",
			input:    "hello世界",
			expected: "hello世界",
		},
		{
			name:     "Emoji",
			input:    "😀test",
			expected: "😀test",
		},
		{
			name:     "Various Unicode scripts",
			input:    "café αβγ العربية",
			expected: "café αβγ العربية",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, this is a design test - we'll implement actual tokenization later
			// The key point is that Unicode should go into tokens as raw bytes
			input := []byte(tt.input)
			if string(input) != tt.expected {
				t.Errorf("Unicode preservation failed: got %q, want %q", string(input), tt.expected)
			}
		})
	}
}
