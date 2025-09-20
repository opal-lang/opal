package decorators

import (
	"context"

	"github.com/aledsdavies/devcmd/core/plan"
)

// ================================================================================================
// CORE DECORATOR INTERFACES - Structural definitions only, no execution logic
// ================================================================================================

// DecoratorBase provides common metadata for all decorators
type DecoratorBase interface {
	Name() string
	Description() string
	ParameterSchema() []ParameterSchema
	Examples() []Example
}

// ArgType represents parameter types independent of AST
type ArgType string

const (
	ArgTypeString               ArgType = "string"
	ArgTypeBool                 ArgType = "bool"
	ArgTypeInt                  ArgType = "int"
	ArgTypeFloat                ArgType = "float"
	ArgTypeDuration             ArgType = "duration"   // Duration strings like "30s", "5m", "1h"
	ArgTypeIdentifier           ArgType = "identifier" // Variable/command identifiers
	ArgTypeList                 ArgType = "list"
	ArgTypeMap                  ArgType = "map"
	ArgTypeBlockFunction        ArgType = "block_function"         // Shell command blocks { commands }
	ArgTypePatternBlockFunction ArgType = "pattern_block_function" // Pattern blocks { branch: commands }
)

// ParameterSchema describes a decorator parameter
type ParameterSchema struct {
	Name        string  `json:"name"`        // Parameter name
	Type        ArgType `json:"type"`        // Parameter type (AST-independent)
	Required    bool    `json:"required"`    // Whether required
	Description string  `json:"description"` // Human-readable description
	Default     any     `json:"default"`     // Default value if not provided
}

// Example provides usage examples
type Example struct {
	Code        string `json:"code"`        // Example code
	Description string `json:"description"` // What it demonstrates
}

// ================================================================================================
// CORE DECORATOR INTERFACES - Plan generation only (execution in runtime)
// ================================================================================================

// ================================================================================================
// LEGACY DECORATOR INTERFACES - Will be removed in Phase 4
// ================================================================================================

// LegacyValueDecorator - Legacy inline value substitution decorators
type LegacyValueDecorator interface {
	DecoratorBase
	// Plan generation - shows how value will be resolved
	Describe(ctx Context, args []Param) plan.ExecutionStep
}

// LegacyActionDecorator - Legacy standalone action decorators
type LegacyActionDecorator interface {
	DecoratorBase
	// Plan generation - shows what action will be executed
	Describe(ctx Context, args []Param) plan.ExecutionStep
	// Execution method - runtime implementations provide this
	Run(ctx Context, args []Param) CommandResult
}

// LegacyBlockDecorator - Legacy execution wrapper decorators
type LegacyBlockDecorator interface {
	DecoratorBase
	// Plan generation - shows how inner commands will be wrapped
	Describe(ctx Context, args []Param, inner plan.ExecutionStep) plan.ExecutionStep
}

// LegacyPatternDecorator - Legacy conditional execution decorators
type LegacyPatternDecorator interface {
	DecoratorBase
	// Plan generation - shows which branch will be selected, with context for env access
	Describe(ctx Context, args []Param, branches map[string]plan.ExecutionStep) plan.ExecutionStep
	// Validation - each decorator validates its own patterns, can return multiple errors
	Validate(patternNames []string) []error
}

// ================================================================================================
// EXECUTION INTERFACES - Runtime contracts that decorators depend on
// ================================================================================================

// Context provides the runtime environment for decorator execution
// Wraps Go context.Context for cancellation plus devcmd-specific functionality
type Context interface {
	// Go context for cancellation and deadlines
	context.Context

	// Shell execution
	ExecShell(command string) CommandResult

	// Environment access
	GetEnv(key string) (string, bool)
	SetEnv(key, value string)
	GetWorkingDir() string
	SetWorkingDir(dir string) error

	// Variable access - CLI variables (@var) - immutable during execution
	GetVar(key string) (string, bool)

	// System information
	SystemInfo() SystemInfo

	// UI interaction
	Prompt(message string) (string, error)
	Confirm(message string) (bool, error)

	// Logging
	Log(level LogLevel, message string)
	Printf(format string, args ...any)
}

// SystemInfo provides system information useful for decorator development
type SystemInfo interface {
	// Hardware information
	GetNumCPU() int
	GetMemoryMB() int64

	// System identification
	GetHostname() string
	GetOS() string   // "linux", "darwin", "windows"
	GetArch() string // "amd64", "arm64", etc.

	// Runtime information
	GetTempDir() string
	GetHomeDir() string
	GetUserName() string
}

// CommandResult represents the result of executing a command
type CommandResult interface {
	GetStdout() string
	GetStderr() string
	GetExitCode() int
	IsSuccess() bool
}

// LogLevel for logging operations
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// ImportRequirement describes external dependencies a decorator needs
type ImportRequirement interface {
	GetPackages() []string     // Go packages to import
	GetBinaries() []string     // External binaries required
	GetEnv() map[string]string // Environment variables required
}

// ================================================================================================
// UNIFIED DECORATOR INTERFACES - Target architecture (2-interface system)
// ================================================================================================

// BlockRequirement describes structural requirements for execution decorators
type BlockRequirement struct {
	Type     BlockType `json:"type"`     // Type of block required
	Required bool      `json:"required"` // Whether block is required
}

// BlockType represents the type of block a decorator requires
type BlockType string

const (
	BlockNone    BlockType = "none"    // No block needed: @cmd(build)
	BlockShell   BlockType = "shell"   // Shell command block: @retry(3) { commands }
	BlockPattern BlockType = "pattern" // Pattern matching block: @when(ENV) { prod: ..., dev: ... }
)

// Resolved represents the result of decorator resolution with type flexibility
type Resolved interface {
	// The actual resolved value (string, bool, int, array, map, etc.)
	Value() any
	// Type classification matching our expression type system
	Type() ResolvedType
	// Hash for contract verification and change detection
	Hash() string
}

// ResolvedType uses our existing expression type system
type ResolvedType string

const (
	ResolvedString   ResolvedType = "string"
	ResolvedNumber   ResolvedType = "number"
	ResolvedDuration ResolvedType = "duration"
	ResolvedBoolean  ResolvedType = "boolean"
	ResolvedArray    ResolvedType = "array"
	ResolvedMap      ResolvedType = "map"
)

// ================================================================================================
// BLOCK EXECUTION DESIGN - Future Implementation
// ================================================================================================

// TODO: Implement full DAG-based value resolution and statement execution
//
// DESIGN OVERVIEW:
// 1. Parse Phase: Source → AST → IR
// 2. DAG Building: Traverse IR, identify value decorators (@var, @env), build dependency graph
// 3. Value Resolution: Execute value decorators in topological order, substitute into IR
// 4. Statement Building: Convert resolved IR into ExecutableStatements
// 5. Execution: Decorators get clean arrays of ready-to-run ExecutableStatements
//
// This enables decorators like @retry to simply loop through pre-resolved statements:
//
//	for _, stmt := range retryParams.Statements {
//	    result, err := stmt.Execute(ctx)
//	}
//
// ExecutableStatement (placeholder interface):
type ExecutableStatement interface {
	// Execute the resolved statement (all values already substituted)
	Execute(ctx Context) (CommandResult, error)
	// Plan generation for resolved statement
	Plan(ctx Context) plan.ExecutionStep
	// Display the resolved command
	String() string
}

// Block parameter type for decorators that need blocks (@retry, @parallel, @timeout, etc.)
type ExecutableBlock []ExecutableStatement

// resolved is a basic implementation of the Resolved interface
type resolved struct {
	value        any
	resolvedType ResolvedType
	hash         string
}

func (r *resolved) Value() any {
	return r.value
}

func (r *resolved) Type() ResolvedType {
	return r.resolvedType
}

func (r *resolved) Hash() string {
	return r.hash
}

// NewResolved creates a new resolved value
func NewResolved(value any, resolvedType ResolvedType, hash string) Resolved {
	return &resolved{
		value:        value,
		resolvedType: resolvedType,
		hash:         hash,
	}
}

// Decorator - Base generic interface for all decorators with validated parameters
type Decorator[T any] interface {
	DecoratorBase
	// Validate parameters and return decorator-specific validated type
	Validate(args []Param) (T, error)
	// Plan generation using validated parameters
	Plan(ctx Context, validated T) plan.ExecutionStep
}

// ValueDecorator - Generic interface for inline value substitution decorators
type ValueDecorator[T any] interface {
	Decorator[T]
	// Value resolution using validated parameters
	Resolve(ctx Context, validated T) (Resolved, error)
	// Performance optimization - expensive decorators resolved lazily
	IsExpensive() bool
}

// ExecutionDecorator - Generic interface for command execution decorators
type ExecutionDecorator[T any] interface {
	Decorator[T]
	// Execution using validated parameters
	Execute(ctx Context, validated T) (CommandResult, error)
}
