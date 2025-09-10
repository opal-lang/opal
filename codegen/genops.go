// Package codegen provides lightweight patterns and utilities for code generation.
// This package is designed to be optionally imported by decorators that want to
// participate in code generation optimization.
package codegen

// GenOps provides lightweight patterns for backend-agnostic code generation.
// This interface focuses on providing helpful patterns and utilities rather than
// being a complex backend abstraction.
type GenOps interface {
	// Basic operations
	Shell(cmd string) TempResult                              // Execute shell command
	CallAction(name string, args []DecoratorParam) TempResult // Call action decorator

	// Chain operations (shell operators)
	And(left, right TempResult) TempResult                // && operator
	Or(left, right TempResult) TempResult                 // || operator
	Pipe(left, right TempResult) TempResult               // | operator
	Append(result TempResult, filename string) TempResult // >> operator

	// Value operations
	Var(name string) TempResult      // Variable reference @var(name)
	Env(name string) TempResult      // Environment variable @env(name)
	Literal(value string) TempResult // String literal

	// Block operations
	Sequential(steps ...TempResult) TempResult // Execute steps in sequence
	Parallel(steps ...TempResult) TempResult   // Execute steps in parallel

	// Context operations
	WithWorkdir(dir string, body func(GenOps) TempResult) TempResult
	WithEnv(key, value string, body func(GenOps) TempResult) TempResult
	WithTimeout(seconds int, body func(GenOps) TempResult) TempResult

	// Advanced operations (for decorators that need them)
	If(condition TempResult, then, else_ TempResult) TempResult
	Loop(times TempResult, body func(GenOps) TempResult) TempResult
	Try(body func(GenOps) TempResult, catch func(GenOps) TempResult) TempResult
}

// TempResult represents an intermediate code generation result.
// This is a lightweight abstraction that can be converted to the target language.
type TempResult interface {
	// ID returns a unique identifier for this result
	ID() string

	// String returns a human-readable representation for debugging
	String() string
}

// DecoratorParam represents a parameter passed to a decorator (parser-agnostic)
type DecoratorParam struct {
	Name  string
	Value interface{}
}

// GenerateHint is an optional interface that decorators can implement to provide
// code generation hints. This is completely optional - decorators work fine without it.
type GenerateHint interface {
	// Generate provides a hint for how this decorator should be code-generated.
	// The IR-based code generator may use this hint for optimization.
	GenerateHint(ops GenOps, args []DecoratorParam) TempResult
}

// BlockGenerateHint is for block decorators that want to provide generation hints
type BlockGenerateHint interface {
	GenerateBlockHint(ops GenOps, args []DecoratorParam, body func(GenOps) TempResult) TempResult
}

// PatternGenerateHint is for pattern decorators that want to provide generation hints
type PatternGenerateHint interface {
	GeneratePatternHint(ops GenOps, args []DecoratorParam, branches map[string]func(GenOps) TempResult) TempResult
}
