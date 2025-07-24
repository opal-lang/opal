package errors

import (
	"fmt"
)

// Error types for different categories of failures
const (
	// Input/File errors
	ErrInputRead    = "INPUT_READ_ERROR"
	ErrFileParse    = "FILE_PARSE_ERROR"
	ErrFileNotFound = "FILE_NOT_FOUND"

	// Command errors
	ErrCommandNotFound   = "COMMAND_NOT_FOUND"
	ErrCommandExecution  = "COMMAND_EXECUTION_ERROR"
	ErrCommandValidation = "COMMAND_VALIDATION_ERROR"
	ErrNoCommandsDefined = "NO_COMMANDS_DEFINED"

	// Variable errors
	ErrVariableNotFound       = "VARIABLE_NOT_FOUND"
	ErrVariableResolution     = "VARIABLE_RESOLUTION_ERROR"
	ErrVariableInitialization = "VARIABLE_INITIALIZATION_ERROR"

	// Decorator errors
	ErrDecoratorNotFound   = "DECORATOR_NOT_FOUND"
	ErrDecoratorValidation = "DECORATOR_VALIDATION_ERROR"
	ErrDecoratorExecution  = "DECORATOR_EXECUTION_ERROR"

	// Generation errors
	ErrCodeGeneration = "CODE_GENERATION_ERROR"
	ErrBuildFailed    = "BUILD_FAILED"

	// System errors
	ErrSystemCommand = "SYSTEM_COMMAND_ERROR"
	ErrPermission    = "PERMISSION_ERROR"
	ErrTimeout       = "TIMEOUT_ERROR"
)

// DevCmdError represents a structured error with type and context
type DevCmdError struct {
	Type    string
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *DevCmdError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap allows error unwrapping
func (e *DevCmdError) Unwrap() error {
	return e.Cause
}

// New creates a new DevCmdError
func New(errorType, message string) *DevCmdError {
	return &DevCmdError{
		Type:    errorType,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// Wrap creates a new DevCmdError wrapping an existing error
func Wrap(errorType, message string, cause error) *DevCmdError {
	return &DevCmdError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (e *DevCmdError) WithContext(key string, value interface{}) *DevCmdError {
	e.Context[key] = value
	return e
}

// GetType returns the error type
func (e *DevCmdError) GetType() string {
	return e.Type
}

// GetContext returns context value by key
func (e *DevCmdError) GetContext(key string) (interface{}, bool) {
	value, exists := e.Context[key]
	return value, exists
}

// Helper functions for common error scenarios

// NewInputError creates an input-related error
func NewInputError(message string, cause error) *DevCmdError {
	return Wrap(ErrInputRead, message, cause)
}

// NewParseError creates a parsing error
func NewParseError(message string, cause error) *DevCmdError {
	return Wrap(ErrFileParse, message, cause)
}

// NewCommandNotFoundError creates a command not found error
func NewCommandNotFoundError(commandName string, availableCommands []string) *DevCmdError {
	return New(ErrCommandNotFound, fmt.Sprintf("Command '%s' not found", commandName)).
		WithContext("command", commandName).
		WithContext("available_commands", availableCommands)
}

// NewCommandExecutionError creates a command execution error
func NewCommandExecutionError(commandName string, cause error) *DevCmdError {
	return Wrap(ErrCommandExecution, fmt.Sprintf("Failed to execute command '%s'", commandName), cause).
		WithContext("command", commandName)
}

// NewVariableNotFoundError creates a variable not found error
func NewVariableNotFoundError(variableName string) *DevCmdError {
	return New(ErrVariableNotFound, fmt.Sprintf("Variable '%s' not defined", variableName)).
		WithContext("variable", variableName)
}

// NewVariableResolutionError creates a variable resolution error
func NewVariableResolutionError(variableName string, cause error) *DevCmdError {
	return Wrap(ErrVariableResolution, fmt.Sprintf("Failed to resolve variable '%s'", variableName), cause).
		WithContext("variable", variableName)
}

// NewDecoratorError creates a decorator-related error
func NewDecoratorError(decoratorName string, cause error) *DevCmdError {
	return Wrap(ErrDecoratorExecution, fmt.Sprintf("Decorator '@%s' failed", decoratorName), cause).
		WithContext("decorator", decoratorName)
}

// NewBuildError creates a build-related error
func NewBuildError(message string, cause error) *DevCmdError {
	return Wrap(ErrBuildFailed, message, cause)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType string) bool {
	if devErr, ok := err.(*DevCmdError); ok {
		return devErr.Type == errorType
	}
	return false
}
