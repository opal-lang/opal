package types

import "fmt"

// Duration Specification
//
// Opal durations use a simple, human-readable format: 1h30m45s
//
// Grammar (from docs/GRAMMAR.md):
//   duration = component+
//   component = number unit
//   number = [0-9]+
//   unit = "y" | "w" | "d" | "h" | "m" | "s" | "ms" | "us" | "ns"
//
// Units:
//   y  = year (365 days)
//   w  = week (7 days)
//   d  = day (24 hours)
//   h  = hour (60 minutes)
//   m  = minute (60 seconds)
//   s  = second
//   ms = millisecond (1/1000 second)
//   us = microsecond (1/1000000 second)
//   ns = nanosecond (1/1000000000 second)
//
// Constraints:
//   1. Components must be in descending order: 1h30m ✓, 30m1h ✗
//   2. Each unit can appear at most once: 1h30m ✓, 1h2h ✗
//   3. Numbers must be non-negative integers: 1h30m ✓, 1.5h ✗, -1h ✗
//   4. Each number must be followed by a unit: 1h30m ✓, 1h30 ✗
//   5. At least one component is required: 1h ✓, "" ✗
//   6. Total duration must not overflow int64: valid range is 0ns to ~292 years
//   7. Zero duration is represented as "0s"
//
// Normalization (Internal Representation):
//   - All durations are normalized to canonical form
//   - Canonical order: y, w, d, h, m, s, ms, us, ns (descending)
//   - Omit zero components (except for "0s")
//   - Overflow handling: 90s → 1m30s, 25h → 1d1h, etc.
//   - Examples:
//     - "90s" → "1m30s"
//     - "90m" → "1h30m"
//     - "25h" → "1d1h"
//     - "8d" → "1w1d"
//     - "1000ms" → "1s"
//     - "1h0m30s" → "1h30s"
//     - "30m90s" → "31m30s"
//
// Display Format:
//   - Same as normalized form
//   - User input can be non-canonical, but we always normalize
//
// Mathematics:
//   - All operations work on nanoseconds internally (highest precision)
//   - Addition: convert both to nanoseconds, add, normalize
//   - Subtraction: convert both to nanoseconds, subtract, clamp to 0ns, normalize
//   - Comparison: convert both to nanoseconds, compare
//
// Clamping:
//   - Negative results are clamped to "0s" at execution time
//   - This prevents errors from subtraction operations
//   - Example: "1s" - "2s" = "0s" (not an error)
//
// Precision:
//   - Internal representation uses nanoseconds (int64)
//   - Maximum duration: 2^63-1 nanoseconds = 9,223,372,036,854,775,807ns ≈ 292.47 years
//   - Overflow detection: inputs exceeding this limit return an error
//   - Sub-nanosecond precision is not supported

// Duration represents a parsed Opal duration
type Duration struct {
	nanos int64 // Total duration in nanoseconds (always >= 0)
}

// Unit multipliers in nanoseconds
const (
	Nanosecond  int64 = 1
	Microsecond int64 = 1000 * Nanosecond
	Millisecond int64 = 1000 * Microsecond
	Second      int64 = 1000 * Millisecond
	Minute      int64 = 60 * Second
	Hour        int64 = 60 * Minute
	Day         int64 = 24 * Hour
	Week        int64 = 7 * Day
	Year        int64 = 365 * Day
)

// MaxDuration is the maximum supported duration (2^63-1 nanoseconds ≈ 292.47 years)
const MaxDuration = int64(^uint64(0) >> 1) // 9223372036854775807

// unitOrder defines the canonical order of units (descending)
var unitOrder = []struct {
	name       string
	multiplier int64
}{
	{"y", Year},
	{"w", Week},
	{"d", Day},
	{"h", Hour},
	{"m", Minute},
	{"s", Second},
	{"ms", Millisecond},
	{"us", Microsecond},
	{"ns", Nanosecond},
}

// ParseDuration parses an Opal duration string
func ParseDuration(s string) (Duration, error) {
	if s == "" {
		return Duration{}, fmt.Errorf("duration cannot be empty")
	}

	nanos, err := parseDurationToNanos(s)
	if err != nil {
		return Duration{}, err
	}

	return Duration{nanos: nanos}, nil
}

// NormalizeDuration converts a duration string to canonical form
// This is the main entry point for normalizing user input
func NormalizeDuration(s string) (string, error) {
	d, err := ParseDuration(s)
	if err != nil {
		return "", err
	}
	return d.String(), nil
}

// String returns the canonical string representation
func (d Duration) String() string {
	return formatDuration(d.nanos)
}

// Nanoseconds returns the total duration in nanoseconds
func (d Duration) Nanoseconds() int64 {
	return d.nanos
}

// Add adds two durations, clamping to maximum duration on overflow
func (d Duration) Add(other Duration) Duration {
	// Check for overflow before addition
	if d.nanos > MaxDuration-other.nanos {
		// Clamp to maximum duration (~292.47 years)
		return Duration{nanos: MaxDuration}
	}
	return Duration{nanos: d.nanos + other.nanos}
}

// Sub subtracts a duration, clamping to 0s if result would be negative
func (d Duration) Sub(other Duration) Duration {
	result := d.nanos - other.nanos
	if result < 0 {
		result = 0
	}
	return Duration{nanos: result}
}

// IsZero returns true if duration is 0s
func (d Duration) IsZero() bool {
	return d.nanos == 0
}

// Compare compares two durations
// Returns -1 if d < other, 0 if d == other, 1 if d > other
func (d Duration) Compare(other Duration) int {
	if d.nanos < other.nanos {
		return -1
	}
	if d.nanos > other.nanos {
		return 1
	}
	return 0
}

// parseDurationToNanos parses Opal duration format to total nanoseconds
// Enforces descending order and no duplicate units
func parseDurationToNanos(s string) (int64, error) {
	var total int64
	var num int64
	var hasDigit bool
	lastUnitIndex := -1 // Start with "no unit seen" (before any valid index)

	i := 0
	for i < len(s) {
		ch := s[i]
		if ch >= '0' && ch <= '9' {
			// Check for overflow before accumulating the digit
			digit := int64(ch - '0')

			// Check if num*10 would overflow
			if num > MaxDuration/10 {
				return 0, fmt.Errorf("invalid duration %q: number too large (overflow)", s)
			}
			num *= 10

			// Check if adding digit would overflow
			if num > MaxDuration-digit {
				return 0, fmt.Errorf("invalid duration %q: number too large (overflow)", s)
			}
			num += digit

			hasDigit = true
			i++
		} else {
			if !hasDigit {
				return 0, fmt.Errorf("invalid duration %q: missing number before unit at position %d", s, i)
			}

			// Try to match a unit (longest match first to handle "ms" vs "m", "us" vs "u", etc.)
			matched := false
			matchedUnitIdx := -1
			matchedUnitLen := 0

			// Find the longest matching unit
			for unitIdx, unit := range unitOrder {
				if i+len(unit.name) <= len(s) && s[i:i+len(unit.name)] == unit.name {
					if len(unit.name) > matchedUnitLen {
						matchedUnitIdx = unitIdx
						matchedUnitLen = len(unit.name)
						matched = true
					}
				}
			}

			if matched {
				unit := unitOrder[matchedUnitIdx]
				// Check descending order (indices must increase: y=0, w=1, ..., ns=8)
				if matchedUnitIdx <= lastUnitIndex {
					return 0, fmt.Errorf("invalid duration %q: units must be in descending order (found %s after larger unit)", s, unit.name)
				}
				lastUnitIndex = matchedUnitIdx

				// Check for overflow before multiplication
				// If num > MaxDuration / multiplier, then num * multiplier will overflow
				if num > MaxDuration/unit.multiplier {
					return 0, fmt.Errorf("invalid duration %q: overflow (duration too large)", s)
				}
				product := num * unit.multiplier

				// Check for overflow before addition
				// Ensure total + product doesn't exceed MaxDuration
				if total > MaxDuration-product {
					return 0, fmt.Errorf("invalid duration %q: overflow (duration too large)", s)
				}
				total += product

				num = 0
				hasDigit = false
				i += matchedUnitLen
			}

			if !matched {
				return 0, fmt.Errorf("invalid duration %q: unknown unit at position %d", s, i)
			}
		}
	}

	if hasDigit {
		return 0, fmt.Errorf("invalid duration %q: missing unit after number", s)
	}

	return total, nil
}

// formatDuration formats nanoseconds into canonical Opal duration format
// Always returns canonical form with units in descending order, omitting zero components
func formatDuration(nanos int64) string {
	if nanos == 0 {
		return "0s"
	}

	var result string
	remaining := nanos

	for _, unit := range unitOrder {
		if remaining >= unit.multiplier {
			count := remaining / unit.multiplier
			remaining %= unit.multiplier
			result += fmt.Sprintf("%d%s", count, unit.name)
		}
	}

	return result
}
