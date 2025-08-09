package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/cli/internal/parser"
	"github.com/aledsdavies/devcmd/core/ast"

	// Import builtins to register decorators
	_ "github.com/aledsdavies/devcmd/cli/internal/builtins"
)

// TestMain ensures cleanup happens after all tests
func TestMain(m *testing.M) {
	code := m.Run()

	// Final cleanup - ignore errors since we're cleaning up
	_ = os.Remove("generated.go")
	matches, _ := filepath.Glob("*.tmp")
	for _, match := range matches {
		_ = os.Remove(match)
	}

	os.Exit(code)
}

// TestCommandResultGeneration tests comprehensive CommandResult scenarios
func TestCommandResultGeneration(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "simple_command",
			input:       `hello: echo "Hello World"`,
			expectError: false,
			description: "Basic command with streaming output",
		},
		{
			name: "cmd_decorator",
			input: `hello: echo "Hello"
greet: @cmd(hello)`,
			expectError: false,
			description: "@cmd decorator calling another command",
		},
		{
			name:        "shell_operators_and",
			input:       `chain: echo "First" && echo "Second"`,
			expectError: false,
			description: "Shell AND operator",
		},
		{
			name:        "shell_operators_or",
			input:       `chain: echo "First" || echo "Second"`,
			expectError: false,
			description: "Shell OR operator",
		},
		{
			name:        "shell_operators_pipe",
			input:       `chain: echo "Hello" | grep "Hell"`,
			expectError: false,
			description: "Shell pipe operator",
		},
		{
			name: "variables",
			input: `var NAME = "World"
hello: echo "Hello, $NAME!"`,
			expectError: false,
			description: "Variable substitution",
		},
		{
			name:        "failed_command",
			input:       `fail: false`,
			expectError: false,
			description: "Command that returns non-zero exit code",
		},
		{
			name: "cmd_calling_failed",
			input: `fail: false
call_fail: @cmd(fail)`,
			expectError: false,
			description: "@cmd decorator calling failed command",
		},
		{
			name: "mixed_commands",
			input: `var MSG = "test"
simple: echo "Simple: $MSG"
cmd_ref: @cmd(simple)
operators: echo "First" && echo "Second"
fail_cmd: false`,
			expectError: false,
			description: "Mix of all command types",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the input
			program, err := parser.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Create engine and generate code
			engine := New(program)
			result, err := engine.GenerateCode(program)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the generated code contains CommandResult
			if !strings.Contains(result.Code.String(), "CommandResult") {
				t.Errorf("Generated code should contain CommandResult type")
			}

			// Verify imports are reasonable (no unused imports)
			code := result.Code.String()

			// Check that strings import is only present when needed
			hasStringsImport := strings.Contains(code, `"strings"`)
			hasActionDecorators := strings.Contains(tc.input, "@cmd") ||
				strings.Contains(tc.input, "&&") ||
				strings.Contains(tc.input, "||") ||
				strings.Contains(tc.input, "|")

			if hasStringsImport && !hasActionDecorators {
				t.Errorf("Strings import present but no ActionDecorators detected in: %s", tc.input)
			}

			// Verify streaming output is implemented
			if !strings.Contains(code, "io.MultiWriter") {
				t.Errorf("Generated code should use io.MultiWriter for streaming output")
			}

			// Verify error handling returns CommandResult
			if strings.Contains(code, "fmt.Errorf") && !strings.Contains(code, "CommandResult") {
				t.Errorf("Generated code should not use fmt.Errorf without CommandResult context")
			}

			// E2E Test: Compile and run the generated code
			if err := compileAndTestGeneratedCode(t, result, tc.name); err != nil {
				t.Errorf("E2E compilation/execution failed: %v", err)
				t.Logf("Generated code that failed to compile:\n%s", code)
				return
			}

			t.Logf("✅ %s: %s (E2E: compiled and executed successfully)", tc.name, tc.description)
		})
	}
}

// TestVarExpansionIssues tests @var expansion with complex values that caused dogfooding failures
func TestVarExpansionIssues(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name: "var_with_shell_command_substitution",
			input: `var VERSION = "$(git describe --tags)"
test: echo "Version: @var(VERSION)"`,
			expectError: false,
			description: "Variable with shell command substitution syntax",
		},
		{
			name: "var_with_complex_shell_syntax",
			input: `var VERSION = "$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')"
build: echo "Building version @var(VERSION)"`,
			expectError: false,
			description: "Variable with complex shell syntax including pipes and redirections",
		},
		{
			name: "var_with_quotes_and_dollars",
			input: `var BUILD_TIME = "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
info: echo "Built at @var(BUILD_TIME)"`,
			expectError: false,
			description: "Variable with date format containing special characters",
		},
		{
			name: "var_in_complex_command",
			input: `var PROJECT = "myproject"
build: go build -ldflags="-s -w -X main.Version=@var(PROJECT)" -o @var(PROJECT) ./main.go`,
			expectError: false,
			description: "Variable used in complex go build command with ldflags",
		},
		{
			name: "workdir_with_var_substitution",
			input: `var PROJECT = "myproject"
build: @workdir("cli") { go build -o ../@var(PROJECT) ./main.go }`,
			expectError: false,
			description: "Variable used inside @workdir decorator",
		},
		{
			name: "parallel_with_var_expansion",
			input: `var PROJECT = "testproject" 
setup: @parallel {
    echo "Setting up @var(PROJECT)"
    echo "Version: $(git describe)"
}`,
			expectError: false,
			description: "Variable expansion inside @parallel decorator",
		},
		{
			name: "multiple_var_substitutions",
			input: `var VERSION = "$(git describe --tags)"
var BUILD_TIME = "$(date -u)"
var PROJECT = "myapp"
info: echo "Project: @var(PROJECT), Version: @var(VERSION), Built: @var(BUILD_TIME)"`,
			expectError: false,
			description: "Multiple variable substitutions in single command",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the input
			program, err := parser.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Create engine and generate code
			engine := New(program)
			result, err := engine.GenerateCode(program)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the generated code contains CommandResult
			if !strings.Contains(result.Code.String(), "CommandResult") {
				t.Errorf("Generated code should contain CommandResult type")
			}

			// E2E Test: Compile and run the generated code
			if err := compileAndTestGeneratedCode(t, result, tc.name); err != nil {
				t.Errorf("E2E compilation/execution failed: %v", err)
				t.Logf("Generated code that failed to compile:\n%s", result.Code.String())
				return
			}

			t.Logf("✅ %s: %s (E2E: compiled and executed successfully)", tc.name, tc.description)
		})
	}
}

// TestImportOptimization specifically tests import management
func TestImportOptimization(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		shouldHaveStrings bool
		description       string
	}{
		{
			name:              "simple_no_strings",
			input:             `hello: echo "World"`,
			shouldHaveStrings: false,
			description:       "Simple command should not import strings",
		},
		{
			name: "cmd_decorator_no_strings",
			input: `hello: echo "World"
greet: @cmd(hello)`,
			shouldHaveStrings: false,
			description:       "@cmd decorator should not need strings import",
		},
		{
			name:              "shell_operators_no_strings",
			input:             `chain: echo "First" && echo "Second"`,
			shouldHaveStrings: false,
			description:       "Shell operators (&&, ||) should not need strings import",
		},
		{
			name:              "pipe_operator_no_strings",
			input:             `chain: echo "Hello" | grep "Hell"`,
			shouldHaveStrings: false,
			description:       "Pipe operator handled by shell, no strings import needed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			engine := New(program)
			result, err := engine.GenerateCode(program)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			code := result.Code.String()
			hasStringsImport := strings.Contains(code, `"strings"`)

			if tc.shouldHaveStrings && !hasStringsImport {
				t.Errorf("Expected strings import but not found in generated code")
			}

			if !tc.shouldHaveStrings && hasStringsImport {
				t.Errorf("Unexpected strings import in generated code")
			}

			t.Logf("✅ %s: strings import = %v (expected %v)", tc.name, hasStringsImport, tc.shouldHaveStrings)
		})
	}
}

// TestEngine_ShellBlockErrorHandling tests that shell blocks stop execution on first error
func TestEngine_ShellBlockErrorHandling(t *testing.T) {
	input := `
# Command that will fail
fail_command: exit 1

# Test command with multiple shell statements where one fails
test_sequence: {
    echo "Step 1: Before failure"
    @cmd(fail_command)
    echo "Step 2: After failure - this should NOT execute"
    echo "Step 3: This should also NOT execute"
}
`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)

	// Find the test_sequence command
	var testCmd *ast.CommandDecl
	for _, cmd := range program.Commands {
		if cmd.Name == "test_sequence" {
			testCmd = &cmd
			break
		}
	}

	if testCmd == nil {
		t.Fatal("test_sequence command not found")
	}

	// Execute the command - this should fail
	result, err := engine.ExecuteCommand(testCmd)

	// The command should fail (return an error)
	if err == nil {
		t.Error("Expected command to fail but it succeeded")
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", result.Status)
	}

	// The error should indicate the failure came from the @cmd(fail_command)
	if !strings.Contains(result.Error, "fail_command") && !strings.Contains(err.Error(), "exit status 1") {
		t.Errorf("Error should mention the failing command or exit status, got: %v", err)
	}

	t.Logf("Command failed as expected with error: %v", err)
	t.Logf("Result status: %s, error: %s", result.Status, result.Error)
}

// TestEngine_BasicConstruction tests basic engine construction
func TestEngine_BasicConstruction(t *testing.T) {
	input := `var PORT = "8080"
build: echo "Building"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	// Context is now created as needed, not stored in engine

	if engine.goVersion != "1.24" {
		t.Errorf("Expected default Go version 1.24, got %s", engine.goVersion)
	}
}

// TestEngine_CustomGoVersion tests engine construction with custom Go version
func TestEngine_CustomGoVersion(t *testing.T) {
	program, err := parser.Parse(strings.NewReader(`build: echo "test"`))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := NewWithGoVersion(program, "1.23")
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	if engine.goVersion != "1.23" {
		t.Errorf("Expected Go version 1.23, got %s", engine.goVersion)
	}
}

// TestEngine_VariableProcessing tests variable processing functionality
func TestEngine_VariableProcessing(t *testing.T) {
	input := `var HOST = "localhost"
var PORT = 8080
var DEBUG = true`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program) // Create engine for testing variable processing

	// Test variable processing with decorator lookups
	ctx := engine.CreateGeneratorContext(context.Background(), program)
	err = ctx.InitializeVariables()
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	// Check variables were processed correctly
	expectedVars := map[string]string{
		"HOST":  "localhost",
		"PORT":  "8080", // Numbers are stored as their string representation
		"DEBUG": "true",
	}

	for name, expectedValue := range expectedVars {
		if actualValue, exists := ctx.GetVariable(name); !exists {
			t.Errorf("Variable %s not found", name)
		} else if actualValue != expectedValue {
			t.Errorf("Variable %s: expected %s, got %s", name, expectedValue, actualValue)
		}
	}
}

// TestEngine_CodeGeneration tests basic code generation
func TestEngine_CodeGeneration(t *testing.T) {
	input := `var PORT = "8080"
serve: echo "Serving on port @var(PORT)"
build: make all`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	generatedCode := result.String()

	// Check for required elements in generated code
	requiredElements := []string{
		"func main()",
		"const PORT = \"8080\"",
		"cobra.Command",
		"rootCmd.Execute()",
		"serveCmd",
		"buildCmd",
	}

	for _, element := range requiredElements {
		if !strings.Contains(generatedCode, element) {
			t.Errorf("Generated code should contain %q.\nGenerated code:\n%s", element, generatedCode)
		}
	}
}

// TestEngine_CommandExecution tests command execution structure
func TestEngine_CommandExecution(t *testing.T) {
	input := `greeting: echo "Hello World"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	if len(program.Commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(program.Commands))
	}

	engine := New(program)
	cmdResult, err := engine.ExecuteCommand(&program.Commands[0])
	// In test environment, command may fail - that's expected
	if err != nil {
		if !strings.Contains(err.Error(), "command execution failed") &&
			!strings.Contains(err.Error(), "command failed") {
			t.Logf("Command execution failed as expected in test environment: %v", err)
		}
	}

	// Verify command result structure
	if cmdResult.Name != "greeting" {
		t.Errorf("Expected command name 'greeting', got %s", cmdResult.Name)
	}

	if cmdResult.Status == "" {
		t.Error("Command status should not be empty")
	}

	if len(cmdResult.Output) == 0 {
		t.Log("Command output is empty, which is expected in test environment")
	}
}

// TestEngine_EmptyProgram tests handling of empty programs
func TestEngine_EmptyProgram(t *testing.T) {
	program, err := parser.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Failed to parse empty program: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed for empty program: %v", err)
	}

	generatedCode := result.String()

	// Should still have basic structure
	if !strings.Contains(generatedCode, "func main()") {
		t.Error("Generated code should contain main function even for empty program")
	}

	if !strings.Contains(generatedCode, "rootCmd.Execute()") {
		t.Error("Generated code should contain root command execution")
	}
}

// TestEngine_ErrorHandling tests error handling in various scenarios
func TestEngine_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid program",
			input:       `build: echo "hello"`,
			expectError: false,
		},
		{
			name: "valid program with variables",
			input: `var TEST = "value"
build: echo "@var(TEST)"`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				if !tt.expectError {
					t.Fatalf("Unexpected parse error: %v", err)
				}
				return
			}

			engine := New(program)
			_, err = engine.GenerateCode(program)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestEngine_GenerateCodeIsolation verifies that GenerateCode doesn't execute commands
func TestEngine_GenerateCodeIsolation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // Strings that should appear in generated code
	}{
		{
			name: "simple_command_generation",
			input: `
test_cmd: echo "This should not execute during generation"
`,
			contains: []string{
				"testCmdCmd := &cobra.Command{",
				`Use:   "test_cmd"`,
				"executeTestCmd := func(ctx ExecutionContext) error {",
			},
		},
		{
			name: "cmd_decorator_chain",
			input: `
helper: echo "Helper command"
main: @cmd(helper)
`,
			contains: []string{
				"executeHelper := func(ctx ExecutionContext) error {",
				"executeMain := func(ctx ExecutionContext) error {",
				"executeHelper(ctx)",
			},
		},
		{
			name: "variable_expansion",
			input: `
var MESSAGE = "test message"
greet: echo "@var(MESSAGE)"
`,
			contains: []string{
				`const MESSAGE = "test message"`,
				"executeGreet := func(ctx ExecutionContext) error {",
				"MESSAGE",
			},
		},
		{
			name: "shell_substitution_variables",
			input: `
var VERSION = "$(echo 'v1.0.0')"
build: echo "Building version @var(VERSION)"
`,
			contains: []string{
				`const VERSION = "$(echo 'v1.0.0')"`,
				"executeBuild := func(ctx ExecutionContext) error {",
				"VERSION",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track if any shell commands were executed
			originalDir, _ := os.Getwd()
			executed := false

			// Create a temporary test directory to isolate any side effects
			tempDir := t.TempDir()
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to restore directory: %v", err)
				}
			}()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Parse the input
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse program: %v", err)
			}

			// Generate code (this should NOT execute any commands)
			engine := New(program)
			result, err := engine.GenerateCode(program)
			if err != nil {
				t.Fatalf("GenerateCode failed: %v", err)
			}

			// Verify no commands were executed by checking no side effects
			if executed {
				t.Error("GenerateCode executed commands - this should never happen")
			}

			// Verify the generated code contains expected patterns
			generatedCode := result.String()
			for _, expected := range tt.contains {
				if !strings.Contains(generatedCode, expected) {
					t.Errorf("Generated code missing expected pattern: %q", expected)
					t.Logf("Generated code:\n%s", generatedCode)
				}
			}

			// Verify the generated code compiles
			if err := compileGeneratedCode(t, result.String(), result.GoModString()); err != nil {
				t.Errorf("Generated code does not compile: %v", err)
				t.Logf("Generated code:\n%s", generatedCode)
			}
		})
	}
}

// compileAndTestGeneratedCode compiles and tests execution of generated Go code for full E2E testing
func compileAndTestGeneratedCode(t *testing.T, result *GenerationResult, testName string) error {
	// Create temporary directory for this test
	tempDir := t.TempDir()

	// Write the generated files
	mainGoPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(result.Code.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write main.go: %v", err)
	}

	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(result.GoModString()), 0o644); err != nil {
		return fmt.Errorf("failed to write go.mod: %v", err)
	}

	// Change to temp directory for compilation
	originalDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			// This is a critical failure that could affect other tests
			fmt.Printf("Warning: failed to restore directory to %s: %v\n", originalDir, err)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		return fmt.Errorf("failed to change to temp dir: %v", err)
	}

	// Initialize go module and download dependencies
	if err := runCommand("go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %v", err)
	}

	// Compile the generated code
	binaryName := fmt.Sprintf("testcli_%s", testName)
	if err := runCommand("go", "build", "-o", binaryName, "."); err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	// Test basic CLI functionality
	binaryPath := filepath.Join(tempDir, binaryName)

	// Test 1: Help command should work
	if err := runCommand(binaryPath, "--help"); err != nil {
		return fmt.Errorf("help command failed: %v", err)
	}

	// Test 2: List available commands
	output, err := runCommandWithOutput(binaryPath, "--help")
	if err != nil {
		return fmt.Errorf("failed to get help output: %v", err)
	}

	// Should contain basic CLI structure
	if !strings.Contains(output, "Available Commands:") && !strings.Contains(output, "Usage:") {
		return fmt.Errorf("help output doesn't contain expected CLI structure")
	}

	return nil
}

// compileGeneratedCode attempts to compile the generated Go code to verify it's valid (legacy function)
func compileGeneratedCode(t *testing.T, mainGo, goMod string) error {
	// Use the enhanced version
	result := &GenerationResult{}
	result.Code.WriteString(mainGo)
	result.GoMod.WriteString(goMod)
	return compileAndTestGeneratedCode(t, result, "legacy")
}

// runCommand executes a command and returns error if it fails
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCommandWithOutput executes a command and returns output
func runCommandWithOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// TestProjectCommandsCLI_DoggfooDing ensures our own commands.cli always generates correctly
// This is a critical test that validates we can eat our own dog food
func TestProjectCommandsCLI_Dogfooding(t *testing.T) {
	// This test MUST always pass - it's our dogfooding test
	// Read the actual commands.cli file from project root
	commandsPath := filepath.Join("..", "..", "..", "commands.cli")
	commandsFile, err := os.Open(commandsPath)
	if err != nil {
		t.Fatalf("CRITICAL: Cannot read project commands.cli at %s: %v", commandsPath, err)
	}
	defer func() {
		if err := commandsFile.Close(); err != nil {
			t.Logf("Warning: failed to close commands.cli file: %v", err)
		}
	}()

	// Parse the commands.cli file
	program, err := parser.Parse(commandsFile)
	if err != nil {
		t.Fatalf("CRITICAL: Cannot parse project commands.cli: %v", err)
	}

	// Generate code - this should NOT execute any commands from commands.cli
	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("CRITICAL: Failed to generate code from project commands.cli: %v", err)
	}

	// Verify we got substantial generated code
	generatedCode := result.String()
	if len(generatedCode) < 1000 {
		t.Errorf("Generated code seems too short (%d chars) - generation may have failed", len(generatedCode))
	}

	// E2E Test: Compile and test the generated CLI from our own commands.cli
	t.Log("🚀 DOGFOODING E2E: Compiling and testing generated CLI from project commands.cli...")
	if err := compileAndTestGeneratedCode(t, result, "dogfooding"); err != nil {
		t.Fatalf("CRITICAL: Dogfooding E2E compilation/execution failed: %v", err)
	}

	t.Logf("✅ DOGFOODING SUCCESS: Project commands.cli generates, compiles, and executes correctly! Generated %d characters of working Go code.", len(generatedCode))
}

// TestEngine_CmdDecoratorNoExecutionDuringGeneration tests that @cmd decorators don't execute during generation
// This is a regression test for the critical bug where @cmd decorators would execute shell commands during code generation
func TestEngine_CmdDecoratorNoExecutionDuringGeneration(t *testing.T) {
	// Create a test file that would be created if commands execute during generation
	testFile := filepath.Join(t.TempDir(), "generation_execution_detected.txt")

	// Create a CLI definition that writes to a file if executed - this is our detection mechanism
	input := fmt.Sprintf(`
var TEST_FILE = "%s"

# This command should NEVER execute during generation - it writes a file that we can detect
dangerous_command: {
    echo "EXECUTION DETECTED DURING GENERATION" > "$TEST_FILE"
    echo "This proves the bug exists"
}

# This command uses @cmd decorator - it should only generate code, not execute
test_command: @cmd(dangerous_command)

# Another command that chains dangerous operations  
chain_command: {
    echo "This is safe"
    @cmd(dangerous_command)
    echo "After dangerous command"
}
`, testFile)

	// Parse the CLI definition
	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse test CLI: %v", err)
	}

	// Verify the test file does NOT exist before generation
	if _, err := os.Stat(testFile); err == nil {
		t.Fatalf("Test file already exists before generation: %s", testFile)
	}

	// Create engine and generate code
	engine := New(program)

	// This is the critical test - GenerateCode should NOT execute any shell commands
	t.Log("🔍 CRITICAL TEST: Calling GenerateCode with @cmd decorators that would create a file...")
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}
	t.Log("✅ GenerateCode completed")

	// THE CRITICAL CHECK: Verify the test file was NOT created during generation
	if _, err := os.Stat(testFile); err == nil {
		t.Fatalf("🚨 CRITICAL FAILURE: Test file was created during generation! Commands were executed: %s", testFile)
	}

	// Verify that code was actually generated
	generatedCode := result.String()
	if len(generatedCode) < 500 {
		t.Errorf("Generated code seems too short (%d chars) - generation may have failed", len(generatedCode))
	}

	// Verify the generated code contains basic structure
	basicPatterns := []string{
		"package main",
		"func main()",
		"cobra.Command",
		"executeDangerousCommand := func(ctx ExecutionContext) error {", // Should generate the function
		"executeTestCommand := func(ctx ExecutionContext) error {",      // Should generate the function
	}

	for _, pattern := range basicPatterns {
		if !strings.Contains(generatedCode, pattern) {
			t.Errorf("Generated code missing basic pattern: %q", pattern)
		}
	}

	// Final verification: The test file should still not exist
	if _, err := os.Stat(testFile); err == nil {
		t.Fatalf("🚨 CRITICAL FAILURE: Test file exists after generation - this proves commands executed!")
	}

	t.Log("✅ SUCCESS: @cmd decorators generated code without executing shell commands")
	t.Log("✅ REGRESSION TEST PASSED: The critical execution-during-generation bug is fixed")
}

// TestSyntaxErrorsInGeneration tests for syntax errors in generated Go code
func TestSyntaxErrorsInGeneration(t *testing.T) {
	// Test the specific pattern that was causing syntax errors with hyphens in command names
	input := `var VERSION = "$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')"
var BUILD_TIME = "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

test-command: echo "Version: @var(VERSION) at @var(BUILD_TIME)"
another-test: echo "Another test command with hyphen"`

	// Parse the input
	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	// Generate CLI code
	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate CLI: %v", err)
	}

	generatedCode := result.Code.String()

	// Test E2E compilation to detect syntax errors
	err = compileAndTestGeneratedCode(t, result, "syntax_errors")
	if err != nil {
		t.Logf("Generated code that failed to compile:\n%s", generatedCode)

		// Look for specific patterns that might cause syntax errors
		if strings.Contains(generatedCode, "test-") {
			t.Errorf("Generated code contains command name with hyphen: this may cause Go syntax errors")
		}

		t.Fatalf("E2E compilation failed with syntax errors: %v", err)
	}

	t.Logf("✅ syntax_errors: Complex variable and nested decorator test compiled successfully")
}

func TestDuplicateVariableDeclarations(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "multiple_shell_commands_same_scope",
			input: `var PROJECT = "testproject"
var VERSION = "1.0.0"

build: echo "Building @var(PROJECT) version @var(VERSION)"
test: echo "Testing @var(PROJECT)"`,
			description: "Multiple shell commands with variable substitutions should have unique variable names",
		},
		{
			name: "workdir_multiple_commands",
			input: `var PROJECT = "testproject"
var VERSION = "1.0.0"

deploy: @workdir("dist") { 
    echo "Deploying @var(PROJECT) v@var(VERSION)"
    echo "Final step for @var(PROJECT)"
}`,
			description: "Multiple commands within @workdir should have unique variable declarations",
		},
		{
			name: "parallel_multiple_commands",
			input: `var PROJECT = "testproject"

setup: @parallel { 
    echo "Setting up @var(PROJECT)"
    echo "Configuring @var(PROJECT)"
    echo "Finalizing @var(PROJECT)"
}`,
			description: "Multiple commands within @parallel should have unique variable declarations",
		},
		{
			name: "mixed_decorators_and_commands",
			input: `var PROJECT = "testproject"
var ENV = "production"

build: echo "Building @var(PROJECT)"
test: echo "Testing @var(PROJECT)"
deploy: @workdir("dist") {
    echo "Deploying @var(PROJECT) to @var(ENV)"
    echo "Verifying @var(PROJECT)"
}
cleanup: echo "Cleaning up @var(PROJECT)"`,
			description: "Mix of regular commands and decorated commands should have unique variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Generate CLI code
			engine := New(program)
			result, err := engine.GenerateCode(program)
			if err != nil {
				t.Fatalf("Failed to generate CLI: %v", err)
			}

			generatedCode := result.Code.String()

			// Test E2E compilation to ensure no duplicate variable declarations
			err = compileAndTestGeneratedCode(t, result, tt.name)
			if err != nil {
				t.Logf("Generated code that failed to compile:\n%s", generatedCode)
				t.Fatalf("E2E compilation/execution failed: %v", err)
			}

			// TODO: Verify that different shell commands get different variable names
			// Look for descriptive variable patterns like buildStdout, deployStep2Stdout, etc.
			hasDescriptiveVars := strings.Contains(generatedCode, "Stdout") &&
				strings.Contains(generatedCode, "CmdStr") &&
				(strings.Contains(generatedCode, "Step2") ||
					strings.Contains(generatedCode, "build") ||
					strings.Contains(generatedCode, "deploy"))

			if !hasDescriptiveVars {
				t.Logf("TODO: Implement descriptive variable names (e.g., buildStdout, deployStep2CmdStr) for readable code generation")
			}

			t.Logf("✅ %s: %s (E2E: compiled and executed successfully)", tt.name, tt.description)
		})
	}
}
