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
	"var":      true, // Variable reference (compile-time expansion)
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

	// Create variable definitions map for expansion
	definitions := createDefinitionMap(cf.Definitions)

	// Group commands by name to find watch/stop pairs
	commandGroups := make(map[string][]parser.Command)
	for _, cmd := range cf.Commands {
		commandGroups[cmd.Name] = append(commandGroups[cmd.Name], cmd)
	}

	// Validate decorators before processing with source context
	if err := validateDecoratorsWithContext(cf.Commands, cf.Lines); err != nil {
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

	// Process command groups with variable expansion and enhanced error reporting
	for name, commands := range commandGroups {
		templateCmd, err := processCommandGroupWithContext(name, commands, definitions, cf.Lines)
		if err != nil {
			return nil, err
		}
		data.Commands = append(data.Commands, templateCmd)
	}

	// Sort commands for consistent output
	sort.Slice(data.Commands, func(i, j int) bool {
		return data.Commands[i].Name < data.Commands[j].Name
	})

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

// createDefinitionMap creates a map from variable definitions for quick lookup
func createDefinitionMap(definitions []parser.Definition) map[string]string {
	defMap := make(map[string]string)
	for _, def := range definitions {
		defMap[def.Name] = def.Value
	}
	return defMap
}

// validateDecoratorsWithContext validates decorators with source line context
func validateDecoratorsWithContext(commands []parser.Command, sourceLines []string) error {
	for _, cmd := range commands {
		if cmd.IsBlock {
			if err := validateBlockDecoratorsWithContext(cmd.Block, cmd.Name, cmd.Line, sourceLines); err != nil {
				return err
			}
		}
		// Also validate elements in simple commands
		if err := validateCommandElementsWithContext(cmd.Elements, cmd.Name, cmd.Line, sourceLines); err != nil {
			return err
		}
	}
	return nil
}

// validateCommandElementsWithContext validates decorators in command elements with source context
func validateCommandElementsWithContext(elements []parser.CommandElement, cmdName string, cmdLine int, sourceLines []string) error {
	for _, elem := range elements {
		if elem.IsDecorator() {
			decorator := elem.(*parser.DecoratorElement)
			if !supportedDecorators[decorator.Name] {
				return createValidationError(
					fmt.Sprintf("unsupported decorator '@%s'. Supported decorators: %s",
						decorator.Name, GetSupportedDecoratorsString()),
					cmdName, cmdLine, sourceLines)
			}

			// Add specific validation for decorator usage patterns
			switch decorator.Name {
			case "var":
				if decorator.Type != "function" {
					return createDecoratorError(
						"var", decorator.Type,
						"@var decorator must be used with function syntax",
						"Use: @var(VARIABLE_NAME)",
						cmdName, cmdLine, sourceLines)
				}
				if len(decorator.Args) == 0 {
					return createDecoratorError(
						"var", "function",
						"@var decorator requires a variable name",
						"Use: @var(VARIABLE_NAME)",
						cmdName, cmdLine, sourceLines)
				}
			case "sh":
				if decorator.Type != "function" {
					return createDecoratorError(
						"sh", decorator.Type,
						"@sh decorator must be used with function syntax",
						"Use: @sh(command)",
						cmdName, cmdLine, sourceLines)
				}
				// Check for invalid nested decorators in @sh
				if err := validateShDecoratorElementsWithContext(decorator.Args, cmdName, cmdLine, sourceLines); err != nil {
					return err
				}
			case "parallel":
				if decorator.Type == "function" {
					return createDecoratorError(
						"parallel", decorator.Type,
						"@parallel decorator cannot be used with function syntax",
						"Use block syntax: @parallel: { command1; command2 }",
						cmdName, cmdLine, sourceLines)
				}
			}

			// Recursively validate decorator arguments
			if err := validateCommandElementsWithContext(decorator.Args, cmdName, cmdLine, sourceLines); err != nil {
				return err
			}

			// Validate block decorators
			if len(decorator.Block) > 0 {
				if err := validateBlockDecoratorsWithContext(decorator.Block, cmdName, cmdLine, sourceLines); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateShDecoratorElementsWithContext checks for invalid nested decorators in @sh with source context
func validateShDecoratorElementsWithContext(elements []parser.CommandElement, cmdName string, cmdLine int, sourceLines []string) error {
	for _, elem := range elements {
		if elem.IsDecorator() {
			decorator := elem.(*parser.DecoratorElement)
			if decorator.Name == "sh" {
				return createDecoratorError(
					"sh", decorator.Type,
					"nested decorator '@sh' not allowed inside @sh",
					"Remove the nested @sh decorator",
					cmdName, cmdLine, sourceLines)
			} else if decorator.Name != "var" {
				return createDecoratorError(
					decorator.Name, decorator.Type,
					fmt.Sprintf("nested decorator '@%s' not allowed inside @sh", decorator.Name),
					"Only @var() is allowed inside @sh() decorators",
					cmdName, cmdLine, sourceLines)
			}

			// Recursively check nested decorators
			if len(decorator.Args) > 0 {
				if err := validateShDecoratorElementsWithContext(decorator.Args, cmdName, cmdLine, sourceLines); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateBlockDecoratorsWithContext validates decorators in block statements with source context
func validateBlockDecoratorsWithContext(statements []parser.BlockStatement, cmdName string, cmdLine int, sourceLines []string) error {
	for _, stmt := range statements {
		if stmt.IsDecorated {
			if !supportedDecorators[stmt.Decorator] {
				return createValidationError(
					fmt.Sprintf("unsupported decorator '@%s'. Supported decorators: %s",
						stmt.Decorator, GetSupportedDecoratorsString()),
					cmdName, cmdLine, sourceLines)
			}

			// Validate decorator usage
			switch stmt.Decorator {
			case "parallel":
				// @parallel is valid when used as a block decorator within block commands
				// No need to reject it here
			case "sh":
				if stmt.DecoratorType != "function" && stmt.DecoratorType != "simple" && stmt.DecoratorType != "block" {
					return createDecoratorError(
						"sh", stmt.DecoratorType,
						"@sh decorator must be used with function or simple syntax",
						"Use: @sh(command) or @sh: command",
						cmdName, cmdLine, sourceLines)
				}
				// Check for nested decorators in @sh content using AST
				if err := validateCommandElementsWithContext(stmt.Elements, cmdName, cmdLine, sourceLines); err != nil {
					return err
				}
			case "var":
				if stmt.DecoratorType != "function" {
					return createDecoratorError(
						"var", stmt.DecoratorType,
						"@var decorator must be used with function syntax",
						"Use: @var(VARIABLE_NAME)",
						cmdName, cmdLine, sourceLines)
				}
			}

			// Recursively validate nested blocks
			if stmt.DecoratorType == "block" && len(stmt.DecoratedBlock) > 0 {
				if err := validateBlockDecoratorsWithContext(stmt.DecoratedBlock, cmdName, cmdLine, sourceLines); err != nil {
					return err
				}
			}
		}

		// Validate elements in non-decorated statements
		if err := validateCommandElementsWithContext(stmt.Elements, cmdName, cmdLine, sourceLines); err != nil {
			return err
		}
	}
	return nil
}

// processCommandGroupWithContext processes a group of commands with enhanced error reporting
func processCommandGroupWithContext(name string, commands []parser.Command, definitions map[string]string, sourceLines []string) (TemplateCommand, error) {
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

	// Determine command type and structure with enhanced error context
	if regularCmd != nil {
		// Regular command (no watch/stop)
		templateCmd.Type = "regular"
		shellCmd, err := buildShellCommandWithContext(*regularCmd, definitions, sourceLines)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build shell command for '%s': %w", name, err)
		}
		templateCmd.ShellCommand = shellCmd
		templateCmd.HelpDescription = name
	} else if watchCmd != nil && stopCmd != nil {
		// Watch/stop pair
		templateCmd.Type = "watch-stop"
		watchShell, err := buildShellCommandWithContext(*watchCmd, definitions, sourceLines)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build watch command for '%s': %w", name, err)
		}
		stopShell, err := buildShellCommandWithContext(*stopCmd, definitions, sourceLines)
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
		watchShell, err := buildShellCommandWithContext(*watchCmd, definitions, sourceLines)
		if err != nil {
			return templateCmd, fmt.Errorf("failed to build watch command for '%s': %w", name, err)
		}
		templateCmd.WatchCommand = watchShell
		templateCmd.IsBackground = true
		templateCmd.HelpDescription = fmt.Sprintf("%s start|stop|logs", name)
	} else if stopCmd != nil {
		// Stop only (unusual, but handle it)
		templateCmd.Type = "stop-only"
		stopShell, err := buildShellCommandWithContext(*stopCmd, definitions, sourceLines)
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

// buildShellCommandWithContext constructs the shell command string with enhanced error reporting
func buildShellCommandWithContext(cmd parser.Command, definitions map[string]string, sourceLines []string) (string, error) {
	if cmd.IsBlock {
		return buildBlockCommandWithContext(cmd.Block, cmd.Name, cmd.Line, definitions, sourceLines)
	}
	// Process Elements if available, otherwise fall back to legacy Command
	if len(cmd.Elements) > 0 {
		return processElementsWithContext(cmd.Elements, definitions, cmd.Name, cmd.Line, sourceLines), nil
	}
	// Expand variables in the command text
	return expandVariablesInText(cmd.Command, definitions), nil
}

// buildBlockCommandWithContext handles block statements with enhanced error reporting
func buildBlockCommandWithContext(statements []parser.BlockStatement, cmdName string, cmdLine int, definitions map[string]string, sourceLines []string) (string, error) {
	var parts []string

	for _, stmt := range statements {
		if stmt.IsDecorated {
			part, err := buildDecoratedStatementWithContext(stmt, cmdName, cmdLine, definitions, sourceLines)
			if err != nil {
				return "", err
			}
			if part != "" {
				parts = append(parts, part)
			}
		} else {
			// Regular command (no decorator) - use Elements if available
			if len(stmt.Elements) > 0 {
				processedCommand := processElementsWithContext(stmt.Elements, definitions, cmdName, cmdLine, sourceLines)
				parts = append(parts, processedCommand)
			} else if stmt.Command != "" {
				// Process the command for variable expansion
				processedCommand := expandVariablesInText(stmt.Command, definitions)
				parts = append(parts, processedCommand)
			}
		}
	}

	return strings.Join(parts, "; "), nil
}

// expandVariablesInText expands @var(NAME) references in text
func expandVariablesInText(text string, definitions map[string]string) string {
	// Simple regex-based replacement for @var(NAME) patterns
	result := text
	for varName, varValue := range definitions {
		oldPattern := fmt.Sprintf("@var(%s)", varName)
		result = strings.ReplaceAll(result, oldPattern, varValue)
	}
	return result
}

// buildDecoratedStatementWithContext handles different decorator types with enhanced error reporting
func buildDecoratedStatementWithContext(stmt parser.BlockStatement, cmdName string, cmdLine int, definitions map[string]string, sourceLines []string) (string, error) {
	switch stmt.Decorator {
	case "sh":
		// Shell command - process Elements if available
		// @sh(command) -> command (executed via shell)
		if len(stmt.Elements) > 0 {
			// For @sh decorators, the Elements should contain a single DecoratorElement
			for _, elem := range stmt.Elements {
				if elem.IsDecorator() {
					decorator := elem.(*parser.DecoratorElement)
					if decorator.Name == "sh" {
						// Process the @sh arguments and expand any @var() decorators within
						return processElementsWithContext(decorator.Args, definitions, cmdName, cmdLine, sourceLines), nil
					}
				}
			}
		}
		// Fallback to the Command field and process it for variable expansion
		command := stmt.Command
		if len(stmt.Elements) > 0 {
			command = processElementsWithContext(stmt.Elements, definitions, cmdName, cmdLine, sourceLines)
		}
		return command, nil

	case "var":
		// Variable reference - this is a compile-time decorator that should be expanded
		// @var(NAME) -> value of NAME variable
		// NOTE: This case should rarely be hit since @var is usually processed within other commands
		varName := stmt.Command
		if value, exists := definitions[varName]; exists {
			return value, nil
		}
		// If variable doesn't exist, preserve the @var() call as a warning
		return fmt.Sprintf("@var(%s)", varName), nil

	case "parallel":
		// Parallel execution - convert to background processes with &
		// @parallel: { cmd1; cmd2; } -> cmd1 &; cmd2 &; wait
		if stmt.DecoratorType == "block" {
			var parallelParts []string
			for _, nestedStmt := range stmt.DecoratedBlock {
				if nestedStmt.IsDecorated {
					// Handle nested decorators
					part, err := buildDecoratedStatementWithContext(nestedStmt, cmdName, cmdLine, definitions, sourceLines)
					if err != nil {
						return "", err
					}
					if part != "" {
						parallelParts = append(parallelParts, part+" &")
					}
				} else {
					// Regular command in parallel block - use Elements if available
					if len(nestedStmt.Elements) > 0 {
						processedCommand := processElementsWithContext(nestedStmt.Elements, definitions, cmdName, cmdLine, sourceLines)
						parallelParts = append(parallelParts, processedCommand+" &")
					} else if nestedStmt.Command != "" {
						// Process the command for variable expansion
						processedCommand := expandVariablesInText(nestedStmt.Command, definitions)
						parallelParts = append(parallelParts, processedCommand+" &")
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
		return "", createValidationError(
			fmt.Sprintf("unsupported decorator '@%s'", stmt.Decorator),
			cmdName, cmdLine, sourceLines)
	}

	return stmt.Command, nil
}

// processElementsWithContext traverses the AST with enhanced error context
func processElementsWithContext(elements []parser.CommandElement, definitions map[string]string, cmdName string, cmdLine int, sourceLines []string) string {
	// With the new parsing strategy, this function is identical to processElements.
	// We can reuse the logic and just change the recursive calls.
	var result strings.Builder

	for _, elem := range elements {
		if decorator, ok := elem.(*parser.DecoratorElement); ok {
			switch decorator.Name {
			case "var":
				if len(decorator.Args) > 0 {
					varName := processElementsWithContext(decorator.Args, definitions, cmdName, cmdLine, sourceLines)
					if value, exists := definitions[varName]; exists {
						result.WriteString(value)
					} else {
						result.WriteString(decorator.String())
					}
				} else {
					result.WriteString(decorator.String())
				}
			case "sh":
				if len(decorator.Args) > 0 {
					result.WriteString(processElementsWithContext(decorator.Args, definitions, cmdName, cmdLine, sourceLines))
				} else {
					result.WriteString(decorator.String())
				}
			default:
				result.WriteString(decorator.String())
			}
		} else {
			result.WriteString(expandVariablesInText(elem.String(), definitions))
		}
	}

	return result.String()
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
