package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFunctionCall_ParsesKnownTopLevelCall(t *testing.T) {
	input := `fun deploy(env String, retries Int = 3) {
	echo "deploy"
}

deploy("prod", retries = 5)`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeShellCommand)); diff != "" {
		t.Fatalf("shell command node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_ParsesCallInBlock(t *testing.T) {
	input := `fun deploy(env String) {
	echo "deploy"
}

if true {
	deploy(env = "prod")
}`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_ParsesShorthandFunctionDefinition(t *testing.T) {
	input := `fun greet() = echo "hello"
greet()`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunction)); diff != "" {
		t.Fatalf("function declaration node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_ParsesShorthandFunctionWithForwardReference(t *testing.T) {
	input := `greet()

fun greet() = echo "hello"`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_ParsesShorthandFunctionWithLineComments(t *testing.T) {
	input := `fun greet() = echo "hello"
# greeting
greet()`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_ParsesForwardReferenceCall(t *testing.T) {
	input := `deploy("prod")

fun deploy(env String) {
	echo "deploy"
}`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_UnknownFunctionReturnsParseError(t *testing.T) {
	input := `missing(env = "prod")`

	tree := ParseString(input)

	if len(tree.Errors) != 1 {
		t.Fatalf("error count mismatch: want 1, got %d (%v)", len(tree.Errors), tree.Errors)
	}

	got := struct {
		Message    string
		Context    string
		Suggestion string
	}{
		Message:    tree.Errors[0].Message,
		Context:    tree.Errors[0].Context,
		Suggestion: tree.Errors[0].Suggestion,
	}

	want := struct {
		Message    string
		Context    string
		Suggestion string
	}{
		Message:    `unknown function "missing"`,
		Context:    "function call",
		Suggestion: `Define it with fun missing(...) { ... } or run a shell command by adding a space: missing (...)`,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unknown function error mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}
}

func TestFunctionCall_IdentifierWithSpaceBeforeParenStaysShell(t *testing.T) {
	input := `echo ("hello")`

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %v", tree.Errors)
	}

	if diff := cmp.Diff(0, countOpenNodes(tree.Events, NodeFunctionCall)); diff != "" {
		t.Fatalf("function call node count mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(1, countOpenNodes(tree.Events, NodeShellCommand)); diff != "" {
		t.Fatalf("shell command node count mismatch (-want +got):\n%s", diff)
	}
}

func countOpenNodes(events []Event, kind NodeKind) int {
	count := 0
	for _, event := range events {
		if event.Kind == EventOpen && NodeKind(event.Data) == kind {
			count++
		}
	}
	return count
}
