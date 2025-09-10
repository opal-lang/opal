package codegen_test

import (
	"testing"

	"github.com/aledsdavies/devcmd/codegen"
)

// Example demonstrates how a decorator might use GenOps for code generation hints
func Example() {
	// This shows how a decorator could optionally implement GenerateHint

	type LogDecorator struct {
		// ... decorator implementation
	}

	// Optional method - decorators work fine without this
	generateHint := func(l *LogDecorator, ops codegen.GenOps, args []codegen.DecoratorParam) codegen.TempResult {
		// Extract message from args
		message := "Hello World" // simplified

		// Use GenOps to generate code patterns
		return ops.CallAction("log", []codegen.DecoratorParam{
			{Value: message},
		})
	}

	_ = generateHint // avoid unused variable warning
}

// Example of GenOps usage patterns
func TestGenOpsPatterns(t *testing.T) {
	// This test shows the intended usage patterns

	// Mock GenOps implementation for testing
	ops := &mockGenOps{}

	// Basic operations
	shellResult := ops.Shell("echo hello")
	actionResult := ops.CallAction("log", []codegen.DecoratorParam{
		{Value: "Starting build"},
	})

	// Chain operations
	andResult := ops.And(shellResult, actionResult)
	varResult := ops.Var("BUILD_DIR")

	// These should generate appropriate code patterns
	if shellResult.ID() == "" {
		t.Error("Shell result should have an ID")
	}

	if andResult.String() == "" {
		t.Error("And result should generate code")
	}

	if varResult.String() == "" {
		t.Error("Var result should generate code")
	}
}

// Mock implementation for testing
type mockGenOps struct{}

func (m *mockGenOps) Shell(cmd string) codegen.TempResult {
	return codegen.NewTempResult("ExecShell(ctx, " + codegen.QuoteString(cmd) + ")")
}

func (m *mockGenOps) CallAction(name string, args []codegen.DecoratorParam) codegen.TempResult {
	return codegen.NewTempResult("callAction(" + codegen.QuoteString(name) + ", args)")
}

func (m *mockGenOps) And(left, right codegen.TempResult) codegen.TempResult {
	return codegen.NewTempResult(left.String() + " && " + right.String())
}

func (m *mockGenOps) Or(left, right codegen.TempResult) codegen.TempResult {
	return codegen.NewTempResult(left.String() + " || " + right.String())
}

func (m *mockGenOps) Pipe(left, right codegen.TempResult) codegen.TempResult {
	return codegen.NewTempResult(left.String() + " | " + right.String())
}

func (m *mockGenOps) Append(result codegen.TempResult, filename string) codegen.TempResult {
	return codegen.NewTempResult(result.String() + " >> " + codegen.QuoteString(filename))
}

func (m *mockGenOps) Var(name string) codegen.TempResult {
	return codegen.NewTempResult("vars." + codegen.SanitizeIdentifier(name))
}

func (m *mockGenOps) Env(name string) codegen.TempResult {
	return codegen.NewTempResult("ctx.Env.Get(" + codegen.QuoteString(name) + ")")
}

func (m *mockGenOps) Literal(value string) codegen.TempResult {
	return codegen.NewTempResult(codegen.QuoteString(value))
}

func (m *mockGenOps) Sequential(steps ...codegen.TempResult) codegen.TempResult {
	return codegen.NewTempResult("sequential(" + codegen.JoinResults(steps, ", ") + ")")
}

func (m *mockGenOps) Parallel(steps ...codegen.TempResult) codegen.TempResult {
	return codegen.NewTempResult("parallel(" + codegen.JoinResults(steps, ", ") + ")")
}

func (m *mockGenOps) WithWorkdir(dir string, body func(codegen.GenOps) codegen.TempResult) codegen.TempResult {
	inner := body(m)
	return codegen.NewTempResult("withWorkdir(" + codegen.QuoteString(dir) + ", " + inner.String() + ")")
}

func (m *mockGenOps) WithEnv(key, value string, body func(codegen.GenOps) codegen.TempResult) codegen.TempResult {
	inner := body(m)
	return codegen.NewTempResult("withEnv(" + codegen.QuoteString(key) + ", " + codegen.QuoteString(value) + ", " + inner.String() + ")")
}

func (m *mockGenOps) WithTimeout(seconds int, body func(codegen.GenOps) codegen.TempResult) codegen.TempResult {
	inner := body(m)
	return codegen.NewTempResult("withTimeout(" + string(rune(seconds)) + ", " + inner.String() + ")")
}

func (m *mockGenOps) If(condition codegen.TempResult, then, else_ codegen.TempResult) codegen.TempResult {
	return codegen.NewTempResult("if " + condition.String() + " then " + then.String() + " else " + else_.String())
}

func (m *mockGenOps) Loop(times codegen.TempResult, body func(codegen.GenOps) codegen.TempResult) codegen.TempResult {
	inner := body(m)
	return codegen.NewTempResult("loop(" + times.String() + ", " + inner.String() + ")")
}

func (m *mockGenOps) Try(body func(codegen.GenOps) codegen.TempResult, catch func(codegen.GenOps) codegen.TempResult) codegen.TempResult {
	bodyResult := body(m)
	catchResult := catch(m)
	return codegen.NewTempResult("try(" + bodyResult.String() + ", " + catchResult.String() + ")")
}
