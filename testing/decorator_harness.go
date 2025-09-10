package testing

import (
	"fmt"
	"strings"
	"time"
)

// ================================================================================================
// DECORATOR TEST UTILITIES - Generic structures for testing decorator behavior
// ================================================================================================

// TestExpectation defines expected results from command execution
type TestExpectation struct {
	Exit    int      // expected exit code
	Stdout  string   // expected stdout content
	Stderr  string   // expected stderr content
	PlanHas []string // substrings that must appear in plan JSON/text
}

// TestScenario describes how to test a decorator
type TestScenario struct {
	// Chain operations around the decorator
	BeforeShell string // optional; executed before decorator with &&
	AfterShell  string // optional; executed after decorator with &&
	PipeTo      string // optional shell that receives stdout from decorator
	AppendTo    string // optional file target for >>

	// For Block/Pattern decorators
	Inner       []string       // newline-separated steps inside the decorator
	PatternVals map[string]any // for Pattern: inputs used to select a branch
}

// TestContext provides context for decorator test cases
type TestContext struct {
	EnvLockPath string            // optional persistent env lock
	Env         map[string]string // additional env values
	WorkDir     string            // working directory
	Timeout     time.Duration     // overall test timeout
}

// Clock interface for time operations (can be mocked)
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

// RealClock uses real time
type RealClock struct{}

func (RealClock) Now() time.Time        { return time.Now() }
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }

// FakeClock uses fake time for testing
type FakeClock struct {
	current time.Time
}

func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{current: start}
}

func (f *FakeClock) Now() time.Time          { return f.current }
func (f *FakeClock) Sleep(d time.Duration)   { f.current = f.current.Add(d) }
func (f *FakeClock) Advance(d time.Duration) { f.current = f.current.Add(d) }

// ================================================================================================
// CLI TEST CONTENT BUILDERS - Generic utilities for building test CLI content
// ================================================================================================

// BuildActionTestCLI creates CLI content for testing an action decorator
func BuildActionTestCLI(decoratorName string, args map[string]interface{}, scenario TestScenario) string {
	var parts []string

	// Add before shell if specified
	if scenario.BeforeShell != "" {
		parts = append(parts, scenario.BeforeShell)
		parts = append(parts, "&&")
	}

	// Add the action decorator
	actionCall := fmt.Sprintf("@%s", decoratorName)
	if len(args) > 0 {
		var argStrs []string
		for name, value := range args {
			if name != "" {
				argStrs = append(argStrs, fmt.Sprintf("%s=%q", name, value))
			} else {
				argStrs = append(argStrs, fmt.Sprintf("%q", value))
			}
		}
		actionCall += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}
	parts = append(parts, actionCall)

	// Add after shell if specified
	if scenario.AfterShell != "" {
		parts = append(parts, "&&")
		parts = append(parts, scenario.AfterShell)
	}

	// Add pipe or append if specified
	if scenario.PipeTo != "" {
		parts = append(parts, "|")
		parts = append(parts, scenario.PipeTo)
	}
	if scenario.AppendTo != "" {
		parts = append(parts, ">>")
		parts = append(parts, scenario.AppendTo)
	}

	command := strings.Join(parts, " ")
	return fmt.Sprintf("test: %s", command)
}

// BuildBlockTestCLI creates CLI content for testing a block decorator
func BuildBlockTestCLI(decoratorName string, args map[string]interface{}, scenario TestScenario) string {
	// Build decorator header
	decoratorCall := fmt.Sprintf("@%s", decoratorName)
	if len(args) > 0 {
		var argStrs []string
		for name, value := range args {
			if name != "" {
				argStrs = append(argStrs, fmt.Sprintf("%s=%q", name, value))
			} else {
				argStrs = append(argStrs, fmt.Sprintf("%q", value))
			}
		}
		decoratorCall += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	// Build inner content
	var innerContent []string
	if len(scenario.Inner) > 0 {
		innerContent = scenario.Inner
	} else {
		innerContent = []string{"echo 'default inner command'"}
	}

	// Format as block
	content := fmt.Sprintf("test: %s {\n", decoratorCall)
	for _, inner := range innerContent {
		content += fmt.Sprintf("    %s\n", inner)
	}
	content += "}"

	return content
}

// BuildValueTestCLI creates CLI content for testing a value decorator
func BuildValueTestCLI(decoratorName string, args map[string]interface{}) string {
	// Build decorator call
	decoratorCall := fmt.Sprintf("@%s", decoratorName)
	if len(args) > 0 {
		var argStrs []string
		for name, value := range args {
			if name != "" {
				argStrs = append(argStrs, fmt.Sprintf("%s=%q", name, value))
			} else {
				argStrs = append(argStrs, fmt.Sprintf("%q", value))
			}
		}
		decoratorCall += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	// Embed in echo command to test value expansion
	return fmt.Sprintf("test: echo \"%s\"", decoratorCall)
}

// BuildPatternTestCLI creates CLI content for testing a pattern decorator
func BuildPatternTestCLI(decoratorName string, args map[string]interface{}, branches map[string][]string, choose map[string]any) string {
	// Build decorator header
	decoratorCall := fmt.Sprintf("@%s", decoratorName)
	if len(args) > 0 {
		var argStrs []string
		for name, value := range args {
			if name != "" {
				argStrs = append(argStrs, fmt.Sprintf("%s=%q", name, value))
			} else {
				argStrs = append(argStrs, fmt.Sprintf("%q", value))
			}
		}
		decoratorCall += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	// Add variables that affect pattern selection
	var varDecls []string
	for key, value := range choose {
		varDecls = append(varDecls, fmt.Sprintf("var %s = %q", key, value))
	}

	// Build branches
	content := strings.Join(varDecls, "\n")
	if len(varDecls) > 0 {
		content += "\n\n"
	}

	content += fmt.Sprintf("test: %s {\n", decoratorCall)
	for branchName, commands := range branches {
		content += fmt.Sprintf("    %s: {\n", branchName)
		for _, cmd := range commands {
			content += fmt.Sprintf("        %s\n", cmd)
		}
		content += "    }\n"
	}
	content += "}"

	return content
}

// ================================================================================================
// ASSERTION HELPERS - Generic assertion utilities
// ================================================================================================

// AssertCommandResult checks that a command result matches expectations
func AssertCommandResult(t TestingT, result CommandResult, expect TestExpectation) {
	t.Helper()

	if result.Exit != expect.Exit {
		t.Errorf("Expected exit code %d, got %d", expect.Exit, result.Exit)
	}

	if result.Stdout != expect.Stdout {
		t.Errorf("Expected stdout %q, got %q", expect.Stdout, result.Stdout)
	}

	if result.Stderr != expect.Stderr {
		t.Errorf("Expected stderr %q, got %q", expect.Stderr, result.Stderr)
	}
}

// AssertStringContains checks that a string contains expected substrings
func AssertStringContains(t TestingT, actual string, expected ...string) {
	t.Helper()

	for _, exp := range expected {
		if !strings.Contains(actual, exp) {
			t.Errorf("String should contain %q. Actual: %s", exp, actual)
		}
	}
}
