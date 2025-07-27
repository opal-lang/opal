package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/execution"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// functionDecoratorAdapter adapts decorators.FunctionDecorator to execution.FunctionDecorator
type functionDecoratorAdapter struct {
	decorator decorators.FunctionDecorator
}

// Expand implements execution.FunctionDecorator by delegating to decorators.FunctionDecorator.Expand
func (a *functionDecoratorAdapter) Expand(ctx *execution.ExecutionContext, params []ast.NamedParameter) *execution.ExecutionResult {
	return a.decorator.Expand(ctx, params)
}

// ProcessGroup represents a group of watch/stop commands for the same identifier
type ProcessGroup struct {
	Identifier   string
	WatchCommand *ast.CommandDecl
	StopCommand  *ast.CommandDecl
}

// CommandGroups holds the analyzed command structure
type CommandGroups struct {
	RegularCommands []*ast.CommandDecl
	ProcessGroups   []ProcessGroup
}

// Engine provides a unified AST walker for both interpreter and generator modes
type Engine struct {
	ctx       *execution.ExecutionContext
	goVersion string // Go version for generated code (e.g., "1.24")
}

// New creates a new execution engine with the execution context
func New(program *ast.Program) *Engine {
	ctx := execution.NewExecutionContext(context.Background(), program)
	engine := &Engine{
		ctx:       ctx,
		goVersion: "1.24", // Default Go version
	}
	// Set up dependency injection for decorators
	engine.setupDependencyInjection()
	return engine
}

// NewWithGoVersion creates a new execution engine with specified Go version
func NewWithGoVersion(program *ast.Program, goVersion string) *Engine {
	ctx := execution.NewExecutionContext(context.Background(), program)
	engine := &Engine{
		ctx:       ctx,
		goVersion: goVersion,
	}
	// Set up dependency injection for decorators
	engine.setupDependencyInjection()
	return engine
}

// ExecuteCommand executes a single command in interpreter mode
func (e *Engine) ExecuteCommand(command *ast.CommandDecl) (*CommandResult, error) {
	// Set execution mode to interpreter
	ctx := e.ctx.WithMode(execution.InterpreterMode)

	// Initialize variables if not already done
	if err := ctx.InitializeVariables(); err != nil {
		return nil, fmt.Errorf("failed to initialize variables: %w", err)
	}

	cmdResult := &CommandResult{
		Name:   command.Name,
		Status: "success",
		Output: []string{},
		Error:  "",
	}

	// Execute the command content directly
	for _, content := range command.Body.Content {
		switch c := content.(type) {
		case *ast.ShellContent:
			// Execute shell content using the execution context
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				cmdResult.Status = "failed"
				cmdResult.Error = result.Error.Error()
				return cmdResult, result.Error
			}
		default:
			err := fmt.Errorf("unsupported command content type in interpreter mode: %T", content)
			cmdResult.Status = "failed"
			cmdResult.Error = err.Error()
			return cmdResult, err
		}
	}

	return cmdResult, nil
}

// ExecuteCommandPlan generates an execution plan for a command without executing it
func (e *Engine) ExecuteCommandPlan(command *ast.CommandDecl) (*plan.ExecutionPlan, error) {
	// Set execution mode to plan mode
	ctx := e.ctx.WithMode(execution.PlanMode)

	// Initialize variables if not already done
	if err := ctx.InitializeVariables(); err != nil {
		return nil, fmt.Errorf("failed to initialize variables: %w", err)
	}

	// Create a new execution plan
	planBuilder := plan.NewPlan()

	// Execute the command content in plan mode to collect plan elements
	for _, content := range command.Body.Content {
		switch c := content.(type) {
		case *ast.ShellContent:
			// Execute shell content in plan mode
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				return nil, fmt.Errorf("failed to create plan for shell content: %w", result.Error)
			}

			// Convert the result to a plan element
			if planData, ok := result.Data.(map[string]interface{}); ok {
				if cmdStr, ok := planData["command"].(string); ok {
					description := "Execute shell command"
					if desc, ok := planData["description"].(string); ok {
						description = desc
					}
					element := plan.Command(cmdStr).WithDescription(description)
					planBuilder.Add(element)
				}
			}
		case *ast.BlockDecorator:
			// Execute block decorator in plan mode
			result, err := e.executeDecoratorPlan(ctx, c)
			if err != nil {
				return nil, fmt.Errorf("failed to create plan for block decorator: %w", err)
			}

			// Add the plan element returned by the decorator
			if planElement, ok := result.Data.(plan.PlanElement); ok {
				planBuilder.Add(planElement)
			}
		default:
			return nil, fmt.Errorf("unsupported command content type in plan mode: %T", content)
		}
	}

	// Build the plan and add command name to context
	execPlan := planBuilder.Build()
	execPlan.Context["command_name"] = command.Name

	return execPlan, nil
}

// executeDecoratorPlan executes a decorator in plan mode
func (e *Engine) executeDecoratorPlan(ctx *execution.ExecutionContext, decorator *ast.BlockDecorator) (*execution.ExecutionResult, error) {
	// Look up the decorator in the registry
	blockDecorator, err := decorators.GetBlock(decorator.Name)
	if err != nil {
		return nil, fmt.Errorf("block decorator @%s not found: %w", decorator.Name, err)
	}

	// Execute the decorator in plan mode
	result := blockDecorator.Execute(ctx, decorator.Args, decorator.Content)
	if result.Error != nil {
		return nil, fmt.Errorf("@%s decorator plan execution failed: %w", decorator.Name, result.Error)
	}

	return result, nil
}

// GenerateCode generates Go code for the entire program using template-based approach
func (e *Engine) GenerateCode(program *ast.Program) (*GenerationResult, error) {
	// Use the new template-based approach
	return e.generateCodeWithTemplate(program)
}

// setupDependencyInjection sets up dependency injection for the execution context
func (e *Engine) setupDependencyInjection() {
	// Set up function decorator lookup - create an adapter to bridge decorator interfaces
	e.ctx.SetFunctionDecoratorLookup(func(name string) (execution.FunctionDecorator, bool) {
		decorator, exists := decorators.GetFunctionDecorator(name)
		if !exists {
			return nil, false
		}

		// Create an adapter that bridges the interface difference
		adapter := &functionDecoratorAdapter{decorator: decorator}
		return adapter, true
	})

	// Command content execution is handled directly by the execution context

	// Set up template functions for code generation
	e.setupTemplateFunctions()
}

// WriteFiles writes the generated Go code and go.mod to the specified directory
func (e *Engine) WriteFiles(result *GenerationResult, targetDir string, moduleName string) error {
	// Write main.go
	mainGoPath := filepath.Join(targetDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(result.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	// Write go.mod with custom module name
	goModContent := result.GoModString()
	if moduleName != "" && moduleName != "devcmd-generated" {
		goModContent = strings.Replace(goModContent, "module devcmd-generated", "module "+moduleName, 1)
	}

	goModPath := filepath.Join(targetDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	return nil
}

// processVariablesIntoContext processes variables and variable groups into the execution context
func (e *Engine) processVariablesIntoContext(program *ast.Program) error {
	// Process individual variables
	for _, variable := range program.Variables {
		value, err := e.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		e.ctx.SetVariable(variable.Name, value)
	}

	// Process variable groups
	for _, group := range program.VarGroups {
		for _, variable := range group.Variables {
			value, err := e.resolveVariableValue(variable.Value)
			if err != nil {
				return fmt.Errorf("failed to resolve variable %s in group: %w", variable.Name, err)
			}
			e.ctx.SetVariable(variable.Name, value)
		}
	}

	return nil
}

// resolveVariableValue resolves a variable value from an expression
func (e *Engine) resolveVariableValue(expr ast.Expression) (string, error) {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value, nil
	case *ast.NumberLiteral:
		return v.Value, nil
	case *ast.BooleanLiteral:
		return fmt.Sprintf("%t", v.Value), nil
	case *ast.Identifier:
		// Handle variable references
		if value, exists := e.ctx.GetVariable(v.Name); exists {
			return value, nil
		}
		return "", fmt.Errorf("undefined variable: %s", v.Name)
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// collectDecoratorImports collects import requirements from all decorators used in the program
func (e *Engine) collectDecoratorImports(program *ast.Program, result *GenerationResult) error {
	// Collect from commands
	for _, cmd := range program.Commands {
		if err := e.collectDecoratorImportsFromContent(cmd.Body.Content, result); err != nil {
			return err
		}
	}
	return nil
}

// collectDecoratorImportsFromContent recursively collects decorator imports from command content
func (e *Engine) collectDecoratorImportsFromContent(content []ast.CommandContent, result *GenerationResult) error {
	for _, item := range content {
		switch c := item.(type) {
		case *ast.ShellContent:
			// Collect from function decorators in shell parts
			for _, part := range c.Parts {
				if fn, ok := part.(*ast.FunctionDecorator); ok {
					if err := e.addDecoratorImports("function", fn.Name, result); err != nil {
						return err
					}
				}
			}
		case *ast.BlockDecorator:
			if err := e.addDecoratorImports("block", c.Name, result); err != nil {
				return err
			}
			// Recursively collect from block content
			if err := e.collectDecoratorImportsFromContent(c.Content, result); err != nil {
				return err
			}
		case *ast.PatternDecorator:
			if err := e.addDecoratorImports("pattern", c.Name, result); err != nil {
				return err
			}
			// Recursively collect from pattern branches
			for _, pattern := range c.Patterns {
				if err := e.collectDecoratorImportsFromContent(pattern.Commands, result); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// addDecoratorImports adds import requirements for a specific decorator
func (e *Engine) addDecoratorImports(decoratorType, name string, result *GenerationResult) error {
	var decorator decorators.Decorator
	var err error

	switch decoratorType {
	case "function":
		decorator, err = decorators.GetFunction(name)
	case "block":
		decorator, err = decorators.GetBlock(name)
	case "pattern":
		decorator, err = decorators.GetPattern(name)
	default:
		return fmt.Errorf("unknown decorator type: %s", decoratorType)
	}

	if err != nil {
		return fmt.Errorf("decorator %s not found: %w", name, err)
	}

	// Get import requirements if the decorator supports it
	if importProvider, ok := decorator.(interface {
		ImportRequirements() decorators.ImportRequirement
	}); ok {
		requirements := importProvider.ImportRequirements()

		// Add standard library imports
		for _, pkg := range requirements.StandardLibrary {
			result.AddStandardImport(pkg)
		}

		// Add third-party imports
		for _, pkg := range requirements.ThirdParty {
			result.AddThirdPartyImport(pkg)
		}

		// Add Go modules
		for module, version := range requirements.GoModules {
			result.AddGoModule(module, version)
		}

		// Check if this decorator uses the plan package and inject devcmd dependency
		needsDevcmdPlan := false
		for _, pkg := range requirements.ThirdParty {
			if strings.Contains(pkg, "github.com/aledsdavies/devcmd/pkgs/plan") {
				needsDevcmdPlan = true
				break
			}
		}

		// If decorator needs plan package, add devcmd dependency with current version
		if needsDevcmdPlan {
			devcmdVersion := e.getDevcmdVersion()
			result.AddGoModule("github.com/aledsdavies/devcmd", devcmdVersion)
		}
	}

	return nil
}

// generateGoMod creates the go.mod file content from collected dependencies
// Go module template
const goModTemplate = `module devcmd-generated

go 1.21

require (
	github.com/spf13/cobra v1.9.1{{if .NeedsDevcmd}}
	github.com/aledsdavies/devcmd {{.DevcmdVersion}}{{end}}
	{{range .Modules}}{{.Module}} {{.Version}}
	{{end}}
)
{{if and .NeedsDevcmd .IsLocalDev}}
// Replace directive for local development
replace github.com/aledsdavies/devcmd => {{.LocalPath}}
{{end}}`

type GoModTemplateData struct {
	NeedsDevcmd   bool
	DevcmdVersion string
	IsLocalDev    bool
	LocalPath     string
	Modules       []ModuleData
}

type ModuleData struct {
	Module  string
	Version string
}

func (e *Engine) generateGoMod(result *GenerationResult) error {
	// Check if we need devcmd module (for plan DSL, etc.)
	needsDevcmd := false
	for module := range result.GoModules {
		if strings.Contains(module, "github.com/aledsdavies/devcmd") {
			needsDevcmd = true
			break
		}
	}

	// Collect other modules (excluding cobra and devcmd)
	var modules []ModuleData
	for module, version := range result.GoModules {
		if module != "github.com/spf13/cobra" && !strings.Contains(module, "github.com/aledsdavies/devcmd") {
			modules = append(modules, ModuleData{
				Module:  module,
				Version: version,
			})
		}
	}

	templateData := GoModTemplateData{
		NeedsDevcmd:   needsDevcmd,
		DevcmdVersion: e.getDevcmdVersion(),
		IsLocalDev:    e.isLocalDevelopment(),
		LocalPath:     e.getDevcmdLocalPath(),
		Modules:       modules,
	}

	tmpl, err := template.New("goMod").Parse(goModTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod template: %w", err)
	}

	if err := tmpl.Execute(&result.GoMod, templateData); err != nil {
		return fmt.Errorf("failed to execute go.mod template: %w", err)
	}

	return nil
}

// isLocalDevelopment checks if we're in local development mode
func (e *Engine) isLocalDevelopment() bool {
	// Check if we're in a git repository and the version looks like a dev version
	version := e.getDevcmdVersion()
	return strings.Contains(version, "dev") || version == "v0.0.0-dev"
}

// getDevcmdLocalPath tries to determine the local path to the devcmd module
func (e *Engine) getDevcmdLocalPath() string {
	// Try to find the devcmd project root by looking for go.mod
	workingDir, err := os.Getwd()
	if err != nil {
		return "../../" // fallback
	}

	// Check if we're in the devcmd project itself
	if _, err := os.Stat(filepath.Join(workingDir, "go.mod")); err == nil {
		// Check if this is the devcmd module
		if goModBytes, err := os.ReadFile(filepath.Join(workingDir, "go.mod")); err == nil {
			if strings.Contains(string(goModBytes), "github.com/aledsdavies/devcmd") {
				return workingDir
			}
		}
	}

	// Try going up directories to find the devcmd project
	currentDir := workingDir
	for i := 0; i < 5; i++ { // Limit search depth
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // reached root
		}

		goModPath := filepath.Join(parentDir, "go.mod")
		if goModBytes, err := os.ReadFile(goModPath); err == nil {
			if strings.Contains(string(goModBytes), "github.com/aledsdavies/devcmd") {
				return parentDir
			}
		}
		currentDir = parentDir
	}

	// Fallback to relative path
	return "../../"
}

// toCamelCase converts a command name to camelCase for variable naming
// Examples: "build" -> "build", "test-all" -> "testAll", "dev_flow" -> "devFlow"
func toCamelCase(name string) string {
	// Handle different separators: hyphens, underscores, and spaces
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	if len(parts) == 0 {
		return name
	}

	// First part stays lowercase, subsequent parts get title case
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += capitalizeFirst(parts[i])
	}

	return result
}

// capitalizeFirst capitalizes the first letter of a string (replacement for deprecated strings.Title)
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// trackVariableUsage recursively tracks which variables are used in command content
func (e *Engine) trackVariableUsage(content ast.CommandContent, usedVars map[string]bool) {
	switch c := content.(type) {
	case *ast.ShellContent:
		for _, part := range c.Parts {
			if funcDec, ok := part.(*ast.FunctionDecorator); ok {
				if funcDec.Name == "var" && len(funcDec.Args) == 1 {
					if ident, ok := funcDec.Args[0].Value.(*ast.Identifier); ok {
						usedVars[ident.Name] = true
					}
				}
			}
		}
	case *ast.BlockDecorator:
		for _, item := range c.Content {
			e.trackVariableUsage(item, usedVars)
		}
	case *ast.PatternDecorator:
		for _, pattern := range c.Patterns {
			for _, cmd := range pattern.Commands {
				e.trackVariableUsage(cmd, usedVars)
			}
		}
	}
}

// trackVariableUsageInBody tracks variable usage in a command body
func (e *Engine) trackVariableUsageInBody(body *ast.CommandBody, usedVars map[string]bool) {
	for _, content := range body.Content {
		e.trackVariableUsage(content, usedVars)
	}
}

// generateShellCommandExpression generates a Go string expression for a shell command with variable expansion
func (e *Engine) generateShellCommandExpression(content *ast.ShellContent) (string, error) {
	// Build Go expression parts for the command (similar to GenerateShellCodeForTemplate)
	var goExprParts []string

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			// Plain text - add as quoted string
			goExprParts = append(goExprParts, fmt.Sprintf("%q", p.Text))

		case *ast.FunctionDecorator:
			// Let the decorator handle its own expansion logic
			decorator, err := decorators.GetFunction(p.Name)
			if err != nil {
				return "", fmt.Errorf("function decorator @%s not found in shell command: %w", p.Name, err)
			}

			// Expand the decorator in generator mode to get Go code
			result := decorator.Expand(e.ctx, p.Args)
			if result.Error != nil {
				return "", fmt.Errorf("decorator @%s expansion failed: %w", p.Name, result.Error)
			}

			// Use the decorator-generated code
			if codeStr, ok := result.Data.(string); ok {
				goExprParts = append(goExprParts, codeStr)
			} else {
				return "", fmt.Errorf("decorator @%s returned invalid data type for shell generation: %T", p.Name, result.Data)
			}

		default:
			return "", fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	// Build the Go expression for the command
	if len(goExprParts) == 1 {
		return goExprParts[0], nil
	} else {
		return strings.Join(goExprParts, " + "), nil
	}
}

// extractShellCommand extracts the raw shell command string from AST ShellContent
func (e *Engine) extractShellCommand(shellContent *ast.ShellContent) string {
	var command strings.Builder
	for _, part := range shellContent.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			command.WriteString(p.Text)
		case *ast.FunctionDecorator:
			// Let the decorator handle its own expansion logic
			decorator, err := decorators.GetFunction(p.Name)
			if err != nil {
				// For now, show the error instead of silently skipping
				command.WriteString(fmt.Sprintf("[ERROR: %v]", err))
				continue
			}

			// Expand the decorator in interpreter mode to get actual value
			result := decorator.Expand(e.ctx, p.Args)
			if result.Error != nil {
				// For now, show the error instead of silently skipping
				command.WriteString(fmt.Sprintf("[ERROR: %v]", result.Error))
				continue
			}

			// Use the decorator-generated value
			if value, ok := result.Data.(string); ok {
				command.WriteString(value)
			}
		}
	}
	return strings.TrimSpace(command.String())
}

// analyzeCommands groups watch/stop commands and separates regular commands
func (e *Engine) analyzeCommands(commands []ast.CommandDecl) CommandGroups {
	groups := CommandGroups{
		RegularCommands: []*ast.CommandDecl{},
		ProcessGroups:   []ProcessGroup{},
	}

	// Track watch/stop commands by identifier
	processMap := make(map[string]ProcessGroup)

	for i, cmd := range commands {
		switch cmd.Type {
		case ast.WatchCommand:
			// Watch command - use the name as identifier
			identifier := cmd.Name
			group := processMap[identifier]
			group.Identifier = identifier
			group.WatchCommand = &commands[i]
			processMap[identifier] = group
		case ast.StopCommand:
			// Stop command - use the name as identifier
			identifier := cmd.Name
			group := processMap[identifier]
			group.Identifier = identifier
			group.StopCommand = &commands[i]
			processMap[identifier] = group
		default:
			// Regular command
			groups.RegularCommands = append(groups.RegularCommands, &commands[i])
		}
	}

	// Convert map to slice
	for _, group := range processMap {
		groups.ProcessGroups = append(groups.ProcessGroups, group)
	}

	return groups
}

// setupTemplateFunctions sets up template functions for the execution context
func (e *Engine) setupTemplateFunctions() {
	templateFuncs := map[string]interface{}{
		"generateShellCode": func(content ast.CommandContent) string {
			// Use the execution context's clean shell code generation
			switch c := content.(type) {
			case *ast.ShellContent:
				code, err := e.ctx.GenerateShellCodeForTemplate(c)
				if err != nil {
					return fmt.Sprintf("return fmt.Errorf(\"shell generation error: %v\")", err)
				}
				return code
			default:
				return fmt.Sprintf("return fmt.Errorf(\"unsupported command content type: %T\")", content)
			}
		},
	}
	e.ctx.SetTemplateFunctions(templateFuncs)
}

// getDevcmdVersion attempts to determine the current devcmd version for go.mod generation
func (e *Engine) getDevcmdVersion() string {
	// Try to get version from build info (when built with go install or go build)
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if we can find our own module version
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}

		// Look for version info in build settings (git commit, etc.)
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
				// Use pseudo-version format with commit hash
				return fmt.Sprintf("v0.0.0-dev-%s", setting.Value[:7])
			}
		}
	}

	// Try to get version from git (if we're in a git repository)
	if gitVersion := e.tryGetGitVersion(); gitVersion != "" {
		return gitVersion
	}

	// Fallback to development version
	return "v0.0.0-dev"
}

// tryGetGitVersion attempts to get version info from git
func (e *Engine) tryGetGitVersion() string {
	// Try to get current commit hash
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	if output, err := cmd.Output(); err == nil {
		commit := strings.TrimSpace(string(output))
		if commit != "" {
			return fmt.Sprintf("v0.0.0-dev-%s", commit)
		}
	}

	return ""
}

// Main CLI template - replaces all the fragile WriteString calls
const mainCLITemplate = `package main

import (
	{{range .StandardImports}}"{{.}}"
	{{end}}{{range .ThirdPartyImports}}"{{.}}"
	{{end}}
)

func main() {
	ctx := context.Background()

	// Variables
	{{range .Variables}}{{if .Used}}{{.Name}} := {{.Value}}
	{{end}}{{end}}

	// Global flags for dry-run mode
	var dryRun bool
	var noColor bool

	rootCmd := &cobra.Command{
		Use:   "cli",
		Short: "Generated CLI from devcmd",
	}
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show execution plan without running commands")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output in dry-run mode")

	{{range .Commands}}
	// Command: {{.Name}}
	{{.FunctionName}} := func(cmd *cobra.Command, args []string) {
		if dryRun {
			// Execute in plan mode using embedded execution plan
			{{if .ExecutionPlan}}
			if noColor {
				fmt.Print({{.ExecutionPlanNoColor}})
			} else {
				fmt.Print({{.ExecutionPlan}})
			}
			{{else}}fmt.Printf("(No plan available)\n"){{end}}
			return
		}
		
		// Normal execution
		{{.ExecutionCode}}
	}

	{{.CommandName}} := &cobra.Command{
		Use:   "{{.Name}}",
		Run:   {{.FunctionName}},
	}
	rootCmd.AddCommand({{.CommandName}})
	{{end}}

	{{range .ProcessGroups}}
	// Process management for {{.Identifier}}
	{{.FunctionName}}Run := func(cmd *cobra.Command, args []string) {
		if dryRun {
			// Execute in plan mode using embedded execution plan
			{{if .WatchExecutionPlan}}
			if noColor {
				fmt.Print({{.WatchExecutionPlanNoColor}})
			} else {
				fmt.Print({{.WatchExecutionPlan}})
			}
			{{else}}fmt.Printf("(No plan available)\n"){{end}}
			return
		}
		
		// Process management with PID tracking and log files
		processName := "{{.Identifier}}"
		pidFile := filepath.Join(os.TempDir(), processName+".pid")
		logFile := filepath.Join(os.TempDir(), processName+".log")
		
		// Check if process is already running
		if pidBytes, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes))); err == nil {
				if process, err := os.FindProcess(pid); err == nil {
					// Send signal 0 to check if process exists
					if err := process.Signal(syscall.Signal(0)); err == nil {
						fmt.Printf("Process %s is already running (PID: %d)\n", processName, pid)
						return
					}
				}
			}
		}
		
		// Create log file
		logFileHandle, err := os.Create(logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create log file: %v\n", err)
			return
		}
		defer logFileHandle.Close()
		
		// Execute the watch command with full decorator support
		// Redirect stdout/stderr to log file for this execution
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		os.Stdout = logFileHandle
		os.Stderr = logFileHandle
		
		// Execute as a background goroutine to simulate process behavior
		// while allowing decorators to work properly
		go func() {
			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
				logFileHandle.Close()
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "Watch command panic: %v\n", r)
				}
			}()
			
			// Execute the full command with decorators
			{{.WatchExecutionCode}}
		}()
		
		// Use current process PID since we're running as goroutines
		// Note: This is a simplified process management approach for decorator support
		pid := os.Getpid()
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write PID file: %v\n", err)
			return
		}
		
		// Restore stdout/stderr immediately for the main process
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		
		fmt.Printf("Started %s process (PID: %d)\n", processName, pid)
		fmt.Printf("Logs: %s\n", logFile)
	}

	{{.CommandName}} := &cobra.Command{
		Use:   "{{.Identifier}}",
		Short: "Manage {{.Identifier}} process",
		{{if .WatchExecutionCode}}Run:   {{.FunctionName}}Run, // Default action is to run{{end}}
	}

	// Run subcommand (explicit)
	{{.FunctionName}}RunCmd := &cobra.Command{
		Use:   "run",
		Short: "Start {{.Identifier}} process (explicit)",
		Run:   {{.FunctionName}}Run,
	}
	{{.CommandName}}.AddCommand({{.FunctionName}}RunCmd)

	// Stop subcommand
	{{.FunctionName}}Stop := func(cmd *cobra.Command, args []string) {
		if dryRun {
			// Execute in plan mode using embedded execution plan
			{{if .StopExecutionPlan}}
			if noColor {
				fmt.Print({{.StopExecutionPlanNoColor}})
			} else {
				fmt.Print({{.StopExecutionPlan}})
			}
			{{else}}fmt.Printf("(No plan available)\n"){{end}}
			return
		}
		
		// Process management with PID tracking
		processName := "{{.Identifier}}"
		pidFile := filepath.Join(os.TempDir(), processName+".pid")
		
		// Read PID from file
		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Printf("Process %s is not running (no PID file found)\n", processName)
			return
		}
		
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid PID in file: %v\n", err)
			return
		}
		
		// Find and kill the process
		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to find process %d: %v\n", pid, err)
			return
		}
		
		{{if .HasCustomStop}}
		// Custom stop command (also terminate the original process)
		{{if .StopCommandString}}cmdStr := {{.StopCommandString}}
		stopCmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
		if err := stopCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Custom stop command failed: %v\n", err)
		}{{else}}{{.StopExecutionCode}}{{end}}
		
		// Also terminate the original process
		if err := process.Signal(syscall.SIGTERM); err != nil {
			// Try SIGKILL if SIGTERM fails
			process.Signal(syscall.SIGKILL)
		}
		{{else}}
		// Default stop: kill the process
		if err := process.Signal(syscall.SIGTERM); err != nil {
			// Try SIGKILL if SIGTERM fails
			if err := process.Signal(syscall.SIGKILL); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to kill process %d: %v\n", pid, err)
				return
			}
		}
		{{end}}
		
		// Clean up PID file
		if err := os.Remove(pidFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove PID file: %v\n", err)
		}
		
		fmt.Printf("Stopped %s process (PID: %d)\n", processName, pid)
	}

	{{.FunctionName}}StopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop {{.Identifier}} process",
		Run:   {{.FunctionName}}Stop,
	}
	{{.CommandName}}.AddCommand({{.FunctionName}}StopCmd)

	// Status subcommand
	{{.FunctionName}}Status := func(cmd *cobra.Command, args []string) {
		if dryRun {
			// Execute in plan mode - status commands use simple default plan
			fmt.Printf("=== Execution Plan ===\n")
			fmt.Printf("Process: {{.Identifier}} (status)\n")
			fmt.Printf("├── Check PID file and process status\n")
			return
		}
		
		// Process management status checking
		processName := "{{.Identifier}}"
		pidFile := filepath.Join(os.TempDir(), processName+".pid")
		logFile := filepath.Join(os.TempDir(), processName+".log")
		
		// Check if PID file exists
		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			fmt.Printf("Process %s is not running (no PID file)\n", processName)
			return
		}
		
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
		if err != nil {
			fmt.Printf("Process %s has invalid PID file\n", processName)
			return
		}
		
		// Check if process is actually running
		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("Process %s (PID: %d) not found\n", processName, pid)
			return
		}
		
		// Send signal 0 to check if process exists without affecting it
		if err := process.Signal(syscall.Signal(0)); err != nil {
			fmt.Printf("Process %s (PID: %d) is not running\n", processName, pid)
			// Clean up stale PID file
			os.Remove(pidFile)
			return
		}
		
		fmt.Printf("Process %s is running (PID: %d)\n", processName, pid)
		fmt.Printf("Log file: %s\n", logFile)
	}

	{{.FunctionName}}StatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show {{.Identifier}} process status",
		Run:   {{.FunctionName}}Status,
	}
	{{.CommandName}}.AddCommand({{.FunctionName}}StatusCmd)

	// Logs subcommand
	{{.FunctionName}}Logs := func(cmd *cobra.Command, args []string) {
		if dryRun {
			// Execute in plan mode - logs commands use simple default plan
			fmt.Printf("=== Execution Plan ===\n")
			fmt.Printf("Process: {{.Identifier}} (logs)\n")
			fmt.Printf("├── Read and display log file\n")
			return
		}
		
		// Process management log reading
		processName := "{{.Identifier}}"
		logFile := filepath.Join(os.TempDir(), processName+".log")
		
		// Check if log file exists
		if _, err := os.Stat(logFile); err != nil {
			fmt.Printf("No log file found for process %s\n", processName)
			fmt.Printf("Expected location: %s\n", logFile)
			return
		}
		
		// Read and display log file contents
		logContent, err := os.ReadFile(logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read log file: %v\n", err)
			return
		}
		
		if len(logContent) == 0 {
			fmt.Printf("Log file for %s is empty\n", processName)
			return
		}
		
		fmt.Printf("=== Logs for %s ===\n", processName)
		fmt.Print(string(logContent))
		if !strings.HasSuffix(string(logContent), "\n") {
			fmt.Println() // Add newline if log doesn't end with one
		}
	}

	{{.FunctionName}}LogsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Show {{.Identifier}} process logs",
		Run:   {{.FunctionName}}Logs,
	}
	{{.CommandName}}.AddCommand({{.FunctionName}}LogsCmd)

	rootCmd.AddCommand({{.CommandName}})
	{{end}}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err, os.Stderr)
		os.Exit(1)
	}
}
`

// Template data structures
type CLITemplateData struct {
	StandardImports   []string
	ThirdPartyImports []string
	Variables         []VariableData
	Commands          []CommandData
	ProcessGroups     []ProcessGroupData
}

type VariableData struct {
	Name  string
	Value string
	Used  bool
}

type CommandData struct {
	Name                 string
	FunctionName         string
	CommandName          string
	ExecutionCode        string
	ExecutionPlan        string // Embedded execution plan for dry-run mode (with colors)
	ExecutionPlanNoColor string // Embedded execution plan for dry-run mode (no colors)
}

type ProcessGroupData struct {
	Identifier                string
	FunctionName              string
	CommandName               string
	RunFunctionName           string
	HasCustomStop             bool
	WatchExecutionCode        string
	StopExecutionCode         string
	WatchExecutionPlan        string // Embedded execution plan for watch command dry-run (with colors)
	WatchExecutionPlanNoColor string // Embedded execution plan for watch command dry-run (no colors)
	StopExecutionPlan         string // Embedded execution plan for stop command dry-run (with colors)
	StopExecutionPlanNoColor  string // Embedded execution plan for stop command dry-run (no colors)
	WatchCommandString        string // Raw shell command for process management
	StopCommandString         string // Raw shell command for stop process management
}

// generateCodeWithTemplate uses a template-based approach instead of fragile WriteString calls
func (e *Engine) generateCodeWithTemplate(program *ast.Program) (*GenerationResult, error) {
	// Set execution mode to generator
	ctx := e.ctx.WithMode(execution.GeneratorMode)

	// Initialize variables in the context first (critical for @var decorator)
	if err := ctx.InitializeVariables(); err != nil {
		return nil, fmt.Errorf("failed to initialize variables: %w", err)
	}

	// Analyze commands to separate regular commands from process management early
	commandGroups := e.analyzeCommands(program.Commands)

	// Initialize the result
	result := &GenerationResult{
		Code:              strings.Builder{},
		GoMod:             strings.Builder{},
		StandardImports:   make(map[string]bool),
		ThirdPartyImports: make(map[string]bool),
		GoModules:         make(map[string]string),
	}

	// Add basic imports needed for generated CLI
	result.AddStandardImport("context")
	result.AddStandardImport("fmt")
	result.AddStandardImport("os")
	result.AddStandardImport("os/exec")

	// Add process management imports if we have process groups
	if len(commandGroups.ProcessGroups) > 0 {
		result.AddStandardImport("strings") // Needed for string operations in process management
		result.AddStandardImport("path/filepath")
		result.AddStandardImport("strconv")
		result.AddStandardImport("syscall")
		// io/ioutil and time are not used in current template implementation
	}

	// Collect imports from all decorators used in the program
	if err := e.collectDecoratorImports(program, result); err != nil {
		return nil, fmt.Errorf("failed to collect decorator imports: %w", err)
	}

	// Add cobra for CLI generation (always needed for generated CLIs)
	result.AddThirdPartyImport("github.com/spf13/cobra")

	// Convert import maps to slices for template
	var standardImports []string
	for imp := range result.StandardImports {
		standardImports = append(standardImports, imp)
	}
	var thirdPartyImports []string
	for imp := range result.ThirdPartyImports {
		thirdPartyImports = append(thirdPartyImports, imp)
	}

	// Prepare template data
	templateData := CLITemplateData{
		StandardImports:   standardImports,
		ThirdPartyImports: thirdPartyImports,
		Variables:         []VariableData{},
		Commands:          []CommandData{},
		ProcessGroups:     []ProcessGroupData{},
	}

	// Track which variables are used across all commands
	usedVariables := make(map[string]bool)
	for _, cmd := range program.Commands {
		e.trackVariableUsageInBody(&cmd.Body, usedVariables)
	}

	// Add variables to template data, only including used ones
	for _, variable := range program.Variables {
		value, err := e.resolveVariableValue(variable.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		templateData.Variables = append(templateData.Variables, VariableData{
			Name:  variable.Name,
			Value: fmt.Sprintf("%q", value), // Quote the string value
			Used:  usedVariables[variable.Name],
		})
	}

	// Add regular commands to template data by executing them in generator mode
	for _, cmd := range commandGroups.RegularCommands {
		// Execute the command content using the execution context to get proper template-generated code
		var executionCode strings.Builder

		for _, content := range cmd.Body.Content {
			// Execute each piece of content and collect the generated code
			switch c := content.(type) {
			case *ast.ShellContent:
				execResult := ctx.ExecuteShell(c)
				if execResult.Error != nil {
					return nil, fmt.Errorf("failed to generate shell code for command %s: %w", cmd.Name, execResult.Error)
				}
				if code, ok := execResult.Data.(string); ok {
					executionCode.WriteString(code)
					executionCode.WriteString("\n")
				}

			case *ast.BlockDecorator:
				// Add decorator marker comment (expected by tests)
				executionCode.WriteString(fmt.Sprintf("\t\t// Block decorator: @%s\n", c.Name))

				// Collect imports for this decorator
				if err := e.addDecoratorImports("block", c.Name, result); err != nil {
					return nil, fmt.Errorf("failed to collect imports for @%s in command %s: %w", c.Name, cmd.Name, err)
				}

				// Execute the decorator to get generated code
				blockDecorator, err := decorators.GetBlock(c.Name)
				if err != nil {
					return nil, fmt.Errorf("block decorator @%s not found for command %s: %w", c.Name, cmd.Name, err)
				}
				decoratorResult := blockDecorator.Execute(ctx, c.Args, c.Content)
				if decoratorResult.Error != nil {
					return nil, fmt.Errorf("@%s decorator execution failed in command %s: %w", c.Name, cmd.Name, decoratorResult.Error)
				}
				if code, ok := decoratorResult.Data.(string); ok {
					executionCode.WriteString(code)
					executionCode.WriteString("\n")
				}

			case *ast.PatternDecorator:
				// Add decorator marker comment (expected by tests)
				executionCode.WriteString(fmt.Sprintf("\t\t// Pattern decorator: @%s\n", c.Name))

				// Collect imports for this decorator
				if err := e.addDecoratorImports("pattern", c.Name, result); err != nil {
					return nil, fmt.Errorf("failed to collect imports for @%s in command %s: %w", c.Name, cmd.Name, err)
				}

				// Also collect imports from pattern branches recursively
				for _, pattern := range c.Patterns {
					if err := e.collectDecoratorImportsFromContent(pattern.Commands, result); err != nil {
						return nil, fmt.Errorf("failed to collect imports from pattern branches in command %s: %w", cmd.Name, err)
					}
				}

				// Execute the pattern decorator to get generated code
				patternDecorator, err := decorators.GetPattern(c.Name)
				if err != nil {
					return nil, fmt.Errorf("pattern decorator @%s not found for command %s: %w", c.Name, cmd.Name, err)
				}
				decoratorResult := patternDecorator.Execute(ctx, c.Args, c.Patterns)
				if decoratorResult.Error != nil {
					return nil, fmt.Errorf("@%s decorator execution failed in command %s: %w", c.Name, cmd.Name, decoratorResult.Error)
				}
				if code, ok := decoratorResult.Data.(string); ok {
					executionCode.WriteString(code)
					executionCode.WriteString("\n")
				}

			default:
				return nil, fmt.Errorf("unsupported command content type %T in command %s", content, cmd.Name)
			}
		}

		// Generate execution plan for this command (both colored and no-color versions)
		executionPlan := ""
		executionPlanNoColor := ""
		if plan, err := e.ExecuteCommandPlan(cmd); err == nil {
			executionPlan = fmt.Sprintf("%q", plan.String())
			executionPlanNoColor = fmt.Sprintf("%q", plan.StringNoColor())
		}

		templateData.Commands = append(templateData.Commands, CommandData{
			Name:                 cmd.Name,
			FunctionName:         toCamelCase(cmd.Name),
			CommandName:          toCamelCase(cmd.Name) + "Cmd",
			ExecutionCode:        executionCode.String(),
			ExecutionPlan:        executionPlan,
			ExecutionPlanNoColor: executionPlanNoColor,
		})
	}

	// Process groups (watch/stop commands)
	for _, group := range commandGroups.ProcessGroups {
		identifier := group.Identifier
		processData := ProcessGroupData{
			Identifier:      identifier,
			FunctionName:    toCamelCase(identifier),
			CommandName:     toCamelCase(identifier) + "Cmd",
			RunFunctionName: toCamelCase(identifier) + "Run",
			HasCustomStop:   group.StopCommand != nil,
		}

		// Generate watch command execution code and extract raw shell commands
		watchCommandString := ""
		if group.WatchCommand != nil {
			var watchCode strings.Builder
			for _, content := range group.WatchCommand.Body.Content {
				switch c := content.(type) {
				case *ast.ShellContent:
					// Extract raw shell command for process management
					if watchCommandString == "" { // Use first shell command for process management
						watchCommandString = e.extractShellCommand(c)
					}

					execResult := ctx.ExecuteShell(c)
					if execResult.Error != nil {
						return nil, fmt.Errorf("failed to generate shell code for watch command %s: %w", identifier, execResult.Error)
					}
					if code, ok := execResult.Data.(string); ok {
						watchCode.WriteString(code)
					}
				case *ast.BlockDecorator:
					if err := e.addDecoratorImports("block", c.Name, result); err != nil {
						return nil, fmt.Errorf("failed to collect imports for @%s in watch command %s: %w", c.Name, identifier, err)
					}
					blockDecorator, err := decorators.GetBlock(c.Name)
					if err != nil {
						return nil, fmt.Errorf("block decorator @%s not found for watch command %s: %w", c.Name, identifier, err)
					}
					decoratorResult := blockDecorator.Execute(ctx, c.Args, c.Content)
					if decoratorResult.Error != nil {
						return nil, fmt.Errorf("@%s decorator execution failed in watch command %s: %w", c.Name, identifier, decoratorResult.Error)
					}
					if code, ok := decoratorResult.Data.(string); ok {
						watchCode.WriteString(code)
					}
				default:
					return nil, fmt.Errorf("unsupported command content type %T in watch command %s", content, identifier)
				}
			}
			processData.WatchExecutionCode = watchCode.String()
		}

		// Generate stop command execution code and extract shell commands
		stopCommandString := ""
		if group.StopCommand != nil {
			var stopCode strings.Builder
			for _, content := range group.StopCommand.Body.Content {
				switch c := content.(type) {
				case *ast.ShellContent:
					// Generate shell command string for process management (with proper variable expansion)
					if stopCommandString == "" { // Use first shell command for process management
						if cmdExpr, err := e.generateShellCommandExpression(c); err == nil {
							stopCommandString = cmdExpr
						} else {
							stopCommandString = e.extractShellCommand(c) // fallback to raw command
						}
					}

					execResult := ctx.ExecuteShell(c)
					if execResult.Error != nil {
						return nil, fmt.Errorf("failed to generate shell code for stop command %s: %w", identifier, execResult.Error)
					}
					if code, ok := execResult.Data.(string); ok {
						stopCode.WriteString(code)
					}
				case *ast.BlockDecorator:
					if err := e.addDecoratorImports("block", c.Name, result); err != nil {
						return nil, fmt.Errorf("failed to collect imports for @%s in stop command %s: %w", c.Name, identifier, err)
					}
					blockDecorator, err := decorators.GetBlock(c.Name)
					if err != nil {
						return nil, fmt.Errorf("block decorator @%s not found for stop command %s: %w", c.Name, identifier, err)
					}
					decoratorResult := blockDecorator.Execute(ctx, c.Args, c.Content)
					if decoratorResult.Error != nil {
						return nil, fmt.Errorf("@%s decorator execution failed in stop command %s: %w", c.Name, identifier, decoratorResult.Error)
					}
					if code, ok := decoratorResult.Data.(string); ok {
						stopCode.WriteString(code)
					}
				default:
					return nil, fmt.Errorf("unsupported command content type %T in stop command %s", content, identifier)
				}
			}
			processData.StopExecutionCode = stopCode.String()
		}

		// Generate execution plans for watch and stop commands (both colored and no-color versions)
		watchExecutionPlan := ""
		watchExecutionPlanNoColor := ""
		if group.WatchCommand != nil {
			if plan, err := e.ExecuteCommandPlan(group.WatchCommand); err == nil {
				watchExecutionPlan = fmt.Sprintf("%q", plan.String())
				watchExecutionPlanNoColor = fmt.Sprintf("%q", plan.StringNoColor())
			}
		}

		stopExecutionPlan := ""
		stopExecutionPlanNoColor := ""
		if group.StopCommand != nil {
			if plan, err := e.ExecuteCommandPlan(group.StopCommand); err == nil {
				stopExecutionPlan = fmt.Sprintf("%q", plan.String())
				stopExecutionPlanNoColor = fmt.Sprintf("%q", plan.StringNoColor())
			}
		} else {
			// Default stop plan (both versions are the same since no colors)
			defaultPlan := fmt.Sprintf("└─ pkill -f '%s'", identifier)
			stopExecutionPlan = fmt.Sprintf("%q", defaultPlan)
			stopExecutionPlanNoColor = fmt.Sprintf("%q", defaultPlan)
		}

		processData.WatchExecutionPlan = watchExecutionPlan
		processData.WatchExecutionPlanNoColor = watchExecutionPlanNoColor
		processData.StopExecutionPlan = stopExecutionPlan
		processData.StopExecutionPlanNoColor = stopExecutionPlanNoColor
		processData.WatchCommandString = watchCommandString
		processData.StopCommandString = stopCommandString

		templateData.ProcessGroups = append(templateData.ProcessGroups, processData)
	}

	// Update template data with collected imports (convert maps to slices)
	standardImports = []string{}
	for imp := range result.StandardImports {
		standardImports = append(standardImports, imp)
	}
	thirdPartyImports = []string{}
	for imp := range result.ThirdPartyImports {
		thirdPartyImports = append(thirdPartyImports, imp)
	}
	templateData.StandardImports = standardImports
	templateData.ThirdPartyImports = thirdPartyImports

	// Execute the template
	tmpl, err := template.New("mainCLI").Parse(mainCLITemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse main CLI template: %w", err)
	}

	var codeBuilder strings.Builder
	if err := tmpl.Execute(&codeBuilder, templateData); err != nil {
		return nil, fmt.Errorf("failed to execute main CLI template: %w", err)
	}

	// Set the generated code
	result.Code.WriteString(codeBuilder.String())

	// Generate go.mod
	if err := e.generateGoMod(result); err != nil {
		return nil, fmt.Errorf("failed to generate go.mod: %w", err)
	}

	return result, nil
}
