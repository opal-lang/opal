package execution

import "github.com/aledsdavies/devcmd/pkgs/ast"

// ExecutionMode represents the different modes of execution
type ExecutionMode int

const (
	InterpreterMode ExecutionMode = iota // Run commands directly
	GeneratorMode                        // Generate Go code for compilation
	PlanMode                             // Generate execution plan for dry-run
)

// String returns a string representation of the execution mode
func (m ExecutionMode) String() string {
	switch m {
	case InterpreterMode:
		return "interpreter"
	case GeneratorMode:
		return "generator"
	case PlanMode:
		return "plan"
	default:
		return "unknown"
	}
}

// ExecutionResult represents the result of executing shell content in different modes
type ExecutionResult struct {
	// Mode is the execution mode that produced this result
	Mode ExecutionMode

	// Data contains the mode-specific result:
	// - InterpreterMode: nil (execution happens directly)
	// - GeneratorMode: string (Go code)
	// - PlanMode: plan.PlanElement (plan element)
	Data interface{}

	// Error contains any execution error
	Error error
}

// FunctionDecorator interface to avoid circular imports
// This mirrors the decorators.FunctionDecorator interface
type FunctionDecorator interface {
	// Expand provides unified expansion for all modes - returns values for command composition
	Expand(ctx *ExecutionContext, params []ast.NamedParameter) *ExecutionResult
}
