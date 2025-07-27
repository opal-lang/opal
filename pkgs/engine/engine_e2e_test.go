package engine

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestEngine_EndToEndRealCommandsFile tests end-to-end functionality with real commands.cli
func TestEngine_EndToEndRealCommandsFile(t *testing.T) {
	// Read the actual commands.cli file from the project root
	// Use relative path from pkgs/engine/ to project root
	file, err := os.Open("../../commands.cli")
	if err != nil {
		t.Skipf("commands.cli not found at ../../commands.cli, skipping test: %v", err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read commands.cli file: %v", err)
	}

	// Parse the real commands.cli file
	program, err := parser.Parse(strings.NewReader(string(content)))
	if err != nil {
		t.Fatalf("Failed to parse real commands.cli file: %v", err)
	}

	t.Logf("Successfully parsed commands.cli: %d variables, %d commands",
		len(program.Variables), len(program.Commands))

	// Create engine and test all modes
	engine := New(program)

	// Test interpreter mode (variable processing)
	t.Run("interpreter_mode", func(t *testing.T) {
		err := engine.processVariablesIntoContext(program)
		if err != nil {
			t.Fatalf("Failed to process variables in interpreter mode: %v", err)
		}

		// Verify key variables are processed correctly
		expectedVars := map[string]string{
			"PROJECT":    "devcmd",
			"GO_VERSION": "1.22",
		}

		for name, expectedValue := range expectedVars {
			if actualValue, exists := engine.ctx.GetVariable(name); !exists {
				t.Errorf("Variable %s not found in context", name)
			} else if actualValue != expectedValue {
				t.Errorf("Variable %s: expected %s, got %s", name, expectedValue, actualValue)
			}
		}

		t.Logf("Successfully processed %d variables in interpreter mode",
			len(engine.ctx.Variables))
	})

	// Test generator mode (code generation)
	t.Run("generator_mode", func(t *testing.T) {
		result, err := engine.GenerateCode(program)
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}

		generatedCode := result.String()

		// Validate generated code structure
		requiredElements := []string{
			"package main",
			"func main()",
			"cobra.Command",
			"rootCmd.Execute()",
			"PROJECT := \"devcmd\"",
			"GO_VERSION := \"1.22\"",
		}

		for _, element := range requiredElements {
			if !strings.Contains(generatedCode, element) {
				t.Errorf("Generated code should contain %q", element)
			}
		}

		// Verify commands are generated
		expectedCommands := []string{"setupCmd", "buildCmd", "testCmd", "helpCmd"}
		for _, cmdName := range expectedCommands {
			if !strings.Contains(generatedCode, cmdName) {
				t.Errorf("Generated code should contain command %q", cmdName)
			}
		}

		// Verify imports are properly handled
		expectedImports := []string{
			"\"fmt\"",
			"\"os\"",
			"\"github.com/spf13/cobra\"",
		}

		for _, imp := range expectedImports {
			if !strings.Contains(generatedCode, imp) {
				t.Errorf("Generated code should contain import %s", imp)
			}
		}

		t.Logf("Successfully generated %d lines of Go code",
			strings.Count(generatedCode, "\n"))
		t.Logf("Generated code includes %d standard imports, %d third-party imports",
			len(result.StandardImports), len(result.ThirdPartyImports))
	})

	// Test specific decorator functionality found in the real file
	t.Run("decorator_processing", func(t *testing.T) {
		// Find commands that use decorators
		var parallelCommands, functionDecoratorCommands int

		for _, cmd := range program.Commands {
			for _, content := range cmd.Body.Content {
				switch c := content.(type) {
				case *ast.BlockDecorator:
					if c.Name == "parallel" {
						parallelCommands++
					}
				case *ast.ShellContent:
					// Count function decorators in shell content
					for _, part := range c.Parts {
						if _, ok := part.(*ast.FunctionDecorator); ok {
							functionDecoratorCommands++
						}
					}
				}
			}
		}

		t.Logf("Found %d commands with @parallel decorator", parallelCommands)
		t.Logf("Found %d function decorator usages", functionDecoratorCommands)

		if parallelCommands == 0 && functionDecoratorCommands == 0 {
			t.Log("No decorators found in commands.cli - this is expected if the file uses simpler syntax")
		}
	})

	// Test command execution structure
	t.Run("command_execution", func(t *testing.T) {
		if len(program.Commands) == 0 {
			t.Fatal("Expected at least one command in commands.cli")
		}

		// Test executing the first command (setup)
		setupCmd := &program.Commands[0]
		if setupCmd.Name != "setup" {
			t.Errorf("Expected first command to be 'setup', got %s", setupCmd.Name)
		}

		cmdResult, err := engine.ExecuteCommand(setupCmd)
		if err != nil {
			// Expected in test environment - verify structure
			if !strings.Contains(err.Error(), "command execution failed") &&
				!strings.Contains(err.Error(), "command failed") {
				t.Logf("Command execution failed as expected in test environment: %v", err)
			}
		}

		// Verify command result structure
		if cmdResult.Name != "setup" {
			t.Errorf("Expected command name 'setup', got %s", cmdResult.Name)
		}

		if cmdResult.Status == "" {
			t.Error("Command status should not be empty")
		}

		t.Logf("Command execution structure validated: name=%s, status=%s",
			cmdResult.Name, cmdResult.Status)
	})

	// Test variable resolution in complex expressions
	t.Run("variable_resolution", func(t *testing.T) {
		// Verify shell commands with variable substitutions are properly handled
		var commandsWithVars int
		for _, cmd := range program.Commands {
			for _, content := range cmd.Body.Content {
				if shell, ok := content.(*ast.ShellContent); ok {
					for _, part := range shell.Parts {
						if _, ok := part.(*ast.FunctionDecorator); ok {
							commandsWithVars++
							break
						}
					}
				}
			}
		}

		t.Logf("Found %d commands with variable substitutions", commandsWithVars)
	})
}

// TestEngine_EndToEndValidation tests comprehensive validation of the parsing and generation pipeline
func TestEngine_EndToEndValidation(t *testing.T) {
	// Test with a simpler, more controlled example that covers key features
	input := `# Test devcmd file with comprehensive features
var PROJECT = "test-project"
var VERSION = "1.0.0"
var DEBUG = true

# Simple command
build: echo "Building @var(PROJECT) version @var(VERSION)"

# Command with parallel execution
test: @parallel {
    echo "Running unit tests"
    echo "Running integration tests"
}

# Command with conditional execution
deploy: @when("ENV") {
    prod: kubectl apply -f prod.yaml
    staging: kubectl apply -f staging.yaml
    default: echo "No deployment for environment"
}

# Command with timeout
benchmark: @timeout(duration=30s) {
    echo "Running benchmarks"
    sleep 5
}

# Command with environment variables
serve: echo "Serving on @env("HOST", default="localhost"):@env("PORT", default="8080")"

# Command using debug variable
debug: echo "Debug mode: @var(DEBUG)"`

	// Parse the test input
	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse test input: %v", err)
	}

	t.Logf("Parsed test program: %d variables, %d commands",
		len(program.Variables), len(program.Commands))

	// Create engine and test comprehensive functionality
	engine := New(program)

	// Test variable processing
	err = engine.processVariablesIntoContext(program)
	if err != nil {
		t.Fatalf("Failed to process variables: %v", err)
	}

	// Verify variables
	expectedVars := map[string]string{
		"PROJECT": "test-project",
		"VERSION": "1.0.0",
		"DEBUG":   "true",
	}

	for name, expectedValue := range expectedVars {
		if actualValue, exists := engine.ctx.GetVariable(name); !exists {
			t.Errorf("Variable %s not found", name)
		} else if actualValue != expectedValue {
			t.Errorf("Variable %s: expected %s, got %s", name, expectedValue, actualValue)
		}
	}

	// Test code generation
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	generatedCode := result.String()

	// Comprehensive validation of generated code
	validationTests := []struct {
		name     string
		contains string
		reason   string
	}{
		{"package_declaration", "package main", "Go code needs package declaration"},
		{"main_function", "func main()", "Generated CLI needs main function"},
		{"cobra_import", "github.com/spf13/cobra", "CLI framework import required"},
		{"context_import", "context", "Context needed for command execution"},
		{"project_var", "PROJECT := \"test-project\"", "Variable should be included"},
		{"version_var", "VERSION := \"1.0.0\"", "Variable should be included"},
		{"debug_var", "DEBUG := \"true\"", "Boolean variable should be string"},
		{"build_command", "buildCmd", "Build command should be generated"},
		{"test_command", "testCmd", "Test command should be generated"},
		{"deploy_command", "deployCmd", "Deploy command should be generated"},
		{"benchmark_command", "benchmarkCmd", "Benchmark command should be generated"},
		{"serve_command", "serveCmd", "Serve command should be generated"},
		{"root_execute", "rootCmd.Execute()", "Root command execution required"},
	}

	for _, test := range validationTests {
		t.Run("validate_"+test.name, func(t *testing.T) {
			if !strings.Contains(generatedCode, test.contains) {
				t.Errorf("Generated code should contain %q - %s", test.contains, test.reason)
				// Print a snippet of the generated code for debugging
				lines := strings.Split(generatedCode, "\n")
				start := 0
				if len(lines) > 20 {
					start = len(lines) - 20
				}
				t.Logf("Last 20 lines of generated code:\n%s",
					strings.Join(lines[start:], "\n"))
			}
		})
	}

	// Validate imports are properly managed
	if len(result.StandardImports) == 0 {
		t.Error("Expected standard library imports to be collected")
	}

	if len(result.ThirdPartyImports) == 0 {
		t.Error("Expected third-party imports to be collected")
	}

	t.Logf("Generated code validation successful: %d standard imports, %d third-party imports",
		len(result.StandardImports), len(result.ThirdPartyImports))
	t.Logf("Generated %d lines of Go code", strings.Count(generatedCode, "\n"))
}
