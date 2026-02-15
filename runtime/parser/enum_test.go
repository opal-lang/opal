package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseEnumDeclaration(t *testing.T) {
	input := `enum DeployStage String {
		Dev
		Prod = "production"
	}`

	tree := ParseString(input)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodesOfKind(tree.Events, NodeEnumDecl)); diff != "" {
		t.Fatalf("enum declaration count mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(2, countOpenNodesOfKind(tree.Events, NodeEnumMember)); diff != "" {
		t.Fatalf("enum member count mismatch (-want +got):\n%s", diff)
	}
}

func TestParseEnumDeclarationInsideBlockRejected(t *testing.T) {
	tree := ParseString(`fun deploy() { enum Stage { Dev } }`)
	if len(tree.Errors) == 0 {
		t.Fatal("expected parse error")
	}

	err := tree.Errors[0]
	if diff := cmp.Diff("enum declarations must be at top level", err.Message); diff != "" {
		t.Fatalf("error message mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("statement", err.Context); diff != "" {
		t.Fatalf("error context mismatch (-want +got):\n%s", diff)
	}
}

func TestParseEnumMemberReferenceExpression(t *testing.T) {
	tree := ParseString(`var os = OS.Windows`)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodesOfKind(tree.Events, NodeQualifiedRef)); diff != "" {
		t.Fatalf("qualified reference node count mismatch (-want +got):\n%s", diff)
	}
}

func TestParseEnumMemberReferenceRejectsExtraSegments(t *testing.T) {
	tree := ParseString(`var os = platform.OS.Windows`)
	if len(tree.Errors) == 0 {
		t.Fatal("expected parse error")
	}

	err := tree.Errors[0]
	if diff := cmp.Diff("qualified reference must use Type.Member", err.Message); diff != "" {
		t.Fatalf("error message mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("expression", err.Context); diff != "" {
		t.Fatalf("error context mismatch (-want +got):\n%s", diff)
	}
}

func TestParseWhenPatternWithEnumMemberReference(t *testing.T) {
	tree := ParseString(`
when @var.os {
  OS.Windows -> echo "ok"
}
`)
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	if diff := cmp.Diff(1, countOpenNodesOfKind(tree.Events, NodeQualifiedRef)); diff != "" {
		t.Fatalf("qualified reference node count mismatch (-want +got):\n%s", diff)
	}
}
