package execution

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// Mock function decorators for testing
type mockVarDecorator struct{}

func (m *mockVarDecorator) Expand(ctx *ExecutionContext, params []ast.NamedParameter) *ExecutionResult {
	if len(params) == 0 {
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@var decorator requires variable name"),
		}
	}

	// Get variable name
	var varName string
	if ident, ok := params[0].Value.(*ast.Identifier); ok {
		varName = ident.Name
	} else {
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@var decorator requires identifier argument"),
		}
	}

	switch ctx.Mode() {
	case InterpreterMode:
		// Look up variable value
		if value, exists := ctx.GetVariable(varName); exists {
			return &ExecutionResult{
				Mode:  InterpreterMode,
				Data:  value,
				Error: nil,
			}
		}
		return &ExecutionResult{
			Mode:  InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("variable '%s' not defined", varName),
		}
	case GeneratorMode:
		// Return variable name for Go code generation
		if _, exists := ctx.GetVariable(varName); exists {
			return &ExecutionResult{
				Mode:  GeneratorMode,
				Data:  varName,
				Error: nil,
			}
		}
		return &ExecutionResult{
			Mode:  GeneratorMode,
			Data:  nil,
			Error: fmt.Errorf("variable '%s' not defined", varName),
		}
	case PlanMode:
		// Return the resolved value for plan display
		if value, exists := ctx.GetVariable(varName); exists {
			return &ExecutionResult{
				Mode:  PlanMode,
				Data:  value,
				Error: nil,
			}
		}
		return &ExecutionResult{
			Mode:  PlanMode,
			Data:  nil,
			Error: fmt.Errorf("variable '%s' not defined", varName),
		}
	default:
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported mode: %v", ctx.Mode()),
		}
	}
}

type mockEnvDecorator struct{}

func (m *mockEnvDecorator) Expand(ctx *ExecutionContext, params []ast.NamedParameter) *ExecutionResult {
	if len(params) == 0 {
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@env decorator requires environment variable name"),
		}
	}

	// Get environment variable name
	var envName string
	if ident, ok := params[0].Value.(*ast.Identifier); ok {
		envName = ident.Name
	} else {
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@env decorator requires identifier argument"),
		}
	}

	switch ctx.Mode() {
	case InterpreterMode:
		// Look up environment variable value
		if value, exists := ctx.GetEnv(envName); exists {
			return &ExecutionResult{
				Mode:  InterpreterMode,
				Data:  value,
				Error: nil,
			}
		}
		// Fall back to system environment
		return &ExecutionResult{
			Mode:  InterpreterMode,
			Data:  os.Getenv(envName),
			Error: nil,
		}
	case GeneratorMode:
		// Generate os.Getenv call
		return &ExecutionResult{
			Mode:  GeneratorMode,
			Data:  fmt.Sprintf("os.Getenv(%q)", envName),
			Error: nil,
		}
	case PlanMode:
		// Return the actual env value for plan display
		if value, exists := ctx.GetEnv(envName); exists {
			return &ExecutionResult{
				Mode:  PlanMode,
				Data:  value,
				Error: nil,
			}
		}
		// Fall back to system environment
		return &ExecutionResult{
			Mode:  PlanMode,
			Data:  os.Getenv(envName),
			Error: nil,
		}
	default:
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported mode: %v", ctx.Mode()),
		}
	}
}

// mockFunctionDecoratorLookup provides mock decorators for testing
func mockFunctionDecoratorLookup(name string) (FunctionDecorator, bool) {
	switch name {
	case "var":
		return &mockVarDecorator{}, true
	case "env":
		return &mockEnvDecorator{}, true
	default:
		return nil, false
	}
}

func TestExecuteShell_InterpreterMode(t *testing.T) {
	// Create test program with variables
	program := ast.NewProgram(
		ast.Var("PROJECT", ast.Str("testproject")),
		ast.Var("VERSION", ast.Str("1.0.0")),
	)

	ctx := NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(InterpreterMode)

	// Set up mock function decorator lookup
	ctx.SetFunctionDecoratorLookup(mockFunctionDecoratorLookup)

	// Initialize variables
	err := ctx.InitializeVariables()
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	tests := []struct {
		name         string
		shellContent *ast.ShellContent
		expectError  bool
	}{
		{
			name: "Simple text command",
			shellContent: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError: false,
		},
		{
			name: "Command with variable expansion",
			shellContent: ast.Shell(
				ast.Text("echo \"Building "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text(" version "),
				ast.At("var", ast.UnnamedParam(ast.Id("VERSION"))),
				ast.Text("\""),
			),
			expectError: false,
		},
		{
			name: "Command with undefined variable",
			shellContent: ast.Shell(
				ast.Text("echo "),
				ast.At("var", ast.UnnamedParam(ast.Id("UNDEFINED"))),
			),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ExecuteShell(tt.shellContent)

			if tt.expectError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if result.Error != nil {
					t.Errorf("Unexpected error: %v", result.Error)
				}
			}

			if result.Mode != InterpreterMode {
				t.Errorf("Expected mode %v, got %v", InterpreterMode, result.Mode)
			}

			// In interpreter mode, Data should be nil
			if result.Data != nil {
				t.Errorf("Expected nil data in interpreter mode, got %v", result.Data)
			}
		})
	}
}

func TestExecuteShell_GeneratorMode(t *testing.T) {
	// Create test program with variables
	program := ast.NewProgram(
		ast.Var("PROJECT", ast.Str("testproject")),
	)

	ctx := NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(GeneratorMode)

	// Set up mock function decorator lookup
	ctx.SetFunctionDecoratorLookup(mockFunctionDecoratorLookup)

	// Initialize variables
	err := ctx.InitializeVariables()
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	tests := []struct {
		name         string
		shellContent *ast.ShellContent
		expectError  bool
		expectedCode string
	}{
		{
			name: "Simple text command",
			shellContent: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError:  false,
			expectedCode: `"echo hello"`,
		},
		{
			name: "Command with variable expansion",
			shellContent: ast.Shell(
				ast.Text("echo \"Building "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text("\""),
			),
			expectError:  false,
			expectedCode: `"echo \"Building " + PROJECT + "\""`,
		},
		{
			name: "Command with environment variable",
			shellContent: ast.Shell(
				ast.Text("echo $"),
				ast.At("env", ast.UnnamedParam(ast.Id("USER"))),
			),
			expectError:  false,
			expectedCode: `"echo $" + os.Getenv("USER")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ExecuteShell(tt.shellContent)

			if tt.expectError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
				return
			}

			if result.Mode != GeneratorMode {
				t.Errorf("Expected mode %v, got %v", GeneratorMode, result.Mode)
			}

			// Check generated code
			code, ok := result.Data.(string)
			if !ok {
				t.Errorf("Expected string data, got %T", result.Data)
				return
			}

			// Check that the expected command expression is in the generated code
			if !strings.Contains(code, tt.expectedCode) {
				t.Errorf("Expected generated code to contain %q, got:\n%s", tt.expectedCode, code)
			}

			// Check that it contains proper Go command execution structure
			expectedParts := []string{
				"cmdStr :=",
				"exec.CommandContext(ctx, \"sh\", \"-c\", cmdStr)",
				"execCmd.Stdout = os.Stdout",
				"execCmd.Run()",
			}

			for _, part := range expectedParts {
				if !strings.Contains(code, part) {
					t.Errorf("Expected generated code to contain %q, got:\n%s", part, code)
				}
			}
		})
	}
}

func TestExecuteShell_PlanMode(t *testing.T) {
	// Create test program with variables
	program := ast.NewProgram(
		ast.Var("PROJECT", ast.Str("testproject")),
	)

	ctx := NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(PlanMode)

	// Set up mock function decorator lookup
	ctx.SetFunctionDecoratorLookup(mockFunctionDecoratorLookup)

	// Initialize variables
	err := ctx.InitializeVariables()
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	tests := []struct {
		name            string
		shellContent    *ast.ShellContent
		expectError     bool
		expectedCommand string
	}{
		{
			name: "Simple text command",
			shellContent: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError:     false,
			expectedCommand: "echo hello",
		},
		{
			name: "Command with variable expansion",
			shellContent: ast.Shell(
				ast.Text("echo \"Building "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text("\""),
			),
			expectError:     false,
			expectedCommand: "echo \"Building testproject\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ExecuteShell(tt.shellContent)

			if tt.expectError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
				return
			}

			if result.Mode != PlanMode {
				t.Errorf("Expected mode %v, got %v", PlanMode, result.Mode)
			}

			// Check plan data
			planData, ok := result.Data.(map[string]interface{})
			if !ok {
				t.Errorf("Expected map[string]interface{} data, got %T", result.Data)
				return
			}

			// Check plan structure
			if planData["type"] != "shell" {
				t.Errorf("Expected plan type 'shell', got %v", planData["type"])
			}

			command, ok := planData["command"].(string)
			if !ok {
				t.Errorf("Expected command to be string, got %T", planData["command"])
				return
			}

			if command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, command)
			}
		})
	}
}

func TestExecuteShell_UnsupportedMode(t *testing.T) {
	ctx := NewExecutionContext(context.Background(), ast.NewProgram())

	// Set up mock function decorator lookup
	ctx.SetFunctionDecoratorLookup(mockFunctionDecoratorLookup)

	ctx.mode = ExecutionMode(999) // Invalid mode

	shellContent := ast.Shell(
		ast.Text("echo test"),
	)

	result := ctx.ExecuteShell(shellContent)

	if result.Error == nil {
		t.Error("Expected error for unsupported mode")
	}

	if !strings.Contains(result.Error.Error(), "unsupported execution mode") {
		t.Errorf("Expected error about unsupported mode, got: %v", result.Error)
	}
}

func TestComposeShellCommand(t *testing.T) {
	// Create test program with variables
	program := ast.NewProgram(
		ast.Var("PROJECT", ast.Str("myproject")),
	)

	ctx := NewExecutionContext(context.Background(), program)

	// Set up mock function decorator lookup
	ctx.SetFunctionDecoratorLookup(mockFunctionDecoratorLookup)

	err := ctx.InitializeVariables()
	if err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	tests := []struct {
		name         string
		shellContent *ast.ShellContent
		expected     string
		expectError  bool
	}{
		{
			name: "Text only",
			shellContent: ast.Shell(
				ast.Text("echo hello world"),
			),
			expected:    "echo hello world",
			expectError: false,
		},
		{
			name: "Mixed text and variable",
			shellContent: ast.Shell(
				ast.Text("echo \"Project: "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text("\""),
			),
			expected:    "echo \"Project: myproject\"",
			expectError: false,
		},
		{
			name: "Undefined variable",
			shellContent: ast.Shell(
				ast.At("var", ast.UnnamedParam(ast.Id("UNDEFINED"))),
			),
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.composeShellCommand(tt.shellContent)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}
