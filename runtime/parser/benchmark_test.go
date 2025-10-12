package parser

import (
	"testing"

	"github.com/aledsdavies/opal/runtime/lexer"
)

// Benchmark suite for parser performance analysis.
//
// Mirrors lexer benchmark structure:
// - BenchmarkParserCore: Primary performance across syntax complexity levels
// - BenchmarkTelemetryModes: Observability overhead (production vs debug)
// - BenchmarkParserScaling: Linear scaling verification across file sizes
//
// Key targets: <2ms for 10K lines parsing, <4ms total (lex + parse), 0 allocs/op hot paths

// BenchmarkParserCore measures pure parsing performance across syntax complexity levels.
// This is the primary performance metric - tracks parser efficiency for different opal syntax patterns.
// Target: <2ms for 10K lines, 0 allocs/op in hot paths.
func BenchmarkParserCore(b *testing.B) {
	scenarios := map[string]string{
		"empty":    "",
		"simple":   "fun greet() {}",
		"function": "fun deploy(env, replicas) { kubectl apply -f k8s/ }",
		"complex":  generateComplexScript(),
	}

	for name, input := range scenarios {
		b.Run(name, func(b *testing.B) {
			inputBytes := []byte(input)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tree := Parse(inputBytes)
				_ = tree
			}
		})
	}
}

// BenchmarkTelemetryModes measures observability overhead for production vs debugging.
// Target: <10% overhead for timing telemetry.
func BenchmarkTelemetryModes(b *testing.B) {
	input := generateComplexScript()
	inputBytes := []byte(input)

	modes := map[string][]ParserOpt{
		"production": {},                      // No telemetry (production default)
		"monitoring": {WithTelemetryBasic()},  // Basic telemetry for monitoring
		"debugging":  {WithTelemetryTiming()}, // Full telemetry for debugging
	}

	for mode, opts := range modes {
		b.Run(mode, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tree := Parse(inputBytes, opts...)
				_ = tree
			}
		})
	}
}

// BenchmarkParserScaling verifies linear O(n) performance scaling across file sizes.
// Should show consistent per-token performance regardless of total input size.
func BenchmarkParserScaling(b *testing.B) {
	sizes := map[string]int{
		"small":  10,    // ~10 functions (~140 lines)
		"medium": 100,   // ~100 functions (~1.4K lines)
		"large":  1000,  // ~1000 functions (~14K lines)
		"xlarge": 10000, // ~10000 functions (~140K lines)
	}

	for size, funcCount := range sizes {
		input := generateScalingInput(funcCount)
		inputBytes := []byte(input)

		b.Run(size, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tree := Parse(inputBytes)
				_ = tree
			}
		})
	}
}

// BenchmarkLexAndParse measures total pipeline performance (lex + parse).
// Target: <4ms for 10K lines total.
func BenchmarkLexAndParse(b *testing.B) {
	input := generate10KLineFile()
	inputBytes := []byte(input)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tree := Parse(inputBytes)
		_ = tree
	}

	// Report throughput
	b.ReportMetric(float64(len(input))/1000000, "MB/s")
}

// BenchmarkParseOnly measures pure parsing performance (excludes lexer cost).
// This isolates parser performance by pre-lexing tokens.
func BenchmarkParseOnly(b *testing.B) {
	scenarios := map[string]string{
		"empty":    "",
		"simple":   "fun greet() {}",
		"function": "fun deploy(env, replicas) { kubectl apply -f k8s/ }",
		"complex":  generateComplexScript(),
	}

	for name, input := range scenarios {
		b.Run(name, func(b *testing.B) {
			inputBytes := []byte(input)

			// Pre-lex tokens (exclude from benchmark)
			lex := lexer.NewLexer()
			lex.Init(inputBytes)
			tokens := lex.GetTokens()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				tree := ParseTokens(inputBytes, tokens)
				_ = tree
			}
		})
	}
}

// Helper functions to generate test inputs

func generateComplexScript() string {
	return `fun deploy(env, replicas) {
	var timeout = 30s
	
	if env == "production" && replicas >= 3 {
		kubectl apply -f k8s/
		kubectl rollout status deployment/app
	}
	
	for service in ["api", "worker", "scheduler"] {
		kubectl scale deployment/@var.service --replicas=@var.replicas
		echo "Deployed ${service} with ${replicas} replicas"
	}
}`
}

func generateScalingInput(funcCount int) string {
	result := ""
	for i := 0; i < funcCount; i++ {
		result += "fun greet() {}\n"
	}
	return result
}

func generate10KLineFile() string {
	// Generate ~10K lines of realistic opal code
	result := ""
	for i := 0; i < 1000; i++ {
		result += `fun deploy() {
	var replicas = 3
	var timeout = 30s
	
	if env == "prod" {
		kubectl apply -f k8s/
	}
	
	for svc in ["api", "worker"] {
		kubectl scale deployment/@var.svc --replicas=@var.replicas
		echo "Deployed ${svc} with ${replicas} replicas"
	}
}

`
	}
	return result
}
