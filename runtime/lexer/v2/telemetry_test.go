package v2

import (
	"testing"
	"time"
)

// TestTelemetryOff_ZeroOverhead tests that TelemetryOff has zero allocation overhead
func TestTelemetryOff_ZeroOverhead(t *testing.T) {
	input := "var test = 123"
	lexer := NewLexer(input) // Default is TelemetryOff

	// Should have no telemetry structures allocated
	if lexer.tokenTelemetry != nil {
		t.Error("TelemetryOff should not allocate tokenTelemetry map")
	}

	// Should have no debug structures allocated
	if lexer.debugEvents != nil {
		t.Error("TelemetryOff should not allocate debugEvents slice")
	}

	// Process tokens
	tokens := lexer.GetTokens()
	if len(tokens) == 0 {
		t.Error("Expected some tokens")
	}

	// Should still have no telemetry after processing
	telemetry := lexer.GetTokenTelemetry()
	if telemetry != nil {
		t.Error("TelemetryOff should return nil telemetry")
	}

	debug := lexer.GetDebugEvents()
	if debug != nil {
		t.Error("TelemetryOff should return nil debug events")
	}
}

// TestTelemetryBasic_TokenCounts tests that TelemetryBasic tracks token counts accurately
func TestTelemetryBasic_TokenCounts(t *testing.T) {
	input := "var test = 123"
	lexer := NewLexer(input, WithTelemetryBasic())

	// Should have telemetry structures allocated
	if lexer.tokenTelemetry == nil {
		t.Error("TelemetryBasic should allocate tokenTelemetry map")
	}

	// Should not have debug structures allocated
	if lexer.debugEvents != nil {
		t.Error("TelemetryBasic should not allocate debugEvents slice")
	}

	// Process tokens
	tokens := lexer.GetTokens()
	expectedTokens := []TokenType{VAR, IDENTIFIER, EQUALS, INTEGER, EOF}

	if len(tokens) != len(expectedTokens) {
		t.Errorf("Expected %d tokens, got %d", len(expectedTokens), len(tokens))
	}

	// Verify token counts
	telemetry := lexer.GetTokenTelemetry()
	if telemetry == nil {
		t.Fatal("TelemetryBasic should return telemetry data")
	}

	// Check specific token counts
	expectedCounts := map[TokenType]int{
		VAR:        1,
		IDENTIFIER: 1,
		EQUALS:     1,
		INTEGER:    1,
		EOF:        1,
	}

	for tokenType, expectedCount := range expectedCounts {
		if tel, exists := telemetry[tokenType]; exists {
			if tel.Count != expectedCount {
				t.Errorf("Expected %d %s tokens, got %d", expectedCount, tokenType, tel.Count)
			}
			if tel.Type != tokenType {
				t.Errorf("Expected telemetry type %s, got %s", tokenType, tel.Type)
			}
			// In basic mode, timing should be zero
			if tel.TotalTime != 0 || tel.AvgTime != 0 || tel.MinTime != 0 || tel.MaxTime != 0 {
				t.Errorf("TelemetryBasic should not collect timing data, got TotalTime=%v", tel.TotalTime)
			}
		} else {
			t.Errorf("Missing telemetry for token type %s", tokenType)
		}
	}
}

// TestTelemetryTiming_PerTokenTypeTiming tests that TelemetryTiming captures per-token-type timing
func TestTelemetryTiming_PerTokenTypeTiming(t *testing.T) {
	input := "var test = 123"
	lexer := NewLexer(input, WithTelemetryTiming())

	// Should have telemetry structures allocated
	if lexer.tokenTelemetry == nil {
		t.Error("TelemetryTiming should allocate tokenTelemetry map")
	}

	// Process tokens
	tokens := lexer.GetTokens()
	if len(tokens) == 0 {
		t.Error("Expected some tokens")
	}

	// Verify timing telemetry
	telemetry := lexer.GetTokenTelemetry()
	if telemetry == nil {
		t.Fatal("TelemetryTiming should return telemetry data")
	}

	// Check that timing data is collected
	for tokenType, tel := range telemetry {
		if tel.Count <= 0 {
			t.Errorf("Expected positive count for %s, got %d", tokenType, tel.Count)
		}

		// Timing should be measured (may be very small but >= 0)
		if tel.TotalTime < 0 {
			t.Errorf("Expected non-negative TotalTime for %s, got %v", tokenType, tel.TotalTime)
		}

		if tel.AvgTime < 0 {
			t.Errorf("Expected non-negative AvgTime for %s, got %v", tokenType, tel.AvgTime)
		}

		// Min/Max should be set
		if tel.MinTime < 0 || tel.MaxTime < 0 {
			t.Errorf("Expected non-negative Min/MaxTime for %s, got Min=%v Max=%v", tokenType, tel.MinTime, tel.MaxTime)
		}

		// For single tokens, min should equal max
		if tel.Count == 1 && tel.MinTime != tel.MaxTime {
			t.Errorf("For single token %s, MinTime should equal MaxTime, got Min=%v Max=%v", tokenType, tel.MinTime, tel.MaxTime)
		}

		// Average should be TotalTime / Count
		expectedAvg := tel.TotalTime / time.Duration(tel.Count)
		if tel.AvgTime != expectedAvg {
			t.Errorf("AvgTime calculation wrong for %s: expected %v, got %v", tokenType, expectedAvg, tel.AvgTime)
		}
	}
}

// TestDebugPaths_MethodTracing tests that DebugPaths captures method entry/exit
func TestDebugPaths_MethodTracing(t *testing.T) {
	input := "123"
	lexer := NewLexer(input, WithDebugPaths())

	// Should have debug structures allocated
	if lexer.debugEvents == nil {
		t.Error("DebugPaths should allocate debugEvents slice")
	}

	// Process tokens
	tokens := lexer.GetTokens()
	if len(tokens) != 2 { // INTEGER + EOF
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}

	// Get debug events
	events := lexer.GetDebugEvents()
	if events == nil {
		t.Fatal("DebugPaths should return debug events")
	}

	// Should have some debug events
	if len(events) == 0 {
		t.Error("Expected some debug events")
	}

	// Verify event structure
	for _, event := range events {
		if event.Timestamp.IsZero() {
			t.Error("Debug event should have timestamp")
		}
		if event.Event == "" {
			t.Error("Debug event should have event description")
		}
		if event.Position.Line <= 0 || event.Position.Column <= 0 {
			t.Error("Debug event should have valid position")
		}
	}
}

// TestMixedTelemetryAndDebug tests that telemetry and debug work together
func TestMixedTelemetryAndDebug(t *testing.T) {
	input := "var x = 42"
	lexer := NewLexer(input, WithTelemetryTiming(), WithDebugPaths())

	// Should have both structures allocated
	if lexer.tokenTelemetry == nil {
		t.Error("Should allocate tokenTelemetry map")
	}
	if lexer.debugEvents == nil {
		t.Error("Should allocate debugEvents slice")
	}

	// Process tokens
	tokens := lexer.GetTokens()
	if len(tokens) == 0 {
		t.Error("Expected some tokens")
	}

	// Should have both telemetry and debug data
	telemetry := lexer.GetTokenTelemetry()
	if telemetry == nil {
		t.Error("Should return telemetry data")
	}

	debug := lexer.GetDebugEvents()
	if debug == nil {
		t.Error("Should return debug events")
	}

	// Verify both are collecting data
	if len(telemetry) == 0 {
		t.Error("Should have telemetry data")
	}
	if len(debug) == 0 {
		t.Error("Should have debug events")
	}
}

// TestTelemetryReset tests that telemetry resets correctly with Init
func TestTelemetryReset(t *testing.T) {
	input1 := "var test"
	lexer := NewLexer(input1, WithTelemetryTiming())

	// Process first input
	tokens1 := lexer.GetTokens()
	if len(tokens1) == 0 {
		t.Error("Expected some tokens from first input")
	}

	telemetry1 := lexer.GetTokenTelemetry()
	if len(telemetry1) == 0 {
		t.Error("Expected telemetry from first input")
	}

	// Reset with new input
	input2 := "123"
	lexer.Init([]byte(input2))

	// Process second input
	tokens2 := lexer.GetTokens()
	if len(tokens2) == 0 {
		t.Error("Expected some tokens from second input")
	}

	telemetry2 := lexer.GetTokenTelemetry()
	if telemetry2 == nil {
		t.Error("Expected telemetry from second input")
	}

	// Should only have telemetry for second input tokens
	for tokenType := range telemetry2 {
		if tokenType != INTEGER && tokenType != EOF {
			t.Errorf("Unexpected token type %s in telemetry after reset", tokenType)
		}
	}

	// Should not have telemetry for first input tokens
	if _, exists := telemetry2[VAR]; exists {
		t.Error("Should not have VAR token telemetry after reset")
	}
	if _, exists := telemetry2[IDENTIFIER]; exists {
		t.Error("Should not have IDENTIFIER token telemetry after reset")
	}
}

// TestTelemetryPerformanceRegression tests that telemetry doesn't significantly impact performance
func TestTelemetryPerformanceRegression(t *testing.T) {
	// Large input to make timing differences measurable
	input := ""
	for i := 0; i < 1000; i++ {
		input += "var test" + string(rune('a'+i%26)) + " = " + string(rune('0'+i%10)) + " "
	}

	// Time without telemetry
	start := time.Now()
	lexerOff := NewLexer(input) // TelemetryOff
	tokensOff := lexerOff.GetTokens()
	timeOff := time.Since(start)

	// Time with basic telemetry
	start = time.Now()
	lexerBasic := NewLexer(input, WithTelemetryBasic())
	tokensBasic := lexerBasic.GetTokens()
	timeBasic := time.Since(start)

	// Time with timing telemetry
	start = time.Now()
	lexerTiming := NewLexer(input, WithTelemetryTiming())
	tokensTiming := lexerTiming.GetTokens()
	timeTiming := time.Since(start)

	// Verify same number of tokens
	if len(tokensOff) != len(tokensBasic) || len(tokensBasic) != len(tokensTiming) {
		t.Error("All lexers should produce same number of tokens")
	}

	// Performance check: timing telemetry should not be more than 50x slower
	if timeTiming > timeOff*50 {
		t.Errorf("Timing telemetry too slow: %v vs %v (base)", timeTiming, timeOff)
	}

	// Basic telemetry should not be more than 5x slower
	if timeBasic > timeOff*5 {
		t.Errorf("Basic telemetry too slow: %v vs %v (base)", timeBasic, timeOff)
	}

	t.Logf("Performance: Off=%v, Basic=%v, Timing=%v", timeOff, timeBasic, timeTiming)
}
