package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// Known decorators that devcmd supports
var supportedDecorators = map[string]bool{
	"sh":       true, // Shell command execution
	"parallel": true, // Parallel execution
}

// TemplateData represents preprocessed data for template generation
type TemplateData struct {
	PackageName      string
	Imports          []string
	HasProcessMgmt   bool
	Commands         []TemplateCommand
	ProcessMgmtFuncs []string
}

// TemplateCommand represents a command ready for template generation
type TemplateCommand struct {
	Name            string // Original command name
	FunctionName    string // Sanitized Go function name
	GoCase          string // Case statement value
	Type            string // "regular", "watch-stop", "watch-only", "stop-only"
	ShellCommand    string // For regular commands
	WatchCommand    string // For watch part of watch-stop commands
	StopCommand     string // For stop part of watch-stop commands
	IsBackground    bool   // For watch commands
	HelpDescription string // Description for help text
}

// TemplateRegistry holds all template components
type TemplateRegistry struct {
	templates map[string]string
}

// NewTemplateRegistry creates a new template registry with all components
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{
		templates: make(map[string]string),
	}
	registry.registerComponents()
	return registry
}

// registerComponents registers all template components
func (tr *TemplateRegistry) registerComponents() {
	// Core templates
	tr.templates["package"] = packageTemplate
	tr.templates["imports"] = importsTemplate
	tr.templates["process-types"] = processTypesTemplate
	tr.templates["process-registry"] = processRegistryTemplate
	tr.templates["cli-struct"] = cliStructTemplate
	tr.templates["main-function"] = mainFunctionTemplate

	// Command templates
	tr.templates["command-switch"] = commandSwitchTemplate
	tr.templates["help-function"] = helpFunctionTemplate
	tr.templates["status-function"] = statusFunctionTemplate
	tr.templates["command-functions"] = commandFunctionsTemplate

	// Command type implementations
	tr.templates["regular-command"] = regularCommandTemplate
	tr.templates["watch-stop-command"] = watchStopCommandTemplate
	tr.templates["watch-only-command"] = watchOnlyCommandTemplate
	tr.templates["stop-only-command"] = stopOnlyCommandTemplate

	// Process management templates
	tr.templates["process-mgmt-functions"] = processMgmtFunctionsTemplate
}

// GetTemplate returns a specific template component
func (tr *TemplateRegistry) GetTemplate(name string) (string, bool) {
	tmpl, exists := tr.templates[name]
	return tmpl, exists
}

// GetMasterTemplate returns the master template that composes all components
func (tr *TemplateRegistry) GetMasterTemplate() string {
	return masterTemplate
}

// GetAllTemplates returns all template components as a single string
func (tr *TemplateRegistry) GetAllTemplates() string {
	var parts []string

	// Add all component templates
	for _, tmpl := range tr.templates {
		parts = append(parts, tmpl)
	}

	// Add master template
	parts = append(parts, tr.GetMasterTemplate())

	return strings.Join(parts, "\n")
}

// PreprocessCommands converts parser commands into template-ready data
func PreprocessCommands(cf *parser.CommandFile) (*TemplateData, error) {
	if cf == nil {
		return nil, fmt.Errorf("command file cannot be nil")
	}

	data := &TemplateData{
		PackageName: "main",
		Imports:     []string{},
		Commands:    []TemplateCommand{},
	}

	// Group commands by name to find watch/stop pairs
	commandGroups := make(map[string][]parser.Command)
	for _, cmd := range cf.Commands {
		commandGroups[cmd.Name] = append(commandGroups[cmd.Name], cmd)
	}

	// Validate decorators before processing
	if err := validateDecorators(cf.Commands); err != nil {
		return nil, err
	}

	// Determine what features we need
	hasWatchCommands := false
	hasRegularCommands := len(cf.Commands) > 0
	for _, cmd := range cf.Commands {
		if cmd.IsWatch {
			hasWatchCommands = true
			break
		}
	}
	data.HasProcessMgmt = hasWatchCommands

	// Set up minimal imports - only include what we actually need
	if hasRegularCommands {
		data.Imports = []string{
			"fmt",
			"os",
		}

		// Only add os/exec if we have actual commands
		if len(cf.Commands) > 0 {
			data.Imports = append(data.Imports, "os/exec")
		}

		if hasWatchCommands {
			additionalImports := []string{
				"encoding/json",
				"io",
				"os/signal",
				"path/filepath",
				"strings",
				"syscall",
				"time",
			}
			data.Imports = append(data.Imports, additionalImports...)
		}
	} else {
		// Minimal imports for empty command files
		data.Imports = []string{"fmt", "os"}
	}

	// Sort imports for consistent output
	sort.Strings(data.Imports)

	// Process command groups
	for name, commands := range commandGroups {
		templateCmd, err := processCommandGroup(name, commands)
		if err != nil {
			return nil, fmt.Errorf("failed to process command group %s: %w", name, err)
		}
		data.Commands = append(data.Commands, templateCmd)
	}

	// Add process management functions if needed
	if hasWatchCommands {
		data.ProcessMgmtFuncs = []string{
			"showStatus",
			"showLogs",
			"stopCommand",
			"runInBackground",
		}
	}

	return data, nil
}

// validateDecorators checks that all decorators used are supported
func validateDecorators(commands []parser.Command) error {
	for _, cmd := range commands {
		if cmd.IsBlock {
			if err := validateBlockDecorators(cmd.Block, cmd.Name, cmd.Line); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateBlockDecorators validates decorators in block statements
func validateBlockDecorators(statements []parser.BlockStatement, cmdName string, cmdLine int) error {
	for _, stmt := range statements {
		if stmt.IsDecorated {
			if !supportedDecorators[stmt.Decorator] {
				return fmt.Errorf("unsupported decorator '@%s' in command '%s' at line %d. Supported decorators: %s",
					stmt.Decorator, cmdName, cmdLine, getSupportedDecoratorsString())
			}

			// Validate decorator usage
			switch stmt.Decorator {
			case "parallel":
				if stmt.DecoratorType != "block" {
					return fmt.Errorf("@parallel decorator must be used with block syntax in command '%s' at line %d. Use: @parallel: { command1; command2 }",
						cmdName, cmdLine)
				}
			case "sh":
				if stmt.DecoratorType != "function" && stmt.DecoratorType != "simple" {
					return fmt.Errorf("@sh decorator must be used with function or simple syntax in command '%s' at line %d. Use: @sh(command) or @sh: command",
						cmdName, cmdLine)
				}
			}

			// Recursively validate nested blocks
			if stmt.DecoratorType == "block" && len(stmt.DecoratedBlock) > 0 {
				if err := validateBlockDecorators(stmt.DecoratedBlock, cmdName, cmdLine); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getSupportedDecoratorsString returns a formatted string of supported decorators
func getSupportedDecoratorsString() string {
	var decorators []string
	for decorator := range supportedDecorators {
		decorators = append(decorators, "@"+decorator)
	}
	sort.Strings(decorators)
	return strings.Join(decorators, ", ")
}

// processCommandGroup processes a group of commands with the same name
func processCommandGroup(name string, commands []parser.Command) (TemplateCommand, error) {
	templateCmd := TemplateCommand{
		Name:         name,
		FunctionName: sanitizeFunctionName(name),
		GoCase:       name,
	}

	var watchCmd, stopCmd *parser.Command
	var regularCmd *parser.Command

	// Categorize commands in the group
	for i, cmd := range commands {
		if cmd.IsWatch {
			watchCmd = &commands[i]
		} else if cmd.IsStop {
			stopCmd = &commands[i]
		} else {
			regularCmd = &commands[i]
		}
	}

	// Determine command type and structure
	if regularCmd != nil {
		// Regular command (no watch/stop)
		templateCmd.Type = "regular"
		shellCmd, err := buildShellCommand(*regularCmd)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build shell command for '%s': %w", name, err)
		}
		templateCmd.ShellCommand = shellCmd
		templateCmd.HelpDescription = name
	} else if watchCmd != nil && stopCmd != nil {
		// Watch/stop pair
		templateCmd.Type = "watch-stop"
		watchShell, err := buildShellCommand(*watchCmd)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build watch command for '%s': %w", name, err)
		}
		stopShell, err := buildShellCommand(*stopCmd)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build stop command for '%s': %w", name, err)
		}
		templateCmd.WatchCommand = watchShell
		templateCmd.StopCommand = stopShell
		templateCmd.IsBackground = true
		templateCmd.HelpDescription = fmt.Sprintf("%s start|stop|logs", name)
	} else if watchCmd != nil {
		// Watch only
		templateCmd.Type = "watch-only"
		watchShell, err := buildShellCommand(*watchCmd)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build watch command for '%s': %w", name, err)
		}
		templateCmd.WatchCommand = watchShell
		templateCmd.IsBackground = true
		templateCmd.HelpDescription = fmt.Sprintf("%s start|stop|logs", name)
	} else if stopCmd != nil {
		// Stop only (unusual, but handle it)
		templateCmd.Type = "stop-only"
		stopShell, err := buildShellCommand(*stopCmd)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build stop command for '%s': %w", name, err)
		}
		templateCmd.StopCommand = stopShell
		templateCmd.HelpDescription = fmt.Sprintf("%s stop", name)
	} else {
		return templateCmd, fmt.Errorf("no valid commands found in group %s", name)
	}

	return templateCmd, nil
}

// sanitizeFunctionName converts command names to valid Go function names
func sanitizeFunctionName(name string) string {
	// Capitalize first letter of each word
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			// Simple capitalize: uppercase first rune, lowercase rest
			runes := []rune(strings.ToLower(part))
			if len(runes) > 0 {
				runes[0] = unicode.ToUpper(runes[0])
			}
			result.WriteString(string(runes))
		}
	}

	funcName := result.String()
	if funcName == "" {
		funcName = "Command"
	}

	return "run" + funcName
}

// buildShellCommand constructs the shell command string from parser command
func buildShellCommand(cmd parser.Command) (string, error) {
	if cmd.IsBlock {
		return buildBlockCommand(cmd.Block, cmd.Name, cmd.Line)
	}
	return cmd.Command, nil
}

// buildBlockCommand handles block statements with decorator support
func buildBlockCommand(statements []parser.BlockStatement, cmdName string, cmdLine int) (string, error) {
	var parts []string

	for _, stmt := range statements {
		if stmt.IsDecorated {
			part, err := buildDecoratedStatement(stmt, cmdName, cmdLine)
			if err != nil {
				return "", err
			}
			if part != "" {
				parts = append(parts, part)
			}
		} else {
			// Regular command (no decorator)
			if stmt.Command != "" {
				parts = append(parts, stmt.Command)
			}
		}
	}

	return strings.Join(parts, "; "), nil
}

// buildDecoratedStatement handles different decorator types
func buildDecoratedStatement(stmt parser.BlockStatement, cmdName string, cmdLine int) (string, error) {
	switch stmt.Decorator {
	case "sh":
		// Shell command - pass through directly (no variable processing)
		if stmt.DecoratorType == "function" {
			// @sh(command) - command is in stmt.Command
			return stmt.Command, nil
		} else {
			// @sh:command - command is in stmt.Command
			return stmt.Command, nil
		}

	case "parallel":
		// Parallel execution - convert to background processes with &
		if stmt.DecoratorType == "block" {
			// @parallel: { cmd1; cmd2; } -> cmd1 &; cmd2 &; wait
			var parallelParts []string
			for _, nestedStmt := range stmt.DecoratedBlock {
				if nestedStmt.IsDecorated {
					// Handle nested decorators
					part, err := buildDecoratedStatement(nestedStmt, cmdName, cmdLine)
					if err != nil {
						return "", err
					}
					if part != "" {
						parallelParts = append(parallelParts, part+" &")
					}
				} else {
					// Regular command in parallel block
					if nestedStmt.Command != "" {
						parallelParts = append(parallelParts, nestedStmt.Command+" &")
					}
				}
			}
			// Add wait to synchronize all background processes
			if len(parallelParts) > 0 {
				parallelParts = append(parallelParts, "wait")
			}
			return strings.Join(parallelParts, "; "), nil
		}

	default:
		// This should not happen due to validation, but handle gracefully
		return "", fmt.Errorf("unsupported decorator '@%s' in command '%s' at line %d", stmt.Decorator, cmdName, cmdLine)
	}

	return stmt.Command, nil
}

// GenerateGo creates a Go CLI from a CommandFile using the composable template system
func GenerateGo(cf *parser.CommandFile) (string, error) {
	// Preprocess the command file into template-ready data
	data, err := PreprocessCommands(cf)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	// Create template registry and get all templates
	registry := NewTemplateRegistry()
	allTemplates := registry.GetAllTemplates()

	// Parse and execute template
	tmpl, err := template.New("go-cli").Parse(allTemplates)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "main", data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	result := buf.String()
	if len(result) == 0 {
		return "", fmt.Errorf("generated empty Go code")
	}

	return result, nil
}

// GenerateGoWithTemplate creates a Go CLI with a custom template (for testing)
func GenerateGoWithTemplate(cf *parser.CommandFile, templateStr string) (string, error) {
	if len(strings.TrimSpace(templateStr)) == 0 {
		return "", fmt.Errorf("template string cannot be empty")
	}

	// Preprocess the command file
	data, err := PreprocessCommands(cf)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	// Parse and execute custom template
	tmpl, err := template.New("custom").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GetTemplateComponent returns a specific template component by name
func GetTemplateComponent(name string) (string, error) {
	registry := NewTemplateRegistry()
	template, exists := registry.GetTemplate(name)
	if !exists {
		return "", fmt.Errorf("template component '%s' not found", name)
	}
	return template, nil
}

// GenerateComponentGo generates Go code using only specific template components
func GenerateComponentGo(cf *parser.CommandFile, componentNames []string) (string, error) {
	// Preprocess the command file into template-ready data
	data, err := PreprocessCommands(cf)
	if err != nil {
		return "", fmt.Errorf("failed to preprocess commands: %w", err)
	}

	registry := NewTemplateRegistry()
	var templateParts []string

	// Collect requested components
	for _, name := range componentNames {
		component, exists := registry.GetTemplate(name)
		if !exists {
			return "", fmt.Errorf("template component '%s' not found", name)
		}
		templateParts = append(templateParts, component)
	}

	// Add a simple execution template
	templateParts = append(templateParts, "{{template \"package\" .}}\n{{template \"imports\" .}}")

	allTemplates := strings.Join(templateParts, "\n")

	// Parse and execute template
	tmpl, err := template.New("component-cli").Parse(allTemplates)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
