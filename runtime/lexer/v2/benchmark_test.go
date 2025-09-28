package v2

import (
	"testing"
)

// Focused benchmark suite for opal V2 lexer performance analysis.
//
// This suite replaces previous micro-benchmarks with actionable, meaningful metrics:
// - BenchmarkLexerCore: Primary performance across syntax complexity levels
// - BenchmarkTelemetryModes: Observability overhead (production vs debug)
// - BenchmarkLexerScaling: Linear scaling verification across file sizes
// - BenchmarkLexerInitialization: Lexer creation cost for IDE integration
//
// Key targets: <250ns/op simple syntax, <500ns/op arithmetic, 0 allocs/op hot paths

// BenchmarkLexerCore measures pure tokenization performance across syntax complexity levels.
// This is the primary performance metric - tracks lexer efficiency for different opal syntax patterns.
// Excludes lexer creation overhead to focus on tokenization speed.
// Target: <250ns/op simple, <500ns/op arithmetic, 0 allocs/op in hot paths.
func BenchmarkLexerCore(b *testing.B) {
	scenarios := map[string]string{
		"simple":     "var x = 5",
		"arithmetic": "count + 1 >= max && timeout <= 30s",
		"complex":    `if env == "prod" && replicas >= 3 { /* deploy */ }`,
		"realistic":  generateTypicalDevcmdScript(),
	}

	for name, input := range scenarios {
		b.Run(name, func(b *testing.B) {
			// Pre-convert to bytes to avoid allocation in benchmark (like BenchmarkLexerZeroAlloc)
			inputBytes := []byte(input)
			lexer := NewLexer("") // Create with empty string, we'll use Init()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Reset lexer state for fresh tokenization
				lexer.Init(inputBytes)

				// Tokenize all input - this is what we're measuring (zero allocations)
				for {
					token := lexer.NextToken()
					if token.Type == EOF {
						break
					}
				}
			}
		})
	}
}

// BenchmarkTelemetryModes measures observability overhead for production vs debugging.
// Critical for understanding the cost of enabling telemetry in different environments.
// Production should have minimal overhead, debugging mode shows full cost of rich telemetry.
// Target: <50% overhead for full debugging mode.
func BenchmarkTelemetryModes(b *testing.B) {
	input := generateTypicalDevcmdScript() // Realistic input
	inputBytes := []byte(input)

	modes := map[string][]LexerOpt{
		"production": {},                      // No telemetry (production default)
		"monitoring": {WithTelemetryBasic()},  // Basic telemetry for monitoring
		"debugging":  {WithTelemetryTiming()}, // Full telemetry for debugging
	}

	for mode, opts := range modes {
		b.Run(mode, func(b *testing.B) {
			// Create lexer outside benchmark
			lexer := NewLexer(input, opts...)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Reset and tokenize - measuring telemetry overhead
				lexer.Init(inputBytes)

				var tokenCount int
				for {
					token := lexer.NextToken()
					tokenCount++
					if token.Type == EOF {
						break
					}
				}
				_ = tokenCount
			}
		})
	}
}

// BenchmarkLexerScaling verifies linear O(n) performance scaling across file sizes.
// Ensures the lexer handles enterprise-scale opal files efficiently.
// Should show consistent per-token performance regardless of total input size.
// Target: Linear scaling, no performance degradation with larger files.
func BenchmarkLexerScaling(b *testing.B) {
	sizes := map[string]int{
		"small":  100,   // ~100 tokens (~10 lines)
		"medium": 1000,  // ~1K tokens (~100 lines)
		"large":  10000, // ~10K tokens (~1K lines)
	}

	for size, tokenCount := range sizes {
		input := generateRealisticInput(tokenCount)
		inputBytes := []byte(input)

		b.Run(size, func(b *testing.B) {
			// Create lexer outside benchmark
			lexer := NewLexer(input)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Reset and tokenize - measuring scaling performance
				lexer.Init(inputBytes)

				var tokenCount int
				for {
					token := lexer.NextToken()
					tokenCount++
					if token.Type == EOF {
						break
					}
				}
				_ = tokenCount
			}
		})
	}
}

// BenchmarkLexerInitialization measures lexer creation cost separately from tokenization.
// Important for IDE/LSP integration where lexers are created frequently.
// One-time cost that should be reasonable for interactive tools.
// Target: <50μs for initialization to feel instant in editors.
func BenchmarkLexerInitialization(b *testing.B) {
	scenarios := map[string]string{
		"simple":    "var x = 5",
		"realistic": generateTypicalDevcmdScript(),
	}

	for name, input := range scenarios {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				lexer := NewLexer(input)
				_ = lexer
			}
		})
	}
}

// generateTypicalDevcmdScript creates a realistic opal deployment script
func generateTypicalDevcmdScript() string {
	return `// Production deployment script
var replicas = 3        /* minimum for HA */
var timeout = 30s       // deployment timeout
var maxCpu = 2          // CPU limit per pod

if env == "production" && replicas >= 3 && !maintenance {
    /* Deploy with high availability settings */
    if cpu <= maxCpu && memory <= maxMemory || force {
        // Proceed with deployment
        deploymentReady = true
        retries = 0
    }
}

// Resource validation chain
while retries <= 5 && !deploymentReady {
    retries++
    timeout *= 2    // exponential backoff
}
`
}
