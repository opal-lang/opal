package testing

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// TIMING AND EXECUTION HELPERS
// =============================================================================

// MeasureExecutionTime extracts execution time from output
func MeasureExecutionTime(output *ExecutionOutput) time.Duration {
	if output == nil {
		return 0
	}
	return output.Duration
}

// ExtractTimestamps finds timestamps in execution output
func ExtractTimestamps(output *ExecutionOutput) []time.Time {
	if output == nil {
		return []time.Time{}
	}

	// Look for timestamp patterns in output
	timestampRegex := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	matches := timestampRegex.FindAllString(output.Stdout, -1)

	var timestamps []time.Time
	for _, match := range matches {
		if t, err := time.Parse("2006-01-02 15:04:05", match); err == nil {
			timestamps = append(timestamps, t)
		}
	}

	return timestamps
}

// VerifyConcurrentExecution checks if commands ran concurrently based on timestamps
func VerifyConcurrentExecution(output *ExecutionOutput) bool {
	timestamps := ExtractTimestamps(output)
	if len(timestamps) < 2 {
		return false
	}

	// Check if timestamps overlap (indicating concurrent execution)
	for i := 0; i < len(timestamps)-1; i++ {
		for j := i + 1; j < len(timestamps); j++ {
			diff := timestamps[j].Sub(timestamps[i])
			if diff < time.Second { // Commands started within 1 second = concurrent
				return true
			}
		}
	}

	return false
}

// CountExecutionAttempts counts retry attempts or similar patterns
func CountExecutionAttempts(output *ExecutionOutput) int {
	if output == nil {
		return 0
	}

	// Look for common retry patterns
	patterns := []string{
		"Attempt",
		"attempt",
		"Retry",
		"retry",
		"Try",
		"try",
	}

	maxCount := 0
	for _, pattern := range patterns {
		count := strings.Count(output.Stdout, pattern) + strings.Count(output.Stderr, pattern)
		if count > maxCount {
			maxCount = count
		}
	}

	return maxCount
}

// MeasureDelaysBetweenAttempts calculates delays between retry attempts
func MeasureDelaysBetweenAttempts(output *ExecutionOutput) []time.Duration {
	timestamps := ExtractTimestamps(output)
	if len(timestamps) < 2 {
		return []time.Duration{}
	}

	var delays []time.Duration
	for i := 1; i < len(timestamps); i++ {
		delay := timestamps[i].Sub(timestamps[i-1])
		delays = append(delays, delay)
	}

	return delays
}

// =============================================================================
// OUTPUT ANALYSIS HELPERS
// =============================================================================

// CountOccurrences counts how many times a pattern appears in output
func CountOccurrences(output *ExecutionOutput, pattern string) int {
	if output == nil {
		return 0
	}

	return strings.Count(output.Stdout, pattern) + strings.Count(output.Stderr, pattern)
}

// ExtractBetween extracts text between start and end markers
func ExtractBetween(output *ExecutionOutput, start, end string) string {
	if output == nil {
		return ""
	}

	text := output.Stdout + output.Stderr
	startIdx := strings.Index(text, start)
	if startIdx == -1 {
		return ""
	}

	startIdx += len(start)
	endIdx := strings.Index(text[startIdx:], end)
	if endIdx == -1 {
		return text[startIdx:]
	}

	return text[startIdx : startIdx+endIdx]
}

// VerifyOutputOrder checks if strings appear in expected order
func VerifyOutputOrder(output *ExecutionOutput, expected []string) bool {
	if output == nil || len(expected) == 0 {
		return len(expected) == 0
	}

	text := output.Stdout + output.Stderr
	lastIndex := -1

	for _, expectedStr := range expected {
		index := strings.Index(text[lastIndex+1:], expectedStr)
		if index == -1 {
			return false
		}
		lastIndex = lastIndex + 1 + index
	}

	return true
}

// ContainsAll checks if output contains all expected strings
func ContainsAll(output *ExecutionOutput, patterns []string) bool {
	if output == nil {
		return len(patterns) == 0
	}

	text := output.Stdout + output.Stderr
	for _, pattern := range patterns {
		if !strings.Contains(text, pattern) {
			return false
		}
	}

	return true
}

// ContainsAny checks if output contains any of the expected strings
func ContainsAny(output *ExecutionOutput, patterns []string) bool {
	if output == nil || len(patterns) == 0 {
		return false
	}

	text := output.Stdout + output.Stderr
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	return false
}

// =============================================================================
// PROCESS AND SYSTEM HELPERS
// =============================================================================

// CountProcessesSpawned estimates processes based on output patterns
func CountProcessesSpawned(output *ExecutionOutput) int {
	if output == nil {
		return 0
	}

	// Look for process indicators
	indicators := []string{
		"PID:",
		"Process",
		"process",
		"Started:",
		"Spawned:",
	}

	count := 0
	text := output.Stdout + output.Stderr
	for _, indicator := range indicators {
		count += strings.Count(text, indicator)
	}

	return count
}

// VerifyProcessIsolation checks for evidence of proper process isolation
func VerifyProcessIsolation(output *ExecutionOutput) bool {
	if output == nil {
		return true // No output = no isolation issues
	}

	// Look for race condition indicators
	raceIndicators := []string{
		"race detected",
		"concurrent access",
		"shared state",
		"WARNING: DATA RACE",
	}

	text := strings.ToLower(output.Stdout + output.Stderr)
	for _, indicator := range raceIndicators {
		if strings.Contains(text, strings.ToLower(indicator)) {
			return false
		}
	}

	return true
}

// =============================================================================
// FILESYSTEM HELPERS
// =============================================================================

// VerifyDirectoryChanged checks if working directory changed based on pwd output
func VerifyDirectoryChanged(output *ExecutionOutput, expectedPath string) bool {
	if output == nil {
		return false
	}

	// Look for pwd output or directory references
	text := output.Stdout + output.Stderr
	return strings.Contains(text, expectedPath)
}

// ExtractWorkingDirectory extracts working directory from pwd output
func ExtractWorkingDirectory(output *ExecutionOutput) string {
	if output == nil {
		return ""
	}

	lines := strings.Split(output.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "/") && !strings.Contains(line, " ") {
			// Looks like a directory path
			return line
		}
	}

	return ""
}

// =============================================================================
// DECORATOR-SPECIFIC HELPERS
// =============================================================================

// VerifyParallelExecution validates that commands ran in parallel
func VerifyParallelExecution(output *ExecutionOutput, expectedConcurrency int) error {
	if !VerifyConcurrentExecution(output) {
		return fmt.Errorf("commands did not execute concurrently")
	}

	if expectedConcurrency > 0 {
		processCount := CountProcessesSpawned(output)
		if processCount < expectedConcurrency {
			return fmt.Errorf("expected at least %d processes, got %d", expectedConcurrency, processCount)
		}
	}

	return nil
}

// VerifyRetryBehavior validates retry decorator behavior
func VerifyRetryBehavior(output *ExecutionOutput, expectedAttempts int, expectedDelay time.Duration) error {
	actualAttempts := CountExecutionAttempts(output)
	if actualAttempts != expectedAttempts {
		return fmt.Errorf("expected %d attempts, got %d", expectedAttempts, actualAttempts)
	}

	if expectedDelay > 0 {
		delays := MeasureDelaysBetweenAttempts(output)
		for i, delay := range delays {
			if delay < expectedDelay-100*time.Millisecond { // Allow 100ms tolerance
				return fmt.Errorf("delay %d too short: %v (expected >= %v)", i, delay, expectedDelay)
			}
		}
	}

	return nil
}

// VerifyTimeoutBehavior validates timeout decorator behavior
func VerifyTimeoutBehavior(output *ExecutionOutput, expectedTimeout time.Duration) error {
	executionTime := MeasureExecutionTime(output)

	// Check if execution was terminated due to timeout
	if executionTime > expectedTimeout+time.Second { // Allow 1 second tolerance
		return fmt.Errorf("execution time %v exceeds timeout %v", executionTime, expectedTimeout)
	}

	// Look for timeout indicators in output
	timeoutIndicators := []string{
		"timeout",
		"timed out",
		"deadline exceeded",
		"context deadline exceeded",
	}

	text := strings.ToLower(output.Stdout + output.Stderr)
	hasTimeoutIndicator := false
	for _, indicator := range timeoutIndicators {
		if strings.Contains(text, indicator) {
			hasTimeoutIndicator = true
			break
		}
	}

	if executionTime >= expectedTimeout-100*time.Millisecond && !hasTimeoutIndicator {
		return fmt.Errorf("execution near timeout duration but no timeout message found")
	}

	return nil
}

// VerifyWorkdirBehavior validates workdir decorator behavior
func VerifyWorkdirBehavior(output *ExecutionOutput, expectedPath string) error {
	actualDir := ExtractWorkingDirectory(output)
	if actualDir == "" {
		return fmt.Errorf("no working directory found in output")
	}

	if !strings.Contains(actualDir, expectedPath) {
		return fmt.Errorf("working directory %q does not contain expected path %q", actualDir, expectedPath)
	}

	return nil
}

// VerifyConditionalExecution validates when/try decorator conditional behavior
func VerifyConditionalExecution(output *ExecutionOutput, condition string, executed bool) error {
	text := output.Stdout + output.Stderr

	// Look for condition evaluation evidence
	conditionFound := strings.Contains(text, condition)

	if executed && !conditionFound {
		return fmt.Errorf("condition %q should have been executed but not found in output", condition)
	}

	if !executed && conditionFound {
		return fmt.Errorf("condition %q should not have been executed but found in output", condition)
	}

	return nil
}

// =============================================================================
// CONTEXT ISOLATION HELPERS
// =============================================================================

// VerifyContextIsolation validates that parallel branches don't share context
func VerifyContextIsolation(output *ExecutionOutput, branches []string) error {
	if !VerifyProcessIsolation(output) {
		return fmt.Errorf("process isolation violated")
	}

	// Check that each branch produced its expected output
	for i, branch := range branches {
		if !strings.Contains(output.Stdout+output.Stderr, branch) {
			return fmt.Errorf("branch %d output %q not found", i, branch)
		}
	}

	// Check for variable bleeding between branches
	if hasVariableInterference(output) {
		return fmt.Errorf("variable interference detected between branches")
	}

	return nil
}

// hasVariableInterference detects if variables leaked between parallel branches
func hasVariableInterference(output *ExecutionOutput) bool {
	if output == nil {
		return false
	}

	// Look for indicators of shared state
	interferencePatterns := []string{
		"variable conflict",
		"shared variable",
		"concurrent modification",
		"race condition",
	}

	text := strings.ToLower(output.Stdout + output.Stderr)
	for _, pattern := range interferencePatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	return false
}

// =============================================================================
// PERFORMANCE HELPERS
// =============================================================================

// VerifyPerformanceWithin checks if execution completed within expected time
func VerifyPerformanceWithin(output *ExecutionOutput, maxDuration time.Duration) error {
	if output == nil {
		return fmt.Errorf("no execution output to measure")
	}

	if output.Duration > maxDuration {
		return fmt.Errorf("execution took %v, expected <= %v", output.Duration, maxDuration)
	}

	return nil
}

// ExtractNumericValue extracts first numeric value from output
func ExtractNumericValue(output *ExecutionOutput) (float64, error) {
	if output == nil {
		return 0, fmt.Errorf("no output")
	}

	// Find first number in output
	re := regexp.MustCompile(`-?\d+\.?\d*`)
	match := re.FindString(output.Stdout + output.Stderr)
	if match == "" {
		return 0, fmt.Errorf("no numeric value found")
	}

	return strconv.ParseFloat(match, 64)
}

// =============================================================================
// ERROR PATTERN HELPERS
// =============================================================================

// VerifyErrorPattern checks if error output matches expected pattern
func VerifyErrorPattern(output *ExecutionOutput, expectedPattern string) error {
	if output == nil {
		return fmt.Errorf("no output to check for error pattern")
	}

	if output.ExitCode == 0 {
		return fmt.Errorf("expected error but command succeeded")
	}

	errorText := output.Stderr
	if errorText == "" {
		errorText = output.Stdout // Some commands write errors to stdout
	}

	if !strings.Contains(strings.ToLower(errorText), strings.ToLower(expectedPattern)) {
		return fmt.Errorf("error output %q does not contain expected pattern %q", errorText, expectedPattern)
	}

	return nil
}

// VerifyNoErrorPattern ensures error pattern is NOT present
func VerifyNoErrorPattern(output *ExecutionOutput, forbiddenPattern string) error {
	if output == nil {
		return nil
	}

	errorText := strings.ToLower(output.Stdout + output.Stderr)
	if strings.Contains(errorText, strings.ToLower(forbiddenPattern)) {
		return fmt.Errorf("forbidden error pattern %q found in output", forbiddenPattern)
	}

	return nil
}
