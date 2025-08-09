package decorators

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component"`
	Decorator   string                 `json:"decorator,omitempty"`
	ExecutionID string                 `json:"execution_id,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Caller      string                 `json:"caller,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
}

// LogFormatter interface for different log output formats
type LogFormatter interface {
	Format(entry *LogEntry) string
}

// JSONFormatter outputs logs in JSON format
type JSONFormatter struct{}

func (f *JSONFormatter) Format(entry *LogEntry) string {
	data, _ := json.Marshal(entry)
	return string(data)
}

// TextFormatter outputs logs in human-readable text format
type TextFormatter struct {
	ShowCaller    bool
	ShowTimestamp bool
	UseColors     bool
}

func (f *TextFormatter) Format(entry *LogEntry) string {
	var parts []string

	if f.ShowTimestamp {
		parts = append(parts, entry.Timestamp.Format("2006-01-02 15:04:05.000"))
	}

	levelStr := entry.Level.String()
	if f.UseColors {
		levelStr = f.colorizeLevel(entry.Level, levelStr)
	}
	parts = append(parts, fmt.Sprintf("[%s]", levelStr))

	if entry.Component != "" {
		parts = append(parts, fmt.Sprintf("(%s)", entry.Component))
	}

	if entry.Decorator != "" {
		parts = append(parts, fmt.Sprintf("@%s", entry.Decorator))
	}

	parts = append(parts, entry.Message)

	if entry.Duration > 0 {
		parts = append(parts, fmt.Sprintf("duration=%v", entry.Duration))
	}

	if entry.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%s", entry.Error))
	}

	if f.ShowCaller && entry.Caller != "" {
		parts = append(parts, fmt.Sprintf("caller=%s", entry.Caller))
	}

	result := strings.Join(parts, " ")

	// Add fields if any
	if len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		result += " " + strings.Join(fieldParts, " ")
	}

	return result
}

func (f *TextFormatter) colorizeLevel(level LogLevel, text string) string {
	if !f.UseColors {
		return text
	}

	switch level {
	case LogLevelTrace:
		return fmt.Sprintf("\033[37m%s\033[0m", text) // White
	case LogLevelDebug:
		return fmt.Sprintf("\033[36m%s\033[0m", text) // Cyan
	case LogLevelInfo:
		return fmt.Sprintf("\033[32m%s\033[0m", text) // Green
	case LogLevelWarn:
		return fmt.Sprintf("\033[33m%s\033[0m", text) // Yellow
	case LogLevelError:
		return fmt.Sprintf("\033[31m%s\033[0m", text) // Red
	case LogLevelFatal:
		return fmt.Sprintf("\033[35m%s\033[0m", text) // Magenta
	default:
		return text
	}
}

// Logger is the main logging interface
type Logger struct {
	mu        sync.RWMutex
	level     LogLevel
	outputs   []io.Writer
	formatter LogFormatter
	component string
	fields    map[string]interface{}
}

// NewLogger creates a new logger instance
func NewLogger(component string) *Logger {
	return &Logger{
		level:     LogLevelInfo,
		outputs:   []io.Writer{os.Stdout},
		formatter: &TextFormatter{ShowTimestamp: true, UseColors: true},
		component: component,
		fields:    make(map[string]interface{}),
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormatter sets the log formatter
func (l *Logger) SetFormatter(formatter LogFormatter) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.formatter = formatter
}

// AddOutput adds an additional output writer
func (l *Logger) AddOutput(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outputs = append(l.outputs, writer)
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		level:     l.level,
		outputs:   l.outputs,
		formatter: l.formatter,
		component: l.component,
		fields:    newFields,
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:     l.level,
		outputs:   l.outputs,
		formatter: l.formatter,
		component: l.component,
		fields:    newFields,
	}
}

// WithDecorator creates a logger with decorator context
func (l *Logger) WithDecorator(decoratorName string) *Logger {
	return l.WithField("decorator", decoratorName)
}

// WithExecutionID creates a logger with execution ID context
func (l *Logger) WithExecutionID(executionID string) *Logger {
	return l.WithField("execution_id", executionID)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, err error, duration time.Duration) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Component: l.component,
		Duration:  duration,
		Fields:    l.fields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Add decorator from fields if present
	if decorator, ok := l.fields["decorator"].(string); ok {
		entry.Decorator = decorator
	}

	// Add execution ID from fields if present
	if execID, ok := l.fields["execution_id"].(string); ok {
		entry.ExecutionID = execID
	}

	// Get caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		funcName := runtime.FuncForPC(pc).Name()
		entry.Caller = fmt.Sprintf("%s:%d (%s)", filepath.Base(file), line, filepath.Base(funcName))
	}

	// Add stack trace for errors and fatal logs
	if level >= LogLevelError {
		entry.StackTrace = getStackTrace()
	}

	formatted := l.formatter.Format(entry)

	for _, output := range l.outputs {
		if _, err := fmt.Fprintln(output, formatted); err != nil {
			// Log to stderr as fallback if primary output fails
			fmt.Fprintf(os.Stderr, "Warning: failed to write log output: %v\n", err)
		}
	}
}

// Trace logs a trace message
func (l *Logger) Trace(message string) {
	l.log(LogLevelTrace, message, nil, 0)
}

// Tracef logs a formatted trace message
func (l *Logger) Tracef(format string, args ...interface{}) {
	l.log(LogLevelTrace, fmt.Sprintf(format, args...), nil, 0)
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message, nil, 0)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LogLevelDebug, fmt.Sprintf(format, args...), nil, 0)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, nil, 0)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LogLevelInfo, fmt.Sprintf(format, args...), nil, 0)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, nil, 0)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LogLevelWarn, fmt.Sprintf(format, args...), nil, 0)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(LogLevelError, message, nil, 0)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LogLevelError, fmt.Sprintf(format, args...), nil, 0)
}

// ErrorWithErr logs an error message with an error object
func (l *Logger) ErrorWithErr(message string, err error) {
	l.log(LogLevelError, message, err, 0)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string) {
	l.log(LogLevelFatal, message, nil, 0)
	os.Exit(1)
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(LogLevelFatal, fmt.Sprintf(format, args...), nil, 0)
	os.Exit(1)
}

// LogDuration logs a message with duration information
func (l *Logger) LogDuration(level LogLevel, message string, duration time.Duration) {
	l.log(level, message, nil, duration)
}

// getStackTrace returns the current stack trace
func getStackTrace() []string {
	var stack []string
	for i := 3; i < 10; i++ { // Skip first few frames
		if pc, file, line, ok := runtime.Caller(i); ok {
			funcName := runtime.FuncForPC(pc).Name()
			frame := fmt.Sprintf("%s:%d (%s)", filepath.Base(file), line, filepath.Base(funcName))
			stack = append(stack, frame)
		} else {
			break
		}
	}
	return stack
}

// LogManager manages multiple loggers and global logging configuration
type LogManager struct {
	mu      sync.RWMutex
	loggers map[string]*Logger
	level   LogLevel
	logFile string
}

var globalLogManager = &LogManager{
	loggers: make(map[string]*Logger),
	level:   LogLevelInfo,
}

// GetLogger returns a logger for the specified component
func GetLogger(component string) *Logger {
	globalLogManager.mu.Lock()
	defer globalLogManager.mu.Unlock()

	if logger, exists := globalLogManager.loggers[component]; exists {
		return logger
	}

	logger := NewLogger(component)
	logger.SetLevel(globalLogManager.level)

	// Add file output if configured
	if globalLogManager.logFile != "" {
		if file, err := os.OpenFile(globalLogManager.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
			logger.AddOutput(file)
		}
	}

	globalLogManager.loggers[component] = logger
	return logger
}

// SetGlobalLogLevel sets the log level for all loggers
func SetGlobalLogLevel(level LogLevel) {
	globalLogManager.mu.Lock()
	defer globalLogManager.mu.Unlock()

	globalLogManager.level = level
	for _, logger := range globalLogManager.loggers {
		logger.SetLevel(level)
	}
}

// SetLogFile configures file logging for all loggers
func SetLogFile(filename string) error {
	globalLogManager.mu.Lock()
	defer globalLogManager.mu.Unlock()

	// Create log directory if needed
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	globalLogManager.logFile = filename

	// Add file output to all existing loggers
	for _, logger := range globalLogManager.loggers {
		logger.AddOutput(file)
	}

	return nil
}

// SetJSONLogging configures all loggers to use JSON format
func SetJSONLogging() {
	globalLogManager.mu.RLock()
	defer globalLogManager.mu.RUnlock()

	formatter := &JSONFormatter{}
	for _, logger := range globalLogManager.loggers {
		logger.SetFormatter(formatter)
	}
}

// GetLogManager returns the global log manager
func GetLogManager() *LogManager {
	return globalLogManager
}

// Convenience functions for global logging
var defaultLogger = GetLogger("devcmd")

// Trace logs a trace message using the default logger
func Trace(message string) {
	defaultLogger.Trace(message)
}

// Debug logs a debug message using the default logger
func Debug(message string) {
	defaultLogger.Debug(message)
}

// Info logs an info message using the default logger
func Info(message string) {
	defaultLogger.Info(message)
}

// Warn logs a warning message using the default logger
func Warn(message string) {
	defaultLogger.Warn(message)
}

// Error logs an error message using the default logger
func Error(message string) {
	defaultLogger.Error(message)
}

// Infof logs a formatted info message using the default logger
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Errorf logs a formatted error message using the default logger
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// ErrorWithErr logs an error message with error using the default logger
func ErrorWithErr(message string, err error) {
	defaultLogger.ErrorWithErr(message, err)
}
