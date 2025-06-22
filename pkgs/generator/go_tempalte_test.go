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
					strings.Contains(lines[j], pattern) {
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
		name     string
		input    string
		expected []string
	}{
		{
			name:  "Basic command imports",
			input: `test: echo hello;`,
			expected: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validators := []func(string) TestResult{mustCompile()}
			for _, imp := range tt.expected {
				validators = append(validators, mustContain(imp))
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

// Command generation tests

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
			expectShell: `exec.Command("sh", "-c", ` + "`go build .`)",
		},
		{
			name:        "Command with dashes",
			input:       `test-unit: go test ./...;`,
			expectFunc:  "func (c *CLI) runTestUnit(args []string)",
			expectShell: `exec.Command("sh", "-c", ` + "`go test ./...`)",
		},
		{
			name: "Block command",
			input: `deploy: {
    echo "Deploying...";
    kubectl apply -f k8s/
}`,
			expectFunc:  "func (c *CLI) runDeploy(args []string)",
			expectShell: `exec.Command("sh", "-c", ` + "`echo \"Deploying...\"; kubectl apply -f k8s/`)",
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
				`pkill node`,
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

// Decorator tests

func TestDecorators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "@sh decorator",
			input:  `test: @sh(echo "Hello World");`,
			expect: `echo "Hello World"`,
		},
		{
			name: "@parallel decorator",
			input: `build: {
    @parallel: {
        go build ./cmd/app1;
        go build ./cmd/app2
    }
}`,
			expect: `go build ./cmd/app1 &; go build ./cmd/app2 &; wait`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertGeneratedCode(t, tt.name, tt.input, allOf(
				mustCompile(),
				mustContain(tt.expect),
			))
		})
	}
}

// Full integration tests

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

# Deployment
deploy: {
    echo "Deploying @var(PROJECT)...";
    @parallel: {
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

		// Check critical features
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

			// Parallel execution
			{"docker build -t myapp:1.0.0 . &", "Parallel docker build missing"},
			{"docker push myapp:1.0.0 &", "Parallel docker push missing"},
			{"wait", "Wait for parallel commands missing"},

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

		return TestResult{Success: true}
	})
}

// Test actual compilation

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

	// Initialize go.mod
	cmd := exec.Command("go", "mod", "init", "testcli")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init go.mod: %v\nOutput: %s", err, output)
	}

	// Try to build
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
		t.Log("✅ Generated code compiles successfully")

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
