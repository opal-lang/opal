package decorators

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestLogLevel(t *testing.T) {
	levels := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelTrace, "TRACE"},
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelFatal, "FATAL"},
	}

	for _, test := range levels {
		if test.level.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.level.String())
		}
	}
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	// Test basic logging
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected log output to contain 'test message', got: %s", output)
	}

	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected log output to contain 'INFO', got: %s", output)
	}

	if !strings.Contains(output, "(test)") {
		t.Errorf("Expected log output to contain '(test)', got: %s", output)
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	// Test logging with fields
	loggerWithFields := logger.WithField("key1", "value1").WithField("key2", 42)
	loggerWithFields.Info("test with fields")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Expected log output to contain 'key1=value1', got: %s", output)
	}

	if !strings.Contains(output, "key2=42") {
		t.Errorf("Expected log output to contain 'key2=42', got: %s", output)
	}
}

func TestLoggerWithDecorator(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	// Test logging with decorator context
	decoratorLogger := logger.WithDecorator("parallel")
	decoratorLogger.Info("decorator execution")

	output := buf.String()
	if !strings.Contains(output, "@parallel") {
		t.Errorf("Expected log output to contain '@parallel', got: %s", output)
	}
}

func TestLoggerWithExecutionID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	// Test logging with execution ID
	execLogger := logger.WithExecutionID("exec_123")
	execLogger.Info("execution message")

	output := buf.String()
	if !strings.Contains(output, "execution_id=exec_123") {
		t.Errorf("Expected log output to contain 'execution_id=exec_123', got: %s", output)
	}
}

func TestLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetLevel(LogLevelWarn)

	// Debug message should not appear
	logger.Debug("debug message")
	debugOutput := buf.String()
	if strings.Contains(debugOutput, "debug message") {
		t.Errorf("Debug message should not appear with WARN level, got: %s", debugOutput)
	}

	// Warn message should appear
	logger.Warn("warn message")
	warnOutput := buf.String()
	if !strings.Contains(warnOutput, "warn message") {
		t.Errorf("Warn message should appear with WARN level, got: %s", warnOutput)
	}
}

func TestLoggerErrorWithErr(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetLevel(LogLevelDebug)

	// Test error logging with error object
	testErr := fmt.Errorf("test error")
	logger.ErrorWithErr("something failed", testErr)

	output := buf.String()
	if !strings.Contains(output, "something failed") {
		t.Errorf("Expected log output to contain 'something failed', got: %s", output)
	}

	if !strings.Contains(output, "error=test error") {
		t.Errorf("Expected log output to contain 'error=test error', got: %s", output)
	}
}

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test")
	logger.AddOutput(&buf)
	logger.SetFormatter(&JSONFormatter{})

	logger.Info("json test")

	output := buf.String()
	if !strings.Contains(output, `"message":"json test"`) {
		t.Errorf("Expected JSON output to contain message field, got: %s", output)
	}

	if !strings.Contains(output, `"level":2`) { // INFO level is 2
		t.Errorf("Expected JSON output to contain level field, got: %s", output)
	}

	if !strings.Contains(output, `"component":"test"`) {
		t.Errorf("Expected JSON output to contain component field, got: %s", output)
	}
}

func TestTextFormatterColors(t *testing.T) {
	formatter := &TextFormatter{UseColors: true, ShowTimestamp: false}

	entry := &LogEntry{
		Level:     LogLevelError,
		Message:   "error message",
		Component: "test",
	}

	output := formatter.Format(entry)
	if !strings.Contains(output, "\033[31m") { // Red color code for ERROR
		t.Errorf("Expected colored output for ERROR level, got: %s", output)
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test that global logger functions work
	Info("global info test")
	Debug("global debug test")
	Warn("global warn test")
	Error("global error test")

	// These should not panic
}

func TestGetLogger(t *testing.T) {
	logger1 := GetLogger("component1")
	logger2 := GetLogger("component1")
	logger3 := GetLogger("component2")

	// Same component should return same logger
	if logger1 != logger2 {
		t.Error("GetLogger should return the same instance for the same component")
	}

	// Different component should return different logger
	if logger1 == logger3 {
		t.Error("GetLogger should return different instances for different components")
	}
}

func TestSetGlobalLogLevel(t *testing.T) {
	var buf bytes.Buffer

	// Create a logger and add our buffer as output
	logger := GetLogger("test_global_level")
	logger.AddOutput(&buf)

	// Set global level to WARN
	SetGlobalLogLevel(LogLevelWarn)

	// Debug message should not appear
	logger.Debug("should not appear")
	debugOutput := buf.String()
	if strings.Contains(debugOutput, "should not appear") {
		t.Errorf("Debug message should not appear with global WARN level")
	}

	// Warn message should appear
	logger.Warn("should appear")
	warnOutput := buf.String()
	if !strings.Contains(warnOutput, "should appear") {
		t.Errorf("Warn message should appear with global WARN level")
	}

	// Reset to INFO for other tests
	SetGlobalLogLevel(LogLevelInfo)
}

func TestLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("concurrent")
	logger.AddOutput(&buf)

	// Test concurrent logging
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Infof("concurrent message %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	if !strings.Contains(output, "concurrent message") {
		t.Errorf("Expected concurrent log messages, got: %s", output)
	}
}

func BenchmarkLogger(b *testing.B) {
	logger := NewLogger("benchmark")
	logger.SetLevel(LogLevelInfo)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	logger := NewLogger("benchmark")
	logger.SetLevel(LogLevelInfo)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.WithField("iteration", i).Info("benchmark message with field")
	}
}

func BenchmarkLoggerFiltered(b *testing.B) {
	logger := NewLogger("benchmark")
	logger.SetLevel(LogLevelWarn) // Filter out INFO messages

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("filtered message") // This should be filtered out
	}
}
