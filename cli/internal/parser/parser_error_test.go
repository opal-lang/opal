package parser

import (
	"strings"
	"testing"
)

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
		category    string
	}{
		// === Synchronization and Error Recovery ===
		{
			name: "missing closing brace - recover at next command",
			input: `broken-cmd: {
    echo "unclosed block"
    
good-cmd: echo "this should parse correctly"`,
			wantErr:     true,
			errContains: "expected '}'",
			category:    "synchronization",
		},
		{
			name: "missing closing brace - recover at block decorator",
			input: `broken: {
    echo "missing brace"
    
@parallel {
    echo "task1"
}`,
			wantErr:     true,
			errContains: "expected '}'",
			category:    "synchronization",
		},

		// === Decorator Context Tests (these should work per spec) ===
		{
			name:        "block decorator in shell context should work per spec",
			input:       `setup: @parallel { @cmd(core-deps) }`,
			wantErr:     false,
			errContains: "",
			category:    "decorator-context",
		},
		{
			name:        "timeout decorator in shell context should work per spec",
			input:       `test: @timeout(30s) { echo "hello" }`,
			wantErr:     false,
			errContains: "",
			category:    "decorator-context",
		},
		{
			name:        "pattern decorator needs correct syntax",
			input:       `test: @when("ENV") { production: echo "hello"; default: echo "dev" }`,
			wantErr:     false,
			errContains: "",
			category:    "decorator-context",
		},
		{
			name:        "pattern decorator with wrong syntax should fail",
			input:       `test: @when("prod") { echo "hello" }`,
			wantErr:     true,
			errContains: "expected ':'",
			category:    "decorator-syntax",
		},

		// === Variable Declaration Errors ===
		{
			name:        "variable @var() in decorator arguments",
			input:       "var TIMEOUT = 30s\ntest: @timeout(@var(TIMEOUT)) { npm test }",
			wantErr:     true,
			errContains: "parameter 'duration' expects duration, got AT",
			category:    "variables",
		},
		{
			name:        "variable @env() in decorator arguments",
			input:       `test: @timeout(@env(DURATION)) { npm test }`,
			wantErr:     true,
			errContains: "parameter 'duration' expects duration, got AT",
			category:    "variables",
		},
		{
			name:        "unquoted identifier in variable",
			input:       "var PATH = ./src",
			wantErr:     true,
			errContains: "variable value must be a quoted string, number, duration, or boolean literal",
			category:    "variables",
		},
		{
			name:        "array syntax in variable",
			input:       "var ITEMS = [1, 2, 3]",
			wantErr:     true,
			errContains: "variable value must be a quoted string, number, duration, or boolean literal",
			category:    "variables",
		},

		// === Bracket Mismatch Errors ===
		{
			name: "unclosed command block",
			input: `test: {
    echo "missing closing brace"`,
			wantErr:     true,
			errContains: "expected '}'",
			category:    "brackets",
		},
		{
			name: "unclosed decorator parentheses",
			input: `test: @timeout(30s {
    echo "missing closing paren"
}`,
			wantErr:     true,
			errContains: "expected ')'",
			category:    "brackets",
		},
		{
			name: "extra closing brace",
			input: `test: {
    echo "hello"
}}`,
			wantErr:     true,
			errContains: "unexpected",
			category:    "brackets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			program, err := Parse(reader)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but parsing succeeded. Program: %+v", program)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errContains, err)
				}
				t.Logf("✅ Expected error [%s]: %v", tt.category, err)
			} else {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				t.Logf("✅ Success [%s]", tt.category)
			}
		})
	}
}

func TestSynchronizationPoints(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		syncPoint   string
		description string
	}{
		{
			name: "command declaration sync",
			input: `broken syntax here
good-cmd: echo "should sync here"`,
			syncPoint:   "command declaration",
			description: "Parser should recover at 'good-cmd:' pattern",
		},
		{
			name: "block decorator sync",
			input: `broken: @invalid(syntax
@parallel {
    echo "should sync here"
}`,
			syncPoint:   "@parallel",
			description: "Parser should recover at @parallel block decorator",
		},
		{
			name: "pattern decorator sync",
			input: `broken: { missing brace
@when("env") {
    production: echo "prod"
    default: echo "dev"  
}`,
			syncPoint:   "@when",
			description: "Parser should recover at @when pattern decorator",
		},
		{
			name: "closing brace sync",
			input: `broken: @invalid syntax
}
good: echo "after brace"`,
			syncPoint:   "}",
			description: "Parser should recover at closing brace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			_, err := Parse(reader)

			// All these cases should produce errors (they contain syntax errors)
			if err == nil {
				t.Error("Expected parsing error but got none")
				return
			}

			t.Logf("✅ Sync test [%s]: %s - Error: %v", tt.syncPoint, tt.description, err)

			// TODO: In the future, we could enhance this to verify that specific
			// parts of the input were parsed correctly after synchronization
		})
	}
}

func TestDecoratorTypeValidation(t *testing.T) {
	tests := []struct {
		name          string
		decoratorName string
		context       string
		input         string
		shouldSucceed bool
	}{
		// Block decorators
		{
			name:          "parallel in command block context",
			decoratorName: "parallel",
			context:       "command block",
			input:         `test: { @parallel { echo "task1"; echo "task2" } }`,
			shouldSucceed: true,
		},
		{
			name:          "parallel in shell context",
			decoratorName: "parallel",
			context:       "shell",
			input:         `test: @parallel { echo "task1"; echo "task2" }`,
			shouldSucceed: true,
		},
		{
			name:          "timeout in command block context",
			decoratorName: "timeout",
			context:       "command block",
			input:         `test: { @timeout(30s) { echo "long task" } }`,
			shouldSucceed: true,
		},
		{
			name:          "timeout in shell context",
			decoratorName: "timeout",
			context:       "shell",
			input:         `test: @timeout(30s) { echo "long task" }`,
			shouldSucceed: true,
		},

		// Action decorators (should work in shell context)
		{
			name:          "log in shell context",
			decoratorName: "log",
			context:       "shell",
			input:         `test: @log("message") echo "hello"`,
			shouldSucceed: true,
		},
		{
			name:          "cmd in shell context",
			decoratorName: "cmd",
			context:       "shell",
			input:         `test: @cmd(helper)`,
			shouldSucceed: true,
		},

		// Value decorators (should work in shell context)
		{
			name:          "var in shell context",
			decoratorName: "var",
			context:       "shell",
			input: `var PROJECT = "test"
test: echo @var(PROJECT)`,
			shouldSucceed: true,
		},
		{
			name:          "env in shell context",
			decoratorName: "env",
			context:       "shell",
			input:         `test: echo @env(HOME)`,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			program, err := Parse(reader)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success for %s in %s context, got error: %v",
						tt.decoratorName, tt.context, err)
				} else {
					t.Logf("✅ %s decorator works in %s context", tt.decoratorName, tt.context)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for %s in %s context, but parsing succeeded. Program: %+v",
						tt.decoratorName, tt.context, program)
				} else if !strings.Contains(err.Error(), "cannot be used in shell context") {
					t.Errorf("Expected 'shell context' error for %s, got: %v", tt.decoratorName, err)
				} else {
					t.Logf("✅ %s decorator correctly rejected in %s context: %v",
						tt.decoratorName, tt.context, err)
				}
			}
		})
	}
}
