package decorators

import (
	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ParameterSchema describes a decorator parameter
type ParameterSchema struct {
	Name        string             // Parameter name (e.g., "key", "default")
	Type        ast.ExpressionType // Parameter type (StringType, NumberType, etc.)
	Required    bool               // Whether this parameter is required
	Description string             // Human-readable description
}

// PatternSchema describes what patterns a pattern decorator accepts
type PatternSchema struct {
	AllowedPatterns     []string // Specific patterns allowed (e.g., ["main", "error", "finally"] for @try)
	RequiredPatterns    []string // Patterns that must be present (e.g., ["main"] for @try)
	AllowsWildcard      bool     // Whether "default" wildcard is allowed (e.g., true for @when)
	AllowsAnyIdentifier bool     // Whether any identifier is allowed (e.g., true for @when)
	Description         string   // Human-readable description of pattern rules
}

// ImportRequirement describes dependencies needed for code generation
type ImportRequirement struct {
	StandardLibrary []string          // Standard library imports (e.g., "time", "context", "sync")
	ThirdParty      []string          // Third-party imports (e.g., "github.com/pkg/errors")
	GoModules       map[string]string // Module dependencies for go.mod (module -> version)
}

// CommandDependencyProvider interface for decorators that reference other commands
// This allows the code generator to determine proper function declaration order
type CommandDependencyProvider interface {
	// GetCommandDependencies returns the names of commands this decorator depends on
	// Parameters: the decorator's parameters as provided in the AST
	// Returns: slice of command names that must be declared before this decorator is used
	GetCommandDependencies(params []ast.NamedParameter) []string
}

// Decorator is a union interface for all decorator types
// Used for registry and common operations
type Decorator interface {
	Name() string
	Description() string
	ParameterSchema() []ParameterSchema

	// ImportRequirements returns the dependencies needed for code generation
	ImportRequirements() ImportRequirement
}

// ValueDecorator represents decorators that provide values for shell interpolation
// Examples: @var, @env - inline value substitution only
type ValueDecorator interface {
	Decorator

	// ExpandInterpreter returns the actual runtime value for interpreter mode
	ExpandInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter) *execution.ExecutionResult

	// GenerateTemplate returns template for Go code expression that evaluates to the value for generator mode
	GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter) (*execution.TemplateResult, error)

	// ExpandPlan returns description for dry-run display in plan mode
	ExpandPlan(ctx execution.PlanContext, params []ast.NamedParameter) *execution.ExecutionResult
}

// ActionDecorator represents decorators that execute commands with structured output
// Examples: @cmd - can be standalone or chained with shell operators
type ActionDecorator interface {
	Decorator

	// ExpandInterpreter executes and returns CommandResult for interpreter mode
	ExpandInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter) *execution.ExecutionResult

	// GenerateTemplate returns template for Go code that produces CommandResult for generator mode
	GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter) (*execution.TemplateResult, error)

	// ExpandPlan returns description for dry-run display in plan mode
	ExpandPlan(ctx execution.PlanContext, params []ast.NamedParameter) *execution.ExecutionResult
}

// BlockDecorator represents decorators that modify command execution behavior
// Examples: @watch, @stop, @parallel
type BlockDecorator interface {
	Decorator

	// ExecuteInterpreter provides execution for interpreter mode
	ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult

	// GenerateTemplate provides template-based code generation
	GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) (*execution.TemplateResult, error)

	// ExecutePlan provides plan generation for plan mode
	ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult
}

// PatternDecorator represents decorators that handle pattern matching
// Examples: @when, @try
type PatternDecorator interface {
	Decorator

	// ExecuteInterpreter provides execution for interpreter mode
	ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult

	// GenerateTemplate provides template-based code generation
	GenerateTemplate(ctx execution.GeneratorContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (*execution.TemplateResult, error)

	// ExecutePlan provides plan generation for plan mode
	ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult

	// PatternSchema defines what patterns this decorator accepts
	PatternSchema() PatternSchema
}

// DecoratorType represents the type of decorator
type DecoratorType int

const (
	ValueType DecoratorType = iota
	ActionType
	BlockType
	PatternType
)

// GetDecoratorType returns the type of a decorator
// Note: Due to interface signature overlap, ActionDecorator and ValueDecorator
// cannot be reliably distinguished through type assertion alone
func GetDecoratorType(d Decorator) DecoratorType {
	switch d.(type) {
	case PatternDecorator:
		return PatternType
	case BlockDecorator:
		return BlockType
	case ActionDecorator:
		return ActionType
	// ValueDecorator case removed due to interface signature collision with ActionDecorator
	// Both interfaces have identical method signatures, making this case unreachable
	default:
		panic("unknown decorator type")
	}
}
