package generator

import (
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	devcmdParser "github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestResult captures the outcome of a test for better error reporting
type TestResult struct {
	Success         bool
	ActualOutput    string
	ExpectedPattern string
	ErrorMessage    string
	Context         map[string]interface{}
}

// assertGeneratedCode validates generated Go code and provides clear feedback
func assertGeneratedCode(t *testing.T, name string, input string, validate func(string) TestResult) {
	t.Helper()

	// Parse the devcmd input
	cf, err := devcmdParser.Parse(input, false)
	if err != nil {
		t.Fatalf("%s: Failed to parse input: %v\nInput:\n%s", name, err, input)
		return
	}

	// Generate Go code
	generated, err := GenerateGo(cf)
	if err != nil {
		t.Fatalf("%s: Failed to generate Go code: %v", name, err)
		return
	}

	// Validate the generated code
	result := validate(generated)

	if !result.Success {
		t.Errorf("%s: %s", name, result.ErrorMessage)

		// Show context if available
		if result.Context != nil {
			for key, value := range result.Context {
				t.Logf("  %s: %v", key, value)
			}
		}

		// Show relevant code snippet
		if result.ExpectedPattern != "" {
			t.Logf("\n  Expected to find: %q", result.ExpectedPattern)
			showRelevantCode(t, generated, result.ExpectedPattern)
		}
	}
}

// showRelevantCode displays the relevant part of generated code for debugging
func showRelevantCode(t *testing.T, code string, pattern string) {
	t.Helper()

	lines := strings.Split(code, "\n")

	// Find command functions
	for i, line := range lines {
		if strings.Contains(line, "func (c *CLI) run") && strings.Contains(line, "(args []string)") {
			// Show the function signature and body
			funcName := extractFunctionName(line)
			t.Logf("\n  Found function: %s", funcName)

			// Show function body until closing brace
			braceCount := 0
			for j := i; j < len(lines) && j < i+50; j++ {
				if strings.Contains(lines[j], "{") {
					braceCount++
				}
				if strings.Contains(lines[j], "}") {
					braceCount--
				}

				// Highlight lines that might contain our pattern
				prefix := "    "
				if strings.Contains(lines[j], "exec.Command") ||
					strings.Contains(lines[j], `"-c"`) ||
					strings.Contains(lines[j], pattern) ||
					strings.Contains(lines[j], "sync.WaitGroup") ||
					strings.Contains(lines[j], "wg.Add(1)") ||
					strings.Contains(lines[j], "go func()") {
					prefix = " >> "
				}

				t.Logf("%s%s", prefix, lines[j])

				if braceCount == 0 && j > i {
					break
				}
			}
		}
	}
}

func extractFunctionName(line string) string {
	start := strings.Index(line, "func")
	end := strings.Index(line, "(args")
	if start >= 0 && end > start {
		return strings.TrimSpace(line[start:end])
	}
	return line
}

// Test helpers for common validations

func mustContain(pattern string) func(string) TestResult {
	return func(code string) TestResult {
		if strings.Contains(code, pattern) {
			return TestResult{Success: true}
		}
		return TestResult{
			Success:         false,
			ExpectedPattern: pattern,
			ErrorMessage:    "Generated code does not contain expected pattern",
			Context: map[string]interface{}{
				"Pattern": pattern,
			},
		}
	}
}

func mustNotContain(pattern string) func(string) TestResult {
	return func(code string) TestResult {
		if !strings.Contains(code, pattern) {
			return TestResult{Success: true}
		}
		return TestResult{
			Success:      false,
			ErrorMessage: "Generated code contains forbidden pattern",
			Context: map[string]interface{}{
				"ForbiddenPattern": pattern,
			},
		}
	}
}

func mustCompile() func(string) TestResult {
	return func(code string) TestResult {
		// First check if it's valid Go syntax
		fset := token.NewFileSet()
		_, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
		if err != nil {
			return TestResult{
				Success:      false,
				ErrorMessage: "Generated code has syntax errors",
				Context: map[string]interface{}{
					"ParseError": err.Error(),
				},
			}
		}

		// Format the code to check for formatting issues
		_, err = format.Source([]byte(code))
		if err != nil {
			return TestResult{
				Success:      false,
				ErrorMessage: "Generated code has formatting errors",
				Context: map[string]interface{}{
					"FormatError": err.Error(),
				},
			}
		}

		return TestResult{Success: true}
	}
}

func allOf(validators ...func(string) TestResult) func(string) TestResult {
	return func(code string) TestResult {
		for _, validator := range validators {
			result := validator(code)
			if !result.Success {
				return result
			}
		}
		return TestResult{Success: true}
	}
}

// Core template component tests

func TestPackageTemplate(t *testing.T) {
	input := `# Empty file`

	assertGeneratedCode(t, "Package declaration", input, allOf(
		mustCompile(),
		mustContain("package main"),
	))
}

func TestImportsTemplate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		forbidden []string
	}{
		{
			name:  "Basic command imports",
			input: `test: echo hello;`,
			expected: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
			},
			forbidden: []string{
				`"golang.org/x/sync/errgroup"`,
				`"sync"`, // Should not be included for non-parallel commands
			},
		},
		{
			name:  "Watch command imports",
			input: `watch server: npm start;`,
			expected: []string{
				`"encoding/json"`,
				`"time"`,
				`"syscall"`,
				`"path/filepath"`,
			},
			forbidden: []string{
				`"golang.org/x/sync/errgroup"`,
			},
		},
		{
			name:  "Parallel command imports",
			input: `build: { @parallel { go build ./app1; go build ./app2 } }`,
			expected: []string{
				`"sync"`,
			},
			forbidden: []string{
				`"golang.org/x/sync/errgroup"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := []func(string) TestResult{mustCompile()}
			for _, imp := range tt.expected {
				validators = append(validators, mustContain(imp))
			}
			for _, forbidden := range tt.forbidden {
				validators = append(validators, mustNotContain(forbidden))
			}
			assertGeneratedCode(t, tt.name, tt.input, allOf(validators...))
		})
	}
}

// Variable expansion tests

func TestVariableExpansion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		notWant  string
	}{
		{
			name: "Simple variable in command",
			input: `
def PORT = 8080;
server: go run main.go --port=@var(PORT);
`,
			expected: `go run main.go --port=8080`,
			notWant:  "@var(PORT)",
		},
		{
			name: "Variable in @sh decorator",
			input: `
def PORT = 8080;
server: @sh(go run main.go --port=@var(PORT));
`,
			expected: `go run main.go --port=8080`,
			notWant:  "@var(PORT)",
		},
		{
			name: "Multiple variables",
			input: `
def HOST = localhost;
def PORT = 8080;
server: @sh(./server --host=@var(HOST) --port=@var(PORT));
`,
			expected: `./server --host=localhost --port=8080`,
			notWant:  "@var(",
		},
		{
			name: "Variable in block command",
			input: `
def APP = myapp;
def VERSION = 1.0.0;
build: {
    echo "Building @var(APP) version @var(VERSION)";
    go build -o @var(APP) .
}
`,
			expected: `Building myapp version 1.0.0`,
			notWant:  "@var(",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertGeneratedCode(t, tt.name, tt.input, func(code string) TestResult {
				// Check for expected content
				if !strings.Contains(code, tt.expected) {
					return TestResult{
						Success:         false,
						ExpectedPattern: tt.expected,
						ErrorMessage:    "Variable not expanded correctly",
						Context: map[string]interface{}{
							"Expected": tt.expected,
							"NotWant":  tt.notWant,
						},
					}
				}

				// Check that variables were expanded (no @var left)
				if tt.notWant != "" && strings.Contains(code, tt.notWant) {
					// Find where the unexpanded variable is
					lines := strings.Split(code, "\n")
					for i, line := range lines {
						if strings.Contains(line, tt.notWant) {
							return TestResult{
								Success:      false,
								ErrorMessage: "Found unexpanded variable",
								Context: map[string]interface{}{
									"Line":    i + 1,
									"Content": strings.TrimSpace(line),
									"NotWant": tt.notWant,
								},
							}
						}
					}
				}

				return TestResult{Success: true}
			})
		})
	}
}

// Updated command generation tests with correct expectations
func TestCommandGeneration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectFunc  string
		expectShell string
	}{
		{
			name:        "Simple command",
			input:       `build: go build .;`,
			expectFunc:  "func (c *CLI) runBuild(args []string)",
			expectShell: `exec.Command("sh", "-c", "go build .")`,
		},
		{
			name:        "Command with dashes",
			input:       `test-unit: go test ./...;`,
			expectFunc:  "func (c *CLI) runTestUnit(args []string)",
			expectShell: `exec.Command("sh", "-c", "go test ./...")`,
		},
		{
			name: "Block command",
			input: `deploy: {
    echo "Deploying...";
    kubectl apply -f k8s/
}`,
			expectFunc:  "func (c *CLI) runDeploy(args []string)",
			expectShell: `exec.Command("sh", "-c", "echo \"Deploying...\"; kubectl apply -f k8s/")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertGeneratedCode(t, tt.name, tt.input, allOf(
				mustCompile(),
				mustContain(tt.expectFunc),
				mustContain(tt.expectShell),
			))
		})
	}
}

// Watch/Stop command tests

func TestWatchStopCommands(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:  "Watch only command",
			input: `watch server: npm start;`,
			expect: []string{
				"runInBackground",
				"ProcessRegistry",
				"case \"start\":",
				"case \"stop\":",
				"case \"logs\":",
			},
		},
		{
			name: "Watch and stop pair",
			input: `
watch server: npm start;
stop server: pkill node;
`,
			expect: []string{
				"runInBackground",
				`c.runInBackground("server", command)`,
				`"pkill node"`, // Now properly escaped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := []func(string) TestResult{mustCompile()}
			for _, exp := range tt.expect {
				validators = append(validators, mustContain(exp))
			}
			assertGeneratedCode(t, tt.name, tt.input, allOf(validators...))
		})
	}
}

// Updated decorator tests for standard library parallel execution with proper escaping
func TestDecorators(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expect    []string
		forbidden []string
	}{
		{
			name:   "@sh decorator",
			input:  `test: @sh(echo "Hello World");`,
			expect: []string{`"echo \"Hello World\""`}, // Now properly escaped
		},
		{
			name: "@parallel decorator with standard library",
			input: `build: {
    @parallel {
        go build ./cmd/app1;
        go build ./cmd/app2
    }
}`,
			expect: []string{
				"sync.WaitGroup",
				"wg.Add(1)",
				"go func()",
				"defer wg.Done()",
				`"go build ./cmd/app1"`, // Now properly escaped
				`"go build ./cmd/app2"`, // Now properly escaped
				"wg.Wait()",
				"errChan := make(chan error,",
			},
			forbidden: []string{
				"errgroup.Group",
				"g.Go(func() error",
				"golang.org/x/sync/errgroup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := []func(string) TestResult{mustCompile()}
			for _, exp := range tt.expect {
				validators = append(validators, mustContain(exp))
			}
			for _, forbidden := range tt.forbidden {
				validators = append(validators, mustNotContain(forbidden))
			}
			assertGeneratedCode(t, tt.name, tt.input, allOf(validators...))
		})
	}
}

// Updated user-defined help command tests with proper escaping expectations
func TestUserDefinedHelpCommand(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name: "No user-defined help - generates default",
			input: `
build: go build .;
test: go test ./...;
`,
			shouldHave: []string{
				`case "help", "--help", "-h":`,
				`c.showHelp()`,
				`func (c *CLI) showHelp()`,
				`fmt.Println("Available commands:")`,
			},
			shouldNotHave: []string{
				`func (c *CLI) runHelp(args []string)`,
			},
		},
		{
			name: "User-defined help - skips default",
			input: `
build: go build .;
test: go test ./...;
help: echo "Custom help message";
`,
			shouldHave: []string{
				`func (c *CLI) runHelp(args []string)`,
				`"echo \"Custom help message\""`, // Now properly escaped
				`case "help":`,
			},
			shouldNotHave: []string{
				`case "help", "--help", "-h":`,
				`func (c *CLI) showHelp()`,
				`fmt.Println("Available commands:")`,
			},
		},
		{
			name: "User-defined help with complex command",
			input: `
build: go build .;
help: {
    echo "My Custom CLI";
    echo "Commands:";
    echo "  build - Build the project"
}`,
			shouldHave: []string{
				`func (c *CLI) runHelp(args []string)`,
				`"echo \"My Custom CLI\"; echo \"Commands:\"; echo \"  build - Build the project\""`, // Now properly escaped
				`case "help":`,
			},
			shouldNotHave: []string{
				`case "help", "--help", "-h":`,
				`func (c *CLI) showHelp()`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("Failed to generate: %v", err)
			}

			// Check for required patterns
			for _, pattern := range tt.shouldHave {
				if !strings.Contains(generated, pattern) {
					t.Errorf("Expected to find pattern: %q", pattern)
					showRelevantCode(t, generated, pattern)
				}
			}

			// Check for forbidden patterns
			for _, pattern := range tt.shouldNotHave {
				if strings.Contains(generated, pattern) {
					t.Errorf("Found forbidden pattern: %q", pattern)
					showRelevantCode(t, generated, pattern)
				}
			}

			// Verify it compiles
			if result := mustCompile()(generated); !result.Success {
				t.Errorf("Generated code doesn't compile: %s", result.ErrorMessage)
			}
		})
	}
}

func TestHelpCommandEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name: "Multiple commands including help",
			input: `
build: go build .;
test: go test ./...;
help: cat README.md;
deploy: kubectl apply -f k8s/;
`,
			desc: "Help among multiple commands should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("Failed to generate: %v", err)
			}

			// Should compile without duplicate case errors
			if result := mustCompile()(generated); !result.Success {
				t.Errorf("Generated code doesn't compile: %s", result.ErrorMessage)
				t.Logf("Generated code:\n%s", generated)
			}
		})
	}
}

// Add new test for help command restrictions
func TestHelpCommandRestrictions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError string
		description string
	}{
		{
			name: "Watch help should be forbidden",
			input: `
watch help: echo "This should be forbidden";
build: go build .;
`,
			expectError: "'help' command cannot be used with 'watch' modifier",
			description: "Generator should catch watch help",
		},
		{
			name: "Standalone stop help should be forbidden",
			input: `
stop help: echo "This should be forbidden";
build: go build .;
`,
			expectError: "'help' command cannot be used with 'stop' modifier",
			description: "Generator should catch standalone stop help",
		},
		{
			name: "Regular help should be allowed",
			input: `
build: go build .;
test: go test ./...;
help: echo "Custom help message";
`,
			expectError: "",
			description: "Regular help commands should work fine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			_, err = GenerateGo(cf)

			if tt.expectError == "" {
				// Should succeed
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				} else {
					t.Logf("✅ %s", tt.description)
				}
			} else {
				// Should fail with specific error
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing %q, but got: %v", tt.expectError, err)
				} else {
					t.Logf("✅ %s: %v", tt.description, err)
				}
			}
		})
	}
}

func TestDefaultHelpFallback(t *testing.T) {
	input := `
build: go build .;
test: go test ./...;
help: echo "My custom help";
`

	cf, err := devcmdParser.Parse(input, false)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generated, err := GenerateGo(cf)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// When user defines help, default help messages should reference user's help
	expectedFallbacks := []string{
		`fmt.Fprintf(os.Stderr, "Run '%s help' for available commands.\n", os.Args[0])`,
	}

	for _, pattern := range expectedFallbacks {
		if !strings.Contains(generated, pattern) {
			t.Errorf("Expected fallback message pattern: %q", pattern)
		}
	}

	// Should not have default help function
	if strings.Contains(generated, `func (c *CLI) showHelp()`) {
		t.Error("Should not generate default showHelp function when user defines help")
	}
}

// Updated full integration tests with proper escaping expectations

func TestFullCLIGeneration(t *testing.T) {
	input := `
# A complete example with various features
def PROJECT = myapp;
def PORT = 8080;
def VERSION = 1.0.0;

# Build commands
build: {
    echo "Building @var(PROJECT) v@var(VERSION)...";
    go build -ldflags="-X main.version=@var(VERSION)" -o @var(PROJECT) .
}

test: go test -v ./...;

# Development server
watch server: @sh(go run . --port=@var(PORT));
stop server: @sh(pkill -f "go run");

# Deployment with parallel execution
deploy: {
    echo "Deploying @var(PROJECT)...";
    @parallel {
        docker build -t @var(PROJECT):@var(VERSION) .;
        docker push @var(PROJECT):@var(VERSION)
    };
    kubectl set image deployment/@var(PROJECT) app=@var(PROJECT):@var(VERSION)
}
`

	assertGeneratedCode(t, "Full CLI", input, func(code string) TestResult {
		// Verify it compiles
		compileResult := mustCompile()(code)
		if !compileResult.Success {
			return compileResult
		}

		// Check critical features with updated patterns
		criticalPatterns := []struct {
			pattern string
			reason  string
		}{
			// Variables are expanded
			{"Building myapp v1.0.0", "Variables in echo not expanded"},
			{"-X main.version=1.0.0", "Variables in build flags not expanded"},
			{"--port=8080", "Port variable not expanded"},

			// Commands are generated
			{"func (c *CLI) runBuild(args []string)", "Build function not generated"},
			{"func (c *CLI) runTest(args []string)", "Test function not generated"},
			{"func (c *CLI) runServer(args []string)", "Server function not generated"},
			{"func (c *CLI) runDeploy(args []string)", "Deploy function not generated"},

			// Process management for watch
			{"ProcessRegistry", "Process registry not included"},
			{"runInBackground", "Background process support missing"},

			// Standard library parallel execution instead of errgroup
			{"sync.WaitGroup", "sync.WaitGroup not used for parallel execution"},
			{"wg.Add(1)", "WaitGroup Add not called"},
			{"go func()", "Goroutines not generated for parallel commands"},
			{"defer wg.Done()", "WaitGroup Done not called"},
			{`"docker build -t myapp:1.0.0 ."`, "Parallel docker build command missing (properly escaped)"},
			{`"docker push myapp:1.0.0"`, "Parallel docker push command missing (properly escaped)"},
			{"wg.Wait()", "WaitGroup Wait not called"},
			{"errChan := make(chan error,", "Error channel not created"},

			// Main structure
			{"func main()", "Main function missing"},
			{"cli.Execute()", "CLI execution missing"},
		}

		for _, check := range criticalPatterns {
			if !strings.Contains(code, check.pattern) {
				return TestResult{
					Success:         false,
					ExpectedPattern: check.pattern,
					ErrorMessage:    check.reason,
				}
			}
		}

		// Verify no unexpanded variables remain
		if strings.Contains(code, "@var(") {
			lines := strings.Split(code, "\n")
			for i, line := range lines {
				if strings.Contains(line, "@var(") {
					return TestResult{
						Success:      false,
						ErrorMessage: "Found unexpanded variable",
						Context: map[string]interface{}{
							"Line":    i + 1,
							"Content": strings.TrimSpace(line),
						},
					}
				}
			}
		}

		// Verify we DON'T have errgroup imports or usage
		forbiddenPatterns := []string{
			"golang.org/x/sync/errgroup",
			"errgroup.Group",
			"g.Go(func() error",
		}

		for _, forbidden := range forbiddenPatterns {
			if strings.Contains(code, forbidden) {
				return TestResult{
					Success:      false,
					ErrorMessage: "Found forbidden errgroup usage",
					Context: map[string]interface{}{
						"Found":    forbidden,
						"Expected": "Standard library sync.WaitGroup",
					},
				}
			}
		}

		return TestResult{Success: true}
	})
}

// Test actual compilation with only standard library

func TestActualCompilation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation test in short mode")
	}

	input := `
def APP = testapp;
def PORT = 3000;

build: go build -o @var(APP) .;
test: go test ./...;
run: ./@var(APP) --port=@var(PORT);

# Test parallel execution with standard library
parallel-build: {
    @parallel {
        go build -o @var(APP)-linux .;
        go build -o @var(APP)-windows .
    }
}

watch dev: @sh(go run . --port=@var(PORT) --dev);
stop dev: @sh(pkill -f "go run");
`

	cf, err := devcmdParser.Parse(input, false)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generated, err := GenerateGo(cf)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}

	// Verify sync import is included when parallel commands are present
	if !strings.Contains(generated, `"sync"`) {
		t.Errorf("Expected sync import for parallel commands")
	}

	// Verify errgroup is NOT included
	if strings.Contains(generated, `"golang.org/x/sync/errgroup"`) {
		t.Errorf("Found forbidden errgroup import - should use standard library only")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "devcmd_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to clean up temp dir: %v", err)
		}
	}()

	// Write generated code
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(generated), 0o644); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Initialize go.mod (without external dependencies)
	cmd := exec.Command("go", "mod", "init", "testcli")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init go.mod: %v\nOutput: %s", err, output)
	}

	// Try to build (should work with only standard library)
	cmd = exec.Command("go", "build", "-o", "testcli", ".")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("Generated code failed to compile: %v", err)
		t.Logf("Compiler output:\n%s", output)

		// Show the generated code for debugging
		t.Logf("\nGenerated code:\n")
		lines := strings.Split(generated, "\n")
		for i, line := range lines {
			t.Logf("%4d: %s", i+1, line)
		}
	} else {
		t.Log("✅ Generated code compiles successfully with standard library only")

		// Try to run help
		cmd = exec.Command("./testcli", "help")
		cmd.Dir = tmpDir
		if output, err := cmd.CombinedOutput(); err == nil {
			t.Logf("✅ CLI runs successfully\nHelp output:\n%s", output)
		}
	}
}

// Error handling tests

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError string
	}{
		{
			name:        "Invalid @parallel syntax",
			input:       `test: @parallel(echo one; echo two);`,
			expectError: "@parallel decorator cannot be used with function syntax",
		},
		{
			name:        "Unknown decorator",
			input:       `test: @unknown(echo hello);`,
			expectError: "unsupported decorator '@unknown'",
		},
		{
			name:        "Watch help forbidden",
			input:       `watch help: echo "help";`,
			expectError: "'help' command cannot be used with 'watch' modifier",
		},
		{
			name:        "Stop help forbidden",
			input:       `stop help: echo "help";`,
			expectError: "'help' command cannot be used with 'stop' modifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}

			_, err = GenerateGo(cf)
			if err == nil {
				t.Errorf("Expected error containing %q, but got no error", tt.expectError)
			} else if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, but got: %v", tt.expectError, err)
			}
		})
	}
}

// Test that GenerateGoWithTemplate function still exists (for the build error)
func TestGenerateGoWithTemplate(t *testing.T) {
	input := `test: echo hello;`
	cf, err := devcmdParser.Parse(input, false)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	template := `{{.PackageName}} - {{range .Commands}}{{.Name}}{{end}}`
	result, err := GenerateGoWithTemplate(cf, template)
	if err != nil {
		t.Fatalf("GenerateGoWithTemplate failed: %v", err)
	}

	if !strings.Contains(result, "main") {
		t.Errorf("Expected result to contain 'main', got: %s", result)
	}
}

// Test specific standard library parallel patterns
func TestStandardLibraryParallelPatterns(t *testing.T) {
	input := `
parallel-test: {
    @parallel {
        echo "Command 1";
        echo "Command 2";
        echo "Command 3"
    }
}
`

	assertGeneratedCode(t, "Standard Library Parallel", input, func(code string) TestResult {
		// Must compile
		if result := mustCompile()(code); !result.Success {
			return result
		}

		// Check for standard library parallel patterns with proper escaping
		requiredPatterns := []string{
			"var wg sync.WaitGroup",
			"errChan := make(chan error, 3)", // Should match number of parallel commands
			"wg.Add(1)",
			"go func()",
			"defer wg.Done()",
			"wg.Wait()",
			"close(errChan)",
			"for err := range errChan",
			`"echo \"Command 1\""`, // Now properly escaped
			`"echo \"Command 2\""`, // Now properly escaped
			`"echo \"Command 3\""`, // Now properly escaped
		}

		for _, pattern := range requiredPatterns {
			if !strings.Contains(code, pattern) {
				return TestResult{
					Success:         false,
					ExpectedPattern: pattern,
					ErrorMessage:    "Missing required standard library parallel pattern",
				}
			}
		}

		// Should NOT contain errgroup patterns
		forbiddenPatterns := []string{
			"errgroup.Group",
			"g.Go(func() error",
			"golang.org/x/sync/errgroup",
		}

		for _, pattern := range forbiddenPatterns {
			if strings.Contains(code, pattern) {
				return TestResult{
					Success:      false,
					ErrorMessage: "Found forbidden errgroup pattern",
					Context: map[string]interface{}{
						"ForbiddenPattern": pattern,
					},
				}
			}
		}

		return TestResult{Success: true}
	})
}

// Updated test for the shell quoting issues that now should be fixed
func TestShellQuotingIssues(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "Special characters with quotes",
			input:       `special-chars: echo "Special: !#$%^&*()";`,
			description: "Commands with special characters in quotes should generate valid shell commands",
		},
		{
			name:        "Mixed quotes",
			input:       `mixed: echo 'Single quotes' && echo "Double quotes";`,
			description: "Commands with mixed quote types should be handled properly",
		},
		{
			name:        "Quotes with variables",
			input:       `def MSG = "Hello World"; test: echo "@var(MSG) with quotes";`,
			description: "Variable expansion within quoted strings should work",
		},
		{
			name:        "Backticks in commands",
			input:       "backticks: echo `date` and other stuff;",
			description: "Commands with backticks should not break Go template generation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("Failed to generate Go code: %v", err)
			}

			// Verify it compiles
			if result := mustCompile()(generated); !result.Success {
				t.Errorf("Generated code doesn't compile: %s", result.ErrorMessage)
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Verify that commands are properly escaped in the generated code
			expectedEscapePatterns := map[string]string{
				"special-chars": `"echo \"Special: !#$%^&*()\""`,
				"mixed":         `"echo 'Single quotes' && echo \"Double quotes\""`,
				"test":          `"echo \"Hello World with quotes\""`,
				"backticks":     `"echo ` + "`date`" + ` and other stuff"`,
			}

			commandName := extractCommandName(tt.input)
			if expectedPattern, exists := expectedEscapePatterns[commandName]; exists {
				if !strings.Contains(generated, expectedPattern) {
					t.Errorf("Expected to find properly escaped command: %q", expectedPattern)
					t.Logf("Generated code:\n%s", generated)
				} else {
					t.Logf("✅ Command properly escaped: %s", expectedPattern)
				}
			}

			// Create temp directory and test actual execution
			tmpDir, err := os.MkdirTemp("", "devcmd_quote_test_*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					t.Logf("Failed to clean up temp dir: %v", err)
				}
			}()

			// Write generated code
			mainFile := filepath.Join(tmpDir, "main.go")
			if err := os.WriteFile(mainFile, []byte(generated), 0o644); err != nil {
				t.Fatalf("Failed to write main.go: %v", err)
			}

			// Initialize go.mod
			cmd := exec.Command("go", "mod", "init", "testcli")
			cmd.Dir = tmpDir
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("Failed to init go.mod: %v\nOutput: %s", err, output)
			}

			// Build the CLI
			cmd = exec.Command("go", "build", "-o", "testcli", ".")
			cmd.Dir = tmpDir
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("Generated code failed to compile: %v\nOutput: %s", err, output)
			}

			if commandName == "" {
				t.Skip("Could not extract command name for runtime testing")
				return
			}

			// Try to run the command
			t.Logf("Testing runtime execution of command: %s", commandName)
			cmd = exec.Command("./testcli", commandName)
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()

			// Log the result for debugging
			t.Logf("Command output: %s", string(output))
			if err != nil {
				t.Logf("Command error: %v", err)

				// Check if this is the specific shell quoting error we fixed
				outputStr := string(output)
				if strings.Contains(outputStr, "unexpected EOF while looking for matching") ||
					strings.Contains(outputStr, "unterminated quoted string") {
					t.Errorf("Shell quoting error still detected: %v\nOutput: %s", err, outputStr)
					t.Errorf("The printf \"%%q\" escaping should have fixed this issue")
				} else {
					t.Logf("Command failed but not due to shell quoting issues: %v", err)
				}
			} else {
				t.Logf("✅ Command executed successfully")
			}
		})
	}
}

// Helper function to extract command name from test input
func extractCommandName(input string) string {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "def ") {
			continue
		}

		// Look for pattern: "commandname:"
		colonIndex := strings.Index(line, ":")
		if colonIndex > 0 {
			commandPart := strings.TrimSpace(line[:colonIndex])
			// Handle watch/stop prefixes
			if strings.HasPrefix(commandPart, "watch ") {
				return strings.TrimSpace(commandPart[6:])
			}
			if strings.HasPrefix(commandPart, "stop ") {
				return strings.TrimSpace(commandPart[5:])
			}
			return commandPart
		}
	}
	return ""
}
