package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestSimpleShellCommand tests parsing of basic shell commands
func TestSimpleShellCommand(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple echo command",
			input: `echo "hello"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // "hello"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "command with flag argument",
			input: `kubectl apply -f deployment.yaml`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // kubectl
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // apply
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 2}, // - (MINUS)
				{Kind: EventToken, Data: 3}, // f
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 4}, // deployment
				{Kind: EventToken, Data: 5}, // . (DOT)
				{Kind: EventToken, Data: 6}, // yaml
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "npm run command",
			input: `npm run build`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // npm
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // run
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 2}, // build
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
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

// TestShellCommandWithOperators tests that shell commands are split by operators
func TestShellCommandWithOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "AND operator",
			input: `echo "first" && echo "second"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				// First command
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // "first"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				// Operator
				{Kind: EventToken, Data: 2}, // &&
				// Second command
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 3}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 4}, // "second"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "OR operator",
			input: `echo "try" || echo "fallback"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // "try"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventToken, Data: 2}, // ||
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 3}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 4}, // "fallback"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "PIPE operator",
			input: `cat file.txt | grep pattern`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // cat
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // file
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // txt
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventToken, Data: 4}, // |
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 5}, // grep
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 6}, // pattern
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "chained operators",
			input: `npm run build && npm test || echo "failed"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				// npm run build
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 0}, // npm
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 1}, // run
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 2}, // build
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventToken, Data: 3}, // &&
				// npm test
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 4}, // npm
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 5}, // test
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventToken, Data: 6}, // ||
				// echo "failed"
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 7}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 8}, // "failed"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
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

// TestShellCommandSplitByNewlines tests that newlines split shell commands
func TestShellCommandSplitByNewlines(t *testing.T) {
	input := `echo "first"
echo "second"`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected errors: %v", tree.Errors)
	}

	expected := []Event{
		{Kind: EventOpen, Data: uint32(NodeSource)},
		// First command
		{Kind: EventOpen, Data: uint32(NodeShellCommand)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 0}, // echo
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 1}, // "first"
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventClose, Data: uint32(NodeShellCommand)},
		// Second command (token 2 is NEWLINE, skipped)
		{Kind: EventOpen, Data: uint32(NodeShellCommand)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 3}, // echo
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 4}, // "second"
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventClose, Data: uint32(NodeShellCommand)},
		{Kind: EventClose, Data: uint32(NodeSource)},
	}

	if diff := cmp.Diff(expected, tree.Events); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

// TestShellCommandInFunctionBody tests shell commands inside function blocks
func TestShellCommandInFunctionBody(t *testing.T) {
	input := `fun deploy {
    kubectl apply -f k8s/
}`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected errors: %v", tree.Errors)
	}

	expected := []Event{
		{Kind: EventOpen, Data: uint32(NodeSource)},
		{Kind: EventOpen, Data: uint32(NodeFunction)},
		{Kind: EventToken, Data: 0}, // fun
		{Kind: EventToken, Data: 1}, // deploy
		{Kind: EventOpen, Data: uint32(NodeBlock)},
		{Kind: EventToken, Data: 2}, // {
		// Token 3 is NEWLINE (skipped in statement parsing)
		// Shell command inside block
		{Kind: EventOpen, Data: uint32(NodeShellCommand)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 4}, // kubectl
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 5}, // apply
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 6}, // -
		{Kind: EventToken, Data: 7}, // f
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 8}, // k8s
		{Kind: EventToken, Data: 9}, // / (DIVIDE)
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventClose, Data: uint32(NodeShellCommand)},
		// Token 10 is NEWLINE
		{Kind: EventToken, Data: 11}, // }
		{Kind: EventClose, Data: uint32(NodeBlock)},
		{Kind: EventClose, Data: uint32(NodeFunction)},
		{Kind: EventClose, Data: uint32(NodeSource)},
	}

	if diff := cmp.Diff(expected, tree.Events); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

// TestShellCommandAfterEquals tests shell commands in single-expression functions
func TestShellCommandAfterEquals(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple function with shell command",
			input: `fun hello = echo "Hello World!"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeFunction)},
				{Kind: EventToken, Data: 0}, // fun
				{Kind: EventToken, Data: 1}, // hello
				{Kind: EventToken, Data: 2}, // =
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 3}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 4}, // "Hello World!"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeFunction)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "function with chained shell commands",
			input: `fun test = echo "first" && echo "second"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeFunction)},
				{Kind: EventToken, Data: 0}, // fun
				{Kind: EventToken, Data: 1}, // test
				{Kind: EventToken, Data: 2}, // =
				// First command
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 3}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 4}, // "first"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventToken, Data: 5}, // &&
				// Second command
				{Kind: EventOpen, Data: uint32(NodeShellCommand)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 6}, // echo
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventOpen, Data: uint32(NodeShellArg)},
				{Kind: EventToken, Data: 7}, // "second"
				{Kind: EventClose, Data: uint32(NodeShellArg)},
				{Kind: EventClose, Data: uint32(NodeShellCommand)},
				{Kind: EventClose, Data: uint32(NodeFunction)},
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

// TestMixedStatementsAndShellCommands tests var declarations mixed with shell commands
func TestMixedStatementsAndShellCommands(t *testing.T) {
	input := `var ENV = "prod"
echo "Deploying"
kubectl apply -f k8s/`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected errors: %v", tree.Errors)
	}

	expected := []Event{
		{Kind: EventOpen, Data: uint32(NodeSource)},
		// Var declaration
		{Kind: EventOpen, Data: uint32(NodeVarDecl)},
		{Kind: EventToken, Data: 0}, // var
		{Kind: EventToken, Data: 1}, // ENV
		{Kind: EventToken, Data: 2}, // =
		{Kind: EventOpen, Data: uint32(NodeLiteral)},
		{Kind: EventToken, Data: 3}, // "prod"
		{Kind: EventClose, Data: uint32(NodeLiteral)},
		{Kind: EventClose, Data: uint32(NodeVarDecl)},
		// First shell command
		{Kind: EventOpen, Data: uint32(NodeShellCommand)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 5}, // echo (token 4 is NEWLINE)
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 6}, // "Deploying"
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventClose, Data: uint32(NodeShellCommand)},
		// Second shell command
		{Kind: EventOpen, Data: uint32(NodeShellCommand)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 8}, // kubectl (token 7 is NEWLINE)
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 9}, // apply
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 10}, // -
		{Kind: EventToken, Data: 11}, // f
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventOpen, Data: uint32(NodeShellArg)},
		{Kind: EventToken, Data: 12}, // k8s
		{Kind: EventToken, Data: 13}, // /
		{Kind: EventClose, Data: uint32(NodeShellArg)},
		{Kind: EventClose, Data: uint32(NodeShellCommand)},
		{Kind: EventClose, Data: uint32(NodeSource)},
	}

	if diff := cmp.Diff(expected, tree.Events); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}
