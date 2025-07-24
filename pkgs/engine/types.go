package engine

import (
	"strings"
)

// ExecutionResult represents the result of executing a program in interpreter mode
type ExecutionResult struct {
	Variables map[string]string // Resolved variable values
	Commands  []CommandResult   // Results from executed commands
}

// CommandResult represents the result of executing a single command
type CommandResult struct {
	Name   string   // Command name
	Status string   // success, failed, skipped
	Output []string // Command output lines
	Error  string   // Error message if failed
}

// GenerationResult represents the result of generating Go code
type GenerationResult struct {
	Code              strings.Builder   // Generated Go code
	GoMod             strings.Builder   // Generated go.mod file
	StandardImports   map[string]bool   // Standard library imports
	ThirdPartyImports map[string]bool   // Third-party imports
	GoModules         map[string]string // Module dependencies (module -> version)
}

// String returns the generated code as a string
func (g *GenerationResult) String() string {
	return g.Code.String()
}

// GoModString returns the generated go.mod content as a string
func (g *GenerationResult) GoModString() string {
	return g.GoMod.String()
}

// AddStandardImport adds a standard library import
func (g *GenerationResult) AddStandardImport(pkg string) {
	g.StandardImports[pkg] = true
}

// AddThirdPartyImport adds a third-party import
func (g *GenerationResult) AddThirdPartyImport(pkg string) {
	g.ThirdPartyImports[pkg] = true
}

// AddGoModule adds a module dependency to go.mod
func (g *GenerationResult) AddGoModule(module, version string) {
	g.GoModules[module] = version
}

// HasStandardImport checks if a standard import is already added
func (g *GenerationResult) HasStandardImport(pkg string) bool {
	return g.StandardImports[pkg]
}

// HasThirdPartyImport checks if a third-party import is already added
func (g *GenerationResult) HasThirdPartyImport(pkg string) bool {
	return g.ThirdPartyImports[pkg]
}

// HasGoModule checks if a module dependency is already added
func (g *GenerationResult) HasGoModule(module string) bool {
	_, exists := g.GoModules[module]
	return exists
}

// Summary returns a summary of execution results
func (e *ExecutionResult) Summary() string {
	var summary strings.Builder

	summary.WriteString("Execution Summary:\n")
	summary.WriteString("Variables:\n")
	for name, value := range e.Variables {
		summary.WriteString("  " + name + " = " + value + "\n")
	}

	summary.WriteString("Commands:\n")
	for _, cmd := range e.Commands {
		summary.WriteString("  " + cmd.Name + ": " + cmd.Status)
		if cmd.Error != "" {
			summary.WriteString(" (" + cmd.Error + ")")
		}
		summary.WriteString("\n")
	}

	return summary.String()
}

// GetSuccessfulCommands returns commands that executed successfully
func (e *ExecutionResult) GetSuccessfulCommands() []CommandResult {
	var successful []CommandResult
	for _, cmd := range e.Commands {
		if cmd.Status == "success" {
			successful = append(successful, cmd)
		}
	}
	return successful
}

// GetFailedCommands returns commands that failed to execute
func (e *ExecutionResult) GetFailedCommands() []CommandResult {
	var failed []CommandResult
	for _, cmd := range e.Commands {
		if cmd.Status == "failed" {
			failed = append(failed, cmd)
		}
	}
	return failed
}

// HasErrors returns true if any commands failed
func (e *ExecutionResult) HasErrors() bool {
	return len(e.GetFailedCommands()) > 0
}
