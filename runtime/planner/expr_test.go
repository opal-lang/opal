package planner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// mapLookup creates a ValueLookup function from a map.
// Used in tests to provide values for EvaluateExpr.
func mapLookup(values map[string]any) ValueLookup {
	return func(name string) (any, bool) {
		val, ok := values[name]
		return val, ok
	}
}

// ========== ExprIR Type Tests ==========

func TestExprIR_Literal_String(t *testing.T) {
	expr := &ExprIR{
		Kind:  ExprLiteral,
		Value: "hello",
	}

	if expr.Kind != ExprLiteral {
		t.Errorf("Kind = %v, want ExprLiteral", expr.Kind)
	}
	if expr.Value != "hello" {
		t.Errorf("Value = %v, want %q", expr.Value, "hello")
	}
}

func TestExprIR_Literal_Int(t *testing.T) {
	expr := &ExprIR{
		Kind:  ExprLiteral,
		Value: 42,
	}

	if expr.Kind != ExprLiteral {
		t.Errorf("Kind = %v, want ExprLiteral", expr.Kind)
	}
	if expr.Value != 42 {
		t.Errorf("Value = %v, want 42", expr.Value)
	}
}

func TestExprIR_Literal_Bool(t *testing.T) {
	expr := &ExprIR{
		Kind:  ExprLiteral,
		Value: true,
	}

	if expr.Kind != ExprLiteral {
		t.Errorf("Kind = %v, want ExprLiteral", expr.Kind)
	}
	if expr.Value != true {
		t.Errorf("Value = %v, want true", expr.Value)
	}
}

func TestExprIR_VarRef(t *testing.T) {
	expr := &ExprIR{
		Kind:    ExprVarRef,
		VarName: "HOME",
	}

	if expr.Kind != ExprVarRef {
		t.Errorf("Kind = %v, want ExprVarRef", expr.Kind)
	}
	if expr.VarName != "HOME" {
		t.Errorf("VarName = %q, want %q", expr.VarName, "HOME")
	}
}

func TestExprIR_EnumMemberRef(t *testing.T) {
	expr := &ExprIR{
		Kind:       ExprEnumMemberRef,
		EnumName:   "OS",
		EnumMember: "Windows",
	}

	if expr.Kind != ExprEnumMemberRef {
		t.Errorf("Kind = %v, want ExprEnumMemberRef", expr.Kind)
	}
	if diff := cmp.Diff("OS", expr.EnumName); diff != "" {
		t.Errorf("EnumName mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("Windows", expr.EnumMember); diff != "" {
		t.Errorf("EnumMember mismatch (-want +got):\n%s", diff)
	}
}

func TestExprIR_DecoratorRef(t *testing.T) {
	expr := &ExprIR{
		Kind: ExprDecoratorRef,
		Decorator: &DecoratorRef{
			Name:     "env",
			Selector: []string{"HOME"},
		},
	}

	if expr.Kind != ExprDecoratorRef {
		t.Errorf("Kind = %v, want ExprDecoratorRef", expr.Kind)
	}
	if expr.Decorator.Name != "env" {
		t.Errorf("Decorator.Name = %q, want %q", expr.Decorator.Name, "env")
	}
	if diff := cmp.Diff([]string{"HOME"}, expr.Decorator.Selector); diff != "" {
		t.Errorf("Decorator.Selector mismatch (-want +got):\n%s", diff)
	}
}

func TestExprIR_DecoratorRef_NestedSelector(t *testing.T) {
	// @aws.secret.api_key
	expr := &ExprIR{
		Kind: ExprDecoratorRef,
		Decorator: &DecoratorRef{
			Name:     "aws",
			Selector: []string{"secret", "api_key"},
		},
	}

	if expr.Decorator.Name != "aws" {
		t.Errorf("Decorator.Name = %q, want %q", expr.Decorator.Name, "aws")
	}
	if diff := cmp.Diff([]string{"secret", "api_key"}, expr.Decorator.Selector); diff != "" {
		t.Errorf("Decorator.Selector mismatch (-want +got):\n%s", diff)
	}
}

func TestExprIR_BinaryOp_Equals(t *testing.T) {
	// @var.ENV == "prod"
	expr := &ExprIR{
		Kind: ExprBinaryOp,
		Op:   "==",
		Left: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "ENV",
		},
		Right: &ExprIR{
			Kind:  ExprLiteral,
			Value: "prod",
		},
	}

	if expr.Kind != ExprBinaryOp {
		t.Errorf("Kind = %v, want ExprBinaryOp", expr.Kind)
	}
	if expr.Op != "==" {
		t.Errorf("Op = %q, want %q", expr.Op, "==")
	}
	if expr.Left.Kind != ExprVarRef {
		t.Errorf("Left.Kind = %v, want ExprVarRef", expr.Left.Kind)
	}
	if expr.Right.Kind != ExprLiteral {
		t.Errorf("Right.Kind = %v, want ExprLiteral", expr.Right.Kind)
	}
}

func TestExprIR_BinaryOp_And(t *testing.T) {
	// @var.A && @var.B
	expr := &ExprIR{
		Kind: ExprBinaryOp,
		Op:   "&&",
		Left: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "A",
		},
		Right: &ExprIR{
			Kind:    ExprVarRef,
			VarName: "B",
		},
	}

	if expr.Op != "&&" {
		t.Errorf("Op = %q, want %q", expr.Op, "&&")
	}
}

func TestExprIR_SourceSpan(t *testing.T) {
	expr := &ExprIR{
		Kind:  ExprLiteral,
		Value: "test",
		Span: SourceSpan{
			File:  "test.opl",
			Start: 10,
			End:   14,
		},
	}

	if expr.Span.File != "test.opl" {
		t.Errorf("Span.File = %q, want %q", expr.Span.File, "test.opl")
	}
	if expr.Span.Start != 10 {
		t.Errorf("Span.Start = %d, want 10", expr.Span.Start)
	}
	if expr.Span.End != 14 {
		t.Errorf("Span.End = %d, want 14", expr.Span.End)
	}
}

// ========== DecoratorRef Tests ==========

func TestDecoratorRef_Simple(t *testing.T) {
	// @env.HOME
	ref := &DecoratorRef{
		Name:     "env",
		Selector: []string{"HOME"},
	}

	if ref.Name != "env" {
		t.Errorf("Name = %q, want %q", ref.Name, "env")
	}
	if len(ref.Selector) != 1 || ref.Selector[0] != "HOME" {
		t.Errorf("Selector = %v, want [HOME]", ref.Selector)
	}
}

func TestDecoratorRef_WithArgs(t *testing.T) {
	// @retry(3, "1s") - parameterized decorator
	ref := &DecoratorRef{
		Name:     "retry",
		Selector: nil,
		Args: []*ExprIR{
			{Kind: ExprLiteral, Value: 3},
			{Kind: ExprLiteral, Value: "1s"},
		},
	}

	if ref.Name != "retry" {
		t.Errorf("Name = %q, want %q", ref.Name, "retry")
	}
	if len(ref.Args) != 2 {
		t.Errorf("len(Args) = %d, want 2", len(ref.Args))
	}
	if ref.Args[0].Value != 3 {
		t.Errorf("Args[0].Value = %v, want 3", ref.Args[0].Value)
	}
}

// ========== CommandExpr Tests ==========

func TestCommandExpr_LiteralOnly(t *testing.T) {
	// echo "hello world"
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "echo \"hello world\""},
		},
	}

	if len(cmd.Parts) != 1 {
		t.Errorf("len(Parts) = %d, want 1", len(cmd.Parts))
	}
	if cmd.Parts[0].Kind != ExprLiteral {
		t.Errorf("Parts[0].Kind = %v, want ExprLiteral", cmd.Parts[0].Kind)
	}
}

func TestCommandExpr_WithVarRef(t *testing.T) {
	// echo "Hello @var.NAME"
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "echo \"Hello "},
			{Kind: ExprVarRef, VarName: "NAME"},
			{Kind: ExprLiteral, Value: "\""},
		},
	}

	if len(cmd.Parts) != 3 {
		t.Errorf("len(Parts) = %d, want 3", len(cmd.Parts))
	}
	if cmd.Parts[1].Kind != ExprVarRef {
		t.Errorf("Parts[1].Kind = %v, want ExprVarRef", cmd.Parts[1].Kind)
	}
	if cmd.Parts[1].VarName != "NAME" {
		t.Errorf("Parts[1].VarName = %q, want %q", cmd.Parts[1].VarName, "NAME")
	}
}

func TestCommandExpr_WithDecoratorRef(t *testing.T) {
	// echo "Home is @env.HOME"
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "echo \"Home is "},
			{
				Kind: ExprDecoratorRef,
				Decorator: &DecoratorRef{
					Name:     "env",
					Selector: []string{"HOME"},
				},
			},
			{Kind: ExprLiteral, Value: "\""},
		},
	}

	if len(cmd.Parts) != 3 {
		t.Errorf("len(Parts) = %d, want 3", len(cmd.Parts))
	}
	if cmd.Parts[1].Kind != ExprDecoratorRef {
		t.Errorf("Parts[1].Kind = %v, want ExprDecoratorRef", cmd.Parts[1].Kind)
	}
}

// ========== EvaluateExpr Tests ==========

func TestEvaluateExpr_Literal_String(t *testing.T) {
	expr := &ExprIR{Kind: ExprLiteral, Value: "hello"}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != "hello" {
		t.Errorf("EvaluateExpr() = %v, want %q", result, "hello")
	}
}

func TestEvaluateExpr_Literal_Int(t *testing.T) {
	expr := &ExprIR{Kind: ExprLiteral, Value: 42}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != 42 {
		t.Errorf("EvaluateExpr() = %v, want 42", result)
	}
}

func TestEvaluateExpr_Literal_Bool(t *testing.T) {
	expr := &ExprIR{Kind: ExprLiteral, Value: true}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_TypeCast_StringToInt(t *testing.T) {
	expr := &ExprIR{
		Kind:     ExprTypeCast,
		TypeName: "Int",
		Left: &ExprIR{
			Kind:  ExprLiteral,
			Value: "42",
		},
	}

	result, err := EvaluateExpr(expr, mapLookup(map[string]any{}))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}

	if diff := cmp.Diff(int64(42), result); diff != "" {
		t.Fatalf("cast result mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateExpr_TypeCast_NoneToOptional(t *testing.T) {
	expr := &ExprIR{
		Kind:     ExprTypeCast,
		TypeName: "String",
		Optional: true,
		Left: &ExprIR{
			Kind:  ExprLiteral,
			Value: nil,
		},
	}

	result, err := EvaluateExpr(expr, mapLookup(map[string]any{}))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}

	if result != nil {
		t.Fatalf("EvaluateExpr() = %v, want nil", result)
	}
}

func TestEvaluateExpr_TypeCast_NoneToRequiredFails(t *testing.T) {
	expr := &ExprIR{
		Kind:     ExprTypeCast,
		TypeName: "Int",
		Left: &ExprIR{
			Kind:  ExprLiteral,
			Value: nil,
		},
	}

	_, err := EvaluateExpr(expr, mapLookup(map[string]any{}))
	if err == nil {
		t.Fatal("expected error")
	}

	want := "cannot cast none to integer"
	if diff := cmp.Diff(want, err.Error()); diff != "" {
		t.Fatalf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateExpr_TypeCast_ObjectNormalizesMap(t *testing.T) {
	expr := &ExprIR{
		Kind:     ExprTypeCast,
		TypeName: "Object",
		Left: &ExprIR{
			Kind: ExprLiteral,
			Value: map[string]string{
				"name": "api",
			},
		},
	}

	result, err := EvaluateExpr(expr, mapLookup(map[string]any{}))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}

	want := map[string]any{"name": "api"}
	if diff := cmp.Diff(want, result); diff != "" {
		t.Fatalf("object cast result mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateExpr_VarRef_Found(t *testing.T) {
	expr := &ExprIR{Kind: ExprVarRef, VarName: "ENV"}
	values := map[string]any{"ENV": "production"}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != "production" {
		t.Errorf("EvaluateExpr() = %v, want %q", result, "production")
	}
}

func TestEvaluateExpr_VarRef_NotFound(t *testing.T) {
	expr := &ExprIR{Kind: ExprVarRef, VarName: "MISSING"}
	values := map[string]any{}

	_, err := EvaluateExpr(expr, mapLookup(values))
	if err == nil {
		t.Fatal("EvaluateExpr() expected error for missing variable")
	}
	// Error should mention the variable name
	if !contains(err.Error(), "MISSING") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "MISSING")
	}
}

func TestEvaluateExpr_EnumMemberRef_Found(t *testing.T) {
	expr := &ExprIR{Kind: ExprEnumMemberRef, EnumName: "OS", EnumMember: "Windows"}

	result, err := EvaluateExpr(expr, mapLookup(map[string]any{"OS.Windows": "windows"}))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}

	if diff := cmp.Diff("windows", result); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateExpr_EnumMemberRef_NotFound(t *testing.T) {
	expr := &ExprIR{Kind: ExprEnumMemberRef, EnumName: "OS", EnumMember: "Darwin"}

	_, err := EvaluateExpr(expr, mapLookup(nil))
	if err == nil {
		t.Fatal("expected error")
	}

	if diff := cmp.Diff("unresolved enum member: OS.Darwin", err.Error()); diff != "" {
		t.Errorf("error mismatch (-want +got):\n%s", diff)
	}
}

func TestEvaluateExpr_BinaryOp_Equals_True(t *testing.T) {
	// @var.ENV == "prod"
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "==",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "ENV"},
		Right: &ExprIR{Kind: ExprLiteral, Value: "prod"},
	}
	values := map[string]any{"ENV": "prod"}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_BinaryOp_Equals_False(t *testing.T) {
	// @var.ENV == "prod"
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "==",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "ENV"},
		Right: &ExprIR{Kind: ExprLiteral, Value: "prod"},
	}
	values := map[string]any{"ENV": "staging"}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != false {
		t.Errorf("EvaluateExpr() = %v, want false", result)
	}
}

func TestEvaluateExpr_BinaryOp_NotEquals(t *testing.T) {
	// @var.ENV != "prod"
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "!=",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "ENV"},
		Right: &ExprIR{Kind: ExprLiteral, Value: "prod"},
	}
	values := map[string]any{"ENV": "staging"}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_BinaryOp_And_True(t *testing.T) {
	// @var.A && @var.B
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "&&",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "A"},
		Right: &ExprIR{Kind: ExprVarRef, VarName: "B"},
	}
	values := map[string]any{"A": true, "B": true}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_BinaryOp_And_False(t *testing.T) {
	// @var.A && @var.B
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "&&",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "A"},
		Right: &ExprIR{Kind: ExprVarRef, VarName: "B"},
	}
	values := map[string]any{"A": true, "B": false}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != false {
		t.Errorf("EvaluateExpr() = %v, want false", result)
	}
}

func TestEvaluateExpr_BinaryOp_Or_True(t *testing.T) {
	// @var.A || @var.B
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "||",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "A"},
		Right: &ExprIR{Kind: ExprVarRef, VarName: "B"},
	}
	values := map[string]any{"A": false, "B": true}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_BinaryOp_Or_False(t *testing.T) {
	// @var.A || @var.B
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "||",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "A"},
		Right: &ExprIR{Kind: ExprVarRef, VarName: "B"},
	}
	values := map[string]any{"A": false, "B": false}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != false {
		t.Errorf("EvaluateExpr() = %v, want false", result)
	}
}

func TestEvaluateExpr_BinaryOp_LessThan(t *testing.T) {
	// @var.COUNT < 10
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "<",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "COUNT"},
		Right: &ExprIR{Kind: ExprLiteral, Value: 10},
	}
	values := map[string]any{"COUNT": 5}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_BinaryOp_GreaterThan(t *testing.T) {
	// @var.COUNT > 10
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    ">",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "COUNT"},
		Right: &ExprIR{Kind: ExprLiteral, Value: 10},
	}
	values := map[string]any{"COUNT": 15}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

func TestEvaluateExpr_Truthiness_NonEmptyString(t *testing.T) {
	// Non-empty string is truthy
	expr := &ExprIR{Kind: ExprLiteral, Value: "hello"}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if !IsTruthy(result) {
		t.Errorf("IsTruthy(%q) = false, want true", result)
	}
}

func TestEvaluateExpr_Truthiness_EmptyString(t *testing.T) {
	// Empty string is falsy
	expr := &ExprIR{Kind: ExprLiteral, Value: ""}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if IsTruthy(result) {
		t.Errorf("IsTruthy(%q) = true, want false", result)
	}
}

func TestEvaluateExpr_Truthiness_Zero(t *testing.T) {
	// Zero is falsy
	expr := &ExprIR{Kind: ExprLiteral, Value: 0}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if IsTruthy(result) {
		t.Errorf("IsTruthy(0) = true, want false")
	}
}

func TestEvaluateExpr_Truthiness_NonZero(t *testing.T) {
	// Non-zero is truthy
	expr := &ExprIR{Kind: ExprLiteral, Value: 42}
	values := map[string]any{}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if !IsTruthy(result) {
		t.Errorf("IsTruthy(42) = false, want true")
	}
}

func TestEvaluateExpr_Truthiness_Nil(t *testing.T) {
	// nil is falsy
	if IsTruthy(nil) {
		t.Errorf("IsTruthy(nil) = true, want false")
	}
}

func TestEvaluateExpr_BinaryOp_LargeInt64Precision(t *testing.T) {
	// Values beyond 2^53 lose precision when converted to float64
	// 9007199254740993 and 9007199254740992 would be equal as float64
	expr := &ExprIR{
		Kind:  ExprBinaryOp,
		Op:    "==",
		Left:  &ExprIR{Kind: ExprVarRef, VarName: "A"},
		Right: &ExprIR{Kind: ExprVarRef, VarName: "B"},
	}
	values := map[string]any{
		"A": int64(9007199254740993),
		"B": int64(9007199254740992),
	}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != false {
		t.Errorf("EvaluateExpr() = %v, want false (large int64 values should not be equal)", result)
	}
}

func TestEvaluateExpr_NestedBinaryOp(t *testing.T) {
	// (@var.A == "x") && (@var.B == "y")
	expr := &ExprIR{
		Kind: ExprBinaryOp,
		Op:   "&&",
		Left: &ExprIR{
			Kind:  ExprBinaryOp,
			Op:    "==",
			Left:  &ExprIR{Kind: ExprVarRef, VarName: "A"},
			Right: &ExprIR{Kind: ExprLiteral, Value: "x"},
		},
		Right: &ExprIR{
			Kind:  ExprBinaryOp,
			Op:    "==",
			Left:  &ExprIR{Kind: ExprVarRef, VarName: "B"},
			Right: &ExprIR{Kind: ExprLiteral, Value: "y"},
		},
	}
	values := map[string]any{"A": "x", "B": "y"}

	result, err := EvaluateExpr(expr, mapLookup(values))
	if err != nil {
		t.Fatalf("EvaluateExpr() error = %v", err)
	}
	if result != true {
		t.Errorf("EvaluateExpr() = %v, want true", result)
	}
}

// ========== RenderExpr Tests ==========

func TestRenderExpr_Literal(t *testing.T) {
	expr := &ExprIR{Kind: ExprLiteral, Value: "hello"}
	displayIDs := map[string]string{}

	result := RenderExpr(expr, displayIDs)
	if result != "hello" {
		t.Errorf("RenderExpr() = %q, want %q", result, "hello")
	}
}

func TestRenderExpr_Literal_Int(t *testing.T) {
	expr := &ExprIR{Kind: ExprLiteral, Value: 42}
	displayIDs := map[string]string{}

	result := RenderExpr(expr, displayIDs)
	if result != "42" {
		t.Errorf("RenderExpr() = %q, want %q", result, "42")
	}
}

func TestRenderExpr_VarRef(t *testing.T) {
	expr := &ExprIR{Kind: ExprVarRef, VarName: "NAME"}
	displayIDs := map[string]string{"NAME": "opal:abc123"}

	result := RenderExpr(expr, displayIDs)
	if result != "opal:abc123" {
		t.Errorf("RenderExpr() = %q, want %q", result, "opal:abc123")
	}
}

func TestRenderExpr_VarRef_Missing(t *testing.T) {
	expr := &ExprIR{Kind: ExprVarRef, VarName: "MISSING"}
	displayIDs := map[string]string{}

	// Missing displayID should return placeholder indicating error
	result := RenderExpr(expr, displayIDs)
	if result != "<unresolved:MISSING>" {
		t.Errorf("RenderExpr() = %q, want %q", result, "<unresolved:MISSING>")
	}
}

func TestRenderExpr_DecoratorRef(t *testing.T) {
	expr := &ExprIR{
		Kind: ExprDecoratorRef,
		Decorator: &DecoratorRef{
			Name:     "env",
			Selector: []string{"HOME"},
		},
	}
	displayIDs := map[string]string{"env.HOME": "opal:def456"}

	result := RenderExpr(expr, displayIDs)
	if result != "opal:def456" {
		t.Errorf("RenderExpr() = %q, want %q", result, "opal:def456")
	}
}

func TestRenderExpr_DecoratorRef_Nested(t *testing.T) {
	expr := &ExprIR{
		Kind: ExprDecoratorRef,
		Decorator: &DecoratorRef{
			Name:     "aws",
			Selector: []string{"secret", "api_key"},
		},
	}
	displayIDs := map[string]string{"aws.secret.api_key": "opal:xyz789"}

	result := RenderExpr(expr, displayIDs)
	if result != "opal:xyz789" {
		t.Errorf("RenderExpr() = %q, want %q", result, "opal:xyz789")
	}
}

// ========== RenderCommand Tests ==========

func TestRenderCommand_LiteralOnly(t *testing.T) {
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "echo hello"},
		},
	}
	displayIDs := map[string]string{}

	result := RenderCommand(cmd, displayIDs)
	if result != "echo hello" {
		t.Errorf("RenderCommand() = %q, want %q", result, "echo hello")
	}
}

func TestRenderCommand_WithVarRef(t *testing.T) {
	// echo "Hello @var.NAME"
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "echo \"Hello "},
			{Kind: ExprVarRef, VarName: "NAME"},
			{Kind: ExprLiteral, Value: "\""},
		},
	}
	displayIDs := map[string]string{"NAME": "opal:abc123"}

	result := RenderCommand(cmd, displayIDs)
	expected := "echo \"Hello opal:abc123\""
	if result != expected {
		t.Errorf("RenderCommand() = %q, want %q", result, expected)
	}
}

func TestRenderCommand_WithDecoratorRef(t *testing.T) {
	// echo "Home is @env.HOME"
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "echo \"Home is "},
			{
				Kind: ExprDecoratorRef,
				Decorator: &DecoratorRef{
					Name:     "env",
					Selector: []string{"HOME"},
				},
			},
			{Kind: ExprLiteral, Value: "\""},
		},
	}
	displayIDs := map[string]string{"env.HOME": "opal:def456"}

	result := RenderCommand(cmd, displayIDs)
	expected := "echo \"Home is opal:def456\""
	if result != expected {
		t.Errorf("RenderCommand() = %q, want %q", result, expected)
	}
}

func TestRenderCommand_MultipleRefs(t *testing.T) {
	// kubectl scale --replicas=@var.COUNT deployment/@var.NAME
	cmd := &CommandExpr{
		Parts: []*ExprIR{
			{Kind: ExprLiteral, Value: "kubectl scale --replicas="},
			{Kind: ExprVarRef, VarName: "COUNT"},
			{Kind: ExprLiteral, Value: " deployment/"},
			{Kind: ExprVarRef, VarName: "NAME"},
		},
	}
	displayIDs := map[string]string{
		"COUNT": "opal:count1",
		"NAME":  "opal:name2",
	}

	result := RenderCommand(cmd, displayIDs)
	expected := "kubectl scale --replicas=opal:count1 deployment/opal:name2"
	if result != expected {
		t.Errorf("RenderCommand() = %q, want %q", result, expected)
	}
}

// Helper for checking error messages
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
