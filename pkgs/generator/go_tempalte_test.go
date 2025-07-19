package generator

import (
	"fmt"
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
	program, err := devcmdParser.Parse(input)
	if err != nil {
		t.Fatalf("%s: Failed to parse input: %v\nInput:\n%s", name, err, input)
		return
	}

	// Debug logging for parallel commands
	if strings.Contains(name, "Parallel") {
		t.Logf("DEBUG %s: Program has %d commands", name, len(program.Commands))
		for i, cmd := range program.Commands {
			t.Logf("DEBUG %s: Command %d: name=%s, type=%v, body content count=%d",
				name, i, cmd.Name, cmd.Type, len(cmd.Body.Content))
			for j, content := range cmd.Body.Content {
				t.Logf("DEBUG %s: Content %d: type=%T", name, j, content)
			}
		}
	}

	// Generate Go code
	generated, err := GenerateGo(program)
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

// showRelevantCode displays only the relevant part of generated code
func showRelevantCode(t *testing.T, code string, pattern string) {
	t.Helper()

	lines := strings.Split(code, "\n")

	// First try to find exec.Command lines containing our pattern
	for i, line := range lines {
		if strings.Contains(line, "exec.Command") && strings.Contains(line, pattern) {
			showCodeContext(t, lines, i, 3, "Found exec.Command with pattern")
			return
		}
	}

	// If not found in exec.Command, look for the pattern in function context
	for i, line := range lines {
		if strings.Contains(line, pattern) {
			// Find the enclosing function
			funcStart := i
			for j := i; j >= 0; j-- {
				if strings.Contains(lines[j], "func (c *CLI) run") {
					funcStart = j
					break
				}
			}

			if funcStart != i {
				t.Logf("\n  Found in function starting at line %d:", funcStart+1)
				showCodeContext(t, lines, i, 5, "Pattern found here")
			} else {
				showCodeContext(t, lines, i, 3, "Pattern found")
			}
			return
		}
	}

	// Pattern not found - show command functions
	t.Logf("\n  Pattern not found. Showing command functions:")
	for i, line := range lines {
		if strings.Contains(line, "func (c *CLI) run") {
			showCodeContext(t, lines, i, 10, "Command function")
		}
	}
}

// showCodeContext shows lines around a specific line with highlighting
func showCodeContext(t *testing.T, lines []string, center int, radius int, label string) {
	t.Helper()

	start := center - radius
	if start < 0 {
		start = 0
	}
	end := center + radius + 1
	if end > len(lines) {
		end = len(lines)
	}

	t.Logf("\n  %s (line %d):", label, center+1)
	for i := start; i < end; i++ {
		prefix := "    "
		if i == center {
			prefix = " >> "
		}
		t.Logf("%s%4d: %s", prefix, i+1, lines[i])
	}
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

		// Find where the forbidden pattern appears
		lines := strings.Split(code, "\n")
		for i, line := range lines {
			if strings.Contains(line, pattern) {
				return TestResult{
					Success:      false,
					ErrorMessage: fmt.Sprintf("Found forbidden pattern at line %d", i+1),
					Context: map[string]interface{}{
						"ForbiddenPattern": pattern,
						"Line":             i + 1,
						"Content":          strings.TrimSpace(line),
					},
				}
			}
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
			// Extract line number from error
			errStr := err.Error()
			lines := strings.Split(code, "\n")

			// Try to parse line number from error
			var lineNum int
			if n, _ := fmt.Sscanf(errStr, "generated.go:%d:", &lineNum); n == 1 && lineNum > 0 && lineNum <= len(lines) {
				return TestResult{
					Success:      false,
					ErrorMessage: "Generated code has syntax errors",
					Context: map[string]interface{}{
						"ParseError": err.Error(),
						"ErrorLine":  lineNum,
						"Code":       lines[lineNum-1],
					},
				}
			}

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
			input: `test: echo hello`,
			expected: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
			},
			forbidden: []string{
				`"sync"`, // Should not be included for non-parallel commands
			},
		},
		{
			name:  "Watch command imports",
			input: `watch server: npm start`,
			expected: []string{
				`"encoding/json"`,
				`"time"`,
				`"syscall"`,
				`"path/filepath"`,
			},
		},
		{
			name:  "Parallel command imports",
			input: `build: @parallel { go build ./app1; go build ./app2 }`,
			expected: []string{
				`"sync"`,
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
			input: `var PORT = 8080
server: go run main.go --port=@var(PORT)`,
			expected: `exec.Command("sh", "-c", "go run main.go --port=8080")`,
			notWant:  "@var(PORT)",
		},
		{
			name: "Multiple variables",
			input: `var HOST = "localhost"
var PORT = 8080
server: ./server --host=@var(HOST) --port=@var(PORT)`,
			expected: `exec.Command("sh", "-c", "./server --host=localhost --port=8080")`,
			notWant:  "@var(",
		},
		{
			name: "Variable in block command - single command per line",
			input: `var APP = "myapp"
var VERSION = "1.0.0"
build: {
    echo "Building @var(APP) version @var(VERSION)"
    go build -o @var(APP) .
}`,
			expected: `exec.Command("sh", "-c", "echo \"Building myapp version 1.0.0\"")`,
			notWant:  "@var(",
		},
		{
			name: "Variable in semicolon-separated command",
			input: `var APP = "myapp"
build: echo "Building @var(APP)"; go build -o @var(APP) .`,
			expected: `exec.Command("sh", "-c", "echo \"Building myapp\"; go build -o myapp .")`,
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
					return TestResult{
						Success:         false,
						ErrorMessage:    "Found unexpanded variable",
						ExpectedPattern: tt.notWant,
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
		description string
	}{
		{
			name:        "Simple command",
			input:       `build: echo hello`,
			expectFunc:  "func (c *CLI) runBuild(args []string)",
			expectShell: `exec.Command("sh", "-c", "echo hello")`,
			description: "Simple commands should generate single exec call",
		},
		{
			name:        "Command with dashes",
			input:       `test-unit: go test ./...`,
			expectFunc:  "func (c *CLI) runTestUnit(args []string)",
			expectShell: `exec.Command("sh", "-c", "go test ./...")`,
			description: "Commands with dashes should be camelCased in function names",
		},
		{
			name:        "Semicolon-separated commands in single line",
			input:       `deploy: echo "Deploying..."; kubectl apply -f k8s/`,
			expectFunc:  "func (c *CLI) runDeploy(args []string)",
			expectShell: `exec.Command("sh", "-c", "echo \"Deploying...\"; kubectl apply -f k8s/")`,
			description: "Semicolon-separated commands on one line = one ShellContent = one exec",
		},
		{
			name: "Block with multiple lines",
			input: `deploy: {
    echo "Starting deployment"
    kubectl apply -f k8s/
    echo "Deployment complete"
}`,
			expectFunc:  "func (c *CLI) runDeploy(args []string)",
			expectShell: `exec.Command("sh", "-c", "echo \"Starting deployment\"")`,
			description: "Block commands with multiple lines should generate multiple exec calls",
		},
		{
			name: "Line continuation creates single command",
			input: `build: echo "Starting build" && \
    npm install && \
    npm run build`,
			expectFunc:  "func (c *CLI) runBuild(args []string)",
			expectShell: `exec.Command("sh", "-c", "echo \"Starting build\" && npm install && npm run build")`,
			description: "Line continuations should create single ShellContent = one exec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
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
		name        string
		input       string
		expect      []string
		description string
	}{
		{
			name:  "Watch only command",
			input: `watch server: npm start`,
			expect: []string{
				"runInBackground",
				"ProcessRegistry",
				"case \"start\":",
				"case \"stop\":",
				"case \"logs\":",
				`c.runInBackground("server", "npm start")`,
			},
			description: "Watch commands should support start/stop/logs subcommands",
		},
		{
			name: "Watch and stop pair",
			input: `watch server: npm start
stop server: pkill node`,
			expect: []string{
				"runInBackground",
				`c.runInBackground("server", "npm start")`,
				`exec.Command("sh", "-c", "pkill node")`,
			},
			description: "Watch/stop pairs should use custom stop command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			validators := []func(string) TestResult{mustCompile()}
			for _, exp := range tt.expect {
				validators = append(validators, mustContain(exp))
			}
			assertGeneratedCode(t, tt.name, tt.input, allOf(validators...))
		})
	}
}

// Parallel decorator tests
func TestParallelDecorator(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expect      []string
		forbidden   []string
		description string
	}{
		{
			name: "Parallel with semicolon-separated commands",
			input: `build: @parallel {
    go build ./cmd/app1; go build ./cmd/app2
}`,
			expect: []string{
				"sync.WaitGroup",
				"wg.Add(1)",
				"go func()",
				"defer wg.Done()",
				`exec.Command("sh", "-c", "go build ./cmd/app1; go build ./cmd/app2")`,
				"wg.Wait()",
				"errChan := make(chan error, 1)", // Only 1 because it's a single ShellContent
			},
			forbidden: []string{
				"errgroup",
			},
			description: "Semicolon-separated = one ShellContent = one goroutine",
		},
		{
			name: "Parallel with multiple lines",
			input: `build: @parallel {
    go build ./cmd/app1
    go build ./cmd/app2
    go build ./cmd/app3
}`,
			expect: []string{
				"sync.WaitGroup",
				"wg.Add(1)",
				"go func()",
				"defer wg.Done()",
				`exec.Command("sh", "-c", "go build ./cmd/app1")`,
				`exec.Command("sh", "-c", "go build ./cmd/app2")`,
				`exec.Command("sh", "-c", "go build ./cmd/app3")`,
				"wg.Wait()",
				"errChan := make(chan error, 3)", // 3 separate ShellContents
			},
			description: "Multiple lines = multiple ShellContents = multiple goroutines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
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

// User-defined help command tests
func TestUserDefinedHelpCommand(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    []string
		shouldNotHave []string
		description   string
	}{
		{
			name: "No user-defined help - generates default",
			input: `build: go build .
test: go test ./...`,
			shouldHave: []string{
				`case "help", "--help", "-h":`,
				`c.showHelp()`,
				`func (c *CLI) showHelp()`,
				`fmt.Println("Available commands:")`,
			},
			shouldNotHave: []string{
				`func (c *CLI) runHelp(args []string)`,
			},
			description: "Should generate default help when not user-defined",
		},
		{
			name: "User-defined help - skips default",
			input: `build: go build .
test: go test ./...
help: echo "Custom help message"`,
			shouldHave: []string{
				`func (c *CLI) runHelp(args []string)`,
				`exec.Command("sh", "-c", "echo \"Custom help message\"")`,
				`case "help":`,
			},
			shouldNotHave: []string{
				`case "help", "--help", "-h":`,
				`func (c *CLI) showHelp()`,
				`fmt.Println("Available commands:")`,
			},
			description: "Should use user-defined help instead of default",
		},
		{
			name: "User-defined help with multiline block",
			input: `build: go build .
help: {
    echo "My Custom CLI"
    echo "Commands:"
    echo "  build - Build the project"
}`,
			shouldHave: []string{
				`func (c *CLI) runHelp(args []string)`,
				`exec.Command("sh", "-c", "echo \"My Custom CLI\"")`,
				`exec.Command("sh", "-c", "echo \"Commands:\"")`,
				`exec.Command("sh", "-c", "echo \"  build - Build the project\"")`,
				`case "help":`,
			},
			shouldNotHave: []string{
				`case "help", "--help", "-h":`,
				`func (c *CLI) showHelp()`,
			},
			description: "Multiline help should generate multiple exec calls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			program, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			generated, err := GenerateGo(program)
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

// Test help command restrictions
func TestHelpCommandRestrictions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError string
		description string
	}{
		{
			name: "Watch help should be forbidden",
			input: `watch help: echo "This should be forbidden"
build: go build .`,
			expectError: "'help' command cannot be used with 'watch' modifier",
			description: "Generator should catch watch help",
		},
		{
			name: "Stop help should be forbidden",
			input: `stop help: echo "This should be forbidden"
build: go build .`,
			expectError: "'help' command cannot be used with 'stop' modifier",
			description: "Generator should catch stop help",
		},
		{
			name: "Regular help should be allowed",
			input: `build: go build .
test: go test ./...
help: echo "Custom help message"`,
			expectError: "",
			description: "Regular help commands should work fine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			program, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			_, err = GenerateGo(program)

			if tt.expectError == "" {
				// Should succeed
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			} else {
				// Should fail with specific error
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.expectError)
				} else if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing %q, but got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// Full integration test
func TestFullCLIGeneration(t *testing.T) {
	input := `# A complete example with various features
var PROJECT = "myapp"
var PORT = 8080
var VERSION = "1.0.0"

# Build commands - multiline block
build: {
    echo "Building @var(PROJECT) v@var(VERSION)..."
    go build -ldflags="-X main.version=@var(VERSION)" -o @var(PROJECT) .
}

# Test with single line
test: go test -v ./...

# Development server
watch server: go run . --port=@var(PORT)
stop server: pkill -f "go run"

# Deployment with parallel execution - semicolon separated
deploy: {
    echo "Deploying @var(PROJECT)..."
    @parallel {
        docker build -t @var(PROJECT):@var(VERSION) .; docker push @var(PROJECT):@var(VERSION)
    }
    kubectl set image deployment/@var(PROJECT) app=@var(PROJECT):@var(VERSION)
}`

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

			// Parallel execution with semicolon-separated commands as single exec
			{"sync.WaitGroup", "sync.WaitGroup not used for parallel execution"},
			{`exec.Command("sh", "-c", "docker build -t myapp:1.0.0 .; docker push myapp:1.0.0")`, "Parallel docker commands not in single exec"},

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
			return TestResult{
				Success:         false,
				ErrorMessage:    "Found unexpanded variable",
				ExpectedPattern: "@var(",
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

	input := `var APP = "testapp"
var PORT = 3000

build: go build -o @var(APP) .
test: go test ./...
run: ./@var(APP) --port=@var(PORT)

# Test parallel with line continuations
parallel-build: @parallel {
    GOOS=linux go build -o @var(APP)-linux . && \
    echo "Linux build complete"
    GOOS=windows go build -o @var(APP)-windows.exe . && \
    echo "Windows build complete"
}

watch dev: go run . --port=@var(PORT) --dev
stop dev: pkill -f "go run"`

	program, err := devcmdParser.Parse(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generated, err := GenerateGo(program)
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
			t.Logf("Failed to remove temp dir: %v", err)
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

		// Show first few errors
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i < 10 && strings.TrimSpace(line) != "" {
				t.Logf("  %s", line)
			}
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

// Test shell content boundaries
func TestShellContentBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectCmds  []string
		description string
	}{
		{
			name: "Newlines create separate commands",
			input: `deploy: {
    echo "Step 1"
    echo "Step 2"
    echo "Step 3"
}`,
			expectCmds: []string{
				`exec.Command("sh", "-c", "echo \"Step 1\"")`,
				`exec.Command("sh", "-c", "echo \"Step 2\"")`,
				`exec.Command("sh", "-c", "echo \"Step 3\"")`,
			},
			description: "Each line in block = separate ShellContent = separate exec",
		},
		{
			name:  "Semicolons keep commands together",
			input: `deploy: echo "Step 1"; echo "Step 2"; echo "Step 3"`,
			expectCmds: []string{
				`exec.Command("sh", "-c", "echo \"Step 1\"; echo \"Step 2\"; echo \"Step 3\"")`,
			},
			description: "Semicolons = single ShellContent = single exec",
		},
		{
			name: "Line continuations keep commands together",
			input: `deploy: echo "Step 1" && \
    echo "Step 2" && \
    echo "Step 3"`,
			expectCmds: []string{
				`exec.Command("sh", "-c", "echo \"Step 1\" && echo \"Step 2\" && echo \"Step 3\"")`,
			},
			description: "Line continuations = single ShellContent = single exec",
		},
		{
			name: "Mixed newlines and semicolons",
			input: `deploy: {
    echo "Group 1 cmd 1"; echo "Group 1 cmd 2"
    echo "Group 2 cmd 1"; echo "Group 2 cmd 2"
}`,
			expectCmds: []string{
				`exec.Command("sh", "-c", "echo \"Group 1 cmd 1\"; echo \"Group 1 cmd 2\"")`,
				`exec.Command("sh", "-c", "echo \"Group 2 cmd 1\"; echo \"Group 2 cmd 2\"")`,
			},
			description: "Newlines separate ShellContents, semicolons stay within",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)

			program, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			generated, err := GenerateGo(program)
			if err != nil {
				t.Fatalf("Failed to generate: %v", err)
			}

			// Check each expected command
			for _, expectedCmd := range tt.expectCmds {
				if !strings.Contains(generated, expectedCmd) {
					t.Errorf("Missing expected command: %s", expectedCmd)
					showRelevantCode(t, generated, "exec.Command")
				}
			}

			// Verify it compiles
			if result := mustCompile()(generated); !result.Success {
				t.Errorf("Generated code doesn't compile: %s", result.ErrorMessage)
			}
		})
	}
}
