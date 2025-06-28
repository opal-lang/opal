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
	PackageName         string
	Imports             []string
	HasProcessMgmt      bool
	HasParallelCommands bool
	HasUserDefinedHelp  bool // NEW: Track if user defined help command
	Commands            []TemplateCommand
	ProcessMgmtFuncs    []string
}

// CommandSegment represents a part of a mixed command (either parallel or sequential)
type CommandSegment struct {
	IsParallel bool
	Commands   []string // For parallel segments
	Command    string   // For sequential segments
}

// TemplateCommand represents a command ready for template generation
type TemplateCommand struct {
	Name             string           // Original command name
	FunctionName     string           // Sanitized Go function name
	GoCase           string           // Case statement value
	Type             string           // "regular", "watch-stop", "watch-only", "stop-only", "parallel", "mixed"
	ShellCommand     string           // For regular commands
	WatchCommand     string           // For watch part of watch-stop commands
	StopCommand      string           // For stop part of watch-stop commands
	ParallelCommands []string         // For pure parallel commands
	CommandSegments  []CommandSegment // For mixed commands
	IsBackground     bool             // For watch commands
	HelpDescription  string           // Description for help text
}

// validateHelpCommandRestrictions ensures help command isn't used with watch/stop
func validateHelpCommandRestrictions(commands []parser.Command, sourceLines []string) error {
	for _, cmd := range commands {
		if cmd.Name == "help" {
			if cmd.IsWatch {
				return createValidationError(
					"'help' command cannot be used with 'watch' modifier. Help is a special reserved command",
					cmd.Name, cmd.Line, sourceLines)
			}
			if cmd.IsStop {
				return createValidationError(
					"'help' command cannot be used with 'stop' modifier. Help is a special reserved command",
					cmd.Name, cmd.Line, sourceLines)
			}
		}
	}
	return nil
}

// PreprocessCommands converts parser commands into template-ready data with standard library support
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

	// Check if user defined a help command
	_, hasUserHelp := commandGroups["help"]
	data.HasUserDefinedHelp = hasUserHelp

	// Validate help command restrictions FIRST
	if err := validateHelpCommandRestrictions(cf.Commands, cf.Lines); err != nil {
		return nil, err
	}

	// Validate decorators before processing with source context
	if err := validateDecoratorsWithContext(cf.Commands, cf.Lines); err != nil {
		return nil, err
	}

	// Determine what features we need
	hasWatchCommands := false
	hasParallelCommands := false
	hasRegularCommands := len(cf.Commands) > 0

	for _, cmd := range cf.Commands {
		if cmd.IsWatch {
			hasWatchCommands = true
		}
		if containsParallelDecorator(cmd) {
			hasParallelCommands = true
		}
	}

	data.HasProcessMgmt = hasWatchCommands
	data.HasParallelCommands = hasParallelCommands

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

		if hasParallelCommands {
			data.Imports = append(data.Imports, "sync")
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

// containsParallelDecorator checks if a command contains @parallel decorator
func containsParallelDecorator(cmd parser.Command) bool {
	if cmd.IsBlock {
		return containsParallelInBlock(cmd.Block)
	}
	return containsParallelInElements(cmd.Elements)
}

// containsParallelInBlock checks for @parallel in block statements
func containsParallelInBlock(statements []parser.BlockStatement) bool {
	for _, stmt := range statements {
		if stmt.IsDecorated && stmt.Decorator == "parallel" {
			return true
		}
		if stmt.IsDecorated && len(stmt.DecoratedBlock) > 0 {
			if containsParallelInBlock(stmt.DecoratedBlock) {
				return true
			}
		}
		if containsParallelInElements(stmt.Elements) {
			return true
		}
	}
	return false
}

// containsParallelInElements checks for @parallel in command elements
func containsParallelInElements(elements []parser.CommandElement) bool {
	for _, elem := range elements {
		if decorator, ok := elem.(*parser.DecoratorElement); ok {
			if decorator.Name == "parallel" {
				return true
			}
			if containsParallelInElements(decorator.Args) {
				return true
			}
			if len(decorator.Block) > 0 && containsParallelInBlock(decorator.Block) {
				return true
			}
		}
	}
	return false
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
		// Check if it's a parallel or mixed command
		if containsParallelDecorator(*regularCmd) {
			segments, err := analyzeCommandStructure(*regularCmd, definitions, sourceLines)
			if err != nil {
				return templateCmd, fmt.Errorf("failed to analyze command structure for '%s': %w", name, err)
			}

			// Determine if it's pure parallel or mixed
			if len(segments) == 1 && segments[0].IsParallel {
				templateCmd.Type = "parallel"
				templateCmd.ParallelCommands = segments[0].Commands
			} else {
				templateCmd.Type = "mixed"
				templateCmd.CommandSegments = segments
			}
		} else {
			// Regular command (no parallel)
			templateCmd.Type = "regular"
			shellCmd, err := buildShellCommandWithContext(*regularCmd, definitions, sourceLines)
			if err != nil {
				return templateCmd, fmt.Errorf("failed to build shell command for '%s': %w", name, err)
			}
			templateCmd.ShellCommand = shellCmd
		}
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

// analyzeCommandStructure analyzes a command and returns segments of parallel and sequential execution
func analyzeCommandStructure(cmd parser.Command, definitions map[string]string, sourceLines []string) ([]CommandSegment, error) {
	if cmd.IsBlock {
		return analyzeBlockStructure(cmd.Block, cmd.Name, cmd.Line, definitions, sourceLines)
	}

	// Simple command - check if it contains parallel decorators
	if containsParallelInElements(cmd.Elements) {
		// This would be unusual but handle it
		commands := []string{processElementsWithContext(cmd.Elements, definitions, cmd.Name, cmd.Line, sourceLines)}
		return []CommandSegment{{IsParallel: true, Commands: commands}}, nil
	}

	// Regular sequential command
	shellCmd := processElementsWithContext(cmd.Elements, definitions, cmd.Name, cmd.Line, sourceLines)
	if shellCmd == "" && cmd.Command != "" {
		shellCmd = expandVariablesInText(cmd.Command, definitions)
	}
	return []CommandSegment{{IsParallel: false, Command: shellCmd}}, nil
}

// analyzeBlockStructure analyzes block statements and groups them into parallel and sequential segments
func analyzeBlockStructure(statements []parser.BlockStatement, cmdName string, cmdLine int, definitions map[string]string, sourceLines []string) ([]CommandSegment, error) {
	var segments []CommandSegment

	for _, stmt := range statements {
		if stmt.IsDecorated && stmt.Decorator == "parallel" {
			// This is a parallel block
			if stmt.DecoratorType == "block" && len(stmt.DecoratedBlock) > 0 {
				var parallelCommands []string
				for _, nestedStmt := range stmt.DecoratedBlock {
					if nestedStmt.IsDecorated {
						// Handle nested decorators within parallel block
						part, err := buildDecoratedStatementWithContext(nestedStmt, cmdName, cmdLine, definitions, sourceLines)
						if err != nil {
							return nil, err
						}
						if part != "" {
							parallelCommands = append(parallelCommands, part)
						}
					} else {
						// Regular command in parallel block
						if len(nestedStmt.Elements) > 0 {
							processedCommand := processElementsWithContext(nestedStmt.Elements, definitions, cmdName, cmdLine, sourceLines)
							parallelCommands = append(parallelCommands, processedCommand)
						} else if nestedStmt.Command != "" {
							processedCommand := expandVariablesInText(nestedStmt.Command, definitions)
							parallelCommands = append(parallelCommands, processedCommand)
						}
					}
				}
				segments = append(segments, CommandSegment{IsParallel: true, Commands: parallelCommands})
			}
		} else {
			// This is a sequential statement
			var command string
			if stmt.IsDecorated {
				part, err := buildDecoratedStatementWithContext(stmt, cmdName, cmdLine, definitions, sourceLines)
				if err != nil {
					return nil, err
				}
				command = part
			} else {
				if len(stmt.Elements) > 0 {
					command = processElementsWithContext(stmt.Elements, definitions, cmdName, cmdLine, sourceLines)
				} else if stmt.Command != "" {
					command = expandVariablesInText(stmt.Command, definitions)
				}
			}

			if command != "" {
				segments = append(segments, CommandSegment{IsParallel: false, Command: command})
			}
		}
	}

	return segments, nil
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

// buildBlockCommandWithContext handles block statements but now excludes @parallel blocks since they're handled separately
func buildBlockCommandWithContext(statements []parser.BlockStatement, cmdName string, cmdLine int, definitions map[string]string, sourceLines []string) (string, error) {
	var parts []string

	for _, stmt := range statements {
		if stmt.IsDecorated {
			// Skip @parallel decorators - they're handled in analyzeCommandStructure
			if stmt.Decorator == "parallel" {
				continue
			}

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

// buildDecoratedStatementWithContext handles different decorator types (updated to skip @parallel)
func buildDecoratedStatementWithContext(stmt parser.BlockStatement, cmdName string, cmdLine int, definitions map[string]string, sourceLines []string) (string, error) {
	switch stmt.Decorator {
	case "sh":
		// Shell command - process Elements if available
		if len(stmt.Elements) > 0 {
			for _, elem := range stmt.Elements {
				if elem.IsDecorator() {
					decorator := elem.(*parser.DecoratorElement)
					if decorator.Name == "sh" {
						return processElementsWithContext(decorator.Args, definitions, cmdName, cmdLine, sourceLines), nil
					}
				}
			}
		}
		command := stmt.Command
		if len(stmt.Elements) > 0 {
			command = processElementsWithContext(stmt.Elements, definitions, cmdName, cmdLine, sourceLines)
		}
		return command, nil

	case "var":
		// Variable reference
		varName := stmt.Command
		if value, exists := definitions[varName]; exists {
			return value, nil
		}
		return fmt.Sprintf("@var(%s)", varName), nil

	case "parallel":
		// @parallel is now handled separately in analyzeCommandStructure
		// This should not be reached, but return empty to be safe
		return "", nil

	default:
		return "", createValidationError(
			fmt.Sprintf("unsupported decorator '@%s'", stmt.Decorator),
			cmdName, cmdLine, sourceLines)
	}
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

			if err := validateCommandElementsWithContext(decorator.Args, cmdName, cmdLine, sourceLines); err != nil {
				return err
			}

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

			switch stmt.Decorator {
			case "parallel":
				// @parallel is valid when used as a block decorator
			case "sh":
				if stmt.DecoratorType != "function" && stmt.DecoratorType != "simple" && stmt.DecoratorType != "block" {
					return createDecoratorError(
						"sh", stmt.DecoratorType,
						"@sh decorator must be used with function or simple syntax",
						"Use: @sh(command) or @sh: command",
						cmdName, cmdLine, sourceLines)
				}
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

			if stmt.DecoratorType == "block" && len(stmt.DecoratedBlock) > 0 {
				if err := validateBlockDecoratorsWithContext(stmt.DecoratedBlock, cmdName, cmdLine, sourceLines); err != nil {
					return err
				}
			}
		}

		if err := validateCommandElementsWithContext(stmt.Elements, cmdName, cmdLine, sourceLines); err != nil {
			return err
		}
	}
	return nil
}

// expandVariablesInText expands @var(NAME) references in text
func expandVariablesInText(text string, definitions map[string]string) string {
	result := text
	for varName, varValue := range definitions {
		oldPattern := fmt.Sprintf("@var(%s)", varName)
		result = strings.ReplaceAll(result, oldPattern, varValue)
	}
	return result
}

// processElementsWithContext traverses the AST with enhanced error context
func processElementsWithContext(elements []parser.CommandElement, definitions map[string]string, cmdName string, cmdLine int, sourceLines []string) string {
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
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
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

// GenerateGo creates a Go CLI from a CommandFile using the composable template system with standard library support
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
	// Core templates from the updated templates
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
	tr.templates["parallel-command"] = parallelCommandTemplate
	tr.templates["mixed-command"] = mixedCommandTemplate

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
