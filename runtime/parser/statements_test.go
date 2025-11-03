package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestSimpleVarDecl tests basic variable declarations
func TestSimpleVarDecl(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "var with integer literal",
			input: "var x = 42",
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeVarDecl)},
				{Kind: EventToken, Data: 0}, // VAR
				{Kind: EventToken, Data: 1}, // x
				{Kind: EventToken, Data: 2}, // =
				{Kind: EventOpen, Data: uint32(NodeLiteral)},
				{Kind: EventToken, Data: 3}, // 42
				{Kind: EventClose, Data: uint32(NodeLiteral)},
				{Kind: EventClose, Data: uint32(NodeVarDecl)},
				{Kind: EventStepExit, Data: 0}, // Step boundary
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "var with string literal",
			input: `var name = "alice"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeVarDecl)},
				{Kind: EventToken, Data: 0}, // VAR
				{Kind: EventToken, Data: 1}, // name
				{Kind: EventToken, Data: 2}, // =
				{Kind: EventOpen, Data: uint32(NodeLiteral)},
				{Kind: EventToken, Data: 3}, // "alice"
				{Kind: EventClose, Data: uint32(NodeLiteral)},
				{Kind: EventClose, Data: uint32(NodeVarDecl)},
				{Kind: EventStepExit, Data: 0}, // Step boundary
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "var with boolean literal",
			input: "var ready = true",
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventStepEnter, Data: 0}, // Step boundary
				{Kind: EventOpen, Data: uint32(NodeVarDecl)},
				{Kind: EventToken, Data: 0},                  // VAR
				{Kind: EventToken, Data: 1},                  // ready
				{Kind: EventToken, Data: 2},                  // =
				{Kind: EventOpen, Data: uint32(NodeLiteral)}, // Now correctly recognized as literal
				{Kind: EventToken, Data: 3},                  // true (lexed as BOOLEAN)
				{Kind: EventClose, Data: uint32(NodeLiteral)},
				{Kind: EventClose, Data: uint32(NodeVarDecl)},
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

			// Compare events
			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestAssignmentOperators tests assignment operators (+=, -=, *=, /=, %=)
func TestAssignmentOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "plus assign with integer",
			input: "fun test { total += 5 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // total
				{EventToken, 4}, // +=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 5
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventStepExit, 0}, // Step boundary
				{EventToken, 6},    // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "minus assign with decorator",
			input: "fun test { remaining -= @var.cost }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // remaining
				{EventToken, 4}, // -=
				{EventOpen, uint32(NodeDecorator)},
				{EventToken, 5}, // @
				{EventToken, 6}, // var
				{EventToken, 7}, // .
				{EventToken, 8}, // cost
				{EventClose, uint32(NodeDecorator)},
				{EventClose, uint32(NodeAssignment)},
				{EventStepExit, 0}, // Step boundary
				{EventToken, 9},    // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "multiply assign",
			input: "fun test { replicas *= 3 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // replicas
				{EventToken, 4}, // *=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 3
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventStepExit, 0}, // Step boundary
				{EventToken, 6},    // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "divide assign",
			input: "fun test { batch_size /= 2 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // batch_size
				{EventToken, 4}, // /=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 2
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventStepExit, 0}, // Step boundary
				{EventToken, 6},    // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "modulo assign",
			input: "fun test { index %= 10 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // index
				{EventToken, 4}, // %=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 10
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventStepExit, 0}, // Step boundary
				{EventToken, 6},    // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "assignment with expression",
			input: "fun test { total += x + y }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // total
				{EventToken, 4}, // +=
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 5}, // x
				{EventClose, uint32(NodeIdentifier)},
				{EventOpen, uint32(NodeBinaryExpr)},
				{EventToken, 6}, // +
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 7}, // y
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodeBinaryExpr)},
				{EventClose, uint32(NodeAssignment)},
				{EventStepExit, 0}, // Step boundary
				{EventToken, 8},    // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) != 0 {
				t.Errorf("Expected no errors, got: %v", tree.Errors)
			}

			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("Events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestObjectLiteralInVarDecl tests parsing object literals in variable declarations
func TestObjectLiteralInVarDecl(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple object",
			input: `var config = {timeout: "5m"}`,
		},
		{
			name:  "object with multiple fields",
			input: `var config = {timeout: "5m", retries: 3}`,
		},
		{
			name:  "empty object",
			input: `var config = {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) > 0 {
				t.Errorf("unexpected parse errors:")
				for _, err := range tree.Errors {
					t.Logf("  %s", err.Message)
				}
			}
		})
	}
}

// TestArrayLiteralInVarDecl tests parsing array literals in variable declarations
func TestArrayLiteralInVarDecl(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple array",
			input: `var hosts = ["web1", "web2"]`,
		},
		{
			name:  "array of integers",
			input: `var ports = [8080, 8081]`,
		},
		{
			name:  "empty array",
			input: `var items = []`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) > 0 {
				t.Errorf("unexpected parse errors:")
				for _, err := range tree.Errors {
					t.Logf("  %s", err.Message)
				}
			}
		})
	}
}
