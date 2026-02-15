package parser

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseStructDeclaration(t *testing.T) {
	input := `struct DeployConfig {
	env String
	replicas Int = 3
	timeout Duration?
}`

	tree := ParseString(input)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodesOfKind(tree.Events, NodeStructDecl)); diff != "" {
		t.Fatalf("struct declaration count mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(3, countOpenNodesOfKind(tree.Events, NodeStructField)); diff != "" {
		t.Fatalf("struct field count mismatch (-want +got):\n%s", diff)
	}
}

func TestParseStructDeclarationInsideBlockRejected(t *testing.T) {
	tree := ParseString(`fun deploy() { struct Config { retries Int } }`)
	if len(tree.Errors) == 0 {
		t.Fatal("expected parse error")
	}

	err := tree.Errors[0]
	if diff := cmp.Diff("struct declarations must be at top level", err.Message); diff != "" {
		t.Fatalf("error message mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("statement", err.Context); diff != "" {
		t.Fatalf("error context mismatch (-want +got):\n%s", diff)
	}
}

func TestParseFunctionColonTypeSyntaxRejected(t *testing.T) {
	tree := ParseString(`fun greet(name: String) {}`)
	if len(tree.Errors) == 0 {
		t.Fatal("expected parse error")
	}

	found := false
	for _, err := range tree.Errors {
		if err.Message == "missing parameter type annotation" {
			found = true
			break
		}
	}

	if diff := cmp.Diff(true, found); diff != "" {
		t.Fatalf("expected missing-type parse error (-want +got):\n%s", diff)
	}
}

func TestParseStructMalformedFieldDoesNotHang(t *testing.T) {
	done := make(chan struct{})

	go func() {
		_ = ParseString(`struct Config { = 1 }`)
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(250 * time.Millisecond):
		t.Fatal("ParseString hung on malformed struct field")
	}
}

func TestParseStructInheritanceRejected(t *testing.T) {
	tree := ParseString(`struct Child : Parent { env String }`)
	if len(tree.Errors) == 0 {
		t.Fatal("expected parse error")
	}

	found := false
	for _, err := range tree.Errors {
		if err.Message == "struct inheritance is not supported" {
			if diff := cmp.Diff("struct declaration", err.Context); diff != "" {
				t.Fatalf("error context mismatch (-want +got):\n%s", diff)
			}
			found = true
			break
		}
	}

	if diff := cmp.Diff(true, found); diff != "" {
		t.Fatalf("expected inheritance rejection error (-want +got):\n%s", diff)
	}
}

func TestParseStructMethodsRejected(t *testing.T) {
	tree := ParseString(`struct Config { fun validate() {} }`)
	if len(tree.Errors) == 0 {
		t.Fatal("expected parse error")
	}

	found := false
	for _, err := range tree.Errors {
		if err.Message == "struct methods are not supported" {
			if diff := cmp.Diff("struct declaration", err.Context); diff != "" {
				t.Fatalf("error context mismatch (-want +got):\n%s", diff)
			}
			found = true
			break
		}
	}

	if diff := cmp.Diff(true, found); diff != "" {
		t.Fatalf("expected method rejection error (-want +got):\n%s", diff)
	}
}

func countOpenNodesOfKind(events []Event, kind NodeKind) int {
	count := 0
	for _, event := range events {
		if event.Kind == EventOpen && NodeKind(event.Data) == kind {
			count++
		}
	}
	return count
}
