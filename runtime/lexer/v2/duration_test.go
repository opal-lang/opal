package v2

import (
	"testing"
)

// TestSimpleDurations tests basic single-unit duration patterns
func TestSimpleDurations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		// Basic time units
		{
			name:  "seconds",
			input: "30s",
			expected: []tokenExpectation{
				{DURATION, "30s", 1, 1},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "minutes",
			input: "5m",
			expected: []tokenExpectation{
				{DURATION, "5m", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "hours",
			input: "2h",
			expected: []tokenExpectation{
				{DURATION, "2h", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "days",
			input: "7d",
			expected: []tokenExpectation{
				{DURATION, "7d", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "weeks",
			input: "1w",
			expected: []tokenExpectation{
				{DURATION, "1w", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "years",
			input: "2y",
			expected: []tokenExpectation{
				{DURATION, "2y", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		// Sub-second units
		{
			name:  "milliseconds",
			input: "500ms",
			expected: []tokenExpectation{
				{DURATION, "500ms", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "microseconds",
			input: "100us",
			expected: []tokenExpectation{
				{DURATION, "100us", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "nanoseconds",
			input: "50ns",
			expected: []tokenExpectation{
				{DURATION, "50ns", 1, 1},
				{EOF, "", 1, 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestCompoundDurations tests multi-unit duration patterns
func TestCompoundDurations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "hours_and_minutes",
			input: "1h30m",
			expected: []tokenExpectation{
				{DURATION, "1h30m", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "days_and_hours",
			input: "2d12h",
			expected: []tokenExpectation{
				{DURATION, "2d12h", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "minutes_and_seconds",
			input: "5m30s",
			expected: []tokenExpectation{
				{DURATION, "5m30s", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "seconds_and_milliseconds",
			input: "1s500ms",
			expected: []tokenExpectation{
				{DURATION, "1s500ms", 1, 1},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "complex_duration",
			input: "1w2d3h4m5s",
			expected: []tokenExpectation{
				{DURATION, "1w2d3h4m5s", 1, 1},
				{EOF, "", 1, 11},
			},
		},
		{
			name:  "skip_units",
			input: "1d30m",
			expected: []tokenExpectation{
				{DURATION, "1d30m", 1, 1},
				{EOF, "", 1, 6},
			},
		},
		{
			name:  "years_to_seconds",
			input: "1y365d24h60m60s",
			expected: []tokenExpectation{
				{DURATION, "1y365d24h60m60s", 1, 1},
				{EOF, "", 1, 16},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestDurationEdgeCases tests boundary conditions and error cases
func TestDurationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "zero_duration",
			input: "0s",
			expected: []tokenExpectation{
				{DURATION, "0s", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "large_number",
			input: "999999h",
			expected: []tokenExpectation{
				{DURATION, "999999h", 1, 1},
				{EOF, "", 1, 8},
			},
		},
		{
			name:  "single_digit",
			input: "1s",
			expected: []tokenExpectation{
				{DURATION, "1s", 1, 1},
				{EOF, "", 1, 3},
			},
		},
		{
			name:  "invalid_unit_stops",
			input: "30x",
			expected: []tokenExpectation{
				{INTEGER, "30", 1, 1},
				{IDENTIFIER, "x", 1, 3},
				{EOF, "", 1, 4},
			},
		},
		{
			name:  "float_not_allowed",
			input: "1.5h",
			expected: []tokenExpectation{
				{FLOAT, "1.5", 1, 1},
				{IDENTIFIER, "h", 1, 4},
				{EOF, "", 1, 5},
			},
		},
		{
			name:  "scientific_not_allowed",
			input: "1e3s",
			expected: []tokenExpectation{
				{SCIENTIFIC, "1e3", 1, 1},
				{IDENTIFIER, "s", 1, 4},
				{EOF, "", 1, 5},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestDurationInContext tests durations in realistic opal usage
func TestDurationInContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "variable_assignment",
			input: "var timeout = 30s",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "timeout", 1, 5},
				{EQUALS, "", 1, 13},
				{DURATION, "30s", 1, 15},
				{EOF, "", 1, 18},
			},
		},
		{
			name:  "duration_arithmetic",
			input: "total = base + 5m30s",
			expected: []tokenExpectation{
				{IDENTIFIER, "total", 1, 1},
				{EQUALS, "", 1, 7},
				{IDENTIFIER, "base", 1, 9},
				{PLUS, "", 1, 14},
				{DURATION, "5m30s", 1, 16},
				{EOF, "", 1, 21},
			},
		},
		{
			name:  "negative_duration",
			input: "var grace = -30s",
			expected: []tokenExpectation{
				{VAR, "var", 1, 1},
				{IDENTIFIER, "grace", 1, 5},
				{EQUALS, "", 1, 11},
				{MINUS, "", 1, 13},
				{DURATION, "30s", 1, 14},
				{EOF, "", 1, 17},
			},
		},
		{
			name:  "multiple_durations",
			input: "1h + 30m - 15s",
			expected: []tokenExpectation{
				{DURATION, "1h", 1, 1},
				{PLUS, "", 1, 4},
				{DURATION, "30m", 1, 6},
				{MINUS, "", 1, 10},
				{DURATION, "15s", 1, 12},
				{EOF, "", 1, 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}

// TestDurationBoundaries tests that durations stop at correct boundaries
func TestDurationBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []tokenExpectation
	}{
		{
			name:  "followed_by_identifier",
			input: "30stimeout",
			expected: []tokenExpectation{
				{DURATION, "30s", 1, 1},
				{IDENTIFIER, "timeout", 1, 4},
				{EOF, "", 1, 11},
			},
		},
		{
			name:  "followed_by_punctuation",
			input: "1h30m,",
			expected: []tokenExpectation{
				{DURATION, "1h30m", 1, 1},
				{COMMA, "", 1, 6},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "followed_by_operator",
			input: "5m+10s",
			expected: []tokenExpectation{
				{DURATION, "5m", 1, 1},
				{PLUS, "", 1, 3},
				{DURATION, "10s", 1, 4},
				{EOF, "", 1, 7},
			},
		},
		{
			name:  "followed_by_space",
			input: "2h 30m",
			expected: []tokenExpectation{
				{DURATION, "2h", 1, 1},
				{DURATION, "30m", 1, 4},
				{EOF, "", 1, 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.name, tt.input, tt.expected)
		})
	}
}
