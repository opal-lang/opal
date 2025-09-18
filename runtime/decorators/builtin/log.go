package builtin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// Register the @log decorator on package import
func init() {
	decorators.RegisterAction(NewLogDecorator())
}

// LogDecorator implements the @log decorator using the core decorator interfaces
type LogDecorator struct{}

// NewLogDecorator creates a new log decorator
func NewLogDecorator() *LogDecorator {
	return &LogDecorator{}
}

// ================================================================================================
// CORE DECORATOR INTERFACE IMPLEMENTATION
// ================================================================================================

// Name returns the decorator name
func (l *LogDecorator) Name() string {
	return "log"
}

// Description returns a human-readable description
func (l *LogDecorator) Description() string {
	return "Output structured logging messages with color support and configurable levels"
}

// ParameterSchema returns the expected parameters for this decorator
func (l *LogDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "message",
			Type:        decorators.ArgTypeString,
			Required:    true,
			Description: "Message to output",
		},
		{
			Name:        "level",
			Type:        decorators.ArgTypeString,
			Required:    false,
			Description: "Log level (info, warn, error, debug) - defaults to info",
		},
		{
			Name:        "plain",
			Type:        decorators.ArgTypeBool,
			Required:    false,
			Description: "If true, output plain text without timestamps and formatting",
		},
	}
}

// Examples returns usage examples
func (l *LogDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{
			Code:        "@log(\"Starting build process\")",
			Description: "Simple info log message",
		},
		{
			Code:        "@log(\"Build failed\", level=\"error\")",
			Description: "Error level log message",
		},
		{
			Code:        "@log(\"{green}Build successful!{/green}\")",
			Description: "Log message with color formatting",
		},
		{
			Code:        "build: @log(\"Building...\") && npm run build && @log(\"Done!\")",
			Description: "Log messages before and after command",
		},
	}
}

// Note: ImportRequirements removed - will be added back when code generation is implemented

// ================================================================================================
// ACTION DECORATOR METHODS
// ================================================================================================

// Run executes the log command and returns appropriate CommandResult
func (l *LogDecorator) Run(ctx decorators.Context, args []decorators.Param) decorators.CommandResult {
	// TODO: Runtime execution - implement when interpreter is rebuilt
	return context.CommandResult{
		Stdout:   "",
		Stderr:   "runtime execution not implemented yet - use plan mode",
		ExitCode: 1,
	}
}

// Describe returns description for dry-run display
func (l *LogDecorator) Describe(ctx decorators.Context, args []decorators.Param) plan.ExecutionStep {
	message, level, plain, err := l.extractDecoratorParameters(args)
	if err != nil {
		return plan.ExecutionStep{
			Type:        plan.StepShell,
			Description: fmt.Sprintf("@log(<error: %v>)", err),
			Command:     "",
		}
	}

	// Show preview of message for plan - truncate at first newline
	preview := message
	if lines := strings.Split(preview, "\n"); len(lines) > 1 {
		preview = lines[0] + " ..."
	}

	// Remove color templates for plan display
	cleanPreview := l.removeColorTemplates(preview)

	// Limit length for plans - need to account for prefix length
	// Plan formatter truncates at 80 chars, so we aim for ~77 total
	var maxMessageLength int
	if plain {
		// "Log (plain): " = 13 characters
		maxMessageLength = 77 - 13
	} else {
		// "Log: [LEVEL] " varies by level, but INFO is 12 chars
		levelUpper := strings.ToUpper(level)
		prefixLength := len(fmt.Sprintf("Log: [%s] ", levelUpper))
		maxMessageLength = 77 - prefixLength
	}

	if len(cleanPreview) > maxMessageLength {
		cleanPreview = cleanPreview[:maxMessageLength] + "..."
	}

	// Format according to test expectations
	var description string
	if plain {
		description = fmt.Sprintf("Log (plain): %s", cleanPreview)
	} else {
		levelUpper := strings.ToUpper(level)
		description = fmt.Sprintf("Log: [%s] %s", levelUpper, cleanPreview)
	}

	return plan.ExecutionStep{
		Type:        plan.StepShell,
		Description: description,
		Command:     description, // Use the formatted description as the command for display
		Metadata: map[string]string{
			"decorator": "log",
			"level":     level,
			"plain":     fmt.Sprintf("%t", plain),
			"lines":     fmt.Sprintf("%d", strings.Count(message, "\n")+1),
		},
	}
}

// ================================================================================================
// HELPER METHODS
// ================================================================================================

// extractDecoratorParameters extracts message, level, and plain flag from decorator parameters
func (l *LogDecorator) extractDecoratorParameters(params []decorators.Param) (message string, level string, plain bool, err error) {
	// Extract message (first positional parameter or named "message")
	message, err = decorators.ExtractPositionalString(params, 0, "")
	if err != nil || message == "" {
		// Try named parameter
		message, err = decorators.ExtractString(params, "message", "")
		if err != nil || message == "" {
			return "", "", false, fmt.Errorf("@log requires a message")
		}
	}

	// Extract optional parameters with defaults
	level, err = decorators.ExtractString(params, "level", "info")
	if err != nil {
		return "", "", false, fmt.Errorf("@log level parameter error: %v", err)
	}

	plain, err = decorators.ExtractBool(params, "plain", false)
	if err != nil {
		return "", "", false, fmt.Errorf("@log plain parameter error: %v", err)
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[level] {
		return "", "", false, fmt.Errorf("invalid log level %q, must be one of: debug, info, warn, error", level)
	}

	return message, level, plain, nil
}

// extractParameters extracts message, level, and plain flag from AST parameters
// Legacy extractParameters method disabled - used AST types
// TODO: Remove when GenerateHint is properly updated

// processColorTemplates processes color template tags like {red}text{/red}
//
//nolint:unused // Will be used for future color template support
func (l *LogDecorator) processColorTemplates(message string) string {
	// Define ANSI color codes
	colorCodes := map[string]string{
		"black":   "\033[30m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"gray":    "\033[90m",
		"grey":    "\033[90m",

		// Bright colors
		"bright_red":     "\033[91m",
		"bright_green":   "\033[92m",
		"bright_yellow":  "\033[93m",
		"bright_blue":    "\033[94m",
		"bright_magenta": "\033[95m",
		"bright_cyan":    "\033[96m",
		"bright_white":   "\033[97m",

		// Text styles
		"bold":      "\033[1m",
		"dim":       "\033[2m",
		"italic":    "\033[3m",
		"underline": "\033[4m",

		// Reset
		"reset": "\033[0m",
	}

	const resetCode = "\033[0m"

	result := message
	for colorName, colorCode := range colorCodes {
		// Pattern for {colorname}text{/colorname}
		pattern := fmt.Sprintf(`\{%s\}(.*?)\{/%s\}`, regexp.QuoteMeta(colorName), regexp.QuoteMeta(colorName))
		re := regexp.MustCompile(pattern)

		result = re.ReplaceAllStringFunc(result, func(match string) string {
			// Extract the text between the tags
			innerPattern := fmt.Sprintf(`\{%s\}(.*?)\{/%s\}`, regexp.QuoteMeta(colorName), regexp.QuoteMeta(colorName))
			innerRe := regexp.MustCompile(innerPattern)
			matches := innerRe.FindStringSubmatch(match)
			if len(matches) > 1 {
				return colorCode + matches[1] + resetCode
			}
			return match
		})
	}

	return result
}

// removeColorTemplates removes color template tags for clean text display
func (l *LogDecorator) removeColorTemplates(message string) string {
	// Remove color template tags like {red}text{/red} -> text
	re := regexp.MustCompile(`\{[^}]*\}([^{]*)\{/[^}]*\}`)
	result := re.ReplaceAllString(message, "$1")

	// Remove any remaining standalone tags
	re = regexp.MustCompile(`\{[^}]*\}`)
	result = re.ReplaceAllString(result, "")

	return result
}
