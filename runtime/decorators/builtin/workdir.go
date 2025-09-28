package builtin

import (
	"fmt"
	"path/filepath"

	"github.com/aledsdavies/opal/core/decorators"
	"github.com/aledsdavies/opal/core/plan"
	"github.com/aledsdavies/opal/runtime/execution/context"
)

// Register the @workdir decorator on package import
func init() {
	decorator := NewWorkdirDecorator()
	decorators.RegisterBlock(decorator)
	decorators.RegisterExecutionDecorator(decorator)
}

// WorkdirDecorator implements the @workdir decorator using the core decorator interfaces
type WorkdirDecorator struct{}

// WorkdirParams represents validated parameters for @workdir decorator
type WorkdirParams struct {
	Path              string `json:"path"`                 // Directory path to change to
	CreateIfNotExists bool   `json:"create_if_not_exists"` // Create directory if it doesn't exist
}

// NewWorkdirDecorator creates a new workdir decorator
func NewWorkdirDecorator() *WorkdirDecorator {
	return &WorkdirDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (w *WorkdirDecorator) Name() string {
	return "workdir"
}

// Description returns a human-readable description
func (w *WorkdirDecorator) Description() string {
	return "Changes working directory for the duration of the block execution"
}

// ParameterSchema returns the expected parameters for this decorator
func (w *WorkdirDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "path",
			Type:        decorators.ArgTypeString,
			Required:    true,
			Description: "Directory path to change to",
		},
		{
			Name:        "createIfNotExists",
			Type:        decorators.ArgTypeBool,
			Required:    false,
			Description: "Create directory if it doesn't exist (default: false)",
		},
	}
}

// Examples returns usage examples
func (w *WorkdirDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code: `build: @workdir("./build") {
    make clean
    make all
}`,
			Description: "Execute commands in the build directory",
		},
		{
			Code: `test: @workdir("./tests", createIfNotExists=true) {
    go test ./...
}`,
			Description: "Create test directory if needed and run tests",
		},
		{
			Code: `deploy: @workdir("/tmp/deploy") {
    kubectl apply -f .
}`,
			Description: "Change to absolute path for deployment",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
// Note: ImportRequirements removed - will be added back when code generation is implemented

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap executes the inner commands with updated working directory context
func (w *WorkdirDecorator) WrapCommands(ctx decorators.Context, args []decorators.Param, inner interface{}) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "runtime execution not implemented yet - use plan mode",
		ExitCode: 1,
	}
}

// Describe returns description for dry-run display
func (w *WorkdirDecorator) Describe(ctx decorators.Context, args []decorators.Param, inner plan.ExecutionStep) plan.ExecutionStep {
	path, createIfNotExists, err := w.extractParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepSequence,
			Description: fmt.Sprintf("@workdir(<error: %v>)", err),
			Command:     "",
		}
	}

	// Resolve path for display
	resolvedPath := path
	if !filepath.IsAbs(path) {
		resolvedPath = filepath.Join(ctx.GetWorkingDir(), path)
	}
	resolvedPath = filepath.Clean(resolvedPath)

	description := fmt.Sprintf("@workdir(%s)", resolvedPath)
	if createIfNotExists {
		description += " (create if needed)"
	}

	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: description,
		Command:     fmt.Sprintf("cd %s", resolvedPath),
		Children:    []plan.ExecutionStep{inner}, // Nested execution
		Metadata: map[string]string{
			"decorator":         "workdir",
			"path":              resolvedPath,
			"createIfNotExists": fmt.Sprintf("%t", createIfNotExists),
			"originalPath":      path,
		},
	}
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractParameters extracts and validates workdir parameters
func (w *WorkdirDecorator) extractParameters(params []decorators.Param) (path string, createIfNotExists bool, err error) {
	if len(params) == 0 {
		return "", false, fmt.Errorf("@workdir requires a path parameter")
	}

	// Extract path (first parameter)
	switch params[0].GetName() {
	case "":
		// Positional parameter
		if val, ok := params[0].GetValue().(string); ok {
			path = val
		} else {
			return "", false, fmt.Errorf("@workdir path must be a string, got %T", params[0].GetValue())
		}
	case "path":
		// Named parameter
		if val, ok := params[0].GetValue().(string); ok {
			path = val
		} else {
			return "", false, fmt.Errorf("@workdir path parameter must be a string, got %T", params[0].GetValue())
		}
	default:
		return "", false, fmt.Errorf("@workdir first parameter must be the path")
	}

	if path == "" {
		return "", false, fmt.Errorf("@workdir path cannot be empty")
	}

	// Set default
	createIfNotExists = false

	// Extract optional parameters
	for i := 1; i < len(params); i++ {
		param := params[i]

		if param.GetName() == "createIfNotExists" {
			if val, ok := param.GetValue().(bool); ok {
				createIfNotExists = val
			} else {
				return "", false, fmt.Errorf("@workdir createIfNotExists parameter must be a boolean, got %T", param.GetValue())
			}
		} else {
			return "", false, fmt.Errorf("@workdir unknown parameter: %s", param.GetName())
		}
	}

	// Basic security check - prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	if cleanPath != path && !filepath.IsAbs(path) {
		return "", false, fmt.Errorf("@workdir path contains invalid characters or traversal")
	}

	return path, createIfNotExists, nil
}

// ================================================================================================
// NEW EXECUTION DECORATOR METHODS (target interface)
// ================================================================================================

// Plan generates an execution plan for the workdir operation

// ================================================================================================
// NEW GENERIC INTERFACE METHODS (ExecutionDecorator[any])
// ================================================================================================

// Validate validates parameters and returns WorkdirParams
func (w *WorkdirDecorator) Validate(args []decorators.Param) (any, error) {
	// Extract path (first positional parameter or named "path")
	path, err := decorators.ExtractPositionalString(args, 0, "")
	if err != nil || path == "" {
		// Try named parameter "path"
		path, err = decorators.ExtractString(args, "path", "")
		if err != nil || path == "" {
			return nil, fmt.Errorf("@workdir requires a path")
		}
	}

	// Extract createIfNotExists flag (optional, defaults to false)
	createIfNotExists, err := decorators.ExtractBool(args, "createIfNotExists", false)
	if err != nil {
		return nil, fmt.Errorf("@workdir createIfNotExists parameter error: %w", err)
	}

	return WorkdirParams{
		Path:              path,
		CreateIfNotExists: createIfNotExists,
	}, nil
}

// Plan generates an execution plan using validated parameters
func (w *WorkdirDecorator) Plan(ctx decorators.Context, validated any) plan.ExecutionStep {
	params, ok := validated.(WorkdirParams)
	if !ok {
		return plan.ExecutionStep{
			Type:        plan.StepDecorator,
			Description: "@workdir(<invalid params>)",
			Command:     "",
			Metadata: map[string]string{
				"decorator": "workdir",
				"error":     "invalid_params",
			},
		}
	}

	description := fmt.Sprintf("@workdir(%q) { ... }", params.Path)
	if params.CreateIfNotExists {
		description = fmt.Sprintf("@workdir(%q, create=true) { ... }", params.Path)
	}

	return plan.ExecutionStep{
		Type:        plan.StepDecorator,
		Description: description,
		Command:     "",
		Metadata: map[string]string{
			"decorator":            "workdir",
			"path":                 params.Path,
			"create_if_not_exists": fmt.Sprintf("%t", params.CreateIfNotExists),
			"status":               "awaiting_executable_block_implementation",
		},
	}
}

// Execute performs the actual workdir logic using validated parameters
func (w *WorkdirDecorator) Execute(ctx decorators.Context, validated any) (decorators.CommandResult, error) {
	_, ok := validated.(WorkdirParams)
	if !ok {
		return nil, fmt.Errorf("@workdir: invalid parameters")
	}

	// TODO: When ExecutableBlock is implemented, this will become:
	// if params.CreateIfNotExists {
	//     err := os.MkdirAll(params.Path, 0755)
	//     if err != nil {
	//         return nil, fmt.Errorf("@workdir: failed to create directory %q: %w", params.Path, err)
	//     }
	// }
	// err := ctx.SetWorkingDir(params.Path)
	// if err != nil {
	//     return nil, fmt.Errorf("@workdir: failed to change directory to %q: %w", params.Path, err)
	// }
	// for _, stmt := range params.Block {
	//     result, err := stmt.Execute(ctx)
	//     if err != nil {
	//         return result, err
	//     }
	// }

	return nil, fmt.Errorf("@workdir: ExecutableBlock not yet implemented - use legacy interface for now")
}

// RequiresBlock returns the block requirements for @workdir
func (w *WorkdirDecorator) RequiresBlock() decorators.BlockRequirement {
	return decorators.BlockRequirement{
		Type:     decorators.BlockShell,
		Required: true,
	}
}
