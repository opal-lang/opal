package parser

import (
	"testing"
)

// TestParseErrors verifies error recovery for invalid syntax
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrors  bool
		description string
	}{
		{
			name:        "missing closing parenthesis",
			input:       "fun greet(name String {",
			wantErrors:  true,
			description: "should report missing )",
		},
		{
			name:        "missing function name",
			input:       "fun () {}",
			wantErrors:  false, // Parser might accept this, semantic analysis rejects
			description: "parser accepts, semantic analysis should reject",
		},
		{
			name:        "missing parameter name before colon",
			input:       "fun greet(: String) {}",
			wantErrors:  false, // Parser might accept this, semantic analysis rejects
			description: "parser accepts, semantic analysis should reject",
		},
		{
			name:        "trailing comma in parameters",
			input:       "fun greet(name String,) {}",
			wantErrors:  false, // Parser accepts trailing comma in parameter list
			description: "parser accepts trailing comma",
		},
		{
			name:        "missing function body",
			input:       "fun greet() }",
			wantErrors:  true,
			description: "should report missing function body (needs = or {)",
		},
		{
			name:        "missing closing brace",
			input:       "fun greet() {",
			wantErrors:  true,
			description: "should report missing }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			hasErrors := len(tree.Errors) > 0

			if hasErrors != tt.wantErrors {
				t.Errorf("Error expectation mismatch: got errors=%v, want errors=%v\nErrors: %v\nDescription: %s",
					hasErrors, tt.wantErrors, tree.Errors, tt.description)
			}

			// Parser should still produce events even with errors (error recovery)
			if len(tree.Events) == 0 {
				t.Error("Parser should produce events even with errors (for error recovery)")
			}
		})
	}
}

// TestErrorMessages verifies error messages are clear and helpful
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		{
			name:           "missing closing parenthesis",
			input:          "fun greet(name String {",
			wantMessage:    "missing ')'",
			wantContext:    "parameter list",
			wantSuggestion: "Add ')' to close the parameter list",
		},
		{
			name:           "missing function body",
			input:          "fun greet() }",
			wantMessage:    "missing '{'",
			wantContext:    "function body",
			wantSuggestion: "Add '{' to start the function body",
		},
		{
			name:           "missing closing brace",
			input:          "fun greet() {",
			wantMessage:    "missing '}'",
			wantContext:    "function body",
			wantSuggestion: "Add '}' to close the function body",
		},

		{
			name:           "missing closing brace",
			input:          "fun greet() {",
			wantMessage:    "missing '}'",
			wantContext:    "function body",
			wantSuggestion: "Add '}' to close the function body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

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
		})
	}
}

// TestMultipleErrors verifies parser reports multiple errors in one pass
func TestMultipleErrors(t *testing.T) {
	input := `fun first(name String {
  echo "hello"
}

fun second() {
  echo "world"

fun third(x String, y String {
  echo "test"
}`

	tree := ParseString(input)

	// Should have multiple errors (missing parens and braces)
	if len(tree.Errors) < 2 {
		t.Errorf("Expected multiple errors, got %d", len(tree.Errors))
	}

	// Should still produce events (partial parse tree for tooling)
	if len(tree.Events) == 0 {
		t.Error("Parser should produce events even with multiple errors")
	}

	// Verify we got errors from different functions
	errorLines := make(map[int]bool)
	for _, err := range tree.Errors {
		errorLines[err.Position.Line] = true
	}

	if len(errorLines) < 2 {
		t.Errorf("Expected errors from multiple lines, got errors on %d lines", len(errorLines))
	}
}

// TestErrorRecoveryProducesUsableTree verifies parser produces usable events even with errors
func TestErrorRecoveryProducesUsableTree(t *testing.T) {
	input := `fun broken(x String, y String {
  echo "test"
}

fun good() {
  echo "works"
}

fun alsoBroken(a String, b String, c String {
  echo "more"
}`

	tree := ParseString(input)

	// Should have errors
	if len(tree.Errors) == 0 {
		t.Error("Expected errors but got none")
	}

	// Should have Source node
	if len(tree.Events) < 2 {
		t.Fatal("Expected at least Source open/close events")
	}

	if tree.Events[0].Kind != EventOpen || NodeKind(tree.Events[0].Data) != NodeSource {
		t.Error("First event should be Source open")
	}

	lastEvent := tree.Events[len(tree.Events)-1]
	if lastEvent.Kind != EventClose || NodeKind(lastEvent.Data) != NodeSource {
		t.Error("Last event should be Source close")
	}

	// Count function nodes - should have all 3 functions even with errors
	functionCount := 0
	for _, event := range tree.Events {
		if event.Kind == EventOpen && NodeKind(event.Data) == NodeFunction {
			functionCount++
		}
	}

	if functionCount != 3 {
		t.Errorf("Expected 3 function nodes, got %d", functionCount)
	}
}
