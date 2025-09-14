package builtin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
)

// Register the @workdir decorator on package import
func init() {
	decorators.RegisterBlock(NewWorkdirDecorator())
}

// WorkdirDecorator implements the @workdir decorator using the core decorator interfaces
type WorkdirDecorator struct{}

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
func (w *WorkdirDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"os", "path/filepath"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ================================================================================================
// BLOCK DECORATOR METHODS
// ================================================================================================

// Wrap executes the inner commands with updated working directory context
func (w *WorkdirDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	path, createIfNotExists, err := w.extractParameters(args)
	if err != nil {
		return decorators.CommandResult{
			Stderr:   fmt.Sprintf("@workdir parameter error: %v", err),
			ExitCode: 1,
		}
	}

	// Resolve path relative to current working directory
	resolvedPath := path
	if !filepath.IsAbs(path) {
		resolvedPath = filepath.Join(ctx.WorkDir, path)
	}

	// Clean the path to normalize it
	resolvedPath = filepath.Clean(resolvedPath)

	// Handle directory creation if requested
	if createIfNotExists {
		if err := os.MkdirAll(resolvedPath, 0o755); err != nil {
			return decorators.CommandResult{
				Stderr:   fmt.Sprintf("failed to create directory %s: %v", resolvedPath, err),
				ExitCode: 1,
			}
		}
	} else {
		// Verify the directory exists
		if _, err := os.Stat(resolvedPath); err != nil {
			return decorators.CommandResult{
				Stderr:   fmt.Sprintf("directory %s does not exist: %v", resolvedPath, err),
				ExitCode: 1,
			}
		}
	}

	// Create new context with updated working directory
	// This follows the "never use os.Chdir()" pattern
	workdirCtx := ctx.WithWorkDir(resolvedPath)

	// Execute inner commands with the new context
	return workdirCtx.ExecSequential(inner.Steps)
}

// Describe returns description for dry-run display
func (w *WorkdirDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
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
		resolvedPath = filepath.Join(ctx.WorkDir, path)
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
func (w *WorkdirDecorator) extractParameters(params []decorators.DecoratorParam) (path string, createIfNotExists bool, err error) {
	if len(params) == 0 {
		return "", false, fmt.Errorf("@workdir requires a path parameter")
	}

	// Extract path (first parameter)
	switch params[0].Name {
	case "":
		// Positional parameter
		if val, ok := params[0].Value.(string); ok {
			path = val
		} else {
			return "", false, fmt.Errorf("@workdir path must be a string, got %T", params[0].Value)
		}
	case "path":
		// Named parameter
		if val, ok := params[0].Value.(string); ok {
			path = val
		} else {
			return "", false, fmt.Errorf("@workdir path parameter must be a string, got %T", params[0].Value)
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

		if param.Name == "createIfNotExists" {
			if val, ok := param.Value.(bool); ok {
				createIfNotExists = val
			} else {
				return "", false, fmt.Errorf("@workdir createIfNotExists parameter must be a boolean, got %T", param.Value)
			}
		} else {
			return "", false, fmt.Errorf("@workdir unknown parameter: %s", param.Name)
		}
	}

	// Basic security check - prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	if cleanPath != path && !filepath.IsAbs(path) {
		return "", false, fmt.Errorf("@workdir path contains invalid characters or traversal")
	}

	return path, createIfNotExists, nil
}
