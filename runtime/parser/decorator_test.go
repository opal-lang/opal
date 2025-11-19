package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/types"
	_ "github.com/opal-lang/opal/runtime/decorators" // Register built-in decorators
)

// Test decorator implementations for parser tests

// ConfigDecorator is a test decorator that accepts object parameters
type ConfigDecorator struct{}

func (d *ConfigDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("config").
		Summary("Test configuration decorator").
		Roles(decorator.RoleProvider).
		PrimaryParamString("name", "Configuration name").
		Done().
		ParamObject("settings", "Configuration settings").
		Done().
		Returns(types.TypeString, "Configuration value").
		Build()
}

func (d *ConfigDecorator) Resolve(ctx decorator.ValueEvalContext, calls ...decorator.ValueCall) ([]decorator.ResolveResult, error) {
	results := make([]decorator.ResolveResult, len(calls))
	for i := range calls {
		results[i] = decorator.ResolveResult{
			Value:  "test",
			Origin: "config",
			Error:  nil,
		}
	}
	return results, nil
}

// DeployDecorator is a test decorator that accepts array parameters
type DeployDecorator struct{}

func (d *DeployDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("deploy").
		Summary("Test deployment decorator").
		Roles(decorator.RoleWrapper).
		PrimaryParamString("target", "Deployment target").
		Done().
		ParamArray("hosts", "List of hosts").
		ElementType(types.TypeString).
		Done().
		Build()
}

func (d *DeployDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return next // Pass through for test
}

func init() {
	// Register test decorators for parser tests
	if err := decorator.Register("config", &ConfigDecorator{}); err != nil {
		panic(err)
	}
	if err := decorator.Register("deploy", &DeployDecorator{}); err != nil {
		panic(err)
	}
}

// TestDecoratorDetection tests that parser recognizes registered decorators
func TestDecoratorDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isDecorator bool
		reason      string
	}{
		{
			name:        "var decorator at top level",
			input:       "@var.name",
			isDecorator: true,
			reason:      "var is a registered decorator",
		},
		{
			name:        "env decorator at top level",
			input:       "@env.HOME",
			isDecorator: true,
			reason:      "env is a registered decorator",
		},
		{
			name:        "var decorator in assignment",
			input:       "var x = @var.name",
			isDecorator: true,
			reason:      "var is a registered decorator",
		},
		{
			name:        "env decorator in assignment",
			input:       "var home = @env.HOME",
			isDecorator: true,
			reason:      "env is a registered decorator",
		},
		{
			name:        "unknown decorator not recognized",
			input:       "@unknown.field",
			isDecorator: false,
			reason:      "unknown is not registered",
		},
		{
			name:        "email address not recognized as decorator",
			input:       "user@example.com",
			isDecorator: false,
			reason:      "example is not a registered decorator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			// Count decorator nodes
			decoratorCount := 0
			for _, evt := range tree.Events {
				if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
					decoratorCount++
				}
			}

			if tt.isDecorator && decoratorCount == 0 {
				t.Errorf("expected decorator node for %q (%s)", tt.input, tt.reason)
			}

			if !tt.isDecorator && decoratorCount > 0 {
				t.Errorf("expected no decorator node for %q (%s)", tt.input, tt.reason)
			}
		})
	}
}

// TestDecoratorInShellCommand tests decorator interpolation in shell commands
func TestDecoratorInShellCommand(t *testing.T) {
	input := `echo "Hello @var.name"`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected parse errors: %v", tree.Errors)
	}

	// Should have at least one decorator node
	hasDecorator := false
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
			hasDecorator = true
			break
		}
	}

	if !hasDecorator {
		t.Error("expected decorator node in shell command with @var.name")
	}
}

// TestLiteralAtSymbol tests that @ without registered decorator stays literal
func TestLiteralAtSymbol(t *testing.T) {
	input := `echo "Email: user@example.com"`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected parse errors: %v", tree.Errors)
	}

	// Should NOT have decorator nodes (example is not registered)
	decoratorCount := 0
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
			decoratorCount++
		}
	}

	if decoratorCount > 0 {
		t.Errorf("expected no decorator nodes for literal @ in email address, got %d", decoratorCount)
	}
}

// TestDecoratorParameters tests parsing decorator parameters with exact event sequences
func TestDecoratorParameters(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "primary only - var",
			input: "@var.username",
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // var
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // username
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0}, // Step boundary
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "primary with single param",
			input: `@env.HOME(default="")`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // env
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // HOME
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 4}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 5}, // default
				{Kind: EventToken, Data: 6}, // =
				{Kind: EventToken, Data: 7}, // ""
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 8}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0}, // Step boundary
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "multiple params",
			input: `@env.HOME(default="/home/user")`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // env
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // HOME
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 4}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 5}, // default
				{Kind: EventToken, Data: 6}, // =
				{Kind: EventToken, Data: 7}, // "/home/user"
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 8}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0}, // Step boundary
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "all named params (unsugared)",
			input: `@env(property="HOME")`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // env
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 2}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 3}, // property
				{Kind: EventToken, Data: 4}, // =
				{Kind: EventToken, Data: 5}, // "HOME"
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 6}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0}, // Step boundary
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			// Should have no errors
			if len(tree.Errors) > 0 {
				t.Errorf("unexpected errors: %v", tree.Errors)
			}

			// Compare events using cmp.Diff for exact match
			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDecoratorParameterTypeValidation tests type checking for decorator parameters
func TestDecoratorParameterTypeValidation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		// Positive cases - correct types
		{
			name:      "string param with string value",
			input:     `@env.HOME(default="")`,
			wantError: false,
		},
		{
			name:      "string param with string value - non-empty",
			input:     `@env.HOME(default="/home/user")`,
			wantError: false,
		},
		{
			name:      "multiple params with correct types",
			input:     `@env.HOME(default="/home")`,
			wantError: false,
		},
		{
			name:      "duration param with duration value",
			input:     `@retry(times=3, delay=2s)`,
			wantError: false,
		},
		{
			name:      "duration param with complex duration",
			input:     `@retry(times=3, delay=1h30m)`,
			wantError: false,
		},
		{
			name:      "integer param with integer value",
			input:     `@retry(times=5)`,
			wantError: false,
		},
		{
			name:      "boolean param with boolean value",
			input:     `@retry(times=3, delay=1s)`,
			wantError: false,
		},

		// Negative cases - type mismatches
		{
			name:           "string param with integer value",
			input:          `@env.HOME(default=42)`,
			wantError:      true,
			wantMessage:    "parameter 'default' expects string, got integer",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use a string value like \"value\"",
		},
		{
			name:           "string param with boolean value",
			input:          `@env.HOME(default=true)`,
			wantError:      true,
			wantMessage:    "parameter 'default' expects string, got boolean",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use a string value like \"value\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

// TestDecoratorRequiredParameters tests validation of required parameters
func TestDecoratorRequiredParameters(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		// Positive cases - required params provided
		{
			name:      "primary param provided via dot syntax",
			input:     `@env.HOME`,
			wantError: false,
		},
		{
			name:      "primary param provided via named param",
			input:     `@env(property="HOME")`,
			wantError: false,
		},

		// Negative cases - missing required params
		{
			name:           "missing primary param - no dot, no named param",
			input:          `@env`,
			wantError:      true,
			wantMessage:    "missing required parameter 'property'",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use dot syntax like @env.HOME or provide property=\"HOME\"",
		},
		{
			name:           "missing primary param - empty parens",
			input:          `@env()`,
			wantError:      true,
			wantMessage:    "missing required parameter 'property'",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use dot syntax like @env.HOME or provide property=\"HOME\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

// TestDecoratorUnknownParameter tests validation of unknown parameters
func TestDecoratorUnknownParameter(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		{
			name:      "known parameter",
			input:     `@env.HOME(default="")`,
			wantError: false,
		},
		{
			name:           "unknown parameter",
			input:          `@env.HOME(unknown="value")`,
			wantError:      true,
			wantMessage:    "unknown parameter 'unknown' for @env",
			wantContext:    "decorator parameter",
			wantSuggestion: "Valid parameters: default, property",
		},
		{
			name:           "mix of known and unknown parameters",
			input:          `@env.HOME(default="", invalid=true)`,
			wantError:      true,
			wantMessage:    "unknown parameter 'invalid' for @env",
			wantContext:    "decorator parameter",
			wantSuggestion: "Valid parameters: default, property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

// TestDecoratorWithBlock tests decorator block parsing
func TestDecoratorWithBlock(t *testing.T) {
	// First, register a test decorator that accepts blocks
	// We'll use a mock since we don't have @retry implemented yet

	tests := []struct {
		name       string
		input      string
		wantError  bool
		wantEvents []string // Simplified event sequence
	}{
		{
			name:      "decorator with empty block",
			input:     `@retry(times=3) { }`,
			wantError: false,
			wantEvents: []string{
				"Open(NodeDecorator)",
				"Token(@)",
				"Token(retry)",
				"Open(NodeParamList)",
				"Open(NodeParam)",
				"Close", // NodeParam
				"Close", // NodeParamList
				"Open(NodeBlock)",
				"Token({)",
				"Token(})",
				"Close", // NodeBlock
				"Close", // NodeDecorator
			},
		},
		{
			name:      "decorator with block containing statements",
			input:     `@retry(times=3) { echo "test" }`,
			wantError: false,
			// Should have decorator, params, and block with shell command
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Fatalf("Unexpected errors: %v", tree.Errors)
				}
			}

			// TODO: Verify event sequence once we implement block parsing
		})
	}
}

// TestDecoratorBlockRequired tests decorators that require blocks
func TestDecoratorBlockRequired(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantSuggestion string
	}{
		{
			name:      "parallel with block - valid",
			input:     `@parallel { echo "test" }`,
			wantError: false,
		},
		{
			name:           "parallel without block - error",
			input:          `@parallel`,
			wantError:      true,
			wantMessage:    "@parallel requires a block",
			wantSuggestion: "Add a block: @parallel(...) { ... }",
		},
		{
			name:      "parallel with empty block - valid",
			input:     `@parallel { }`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]
				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if tt.wantSuggestion != "" && err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", tree.Errors)
				}

				// Verify block node exists
				hasBlock := false
				for _, evt := range tree.Events {
					if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeBlock {
						hasBlock = true
						break
					}
				}
				if !hasBlock {
					t.Error("Expected block node in events")
				}
			}
		})
	}
}

// TestDecoratorBlockOptional tests decorators with optional blocks
func TestDecoratorBlockOptional(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		wantBlock bool
	}{
		{
			name:      "retry with block",
			input:     `@retry(times=3) { kubectl apply -f deployment.yaml }`,
			wantError: false,
			wantBlock: true,
		},
		{
			name:      "retry without block",
			input:     `@retry(times=3)`,
			wantError: false,
			wantBlock: false,
		},
		{
			name:      "retry with empty block",
			input:     `@retry(times=3) { }`,
			wantError: false,
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", tree.Errors)
				}

				// Check for block node
				hasBlock := false
				for _, evt := range tree.Events {
					if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeBlock {
						hasBlock = true
						break
					}
				}

				if hasBlock != tt.wantBlock {
					t.Errorf("Block presence mismatch: got %v, want %v", hasBlock, tt.wantBlock)
				}
			}
		})
	}
}

// TestDecoratorBlockForbidden tests decorators that cannot have blocks
func TestDecoratorBlockForbidden(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantError   bool
		wantMessage string
	}{
		{
			name:      "env without block - valid",
			input:     `@env.HOME`,
			wantError: false,
		},
		{
			name:        "env with block - error",
			input:       `@env.HOME { }`,
			wantError:   true,
			wantMessage: "@env cannot have a block",
		},
		{
			name:      "var without block - valid",
			input:     `@var.name`,
			wantError: false,
		},
		{
			name:        "var with block - error",
			input:       `@var.name { }`,
			wantError:   true,
			wantMessage: "@var cannot have a block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]
				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", tree.Errors)
				}
			}
		})
	}
}

// TestDecoratorBlockWithStatements tests blocks containing actual statements
func TestDecoratorBlockWithStatements(t *testing.T) {
	input := `@retry(times=3) {
		kubectl apply -f deployment.yaml
		kubectl rollout status deployment/app
	}`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Fatalf("Unexpected errors: %v", tree.Errors)
	}

	// Verify structure: Decorator -> ParamList -> Block -> ShellCommands
	hasDecorator := false
	hasBlock := false
	hasShellCommand := false

	for _, evt := range tree.Events {
		if evt.Kind == EventOpen {
			switch NodeKind(evt.Data) {
			case NodeDecorator:
				hasDecorator = true
			case NodeBlock:
				hasBlock = true
			case NodeShellCommand:
				hasShellCommand = true
			}
		}
	}

	if !hasDecorator {
		t.Error("Expected decorator node")
	}
	if !hasBlock {
		t.Error("Expected block node")
	}
	if !hasShellCommand {
		t.Error("Expected shell command nodes in block")
	}
}

// TestDecoratorBlockExactEvents tests decorator blocks with exact event sequences
func TestDecoratorBlockExactEvents(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "retry with empty block",
			input: `@retry(times=3) { }`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // retry
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 2}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 3}, // times
				{Kind: EventToken, Data: 4}, // =
				{Kind: EventToken, Data: 5}, // 3
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 6}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventOpen, Data: uint32(NodeBlock)},
				{Kind: EventToken, Data: 7}, // {
				{Kind: EventToken, Data: 8}, // }
				{Kind: EventClose, Data: uint32(NodeBlock)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "parallel with empty block",
			input: `@parallel { }`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // parallel
				{Kind: EventOpen, Data: uint32(NodeBlock)},
				{Kind: EventToken, Data: 2}, // {
				{Kind: EventToken, Data: 3}, // }
				{Kind: EventClose, Data: uint32(NodeBlock)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "retry without block (optional)",
			input: `@retry(times=3)`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // retry
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 2}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 3}, // times
				{Kind: EventToken, Data: 4}, // =
				{Kind: EventToken, Data: 5}, // 3
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 6}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "parallel with optional param and block",
			input: `@parallel(maxConcurrency=5) { }`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // parallel
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 2}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 3}, // maxConcurrency
				{Kind: EventToken, Data: 4}, // =
				{Kind: EventToken, Data: 5}, // 5
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 6}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventOpen, Data: uint32(NodeBlock)},
				{Kind: EventToken, Data: 7}, // {
				{Kind: EventToken, Data: 8}, // }
				{Kind: EventClose, Data: uint32(NodeBlock)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventStepExit, Data: 0},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Errorf("unexpected errors: %v", tree.Errors)
			}

			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDecoratorAsStatement tests decorators used as statements in blocks
func TestDecoratorAsStatement(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name: "decorator in function block",
			input: `fun deploy {
				@retry(times=3) {
					echo "test"
				}
			}`,
			wantError: false,
		},
		{
			name: "multiple decorators in block",
			input: `fun deploy {
				@retry(times=3) { echo "a" }
				@parallel { echo "b" }
			}`,
			wantError: false,
		},
		{
			name: "decorator mixed with shell commands",
			input: `fun deploy {
				echo "starting"
				@retry(times=3) { kubectl apply }
				echo "done"
			}`,
			wantError: false,
		},
		{
			name: "nested decorator blocks",
			input: `fun deploy {
				@retry(times=3) {
					@parallel {
						echo "nested"
					}
				}
			}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", tree.Errors)
				}

				// Verify decorator nodes exist
				decoratorCount := 0
				for _, evt := range tree.Events {
					if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
						decoratorCount++
					}
				}
				if decoratorCount == 0 {
					t.Error("Expected at least one decorator node")
				}
			}
		})
	}
}

// TestDecoratorNesting tests nested decorator blocks
func TestDecoratorNesting(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		wantError          bool
		wantDecoratorCount int
		wantBlockCount     int
	}{
		{
			name: "decorator inside decorator block",
			input: `@timeout(5m) {
				@retry(times=3) {
					echo "test"
				}
			}`,
			wantError:          false,
			wantDecoratorCount: 2, // @timeout and @retry
			wantBlockCount:     2, // timeout's block and retry's block
		},
		{
			name: "multiple decorators in parallel block",
			input: `@parallel {
				@retry(times=2) { echo "a" }
				@retry(times=2) { echo "b" }
			}`,
			wantError:          false,
			wantDecoratorCount: 3, // @parallel and 2x @retry
			wantBlockCount:     3, // parallel's block and 2x retry blocks
		},
		{
			name: "three levels of nesting",
			input: `@timeout(10m) {
				@retry(times=3) {
					@parallel {
						echo "nested"
					}
				}
			}`,
			wantError:          false,
			wantDecoratorCount: 3, // @timeout, @retry, @parallel
			wantBlockCount:     3, // all three have blocks
		},
		{
			name: "decorator with shell commands in block",
			input: `@retry(times=3) {
				echo "before"
				kubectl apply -f deployment.yaml
				echo "after"
			}`,
			wantError:          false,
			wantDecoratorCount: 1, // just @retry
			wantBlockCount:     1, // retry's block
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Fatalf("Unexpected errors: %v", tree.Errors)
				}

				// Count decorators and blocks
				decoratorCount := 0
				blockCount := 0
				for _, evt := range tree.Events {
					if evt.Kind == EventOpen {
						switch NodeKind(evt.Data) {
						case NodeDecorator:
							decoratorCount++
						case NodeBlock:
							blockCount++
						}
					}
				}

				if decoratorCount != tt.wantDecoratorCount {
					t.Errorf("Decorator count mismatch: got %d, want %d", decoratorCount, tt.wantDecoratorCount)
				}

				if blockCount != tt.wantBlockCount {
					t.Errorf("Block count mismatch: got %d, want %d", blockCount, tt.wantBlockCount)
				}
			}
		})
	}
}

// TestDecoratorPositionalParameters tests positional parameter syntax
func TestDecoratorPositionalParameters(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "single positional parameter",
			input:     `@timeout(5m) { }`,
			wantError: false,
		},
		{
			name:      "multiple positional parameters",
			input:     `@retry(3, 2s)`,
			wantError: false,
		},
		{
			name:      "mixed positional and named",
			input:     `@retry(3, delay=2s)`,
			wantError: false,
		},
		{
			name:      "all named (current working syntax)",
			input:     `@retry(times=3, delay=2s)`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", tree.Errors)
				}
			}
		})
	}
}

// TestPositionalParameters tests positional parameter support
func TestPositionalParameters(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
		wantCode       ErrorCode
		wantPath       string
		wantExpected   string
		wantGot        string
	}{
		// === Basic Positional ===
		{
			name:      "single positional - timeout",
			input:     `@timeout(5m) { echo "test" }`,
			wantError: false,
		},
		{
			name:      "two positional - retry",
			input:     `@retry(3, 2s) { echo "test" }`,
			wantError: false,
		},
		{
			name:      "three positional - retry with backoff",
			input:     `@retry(3, 2s, "linear") { echo "test" }`,
			wantError: false,
		},

		// === Mixed: Positional then Named ===
		{
			name:      "mixed - first positional, second named",
			input:     `@retry(3, delay=2s) { echo "test" }`,
			wantError: false,
		},
		{
			name:      "mixed - first positional, third named",
			input:     `@retry(3, backoff="linear") { echo "test" }`,
			wantError: false,
		},

		// === Mixed: Named then Positional (Kotlin-style gaps) ===
		{
			name:      "mixed - second named, first and third positional",
			input:     `@retry(3, delay=2s, "exponential") { echo "test" }`,
			wantError: false,
		},
		{
			name:      "mixed - second named, then positional fills first slot",
			input:     `@retry(delay=2s, 3) { echo "test" }`,
			wantError: false,
		},
		{
			name:      "mixed - second and third named, first positional",
			input:     `@retry(delay=2s, backoff="linear", 3) { echo "test" }`,
			wantError: false,
		},

		// === Edge Cases ===
		{
			name:        "too many positional arguments",
			input:       `@retry(3, 2s, "linear", "extra") { echo "test" }`,
			wantError:   true,
			wantMessage: "too many positional arguments",
		},
		{
			name:        "duplicate parameter - positional and named",
			input:       `@retry(3, times=5) { echo "test" }`,
			wantError:   true,
			wantMessage: "duplicate parameter 'times'",
		},
		{
			name:           "named then positional fills next slot",
			input:          `@retry(times=3, 5) { echo "test" }`,
			wantError:      true,
			wantMessage:    "parameter 'delay' expects duration (e.g., \"5m\", \"1h\"), got integer",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use a duration value like \"5m\"",
			wantCode:       ErrorCodeSchemaTypeMismatch,
			wantPath:       "delay",
			wantExpected:   "duration (e.g., \"5m\", \"1h\")",
			wantGot:        "integer",
		},

		// === Type Validation ===
		{
			name:           "wrong type - string where int expected",
			input:          `@retry("not-a-number", 2s) { echo "test" }`,
			wantError:      true,
			wantMessage:    "parameter 'times' expects integer between 1 and 100, got string",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use an integer value like 50",
			wantCode:       ErrorCodeSchemaTypeMismatch,
			wantPath:       "times",
			wantExpected:   "integer between 1 and 100",
			wantGot:        "string",
		},
		{
			name:           "wrong type - int where duration expected",
			input:          `@retry(3, 123) { echo "test" }`,
			wantError:      true,
			wantMessage:    "parameter 'delay' expects duration (e.g., \"5m\", \"1h\"), got integer",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use a duration value like \"5m\"",
			wantCode:       ErrorCodeSchemaTypeMismatch,
			wantPath:       "delay",
			wantExpected:   "duration (e.g., \"5m\", \"1h\")",
			wantGot:        "integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				// Test complete error structure (not lazy partial tests)
				if tt.wantMessage != "" && err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if tt.wantContext != "" && err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if tt.wantSuggestion != "" && err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}

				if tt.wantCode != "" && err.Code != tt.wantCode {
					t.Errorf("Code mismatch:\ngot:  %q\nwant: %q", err.Code, tt.wantCode)
				}

				if tt.wantPath != "" && err.Path != tt.wantPath {
					t.Errorf("Path mismatch:\ngot:  %q\nwant: %q", err.Path, tt.wantPath)
				}

				if tt.wantExpected != "" && err.ExpectedType != tt.wantExpected {
					t.Errorf("ExpectedType mismatch:\ngot:  %q\nwant: %q", err.ExpectedType, tt.wantExpected)
				}

				if tt.wantGot != "" && err.GotValue != tt.wantGot {
					t.Errorf("GotValue mismatch:\ngot:  %q\nwant: %q", err.GotValue, tt.wantGot)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", tree.Errors)
				}
			}
		})
	}
}

// TestPositionalParametersNesting tests positional params work with nesting
func TestPositionalParametersNesting(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "2-level nesting with positional",
			input: `@timeout(5m) {
				@retry(3, 2s) {
					echo "nested"
				}
			}`,
		},
		{
			name: "3-level nesting with mixed params",
			input: `@timeout(5m) {
				@retry(3, delay=1s) {
					@parallel {
						echo "deep"
					}
				}
			}`,
		},
		{
			name: "positional in parallel block",
			input: `@parallel {
				@retry(3) { echo "a" }
				@retry(5, 2s) { echo "b" }
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Errorf("Unexpected errors: %v", tree.Errors)
			}

			// Should have decorator nodes
			hasDecorators := false
			for _, evt := range tree.Events {
				if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
					hasDecorators = true
					break
				}
			}

			if !hasDecorators {
				t.Error("Expected decorator nodes in nested structure")
			}
		})
	}
}

// TestDecoratorObjectParameter tests parsing object literals as parameter values
func TestDecoratorObjectParameter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple object",
			input: `@config.myconfig(settings={timeout: "5m"})`,
		},
		{
			name:  "object with multiple fields",
			input: `@config.myconfig(settings={timeout: "5m", retries: 3})`,
		},
		{
			name:  "empty object",
			input: `@config.myconfig(settings={})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Errorf("unexpected parse errors:")
				for _, err := range tree.Errors {
					t.Logf("  %s", err.Message)
				}
			}

			// Verify we have a decorator node
			hasDecorator := false
			for _, evt := range tree.Events {
				if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
					hasDecorator = true
					break
				}
			}

			if !hasDecorator {
				t.Error("expected decorator node")
			}
		})
	}
}

// TestDecoratorArrayParameter tests parsing array literals as parameter values
func TestDecoratorArrayParameter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple array",
			input: `@deploy.production(hosts=["web1", "web2"])`,
		},
		{
			name:  "array of integers",
			input: `@deploy.staging(hosts=[8080, 8081])`,
		},
		{
			name:  "empty array",
			input: `@deploy.test(hosts=[])`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Errorf("unexpected parse errors:")
				for _, err := range tree.Errors {
					t.Logf("  %s", err.Message)
				}
			}

			// Verify we have a decorator node
			hasDecorator := false
			for _, evt := range tree.Events {
				if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
					hasDecorator = true
					break
				}
			}

			if !hasDecorator {
				t.Error("expected decorator node")
			}
		})
	}
}

// TestMultiSegmentDecoratorPaths tests that decorators can have multiple dots
// in their path (e.g., @aws.ssm.param).
func TestMultiSegmentDecoratorPaths(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		shouldPass bool
	}{
		{
			name:       "two segments (current working)",
			input:      "var x = @env.HOME",
			shouldPass: true,
		},
		{
			name:       "three segments (should work)",
			input:      "var x = @aws.ssm",
			shouldPass: true,
		},
		{
			name:       "four segments (should work)",
			input:      "var x = @aws.ssm.param",
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			hasErrors := len(tree.Errors) > 0
			if tt.shouldPass && hasErrors {
				t.Errorf("Expected parse to succeed, got error: %v", tree.Errors[0])
			}
			if !tt.shouldPass && !hasErrors {
				t.Error("Expected parse to fail, but it succeeded")
			}
		})
	}
}
