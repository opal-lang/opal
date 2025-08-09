package lexer

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/types"
)

// Real-world Devcmd examples based on the ACTUAL language specification
func BenchmarkRealWorldScenarios(b *testing.B) {
	scenarios := []struct {
		name  string
		input string
	}{
		{
			"Frontend_Development",
			`// Frontend development workflow - spec compliant
var (
	NODE_ENV = "development"
	WEBPACK_MODE = "development"
	API_URL = "http://localhost:3000"
	HOT_RELOAD = true
)

// Simple commands
install: npm install
clean: rm -rf dist node_modules

// Development server with single decorator
dev: @timeout(30s) {
	webpack serve --mode development --hot
}

// Build process with timeout
build: @timeout(2m) {
	echo "Building for production..."
	NODE_ENV=production webpack --mode production
	echo "Build complete"
}

// Testing with parallel execution
test: @parallel {
	npm run test:unit
	npm run test:e2e
	npm run lint
}`,
		},
		{
			"DevOps_Deployment",
			`// DevOps deployment example - only valid decorators
var (
	CLUSTER = "production"
	NAMESPACE = "myapp"
	IMAGE_TAG = "latest"
	AUTO_ROLLBACK = true
)

// Deployment with timeout (no chaining allowed)
deploy: @timeout(10m) {
	echo "Deploying to cluster..."
	kubectl config use-context production
	kubectl apply -f k8s/ -n myapp
	kubectl rollout status deployment/api -n myapp
	echo "Deployment successful"
}

// Log monitoring - simple command
logs: kubectl logs -f deployment/api -n myapp

// Rollback with retry
rollback: @retry(3) {
	kubectl rollout undo deployment/api -n myapp
	kubectl rollout status deployment/api -n myapp
}

// Cleanup - simple command
stop: kubectl delete deployment --all -n myapp`,
		},
		{
			"Pattern_Matching_When",
			`// Pattern matching with @when decorator
var (
	ENV = "production"
	REGION = "us-east-1"
	FEATURE_FLAGS_ENABLED = true
)

build: @when(ENV) {
	prod: npm run build:production
	dev: npm run build:dev
	test: npm run build:test
	*: npm run build
}

deploy: @when(REGION) {
	us-east-1: kubectl apply -f k8s/us-east.yaml
	eu-west-1: kubectl apply -f k8s/eu-west.yaml
	ap-south-1: kubectl apply -f k8s/ap-south.yaml
	*: echo "Unsupported region"
}

server: @when(NODE_ENV) {
	production: node server.js --port 80
	development: nodemon server.js --port 3000
	test: node server.js --port 8080 --test
}`,
		},
		{
			"Pattern_Matching_Try",
			`// Error handling with @try decorator
var (
	BUILD_CMD = "npm run build"
	TEST_CMD = "npm test"
	FAIL_FAST = false
)

ci: @try {
	main: {
		echo "Starting CI pipeline..."
		npm run build
		npm test
		echo "CI pipeline completed successfully"
	}
	error: {
		echo "CI pipeline failed"
		echo "Cleaning up..."
		rm -rf dist node_modules/.cache
		exit 1
	}
	finally: {
		echo "CI pipeline finished"
		echo "Uploading logs..."
		aws s3 cp logs/ s3://ci-logs/ --recursive
	}
}

deploy: @try {
	main: {
		kubectl apply -f k8s/
		kubectl rollout status deployment/app
	}
	error: {
		echo "Deployment failed, rolling back..."
		kubectl rollout undo deployment/app
	}
	finally: {
		kubectl get pods
		kubectl get services
	}
}`,
		},
		{
			"Variable_References",
			`// Variable substitution with @var() function decorator
var (
	PORT = "8080"
	HOST = "localhost"
	DATABASE_URL = "postgresql://user:pass@localhost:5432/db"
	SSL_ENABLED = true
)

server: echo "Starting server on @var(PORT) at @var(HOST)"

database: psql "@var(DATABASE_URL)"

config: echo "Server: @var(HOST):@var(PORT), DB: @var(DATABASE_URL)"`,
		},
		{
			"Process_Management",
			`// Process management commands
var (
	APP_PORT = "3000"
	LOG_FILE = "/var/log/app.log"
	DAEMON_MODE = false
)

// Watch command - starts and manages process
watch server: node app.js --port @var(APP_PORT)

watch logs: tail -f @var(LOG_FILE)

// Stop commands - cleanup processes
stop server: pkill -f "node app.js"

stop logs: pkill -f "tail -f"`,
		},
		{
			"Complex_Shell_Commands",
			`// Complex shell command patterns - no invalid decorators
var (
	LOG_LEVEL = "info"
	DATABASE_URL = "postgresql://user:pass@localhost:5432/db"
	PARALLEL_JOBS = 4
	DRY_RUN = false
)

// Complex shell with pipes and redirections
process-logs: {
	tail -f /var/log/app.log | grep ERROR | while read line; do
		echo "[$(date)] $line" >> error.log
		curl -X POST webhook.com/alert -d "$line" &
	done
}

// Database operations with error handling
db-migrate: {
	export DATABASE_URL="@var(DATABASE_URL)"
	if ! pg_isready -d "$DATABASE_URL"; then
		echo "Database not ready" >&2
		exit 1
	fi
	npm run migrate && npm run seed || {
		echo "Migration failed, rolling back..."
		npm run migrate:rollback
		exit 1
	}
}

// File processing with find and xargs
process-files: {
	find ./src -name "*.js" -mtime -1 | \
	xargs -I {} sh -c 'echo "Processing {}"; eslint {} --fix'
}`,
		},
		{
			"Large_Monorepo_Config",
			generateMonorepoConfig(20), // 20 services
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				lexer := New(strings.NewReader(scenario.input))
				tokenCount := 0
				for {
					token := lexer.NextToken()
					tokenCount++
					if token.Type == types.EOF {
						break
					}
				}
			}
		})
	}
}

// generateMonorepoConfig creates a realistic monorepo configuration using only valid syntax
func generateMonorepoConfig(serviceCount int) string {
	var result strings.Builder

	// Global variables with strings and booleans
	result.WriteString(`// Monorepo microservices configuration - spec compliant
var (
	ENVIRONMENT = "production"
	LOG_LEVEL = "info"
	BASE_PORT = 3000
	DATABASE_HOST = "localhost"
	REDIS_HOST = "localhost"
	ENABLE_METRICS = true
	USE_CACHE = true
)

`)

	// Service categories
	categories := []string{"api", "web", "worker", "auth", "notifications"}

	for i := 0; i < serviceCount; i++ {
		category := categories[i%len(categories)]
		serviceId := i + 1
		port := 3000 + i

		// Each service gets its own commands with valid decorators only
		result.WriteString(fmt.Sprintf(`// %s service %d
%s%d: @when(ENVIRONMENT) {
	production: node services/%s%d/server.js --port %d --env production
	staging: node services/%s%d/server.js --port %d --env staging
	development: nodemon services/%s%d/server.js --port %d --env development
	*: echo "Unknown environment"
}

watch %s%d: nodemon services/%s%d/server.js --port %d --watch

test-%s%d: npm test --prefix services/%s%d

`, category, serviceId, category, serviceId, category, serviceId, port, category, serviceId, port, category, serviceId, port, category, serviceId, category, serviceId, port, category, serviceId, category, serviceId))
	}

	// Global commands - no decorator chaining
	result.WriteString(`
// Global operations
start-all: @parallel {
`)

	for i := 0; i < serviceCount; i++ {
		category := categories[i%len(categories)]
		result.WriteString(fmt.Sprintf(`	node services/%s%d/server.js --port %d &
`, category, i+1, 3000+i))
	}

	result.WriteString(`	wait
}

test-all: @parallel {
`)

	for i := 0; i < serviceCount; i++ {
		category := categories[i%len(categories)]
		result.WriteString(fmt.Sprintf(`	npm test --prefix services/%s%d
`, category, i+1))
	}

	result.WriteString(`}

stop all: {
	echo "Stopping all services..."
	pkill -f "services/"
	echo "All services stopped"
}

deploy: @try {
	main: {
		echo "Starting deployment..."
		kubectl config use-context production
		kubectl apply -f k8s/
		kubectl rollout status deployment --all
		echo "Deployment successful"
	}
	error: {
		echo "Deployment failed, rolling back..."
		kubectl rollout undo deployment --all
		exit 1
	}
	finally: {
		kubectl get pods
		kubectl get services
	}
}
`)

	return result.String()
}

// Test real-world performance with ONLY legal Devcmd syntax
func TestRealWorldPerformanceContracts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world performance tests in short mode")
	}

	t.Log("=== SPEC-COMPLIANT DEVCMD PERFORMANCE CONTRACTS ===")

	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			"Variable_Substitution",
			`var (
	PORT = "8080"
	HOST = "localhost"
	SECURE = true
)

server: echo "Starting on @var(HOST):@var(PORT)"
connect: ssh user@@var(HOST) -p @var(PORT)`,
			"Variable substitution with @var() function decorator and boolean support",
		},
		{
			"When_Pattern_Matching",
			`var ENV = "production"

deploy: @when(ENV) {
	production: kubectl apply -f k8s/prod.yaml
	staging: kubectl apply -f k8s/staging.yaml
	development: echo "Local development, no deployment"
	*: echo "Unknown environment"
}`,
			"Pattern matching with @when decorator",
		},
		{
			"Try_Error_Handling",
			`build: @try {
	main: {
		npm run build
		npm test
	}
	error: {
		echo "Build failed"
		exit 1
	}
	finally: {
		echo "Cleanup complete"
	}
}`,
			"Error handling with @try decorator",
		},
		{
			"Timeout_Decorator",
			`deploy: @timeout(5m) {
	kubectl apply -f k8s/
	kubectl rollout status deployment/app
}`,
			"Timeout decorator with shell commands",
		},
		{
			"Parallel_Execution",
			`test-all: @parallel {
	npm run test:unit
	npm run test:integration
	npm run lint
	npm run type-check
}`,
			"Parallel execution decorator",
		},
		{
			"Retry_Mechanism",
			`flaky-test: @retry(3) {
	npm test -- --testNamePattern="flaky"
}`,
			"Retry decorator for unreliable commands",
		},
		{
			"Process_Management",
			`watch server: node app.js --port 3000

stop server: pkill -f "node app.js"`,
			"Process management with watch and stop",
		},
		{
			"Complex_Shell",
			`monitor: {
	while true; do
		if ! pgrep -f "node server.js" > /dev/null; then
			echo "Server down, restarting..."
			node server.js &
			sleep 5
		fi
		sleep 10
	done
}`,
			"Complex shell commands with control flow",
		},
		{
			"Boolean_Variables",
			`var (
	DEBUG = true
	PRODUCTION = false
	FEATURE_X = true
)

build: echo "Debug: @var(DEBUG), Production: @var(PRODUCTION)"`,
			"Boolean variable declarations and usage",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Testing: %s", test.description)

			// Performance contract: spec-compliant examples should lex quickly
			start := time.Now()
			lexer := New(strings.NewReader(test.input))
			tokenCount := 0

			for {
				token := lexer.NextToken()
				tokenCount++
				if token.Type == types.EOF {
					break
				}
			}

			duration := time.Since(start)

			// Contract: individual examples should be imperceptible (< 10ms)
			const maxDuration = 10 * time.Millisecond
			if duration > maxDuration {
				t.Errorf("Spec-compliant performance regression: %s took %v, expected < %v",
					test.name, duration, maxDuration)
			}

			t.Logf("Performance: %d tokens in %v (%.1f tokens/μs)",
				tokenCount, duration, float64(tokenCount)/float64(duration.Microseconds()))
		})
	}
}

// Test nested decorator performance - spec compliant nesting only
func TestValidNestedDecoratorPerformance(t *testing.T) {
	// Valid nesting: decorators inside @when and @try patterns
	input := `deploy: @when(ENVIRONMENT) {
	production: @timeout(10m) {
		kubectl apply -f k8s/prod/
		kubectl rollout status deployment/api
	}
	staging: @timeout(5m) {
		kubectl apply -f k8s/staging/
	}
	development: echo "Local development"
	*: echo "Unknown environment"
}

ci: @try {
	main: @parallel {
		npm run build
		npm run test
		npm run lint
	}
	error: @retry(2) {
		echo "Retrying failed CI..."
		npm run build
		npm test
	}
	finally: {
		echo "CI completed"
	}
}`

	start := time.Now()
	lexer := New(strings.NewReader(input))
	tokenCount := 0
	decoratorCount := 0

	for {
		token := lexer.NextToken()
		tokenCount++
		if token.Type == types.AT {
			decoratorCount++
		}
		if token.Type == types.EOF {
			break
		}
	}

	duration := time.Since(start)

	// Contract: nested decorators should be imperceptible (< 10ms)
	// Current CI performance: ~880µs, well under human perception
	const maxDuration = 10 * time.Millisecond
	if duration > maxDuration {
		t.Errorf("Valid nested decorator performance regression: took %v, expected < %v",
			duration, maxDuration)
	}

	t.Logf("Valid nested decorators: %d tokens (%d decorators) in %v",
		tokenCount, decoratorCount, duration)
}

// Test only the standard library decorators that are actually defined
func TestStandardDecoratorPerformance(t *testing.T) {
	// Using current decorators: @var, @env, @parallel, @timeout, @retry, @when, @try
	input := `var DATABASE_URL = "postgresql://localhost:5432/db"
var API_KEY = "secret-key"

// Function decorators: @var, @env
connect: psql "@var(DATABASE_URL)"
api: curl -H "Authorization: @env(API_KEY)" api.example.com

// Block decorators: @parallel, @timeout, @retry
services: @parallel {
	node api.js
	node worker.js
}

deploy: @timeout(30s) {
	kubectl apply -f k8s/
}

flaky: @retry(3) {
	npm test -- --flaky
}

// Pattern decorators: @when, @try
env-deploy: @when(ENV) {
	prod: kubectl apply -f prod.yaml
	dev: echo "Development mode"
	*: echo "Unknown environment"
}

safe-deploy: @try {
	main: kubectl apply -f k8s/
	error: kubectl rollout undo deployment/app
	finally: kubectl get pods
}`

	// Use benchmark-style timing for more reliable results
	const iterations = 100
	start := time.Now()

	var totalTokens int
	for i := 0; i < iterations; i++ {
		lexer := New(strings.NewReader(input))
		tokenCount := 0

		for {
			token := lexer.NextToken()
			tokenCount++
			if token.Type == types.EOF {
				break
			}
		}
		totalTokens = tokenCount // Same for each iteration
	}

	duration := time.Since(start)
	avgDuration := duration / iterations

	// Human perception threshold: avg should be imperceptible (< 5ms for GitHub Actions)
	// GitHub Actions performance: ~1.4ms avg, so 5ms allows headroom
	const maxAvgDuration = 5 * time.Millisecond
	if avgDuration > maxAvgDuration {
		t.Errorf("Standard decorator performance regression: avg %v per parse, expected < %v",
			avgDuration, maxAvgDuration)
		t.Errorf("Total: %d iterations of %d tokens in %v", iterations, totalTokens, duration)
	}

	t.Logf("Standard decorators: %d tokens, avg %v per parse (%d iterations)",
		totalTokens, avgDuration, iterations)
}

// Benchmark memory usage with spec-compliant patterns only
func BenchmarkSpecCompliantMemory(b *testing.B) {
	// Large spec-compliant configuration
	input := generateMonorepoConfig(15) // 15 services

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lexer := New(strings.NewReader(input))
		for {
			token := lexer.NextToken()
			if token.Type == types.EOF {
				break
			}
		}
	}

	runtime.ReadMemStats(&m2)
	allocatedBytes := m2.TotalAlloc - m1.TotalAlloc

	b.ReportMetric(float64(allocatedBytes)/float64(b.N), "bytes/op")
	b.ReportMetric(float64(len(input)), "input-bytes")
}

// Test boolean token performance
func TestBooleanTokenPerformance(t *testing.T) {
	input := `var (
	DEBUG = true
	PRODUCTION = false
	FEATURE_A = true
	FEATURE_B = false
	FEATURE_C = true
	ENABLE_LOGGING = true
	ENABLE_METRICS = false
	ENABLE_TRACING = true
	USE_CACHE = false
	USE_CDN = true
)

config: @when(PRODUCTION) {
	true: echo "Production mode enabled"
	false: echo "Development mode"
}`

	start := time.Now()
	lexer := New(strings.NewReader(input))
	tokenCount := 0
	booleanCount := 0

	for {
		token := lexer.NextToken()
		tokenCount++
		if token.Type == types.BOOLEAN {
			booleanCount++
		}
		if token.Type == types.EOF {
			break
		}
	}

	duration := time.Since(start)

	// Contract: boolean tokens should be imperceptible (< 10ms)
	// Current CI performance: ~450µs, well under human perception
	const maxDuration = 10 * time.Millisecond
	if duration > maxDuration {
		t.Errorf("Boolean token performance regression: took %v, expected < %v",
			duration, maxDuration)
	}

	t.Logf("Boolean tokens: %d tokens (%d booleans) in %v",
		tokenCount, booleanCount, duration)
}
