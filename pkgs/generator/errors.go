package generator

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// GeneratorError provides enhanced error reporting with source context
type GeneratorError struct {
	Message     string
	SourceLine  int
	SourceText  string
	CommandName string
	ErrorType   string // "validation", "processing", "template", etc.
}

func (e *GeneratorError) Error() string {
	var builder strings.Builder

	// Add error type prefix if specified
	if e.ErrorType != "" {
		builder.WriteString(fmt.Sprintf("[%s] ", e.ErrorType))
	}

	if e.SourceLine > 0 && e.SourceText != "" && e.CommandName != "" {
		builder.WriteString(fmt.Sprintf("error in command '%s' at line %d: %s\nSource: %s",
			e.CommandName, e.SourceLine, e.Message, e.SourceText))
	} else if e.CommandName != "" {
		builder.WriteString(fmt.Sprintf("error in command '%s': %s", e.CommandName, e.Message))
	} else {
		builder.WriteString(fmt.Sprintf("generator error: %s", e.Message))
	}

	return builder.String()
}

// ValidationError represents a validation error during generator processing
type ValidationError struct {
	*GeneratorError
}

func NewValidationError(message, commandName string, sourceLine int, sourceText string) *ValidationError {
	return &ValidationError{
		GeneratorError: &GeneratorError{
			Message:     message,
			CommandName: commandName,
			SourceLine:  sourceLine,
			SourceText:  sourceText,
			ErrorType:   "validation",
		},
	}
}

// ProcessingError represents an error during command processing
type ProcessingError struct {
	*GeneratorError
}

func NewProcessingError(message, commandName string, sourceLine int, sourceText string) *ProcessingError {
	return &ProcessingError{
		GeneratorError: &GeneratorError{
			Message:     message,
			CommandName: commandName,
			SourceLine:  sourceLine,
			SourceText:  sourceText,
			ErrorType:   "processing",
		},
	}
}

// TemplateError represents an error during template generation
type TemplateError struct {
	*GeneratorError
	TemplateName string
}

func NewTemplateError(message, templateName string) *TemplateError {
	return &TemplateError{
		GeneratorError: &GeneratorError{
			Message:   message,
			ErrorType: "template",
		},
		TemplateName: templateName,
	}
}

func (e *TemplateError) Error() string {
	return fmt.Sprintf("[template] error in template '%s': %s", e.TemplateName, e.Message)
}

// DecoratorError represents errors specific to decorator usage
type DecoratorError struct {
	*GeneratorError
	DecoratorName string
	DecoratorType string // "function", "simple", "block"
	Suggestion    string
}

func NewDecoratorError(decoratorName, decoratorType, message, suggestion, commandName string, sourceLine int, sourceText string) *DecoratorError {
	return &DecoratorError{
		GeneratorError: &GeneratorError{
			Message:     message,
			CommandName: commandName,
			SourceLine:  sourceLine,
			SourceText:  sourceText,
			ErrorType:   "decorator",
		},
		DecoratorName: decoratorName,
		DecoratorType: decoratorType,
		Suggestion:    suggestion,
	}
}

func (e *DecoratorError) Error() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[decorator] error with @%s", e.DecoratorName))
	if e.DecoratorType != "" {
		builder.WriteString(fmt.Sprintf(" (%s)", e.DecoratorType))
	}

	if e.CommandName != "" && e.SourceLine > 0 {
		builder.WriteString(fmt.Sprintf(" in command '%s' at line %d", e.CommandName, e.SourceLine))
	}

	builder.WriteString(fmt.Sprintf(": %s", e.Message))

	if e.SourceText != "" {
		builder.WriteString(fmt.Sprintf("\nSource: %s", e.SourceText))
	}

	if e.Suggestion != "" {
		builder.WriteString(fmt.Sprintf("\nSuggestion: %s", e.Suggestion))
	}

	return builder.String()
}

// VariableError represents errors related to variable handling
type VariableError struct {
	*GeneratorError
	VariableName string
}

func NewVariableError(variableName, message, commandName string, sourceLine int, sourceText string) *VariableError {
	return &VariableError{
		GeneratorError: &GeneratorError{
			Message:     message,
			CommandName: commandName,
			SourceLine:  sourceLine,
			SourceText:  sourceText,
			ErrorType:   "variable",
		},
		VariableName: variableName,
	}
}

func (e *VariableError) Error() string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[variable] error with variable '%s'", e.VariableName))

	if e.CommandName != "" && e.SourceLine > 0 {
		builder.WriteString(fmt.Sprintf(" in command '%s' at line %d", e.CommandName, e.SourceLine))
	}

	builder.WriteString(fmt.Sprintf(": %s", e.Message))

	if e.SourceText != "" {
		builder.WriteString(fmt.Sprintf("\nSource: %s", e.SourceText))
	}

	return builder.String()
}

// ErrorCollector collects multiple errors during processing
type ErrorCollector struct {
	errors []error
}

func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

func (ec *ErrorCollector) Error() error {
	if !ec.HasErrors() {
		return nil
	}

	if len(ec.errors) == 1 {
		return ec.errors[0]
	}

	var messages []string
	for i, err := range ec.errors {
		messages = append(messages, fmt.Sprintf("%d. %s", i+1, err.Error()))
	}

	return fmt.Errorf("multiple generator errors:\n%s", strings.Join(messages, "\n"))
}

// Common error messages and helpers

// GetSupportedDecoratorsString returns a formatted string of supported decorators using stdlib registry
func GetSupportedDecoratorsString() string {
	allDecorators := stdlib.GetAllDecorators()
	var decoratorNames []string
	for _, decorator := range allDecorators {
		decoratorNames = append(decoratorNames, "@"+decorator.Name)
	}
	return strings.Join(decoratorNames, ", ")
}

// IsUnsupportedDecoratorError checks if an error is about an unsupported decorator
func IsUnsupportedDecoratorError(err error) bool {
	if genErr, ok := err.(*GeneratorError); ok {
		return strings.Contains(genErr.Message, "unsupported decorator")
	}
	if decErr, ok := err.(*DecoratorError); ok {
		return strings.Contains(decErr.Message, "unsupported")
	}
	return false
}

// IsVariableError checks if an error is related to variable handling
func IsVariableError(err error) bool {
	_, ok := err.(*VariableError)
	return ok
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
