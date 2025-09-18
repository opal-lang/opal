package execution

import (
	"github.com/aledsdavies/devcmd/core/ir"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// ================================================================================================
// RUNTIME EXECUTION INTERFACES - Extend core interfaces with execution behavior
// ================================================================================================

// Runtime decorators implement core interfaces and add execution capability

// ImportRequirement describes external dependencies a decorator needs
type ImportRequirement struct {
	Packages []string          `json:"packages"` // Go packages to import
	Binaries []string          `json:"binaries"` // External binaries required
	Env      map[string]string `json:"env"`      // Environment variables required
}

// ================================================================================================
// TYPE ALIASES - Backward compatibility during migration
// ================================================================================================

// Type aliases for easier migration from old decorators package
type (
	// Core IR types (already in right place)
	CommandSeq   = ir.CommandSeq
	CommandStep  = ir.CommandStep
	ChainElement = ir.ChainElement
	ElementKind  = ir.ElementKind
	ChainOp      = ir.ChainOp

	// Execution types (from context package)
	Ctx           = context.Ctx
	CommandResult = context.CommandResult
	UIConfig      = context.UIConfig

	// Support for pattern decorators
	PatternSchema = map[string]any

	// Support for parallel decorators
	ParallelMode = string
)

// Parallel mode constants
const (
	ParallelModeFailFast      ParallelMode = "fail-fast"
	ParallelModeFailImmediate ParallelMode = "fail-immediate"
	ParallelModeAll           ParallelMode = "all"
)

// Element kind constants (from core/ir)
const (
	ElementKindShell   = ir.ElementKindShell
	ElementKindAction  = ir.ElementKindAction
	ElementKindBlock   = ir.ElementKindBlock
	ElementKindPattern = ir.ElementKindPattern
)

// Chain operation constants (from core/ir)
const (
	ChainOpAnd = ir.ChainOpAnd
	ChainOpOr  = ir.ChainOpOr
)
