package types

import "testing"

// TestNormalizeDuration tests duration normalization to canonical form
func TestNormalizeDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic units
		{"1s", "1s"},
		{"1m", "1m"},
		{"1h", "1h"},
		{"1d", "1d"},
		{"1w", "1w"},
		{"1y", "1y"},
		{"1ms", "1ms"},
		{"1us", "1us"},
		{"1ns", "1ns"},

		// Overflow normalization
		{"60s", "1m"},
		{"90s", "1m30s"},
		{"60m", "1h"},
		{"90m", "1h30m"},
		{"24h", "1d"},
		{"25h", "1d1h"},
		{"7d", "1w"},
		{"8d", "1w1d"},
		{"365d", "1y"},
		{"1000ms", "1s"},
		{"1000us", "1ms"},
		{"1000ns", "1us"},

		// Complex durations
		{"1h30m", "1h30m"},
		{"1h30m45s", "1h30m45s"},
		{"1d2h3m4s", "1d2h3m4s"},
		{"1w2d3h4m5s", "1w2d3h4m5s"},
		{"1y1w1d1h1m1s", "1y1w1d1h1m1s"},

		// Non-canonical input (needs normalization)
		{"3600s", "1h"},
		{"3661s", "1h1m1s"},
		{"90m30s", "1h30m30s"},
		{"30m90s", "31m30s"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeDuration(tt.input)
			if err != nil {
				t.Fatalf("NormalizeDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("NormalizeDuration(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestNormalizeDuration_Invalid tests invalid duration inputs
func TestNormalizeDuration_Invalid(t *testing.T) {
	tests := []struct {
		input       string
		description string
	}{
		{"", "empty string"},
		{"invalid", "no unit"},
		{"1x", "unknown unit"},
		{"-1s", "negative (starts with -)"},
		{"1.5s", "decimal"},
		{"1s1h", "wrong order (s before h)"},
		{"1h1h", "duplicate unit"},
		{"1m1s1h", "wrong order (h after m and s)"},
		{"s", "missing number"},
		{"1", "missing unit"},
		{"1h2", "missing unit after second number"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			_, err := NormalizeDuration(tt.input)
			if err == nil {
				t.Errorf("NormalizeDuration(%q) expected error for %s, got nil", tt.input, tt.description)
			}
		})
	}
}

// TestParseDuration tests parsing durations
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input         string
		expectedNanos int64
	}{
		{"1ns", 1},
		{"1us", 1000},
		{"1ms", 1000000},
		{"1s", 1000000000},
		{"1m", 60000000000},
		{"1h", 3600000000000},
		{"1d", 86400000000000},
		{"1w", 604800000000000},
		{"1y", 31536000000000000},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if d.Nanoseconds() != tt.expectedNanos {
				t.Errorf("ParseDuration(%q).Nanoseconds() = %d, want %d", tt.input, d.Nanoseconds(), tt.expectedNanos)
			}
		})
	}
}

// TestDuration_Add tests duration addition
func TestDuration_Add(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected string
	}{
		{"1s", "1s", "2s"},
		{"30s", "30s", "1m"},
		{"1h", "30m", "1h30m"},
		{"1d", "1d", "2d"},
	}

	for _, tt := range tests {
		t.Run(tt.a+"+"+tt.b, func(t *testing.T) {
			da, _ := ParseDuration(tt.a)
			db, _ := ParseDuration(tt.b)
			result := da.Add(db)
			if result.String() != tt.expected {
				t.Errorf("%s + %s = %s, want %s", tt.a, tt.b, result.String(), tt.expected)
			}
		})
	}
}

// TestDuration_Sub tests duration subtraction with clamping
func TestDuration_Sub(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected string
	}{
		{"2s", "1s", "1s"},
		{"1m", "30s", "30s"},
		{"1h30m", "30m", "1h"},
		{"1s", "2s", "0s"}, // Clamped to 0
		{"1s", "1s", "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.a+"-"+tt.b, func(t *testing.T) {
			da, _ := ParseDuration(tt.a)
			db, _ := ParseDuration(tt.b)
			result := da.Sub(db)
			if result.String() != tt.expected {
				t.Errorf("%s - %s = %s, want %s", tt.a, tt.b, result.String(), tt.expected)
			}
		})
	}
}

// TestDuration_Compare tests duration comparison
func TestDuration_Compare(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"1s", "1s", 0},
		{"1s", "2s", -1},
		{"2s", "1s", 1},
		{"1m", "60s", 0},
		{"1h", "59m", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			da, _ := ParseDuration(tt.a)
			db, _ := ParseDuration(tt.b)
			result := da.Compare(db)
			if result != tt.expected {
				t.Errorf("%s.Compare(%s) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestDuration_IsZero tests zero duration detection
func TestDuration_IsZero(t *testing.T) {
	zero, _ := ParseDuration("0s")
	if !zero.IsZero() {
		t.Error("0s should be zero")
	}

	nonZero, _ := ParseDuration("1s")
	if nonZero.IsZero() {
		t.Error("1s should not be zero")
	}

	// Subtraction clamped to zero
	d1, _ := ParseDuration("1s")
	d2, _ := ParseDuration("2s")
	clamped := d1.Sub(d2)
	if !clamped.IsZero() {
		t.Error("1s - 2s should be zero (clamped)")
	}
}

// TestDuration_String tests string formatting
func TestDuration_String(t *testing.T) {
	tests := []struct {
		nanos    int64
		expected string
	}{
		{0, "0s"},
		{1, "1ns"},
		{1000, "1us"},
		{1000000, "1ms"},
		{1000000000, "1s"},
		{60000000000, "1m"},
		{3600000000000, "1h"},
		{86400000000000, "1d"},
		{604800000000000, "1w"},
		{31536000000000000, "1y"},
		{3661000000000, "1h1m1s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			d := Duration{nanos: tt.nanos}
			if d.String() != tt.expected {
				t.Errorf("Duration{%d}.String() = %q, want %q", tt.nanos, d.String(), tt.expected)
			}
		})
	}
}

// TestDuration_DescendingOrder tests that units must be in descending order
func TestDuration_DescendingOrder(t *testing.T) {
	valid := []string{
		"1y1w1d1h1m1s1ms1us1ns",
		"1h30m",
		"1d12h",
	}

	for _, input := range valid {
		t.Run("valid:"+input, func(t *testing.T) {
			_, err := ParseDuration(input)
			if err != nil {
				t.Errorf("ParseDuration(%q) should be valid, got error: %v", input, err)
			}
		})
	}

	invalid := []string{
		"1s1m",   // s before m
		"1m1h",   // m before h
		"1h1d",   // h before d
		"1ns1ms", // ns before ms
	}

	for _, input := range invalid {
		t.Run("invalid:"+input, func(t *testing.T) {
			_, err := ParseDuration(input)
			if err == nil {
				t.Errorf("ParseDuration(%q) should be invalid (wrong order), got nil error", input)
			}
		})
	}
}

// TestDuration_NoDuplicates tests that each unit can appear at most once
func TestDuration_NoDuplicates(t *testing.T) {
	invalid := []string{
		"1h2h",
		"1m2m",
		"1s2s",
		"1d2d",
	}

	for _, input := range invalid {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDuration(input)
			if err == nil {
				t.Errorf("ParseDuration(%q) should be invalid (duplicate unit), got nil error", input)
			}
		})
	}
}

// TestDuration_EdgeCases tests edge cases and boundary conditions
func TestDuration_EdgeCases(t *testing.T) {
	t.Run("zero duration", func(t *testing.T) {
		d, err := ParseDuration("0s")
		if err != nil {
			t.Fatalf("ParseDuration(\"0s\") error: %v", err)
		}
		if !d.IsZero() {
			t.Error("0s should be zero")
		}
		if d.String() != "0s" {
			t.Errorf("0s.String() = %q, want \"0s\"", d.String())
		}
	})

	t.Run("single nanosecond", func(t *testing.T) {
		d, err := ParseDuration("1ns")
		if err != nil {
			t.Fatalf("ParseDuration(\"1ns\") error: %v", err)
		}
		if d.Nanoseconds() != 1 {
			t.Errorf("1ns.Nanoseconds() = %d, want 1", d.Nanoseconds())
		}
	})

	t.Run("large duration (years)", func(t *testing.T) {
		d, err := ParseDuration("100y")
		if err != nil {
			t.Fatalf("ParseDuration(\"100y\") error: %v", err)
		}
		expected := int64(100) * Year
		if d.Nanoseconds() != expected {
			t.Errorf("100y.Nanoseconds() = %d, want %d", d.Nanoseconds(), expected)
		}
	})

	t.Run("all units combined", func(t *testing.T) {
		input := "1y2w3d4h5m6s7ms8us9ns"
		d, err := ParseDuration(input)
		if err != nil {
			t.Fatalf("ParseDuration(%q) error: %v", input, err)
		}
		expected := Year + 2*Week + 3*Day + 4*Hour + 5*Minute + 6*Second + 7*Millisecond + 8*Microsecond + 9*Nanosecond
		if d.Nanoseconds() != expected {
			t.Errorf("%s.Nanoseconds() = %d, want %d", input, d.Nanoseconds(), expected)
		}
	})

	t.Run("maximum safe duration", func(t *testing.T) {
		// ~292 years is the max (2^63 - 1 nanoseconds)
		d, err := ParseDuration("290y")
		if err != nil {
			t.Fatalf("ParseDuration(\"290y\") error: %v", err)
		}
		if d.Nanoseconds() < 0 {
			t.Error("290y should not overflow to negative")
		}
	})

	t.Run("unit ambiguity - ms vs m", func(t *testing.T) {
		// "1ms" should parse as milliseconds, not "1m" + "s"
		d, err := ParseDuration("1ms")
		if err != nil {
			t.Fatalf("ParseDuration(\"1ms\") error: %v", err)
		}
		if d.Nanoseconds() != Millisecond {
			t.Errorf("1ms.Nanoseconds() = %d, want %d (millisecond)", d.Nanoseconds(), Millisecond)
		}
		if d.String() != "1ms" {
			t.Errorf("1ms.String() = %q, want \"1ms\"", d.String())
		}
	})

	t.Run("unit ambiguity - us vs u", func(t *testing.T) {
		// "1us" should parse as microseconds
		d, err := ParseDuration("1us")
		if err != nil {
			t.Fatalf("ParseDuration(\"1us\") error: %v", err)
		}
		if d.Nanoseconds() != Microsecond {
			t.Errorf("1us.Nanoseconds() = %d, want %d (microsecond)", d.Nanoseconds(), Microsecond)
		}
	})

	t.Run("leading zeros", func(t *testing.T) {
		d, err := ParseDuration("01s")
		if err != nil {
			t.Fatalf("ParseDuration(\"01s\") error: %v", err)
		}
		if d.Nanoseconds() != Second {
			t.Errorf("01s.Nanoseconds() = %d, want %d", d.Nanoseconds(), Second)
		}
	})

	t.Run("large numbers", func(t *testing.T) {
		d, err := ParseDuration("999999s")
		if err != nil {
			t.Fatalf("ParseDuration(\"999999s\") error: %v", err)
		}
		expected := int64(999999) * Second
		if d.Nanoseconds() != expected {
			t.Errorf("999999s.Nanoseconds() = %d, want %d", d.Nanoseconds(), expected)
		}
	})
}

// TestDuration_InvalidEdgeCases tests invalid edge cases
func TestDuration_InvalidEdgeCases(t *testing.T) {
	tests := []struct {
		input       string
		description string
	}{
		{"", "empty string"},
		{" ", "whitespace only"},
		{"1", "number without unit"},
		{"s", "unit without number"},
		{"1x", "invalid unit"},
		{"1sec", "invalid unit (sec instead of s)"},
		{"1second", "invalid unit (second instead of s)"},
		{"1min", "invalid unit (min instead of m)"},
		{"1hr", "invalid unit (hr instead of h)"},
		{"1.5s", "decimal number"},
		{"1,5s", "comma in number"},
		{"-1s", "negative number"},
		{"1s-1m", "negative in middle"},
		{"1s ", "trailing space"},
		{" 1s", "leading space"},
		{"1 s", "space between number and unit"},
		{"1s 1m", "space between components"},
		{"1s1", "number without unit at end"},
		{"1h1m1", "number without unit at end (complex)"},
		{"1ss", "double unit"},
		{"1sm", "wrong order (s before m)"},
		{"1ms1s", "wrong order (ms before s)"},
		{"1us1ms", "wrong order (us before ms)"},
		{"1ns1us", "wrong order (ns before us)"},
		{"1h1h", "duplicate unit"},
		{"1m1m1m", "triple duplicate"},
		{"0", "zero without unit"},
		{"1y1y", "duplicate year"},
		{"1w1w", "duplicate week"},
		{"1d1d", "duplicate day"},
		{"abc", "non-numeric"},
		{"1sabc", "garbage after valid duration"},
		{"1h30x", "invalid unit in middle"},
		{"1000000000000000000m", "overflow (huge minutes)"},
		{"1000000000000000000s", "overflow (huge seconds)"},
		{"999999999999999999y", "overflow (huge years)"},
		{"9223372036854775808ns", "overflow (number exceeds int64)"},
		{"99999999999999999999ns", "overflow (very large number)"},
		{"18446744073709551616ns", "overflow (exceeds uint64)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			_, err := ParseDuration(tt.input)
			if err == nil {
				t.Errorf("ParseDuration(%q) should be invalid (%s), got nil error", tt.input, tt.description)
			}
		})
	}
}

// TestDuration_Normalization_EdgeCases tests normalization edge cases
func TestDuration_Normalization_EdgeCases(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		description string
	}{
		// Exact conversions
		{"60s", "1m", "60 seconds = 1 minute"},
		{"3600s", "1h", "3600 seconds = 1 hour"},
		{"86400s", "1d", "86400 seconds = 1 day"},
		{"604800s", "1w", "604800 seconds = 1 week"},
		{"31536000s", "1y", "31536000 seconds = 1 year"},

		// Overflow with remainder
		{"61s", "1m1s", "61 seconds = 1 minute 1 second"},
		{"3661s", "1h1m1s", "3661 seconds = 1 hour 1 minute 1 second"},
		{"90061s", "1d1h1m1s", "complex overflow"},

		// Multiple overflows
		{"1000ms", "1s", "1000 milliseconds = 1 second"},
		{"1000000us", "1s", "1000000 microseconds = 1 second"},
		{"1000000000ns", "1s", "1000000000 nanoseconds = 1 second"},

		// Mixed units with overflow
		{"1m60s", "2m", "1 minute + 60 seconds = 2 minutes"},
		{"1h60m", "2h", "1 hour + 60 minutes = 2 hours"},
		{"1d24h", "2d", "1 day + 24 hours = 2 days"},
		{"1w7d", "2w", "1 week + 7 days = 2 weeks"},

		// Complex normalization
		{"1h90m", "2h30m", "1 hour + 90 minutes = 2 hours 30 minutes"},
		{"1m90s", "2m30s", "1 minute + 90 seconds = 2 minutes 30 seconds"},
		{"1s1000ms", "2s", "1 second + 1000 milliseconds = 2 seconds"},

		// Zero components omitted
		{"1h0m30s", "1h30s", "zero minutes omitted"},
		{"1d0h0m1s", "1d1s", "zero hours and minutes omitted"},

		// Large numbers
		{"10000s", "2h46m40s", "10000 seconds normalized"},
		{"100000s", "1d3h46m40s", "100000 seconds normalized"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := NormalizeDuration(tt.input)
			if err != nil {
				t.Fatalf("NormalizeDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("NormalizeDuration(%q) = %q, want %q (%s)", tt.input, got, tt.expected, tt.description)
			}
		})
	}
}

// TestDuration_Arithmetic_EdgeCases tests arithmetic edge cases
func TestDuration_Arithmetic_EdgeCases(t *testing.T) {
	t.Run("add zero", func(t *testing.T) {
		d1, _ := ParseDuration("1h")
		d2, _ := ParseDuration("0s")
		result := d1.Add(d2)
		if result.String() != "1h" {
			t.Errorf("1h + 0s = %s, want 1h", result.String())
		}
	})

	t.Run("subtract zero", func(t *testing.T) {
		d1, _ := ParseDuration("1h")
		d2, _ := ParseDuration("0s")
		result := d1.Sub(d2)
		if result.String() != "1h" {
			t.Errorf("1h - 0s = %s, want 1h", result.String())
		}
	})

	t.Run("subtract equal durations", func(t *testing.T) {
		d1, _ := ParseDuration("1h")
		d2, _ := ParseDuration("1h")
		result := d1.Sub(d2)
		if !result.IsZero() {
			t.Errorf("1h - 1h = %s, want 0s", result.String())
		}
	})

	t.Run("subtract larger from smaller (clamping)", func(t *testing.T) {
		d1, _ := ParseDuration("1s")
		d2, _ := ParseDuration("1h")
		result := d1.Sub(d2)
		if !result.IsZero() {
			t.Errorf("1s - 1h = %s, want 0s (clamped)", result.String())
		}
	})

	t.Run("add with overflow normalization", func(t *testing.T) {
		d1, _ := ParseDuration("30m")
		d2, _ := ParseDuration("40m")
		result := d1.Add(d2)
		if result.String() != "1h10m" {
			t.Errorf("30m + 40m = %s, want 1h10m", result.String())
		}
	})

	t.Run("compare equivalent durations", func(t *testing.T) {
		d1, _ := ParseDuration("60s")
		d2, _ := ParseDuration("1m")
		if d1.Compare(d2) != 0 {
			t.Errorf("60s.Compare(1m) = %d, want 0 (equal)", d1.Compare(d2))
		}
	})

	t.Run("compare zero durations", func(t *testing.T) {
		d1, _ := ParseDuration("0s")
		d2 := Duration{nanos: 0}
		if d1.Compare(d2) != 0 {
			t.Errorf("0s.Compare(0) = %d, want 0 (equal)", d1.Compare(d2))
		}
	})

	t.Run("add overflow clamping", func(t *testing.T) {
		// Create two durations that would overflow when added
		d1 := Duration{nanos: MaxDuration - 1000}
		d2 := Duration{nanos: 2000}
		result := d1.Add(d2)
		// Should clamp to MaxDuration, not wrap around
		if result.Nanoseconds() != MaxDuration {
			t.Errorf("Add overflow should clamp to MaxDuration, got %d", result.Nanoseconds())
		}
		if result.Nanoseconds() < 0 {
			t.Error("Add overflow should not produce negative duration")
		}
	})

	t.Run("add large durations", func(t *testing.T) {
		d1, _ := ParseDuration("200y")
		d2, _ := ParseDuration("200y")
		result := d1.Add(d2)
		// 400 years exceeds max (~292 years), should clamp
		if result.Nanoseconds() != MaxDuration {
			t.Errorf("200y + 200y should clamp to MaxDuration, got %d", result.Nanoseconds())
		}
	})
}
