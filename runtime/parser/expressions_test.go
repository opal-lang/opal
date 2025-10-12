package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestUnaryExpression tests unary operators (! and -)
func TestUnaryExpression(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "negation with integer in var",
			input: "fun test { var x = -5 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // x
				{EventToken, 5}, // =
				{EventOpen, uint32(NodeUnaryExpr)},
				{EventToken, 6}, // -
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 7}, // 5
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeUnaryExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 8}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "unary minus with binary addition",
			input: "fun test { var result = -x + y }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // result
				{EventToken, 5}, // =
				{EventOpen, uint32(NodeUnaryExpr)},
				{EventToken, 6}, // -
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 7}, // x
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodeUnaryExpr)},
				{EventOpen, uint32(NodeBinaryExpr)},
				{EventToken, 8}, // +
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 9}, // y
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodeBinaryExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 10}, // }
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

// TestPrefixIncrementDecrement tests prefix ++ and -- operators
func TestPrefixIncrementDecrement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "prefix increment with identifier",
			input: "fun test { var x = ++counter }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // x
				{EventToken, 5}, // =
				{EventOpen, uint32(NodePrefixExpr)},
				{EventToken, 6}, // ++
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 7}, // counter
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodePrefixExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 8}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "prefix decrement with decorator",
			input: "fun test { var y = --@var.count }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // y
				{EventToken, 5}, // =
				{EventOpen, uint32(NodePrefixExpr)},
				{EventToken, 6}, // --
				{EventOpen, uint32(NodeDecorator)},
				{EventToken, 7},  // @
				{EventToken, 8},  // var
				{EventToken, 9},  // .
				{EventToken, 10}, // count
				{EventClose, uint32(NodeDecorator)},
				{EventClose, uint32(NodePrefixExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 11}, // }
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

// TestPostfixIncrementDecrement tests postfix ++ and -- operators
func TestPostfixIncrementDecrement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "postfix increment with identifier",
			input: "fun test { var x = counter++ }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // x
				{EventToken, 5}, // =
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 6}, // counter
				{EventClose, uint32(NodeIdentifier)},
				{EventOpen, uint32(NodePostfixExpr)},
				{EventToken, 7}, // ++
				{EventClose, uint32(NodePostfixExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 8}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "postfix decrement with decorator",
			input: "fun test { var y = @var.count-- }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // y
				{EventToken, 5}, // =
				{EventOpen, uint32(NodeDecorator)},
				{EventToken, 6}, // @
				{EventToken, 7}, // var
				{EventToken, 8}, // .
				{EventToken, 9}, // count
				{EventClose, uint32(NodeDecorator)},
				{EventOpen, uint32(NodePostfixExpr)},
				{EventToken, 10}, // --
				{EventClose, uint32(NodePostfixExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 11}, // }
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

// TestIncrementDecrementPrecedence tests operator precedence
func TestIncrementDecrementPrecedence(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "postfix before addition",
			input: "fun test { var result = x++ + y }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // result
				{EventToken, 5}, // =
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 6}, // x
				{EventClose, uint32(NodeIdentifier)},
				{EventOpen, uint32(NodePostfixExpr)},
				{EventToken, 7}, // ++
				{EventClose, uint32(NodePostfixExpr)},
				{EventOpen, uint32(NodeBinaryExpr)},
				{EventToken, 8}, // +
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 9}, // y
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodeBinaryExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 10}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "prefix before addition",
			input: "fun test { var result = ++x + y }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeVarDecl)},
				{EventToken, 3}, // var
				{EventToken, 4}, // result
				{EventToken, 5}, // =
				{EventOpen, uint32(NodePrefixExpr)},
				{EventToken, 6}, // ++
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 7}, // x
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodePrefixExpr)},
				{EventOpen, uint32(NodeBinaryExpr)},
				{EventToken, 8}, // +
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 9}, // y
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodeBinaryExpr)},
				{EventClose, uint32(NodeVarDecl)},
				{EventToken, 10}, // }
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
