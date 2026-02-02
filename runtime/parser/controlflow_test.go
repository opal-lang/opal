package parser

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIfStatement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple if with boolean",
			input: "fun test { if true { echo \"yes\" } }",
			events: []Event{
				{EventOpen, 0},      // Source
				{EventOpen, 1},      // Function
				{EventToken, 0},     // fun
				{EventToken, 1},     // test
				{EventOpen, 3},      // Block
				{EventToken, 2},     // {
				{EventOpen, 10},     // If
				{EventToken, 3},     // if
				{EventOpen, 13},     // Literal (condition wrapped by expression())
				{EventToken, 4},     // true
				{EventClose, 13},    // Literal
				{EventOpen, 3},      // Block
				{EventToken, 5},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 6},     // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 7},     // "yes"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 8},     // }
				{EventClose, 3},     // Block
				{EventClose, 10},    // If
				{EventToken, 9},     // }
				{EventClose, 3},     // Block
				{EventClose, 1},     // Function
				{EventClose, 0},     // Source
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

func TestIfElseStatement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "if-else",
			input: "fun test { if true { echo \"yes\" } else { echo \"no\" } }",
			events: []Event{
				{EventOpen, 0},      // Source
				{EventOpen, 1},      // Function
				{EventToken, 0},     // fun
				{EventToken, 1},     // test
				{EventOpen, 3},      // Block
				{EventToken, 2},     // {
				{EventOpen, 10},     // If
				{EventToken, 3},     // if
				{EventOpen, 13},     // Literal (condition)
				{EventToken, 4},     // true
				{EventClose, 13},    // Literal
				{EventOpen, 3},      // Block
				{EventToken, 5},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 6},     // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 7},     // "yes"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 8},     // }
				{EventClose, 3},     // Block
				{EventOpen, 11},     // Else
				{EventToken, 9},     // else
				{EventOpen, 3},      // Block
				{EventToken, 10},    // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 11},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 12},    // "no"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 13},    // }
				{EventClose, 3},     // Block
				{EventClose, 11},    // Else
				{EventClose, 10},    // If
				{EventToken, 14},    // }
				{EventClose, 3},     // Block
				{EventClose, 1},     // Function
				{EventClose, 0},     // Source
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

func TestIfElseIfChain(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "if-else-if-else",
			input: "fun test { if true { echo \"a\" } else if false { echo \"b\" } else { echo \"c\" } }",
			events: []Event{
				{EventOpen, 0},      // Source
				{EventOpen, 1},      // Function
				{EventToken, 0},     // fun
				{EventToken, 1},     // test
				{EventOpen, 3},      // Block
				{EventToken, 2},     // {
				{EventOpen, 10},     // If
				{EventToken, 3},     // if
				{EventOpen, 13},     // Literal (condition)
				{EventToken, 4},     // true
				{EventClose, 13},    // Literal
				{EventOpen, 3},      // Block
				{EventToken, 5},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 6},     // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 7},     // "a"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 8},     // }
				{EventClose, 3},     // Block
				{EventOpen, 11},     // Else
				{EventToken, 9},     // else
				{EventOpen, 10},     // If (nested)
				{EventToken, 10},    // if
				{EventOpen, 13},     // Literal (condition)
				{EventToken, 11},    // false
				{EventClose, 13},    // Literal
				{EventOpen, 3},      // Block
				{EventToken, 12},    // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 13},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 14},    // "b"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 15},    // }
				{EventClose, 3},     // Block
				{EventOpen, 11},     // Else
				{EventToken, 16},    // else
				{EventOpen, 3},      // Block
				{EventToken, 17},    // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 18},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 19},    // "c"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 20},    // }
				{EventClose, 3},     // Block
				{EventClose, 11},    // Else
				{EventClose, 10},    // If (nested)
				{EventClose, 11},    // Else
				{EventClose, 10},    // If
				{EventToken, 21},    // }
				{EventClose, 3},     // Block
				{EventClose, 1},     // Function
				{EventClose, 0},     // Source
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

func TestIfAtTopLevel(t *testing.T) {
	// If statements ARE allowed at top level (script mode)
	input := "if true { echo \"hello\" }"

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors for top-level if (script mode), got: %v", tree.Errors)
	}

	// Should have events for the if statement
	if len(tree.Events) == 0 {
		t.Error("Expected events for if statement, got none")
	}
}

func TestFunInsideControlFlow(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "fun inside if block",
			input: "fun test { if true { fun helper() { } } }",
		},
		{
			name:  "fun inside else block",
			input: "fun test { if true { } else { fun helper() { } } }",
		},
		{
			name:  "fun inside for loop",
			input: "fun test { for item in items { fun helper() { } } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Fatal("Expected error for fun inside control flow, got none")
			}

			err := tree.Errors[0]
			if err.Message != "function declarations must be at top level" {
				t.Errorf("Expected error about fun at top level, got: %s", err.Message)
			}
		})
	}
}

func TestElseWithoutIf(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "else at start of block",
			input: "fun test { else { echo \"hello\" } }",
		},
		{
			name:  "else after shell command",
			input: "fun test { echo \"hello\" \n else { echo \"world\" } }",
		},
		{
			name:  "else after var declaration",
			input: "fun test { var x = 5 \n else { echo \"world\" } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Fatal("Expected error for else without if, got none")
			}

			err := tree.Errors[0]
			if err.Message != "else without matching if" {
				t.Errorf("Expected error 'else without matching if', got: %s", err.Message)
			}
			if err.Context != "statement" {
				t.Errorf("Expected context 'statement', got: %s", err.Context)
			}
		})
	}
}

func TestIfStatementErrorRecovery(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		minErrorCount int
		containsError string
	}{
		{
			name:          "missing condition - block immediately after if",
			input:         "fun test { if { echo \"hello\" } }",
			minErrorCount: 1,
			containsError: "missing condition after 'if'",
		},
		{
			name:          "missing block after condition",
			input:         "fun test { if true }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
		{
			name:          "missing block after else",
			input:         "fun test { if true { } else }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
		{
			name:          "nested if missing condition",
			input:         "fun test { if true { if { } } }",
			minErrorCount: 1,
			containsError: "missing condition after 'if'",
		},
		{
			name:          "else if missing condition",
			input:         "fun test { if true { } else if { } }",
			minErrorCount: 1,
			containsError: "missing condition after 'if'",
		},
		{
			name:          "orphaned else with type error",
			input:         "fun test { else if 42 { } }",
			minErrorCount: 1, // Only orphaned else error (type checking moved to plan time)
			containsError: "else without matching if",
		},
		{
			name:          "missing block after string condition",
			input:         "fun test { if \"string\" }",
			minErrorCount: 1, // Missing block error (type checking moved to plan time)
			containsError: "missing '{'",
		},
		{
			name:          "multiple if statements valid",
			input:         "fun test { if true { } if 42 { } }",
			minErrorCount: 0, // No errors (type checking moved to plan time)
			containsError: "",
		},
		{
			name:          "if with statement instead of block",
			input:         "fun test { if true echo \"hi\" }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
		{
			name:          "else with statement instead of block",
			input:         "fun test { if true { } else echo \"hi\" }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) < tt.minErrorCount {
				t.Errorf("Expected at least %d error(s), got %d: %v",
					tt.minErrorCount, len(tree.Errors), tree.Errors)
				return
			}

			// Check that at least one error contains the expected message (if we expect errors)
			if tt.minErrorCount > 0 && tt.containsError != "" {
				found := false
				for _, err := range tree.Errors {
					if containsSubstring(err.Message, tt.containsError) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected error containing '%s', got errors: %v",
						tt.containsError, tree.Errors)
				}
			}

			// Verify parser didn't panic and produced some events
			if len(tree.Events) == 0 {
				t.Error("Parser produced no events (possible panic or early exit)")
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIfConditionTypeChecking(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "string literal condition",
			input:     `fun test { if "hello" { echo "yes" } }`,
			expectErr: false, // String literals are valid expressions (evaluated at plan time)
		},
		{
			name:      "integer literal condition",
			input:     `fun test { if 42 { echo "yes" } }`,
			expectErr: false, // Integer literals are valid expressions (evaluated at plan time)
		},
		{
			name:      "boolean true",
			input:     `fun test { if true { echo "yes" } }`,
			expectErr: false,
		},
		{
			name:      "boolean false",
			input:     `fun test { if false { echo "yes" } }`,
			expectErr: false,
		},
		{
			name:      "identifier (could be boolean)",
			input:     `fun test { if isReady { echo "yes" } }`,
			expectErr: false, // Identifiers are allowed (runtime check)
		},
		{
			name:      "decorator (could be boolean)",
			input:     `fun test { if @var.enabled { echo "yes" } }`,
			expectErr: false, // Decorators are allowed (runtime check)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if tt.expectErr {
				if len(tree.Errors) == 0 {
					t.Fatalf("Expected error for non-boolean condition, got none")
				}
				err := tree.Errors[0]
				if err.Message != tt.errMsg {
					t.Errorf("Expected error '%s', got: %s", tt.errMsg, err.Message)
				}
			} else {
				if len(tree.Errors) != 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

func TestForLoop(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple for loop",
			input: "fun test { for item in items { echo @var.item } }",
			events: []Event{
				{EventOpen, 0},      // Source
				{EventOpen, 1},      // Function
				{EventToken, 0},     // fun
				{EventToken, 1},     // test
				{EventOpen, 3},      // Block
				{EventToken, 2},     // {
				{EventOpen, 12},     // For (NodeFor = 12)
				{EventToken, 3},     // for
				{EventToken, 4},     // item (loop variable)
				{EventToken, 5},     // in
				{EventToken, 6},     // items (collection)
				{EventOpen, 3},      // Block
				{EventToken, 7},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 8},     // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventOpen, 18},     // Decorator (@var.item)
				{EventToken, 9},     // @
				{EventToken, 10},    // var
				{EventToken, 11},    // .
				{EventToken, 12},    // item
				{EventClose, 18},    // Decorator
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 13},    // }
				{EventClose, 3},     // Block
				{EventClose, 12},    // For
				{EventToken, 14},    // }
				{EventClose, 3},     // Block
				{EventClose, 1},     // Function
				{EventClose, 0},     // Source
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

func TestForLoopWithDecorator(t *testing.T) {
	input := "fun test { for item in @var.items { echo @var.item } }"
	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors, got: %v", tree.Errors)
	}

	// Verify NodeFor and NodeDecorator are present
	hasFor := false
	hasDecorator := false
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen && evt.Data == uint32(NodeFor) {
			hasFor = true
		}
		if evt.Kind == EventOpen && evt.Data == uint32(NodeDecorator) {
			hasDecorator = true
		}
	}

	if !hasFor {
		t.Error("Expected NodeFor in events")
	}
	if !hasDecorator {
		t.Error("Expected NodeDecorator in events")
	}
}

func TestForLoopErrorRecovery(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr ParseError
	}{
		{
			name:  "missing loop variable",
			input: "fun test { for in items { } }",
			expectedErr: ParseError{
				Message:    "missing loop variable after 'for'",
				Context:    "for loop",
				Suggestion: "Add a variable name to hold each item",
				Example:    "for item in items { ... }",
				Note:       "for loops unroll at plan-time; the loop variable is resolved during planning",
			},
		},
		{
			name:  "missing 'in' keyword",
			input: "fun test { for item items { } }",
			expectedErr: ParseError{
				Message:    "missing 'in' keyword in for loop",
				Context:    "for loop",
				Suggestion: "Add 'in' between loop variable and collection",
				Example:    "for item in items { ... }",
			},
		},
		{
			name:  "missing collection",
			input: "fun test { for item in { } }",
			expectedErr: ParseError{
				Message:    "missing collection expression in for loop",
				Context:    "for loop",
				Suggestion: "Provide a collection to iterate over",
				Example:    "for item in items { ... } or for i in 1...10 { ... } or for x in [1, 2, 3] { ... }",
				Note:       "collection must resolve at plan-time to a list of concrete values",
			},
		},
		{
			name:  "missing block",
			input: "fun test { for item in items }",
			expectedErr: ParseError{
				Message:    "missing block after for loop header",
				Context:    "for loop body",
				Suggestion: "Add a block with the loop body",
				Example:    "for item in items { echo @var.item }",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Fatalf("Expected error, got none")
			}

			actual := tree.Errors[0]
			expected := tt.expectedErr

			// Compare only the fields we care about (ignore Position, Expected, Got)
			if actual.Message != expected.Message {
				t.Errorf("Message mismatch:\nwant: %s\ngot:  %s", expected.Message, actual.Message)
			}
			if actual.Context != expected.Context {
				t.Errorf("Context mismatch:\nwant: %s\ngot:  %s", expected.Context, actual.Context)
			}
			if actual.Suggestion != expected.Suggestion {
				t.Errorf("Suggestion mismatch:\nwant: %s\ngot:  %s", expected.Suggestion, actual.Suggestion)
			}
			if actual.Example != expected.Example {
				t.Errorf("Example mismatch:\nwant: %s\ngot:  %s", expected.Example, actual.Example)
			}
			if actual.Note != expected.Note {
				t.Errorf("Note mismatch:\nwant: %s\ngot:  %s", expected.Note, actual.Note)
			}
		})
	}
}

// TestTryCatch tests basic try-catch parsing
func TestTryCatch(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple try-catch",
			input: "fun test { try { echo \"attempting\" } catch { echo \"failed\" } }",
			events: []Event{
				{EventOpen, 0},      // Source
				{EventOpen, 1},      // Function
				{EventToken, 0},     // fun
				{EventToken, 1},     // test
				{EventOpen, 3},      // Block
				{EventToken, 2},     // {
				{EventOpen, 19},     // Try (NodeTry = 19)
				{EventToken, 3},     // try
				{EventOpen, 3},      // Block
				{EventToken, 4},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 5},     // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 6},     // "attempting"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 7},     // }
				{EventClose, 3},     // Block
				{EventOpen, 20},     // Catch (NodeCatch = 20)
				{EventToken, 8},     // catch
				{EventOpen, 3},      // Block
				{EventToken, 9},     // {
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 10},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 11},    // "failed"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventToken, 12},    // }
				{EventClose, 3},     // Block
				{EventClose, 20},    // Catch
				{EventClose, 19},    // Try
				{EventToken, 13},    // }
				{EventClose, 3},     // Block
				{EventClose, 1},     // Function
				{EventClose, 0},     // Source
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

// TestTryCatchCombinations tests all valid combinations of try/catch/finally
func TestTryCatchCombinations(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		hasTry     bool
		hasCatch   bool
		hasFinally bool
	}{
		{
			name:       "try-catch",
			input:      "fun test { try { echo \"a\" } catch { echo \"b\" } }",
			hasTry:     true,
			hasCatch:   true,
			hasFinally: false,
		},
		{
			name:       "try-finally",
			input:      "fun test { try { echo \"a\" } finally { echo \"c\" } }",
			hasTry:     true,
			hasCatch:   false,
			hasFinally: true,
		},
		{
			name:       "try-catch-finally",
			input:      "fun test { try { echo \"a\" } catch { echo \"b\" } finally { echo \"c\" } }",
			hasTry:     true,
			hasCatch:   true,
			hasFinally: true,
		},
		{
			name:       "try only (valid - catch/finally optional)",
			input:      "fun test { try { echo \"a\" } }",
			hasTry:     true,
			hasCatch:   false,
			hasFinally: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) != 0 {
				t.Errorf("Expected no errors, got: %v", tree.Errors)
			}

			// Check for presence of expected nodes
			foundTry := false
			foundCatch := false
			foundFinally := false

			for _, event := range tree.Events {
				if event.Kind == EventOpen {
					switch event.Data {
					case 19: // NodeTry
						foundTry = true
					case 20: // NodeCatch
						foundCatch = true
					case 21: // NodeFinally
						foundFinally = true
					}
				}
			}

			if foundTry != tt.hasTry {
				t.Errorf("Expected hasTry=%v, got %v", tt.hasTry, foundTry)
			}
			if foundCatch != tt.hasCatch {
				t.Errorf("Expected hasCatch=%v, got %v", tt.hasCatch, foundCatch)
			}
			if foundFinally != tt.hasFinally {
				t.Errorf("Expected hasFinally=%v, got %v", tt.hasFinally, foundFinally)
			}
		})
	}
}

// TestTryCatchErrorRecovery tests error handling for malformed try/catch
func TestTryCatchErrorRecovery(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantError     bool
		errorContains string
	}{
		{
			name:          "missing try block",
			input:         "fun test { try catch { echo \"b\" } }",
			wantError:     true,
			errorContains: "missing block after 'try'",
		},
		{
			name:          "missing catch block",
			input:         "fun test { try { echo \"a\" } catch }",
			wantError:     true,
			errorContains: "missing block after 'catch'",
		},
		{
			name:          "missing finally block",
			input:         "fun test { try { echo \"a\" } finally }",
			wantError:     true,
			errorContains: "missing block after 'finally'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Errorf("Expected error containing %q, got no errors", tt.errorContains)
					return
				}

				found := false
				for _, err := range tree.Errors {
					if strings.Contains(err.Message, tt.errorContains) {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Expected error containing %q, got errors: %v", tt.errorContains, tree.Errors)
				}
			} else {
				if len(tree.Errors) != 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

// TestFunInsideTryCatch tests that fun declarations are rejected inside try/catch/finally
func TestFunInsideTryCatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "fun inside try block",
			input: "fun test { try { fun helper { } } }",
		},
		{
			name:  "fun inside catch block",
			input: "fun test { try { echo \"a\" } catch { fun helper { } } }",
		},
		{
			name:  "fun inside finally block",
			input: "fun test { try { echo \"a\" } finally { fun helper { } } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Errorf("Expected error about fun at top level, got no errors")
				return
			}

			err := tree.Errors[0]
			if err.Message != "function declarations must be at top level" {
				t.Errorf("Expected error about fun at top level, got: %s", err.Message)
			}
		})
	}
}

// TestOrphanCatchFinally tests that catch/finally without try are rejected
func TestOrphanCatchFinally(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		errorContains string
	}{
		{
			name:          "orphan catch",
			input:         "fun test { catch { echo \"error\" } }",
			errorContains: "catch without try",
		},
		{
			name:          "orphan finally",
			input:         "fun test { finally { echo \"cleanup\" } }",
			errorContains: "finally without try",
		},
		{
			name:          "catch before try",
			input:         "fun test { catch { } try { } }",
			errorContains: "catch without try",
		},
		{
			name:          "finally before try",
			input:         "fun test { finally { } try { } }",
			errorContains: "finally without try",
		},
		{
			name:          "orphan catch at top level",
			input:         "catch { echo \"error\" }",
			errorContains: "catch without try",
		},
		{
			name:          "orphan finally at top level",
			input:         "finally { echo \"cleanup\" }",
			errorContains: "finally without try",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Errorf("Expected error containing %q, got no errors", tt.errorContains)
				return
			}

			found := false
			for _, err := range tree.Errors {
				if strings.Contains(err.Message, tt.errorContains) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error containing %q, got errors: %v", tt.errorContains, tree.Errors)
			}
		})
	}
}

// TestWhenStatement tests basic when pattern matching with string literals and else
func TestWhenStatement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple when with string patterns",
			input: `fun test { when @var.ENV { "production" -> echo "prod" else -> echo "other" } }`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 22},  // When (NodeWhen = 22)
				{EventToken, 3},  // when
				{EventOpen, 18},  // Decorator (NodeDecorator = 18)
				{EventToken, 4},  // @
				{EventToken, 5},  // var
				{EventToken, 6},  // .
				{EventToken, 7},  // ENV
				{EventClose, 18}, // Decorator
				{EventToken, 8},  // {

				// First arm: "production" -> echo "prod"
				{EventOpen, 23},     // WhenArm (NodeWhenArm = 23)
				{EventOpen, 24},     // PatternLiteral (NodePatternLiteral = 24)
				{EventToken, 9},     // "production"
				{EventClose, 24},    // PatternLiteral
				{EventToken, 10},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 11},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 12},    // "prod"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				// Second arm: else -> echo "other"
				{EventOpen, 23},     // WhenArm
				{EventOpen, 25},     // PatternElse (NodePatternElse = 25)
				{EventToken, 13},    // else
				{EventClose, 25},    // PatternElse
				{EventToken, 14},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 15},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 16},    // "other"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				{EventToken, 17}, // }
				{EventClose, 22}, // When
				{EventToken, 18}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
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

// TestWhenStatementMultiplePatterns tests when with multiple string patterns
func TestWhenStatementMultiplePatterns(t *testing.T) {
	input := `fun test { 
		when @var.ENV { 
			"prod" -> echo "p"
			"staging" -> echo "s"
			"dev" -> echo "d"
			else -> echo "x"
		}
	}`
	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors, got: %v", tree.Errors)
	}

	// Check for 4 when arms
	armCount := 0
	for _, event := range tree.Events {
		if event.Kind == EventOpen && event.Data == 23 { // NodeWhenArm
			armCount++
		}
	}

	if armCount != 4 {
		t.Errorf("Expected 4 when arms, got %d", armCount)
	}
}

// TestWhenStatementWithBlock tests when arm with block body
func TestWhenStatementWithBlock(t *testing.T) {
	input := `fun test { when @var.ENV { "production" -> { kubectl apply echo "done" } else -> echo "skip" } }`
	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors, got: %v", tree.Errors)
	}

	// Verify we have blocks
	hasBlock := false
	for _, event := range tree.Events {
		if event.Kind == EventOpen && event.Data == 3 { // NodeBlock
			hasBlock = true
			break
		}
	}

	if !hasBlock {
		t.Error("Expected to find block in when arm")
	}
}

// TestWhenStatementErrorRecovery tests error handling for malformed when statements
func TestWhenStatementErrorRecovery(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		errorContains string
	}{
		{
			name:          "missing expression",
			input:         "fun test { when { } }",
			errorContains: "missing expression after 'when'",
		},
		{
			name:          "missing opening brace",
			input:         "fun test { when @var.ENV }",
			errorContains: "missing '{' after when expression",
		},
		{
			name:          "missing arrow",
			input:         `fun test { when @var.ENV { "prod" echo "x" } }`,
			errorContains: "missing '->' after pattern",
		},
		{
			name:          "missing closing brace",
			input:         `fun test { when @var.ENV { "prod" -> echo "x" `,
			errorContains: "missing '}' after when arms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Errorf("Expected error containing %q, got no errors", tt.errorContains)
				return
			}

			found := false
			for _, err := range tree.Errors {
				if strings.Contains(err.Message, tt.errorContains) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error containing %q, got errors: %v", tt.errorContains, tree.Errors)
			}
		})
	}
}

// TestFunInsideWhen tests that fun declarations are rejected inside when blocks
func TestFunInsideWhen(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "fun inside when arm body",
			input: `fun test { when @var.ENV { "prod" -> fun helper { } } }`,
		},
		{
			name:  "fun inside when arm block",
			input: `fun test { when @var.ENV { "prod" -> { fun helper { } } } }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Errorf("Expected error about fun at top level, got no errors")
				return
			}

			err := tree.Errors[0]
			if err.Message != "function declarations must be at top level" {
				t.Errorf("Expected error about fun at top level, got: %s", err.Message)
			}
		})
	}
}

// TestWhenAtTopLevel tests when statement at top level (script mode)
func TestWhenAtTopLevel(t *testing.T) {
	input := `when @var.ENV { "production" -> kubectl apply else -> echo "skip" }`
	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors, got: %v", tree.Errors)
	}

	// Verify we have a when statement
	hasWhen := false
	for _, event := range tree.Events {
		if event.Kind == EventOpen && event.Data == 22 { // NodeWhen
			hasWhen = true
			break
		}
	}

	if !hasWhen {
		t.Error("Expected to find when statement at top level")
	}
}

// TestWhenRegexPatterns tests regex pattern matching
func TestWhenRegexPatterns(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple regex pattern",
			input: `fun test { when @var.branch { r"^release/" -> echo "release" else -> echo "other" } }`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 22},  // When (NodeWhen = 22)
				{EventToken, 3},  // when
				{EventOpen, 18},  // Decorator (NodeDecorator = 18)
				{EventToken, 4},  // @
				{EventToken, 5},  // var
				{EventToken, 6},  // .
				{EventToken, 7},  // branch
				{EventClose, 18}, // Decorator
				{EventToken, 8},  // {

				// First arm: r"^release/" -> echo "release"
				{EventOpen, 23},     // WhenArm (NodeWhenArm = 23)
				{EventOpen, 26},     // PatternRegex (NodePatternRegex = 26) - NEW NODE TYPE
				{EventToken, 9},     // r
				{EventToken, 10},    // "^release/"
				{EventClose, 26},    // PatternRegex
				{EventToken, 11},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 12},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 13},    // "release"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				// Second arm: else -> echo "other"
				{EventOpen, 23},     // WhenArm
				{EventOpen, 25},     // PatternElse (NodePatternElse = 25)
				{EventToken, 14},    // else
				{EventClose, 25},    // PatternElse
				{EventToken, 15},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 16},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 17},    // "other"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				{EventToken, 18}, // }
				{EventClose, 22}, // When
				{EventToken, 19}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
			},
		},
		{
			name: "multiple regex patterns",
			input: `fun test { when @var.branch {
				r"^main$" -> echo "main"
				r"^dev-" -> echo "dev"
				else -> echo "other"
			} }`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 22},  // When
				{EventToken, 3},  // when
				{EventOpen, 18},  // Decorator
				{EventToken, 4},  // @
				{EventToken, 5},  // var
				{EventToken, 6},  // .
				{EventToken, 7},  // branch
				{EventClose, 18}, // Decorator
				{EventToken, 8},  // {

				// First arm: r"^main$" -> echo "main"
				{EventOpen, 23},     // WhenArm
				{EventOpen, 26},     // PatternRegex
				{EventToken, 10},    // r (token 9 is newline, skipped)
				{EventToken, 11},    // "^main$"
				{EventClose, 26},    // PatternRegex
				{EventToken, 12},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 13},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 14},    // "main"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				// Second arm: r"^dev-" -> echo "dev"
				{EventOpen, 23},     // WhenArm
				{EventOpen, 26},     // PatternRegex
				{EventToken, 16},    // r (token 15 is newline, skipped)
				{EventToken, 17},    // "^dev-"
				{EventClose, 26},    // PatternRegex
				{EventToken, 18},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 19},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 20},    // "dev"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				// Third arm: else -> echo "other"
				{EventOpen, 23},     // WhenArm
				{EventOpen, 25},     // PatternElse
				{EventToken, 22},    // else (token 21 is newline, skipped)
				{EventClose, 25},    // PatternElse
				{EventToken, 23},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 24},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 25},    // "other"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				{EventToken, 27}, // } (token 26 is newline, skipped)
				{EventClose, 22}, // When
				{EventToken, 28}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
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

// TestWhenRangePatterns tests numeric range pattern matching
func TestWhenRangePatterns(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple integer range",
			input: `fun test { when @var.status { 200...299 -> echo "success" else -> echo "error" } }`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 22},  // When (NodeWhen = 22)
				{EventToken, 3},  // when
				{EventOpen, 18},  // Decorator (NodeDecorator = 18)
				{EventToken, 4},  // @
				{EventToken, 5},  // var
				{EventToken, 6},  // .
				{EventToken, 7},  // status
				{EventClose, 18}, // Decorator
				{EventToken, 8},  // {

				// First arm: 200...299 -> echo "success"
				{EventOpen, 23},     // WhenArm (NodeWhenArm = 23)
				{EventOpen, 27},     // PatternRange (NodePatternRange = 27) - NEW NODE TYPE
				{EventToken, 9},     // 200
				{EventToken, 10},    // ...
				{EventToken, 11},    // 299
				{EventClose, 27},    // PatternRange
				{EventToken, 12},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 13},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 14},    // "success"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				// Second arm: else -> echo "error"
				{EventOpen, 23},     // WhenArm
				{EventOpen, 25},     // PatternElse (NodePatternElse = 25)
				{EventToken, 15},    // else
				{EventClose, 25},    // PatternElse
				{EventToken, 16},    // ->
				{EventStepEnter, 0}, // Step boundary
				{EventOpen, 8},      // ShellCommand
				{EventOpen, 9},      // ShellArg
				{EventToken, 17},    // echo
				{EventClose, 9},     // ShellArg
				{EventOpen, 9},      // ShellArg
				{EventToken, 18},    // "error"
				{EventClose, 9},     // ShellArg
				{EventClose, 8},     // ShellCommand
				{EventStepExit, 0},  // Step boundary
				{EventClose, 23},    // WhenArm

				{EventToken, 19}, // }
				{EventClose, 22}, // When
				{EventToken, 20}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
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

// TestForLoopRanges tests for loop with range expressions (1...10)

// TestForLoopRanges tests for loop with range expressions (1...10)
func TestForLoopRanges(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple integer range", "fun test { for i in 1...10 { echo @var.i } }"},
		{"decorator as range start", "fun test { for i in @var.start...10 { } }"},
		{"decorator as range end", "fun test { for i in 1...@var.end { } }"},
		{"both decorators", "fun test { for i in @var.start...@var.end { } }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) > 0 {
				t.Fatalf("Unexpected errors: %v", tree.Errors)
			}

			// Check that NodeRange appears in events
			hasRange := false
			for _, ev := range tree.Events {
				if ev.Kind == EventOpen && NodeKind(ev.Data) == NodeRange {
					hasRange = true
					break
				}
			}

			if !hasRange {
				t.Errorf("Expected NodeRange in events, but not found")
			}
		})
	}
}

// TestWhenOrPatterns tests OR pattern matching
func TestWhenOrPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple OR", `fun test { when @var.env { "prod" | "production" -> echo "p" else -> echo "x" } }`},
		{"multiple OR", `fun test { when @var.env { "dev" | "development" | "local" -> echo "d" else -> echo "x" } }`},
		{"mixed patterns", `fun test { when @var.env { "prod" | r"^staging-" -> echo "deploy" else -> echo "skip" } }`},
		{"OR with ranges", `fun test { when @var.code { 200...299 | 300...399 -> echo "ok" else -> echo "err" } }`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) > 0 {
				t.Fatalf("Unexpected errors: %v", tree.Errors)
			}

			// Check that NodePatternOr appears in events
			hasOr := false
			for _, ev := range tree.Events {
				if ev.Kind == EventOpen && NodeKind(ev.Data) == NodePatternOr {
					hasOr = true
					break
				}
			}

			if !hasOr {
				t.Errorf("Expected NodePatternOr in events, but not found")
			}
		})
	}
}
