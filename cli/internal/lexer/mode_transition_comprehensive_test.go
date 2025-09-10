package lexer

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/types"
)

// TestModeTransitionScenarios tests comprehensive nesting scenarios found in the specification
// This serves as TDD for implementing proper mode stack or context tracking
func TestModeTransitionScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		input       string
		description string
		expected    []tokenExpectation
	}{
		{
			name:        "Scenario 1: Simple pattern with mixed syntax",
			description: "Pattern decorator with both simple (dev:) and block (prod: {}) syntax",
			input: `deploy: @when(ENV) {
  prod: { npm run build && npm run deploy }
  dev: npm run dev-deploy
  default: { echo "Unknown env: $ENV"; exit 1 }
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				// First pattern branch with block
				{types.IDENTIFIER, "prod"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build"},
				{types.AND, "&&"},
				{types.SHELL_TEXT, " npm run deploy"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Second pattern branch simple syntax
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "npm run dev-deploy"},
				{types.SHELL_END, ""},
				// Third pattern branch with block
				{types.IDENTIFIER, "default"},
				{types.COLON, ":"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Unknown env: $ENV"},
				{types.STRING_END, "\""},
				{types.SHELL_TEXT, "; exit 1"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 2: Nested decorators in pattern branches",
			description: "Pattern decorator containing block decorators in different branches",
			input: `deploy: @when(ENV) {
  prod: @timeout(60s) { deploy prod }
  dev: @timeout(30s) { deploy dev }
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				// First pattern branch with nested decorator
				{types.IDENTIFIER, "prod"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "60s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "deploy prod"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Second pattern branch with nested decorator
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "deploy dev"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 3: Deep nesting from specification",
			description: "From docs: @timeout containing @retry with multiple commands",
			input: `deploy: @timeout(5m) {
    echo "Starting deployment"
    @retry(attempts = 3) {
        kubectl apply -f k8s/
        kubectl rollout status
    }
    echo "Deployment completed"
    @parallel {
        kubectl logs deployment/api
        kubectl logs deployment/worker
    }
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "deploy"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "5m"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Starting deployment"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "attempts"},
				{types.EQUALS, "="},
				{types.NUMBER, "3"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "kubectl apply -f k8s/"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "kubectl rollout status"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Deployment completed"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "kubectl logs deployment/api"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "kubectl logs deployment/worker"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 4: Complex parallel nesting from specification",
			description: "From docs: @parallel containing multiple @timeout decorators",
			input: `release: @parallel {
    @timeout(2m) {
        npm run build
        npm run test:unit
    }
    @timeout(1m) {
        npm run test:e2e
        npm run lint
    }
    @retry(attempts = 2) {
        npm run deploy
        npm run smoke-test
    }
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "release"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				// First nested timeout
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "2m"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run build"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run test:unit"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Second nested timeout
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "1m"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run test:e2e"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run lint"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Third nested retry
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "attempts"},
				{types.EQUALS, "="},
				{types.NUMBER, "2"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "npm run deploy"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run smoke-test"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 5: Pattern decorator with nested decorators in multi-line blocks",
			description: "From docs: @when with @timeout in multi-line pattern branches",
			input: `dev: @when(MODE) {
    production: @timeout(BUILD_TIMEOUT) {
        echo "Building for production..."
        NODE_ENV=production webpack --mode production
        echo "Build completed"
        serve -s dist -l @var(PORT)
    }
    development: @timeout(30s) {
        echo "Starting development server..."
        NODE_ENV=@env("NODE_ENV") webpack serve --mode @var(WEBPACK_MODE) --hot
    }
    default: echo "Unknown mode"
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "dev"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "MODE"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				// Production branch with nested @timeout
				{types.IDENTIFIER, "production"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "BUILD_TIMEOUT"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Building for production..."},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "NODE_ENV=production webpack --mode production"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Build completed"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "serve -s dist -l "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "PORT"},
				{types.RPAREN, ")"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Development branch with nested @timeout
				{types.IDENTIFIER, "development"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.DURATION, "30s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Starting development server..."},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "NODE_ENV="},
				{types.AT, "@"},
				{types.IDENTIFIER, "env"},
				{types.LPAREN, "("},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "NODE_ENV"},
				{types.STRING_END, "\""},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, " webpack serve --mode "},
				{types.AT, "@"},
				{types.IDENTIFIER, "var"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "WEBPACK_MODE"},
				{types.RPAREN, ")"},
				{types.SHELL_TEXT, " --hot"},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Default branch simple syntax
				{types.IDENTIFIER, "default"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Unknown mode"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 6: Try pattern with nested decorators",
			description: "From docs: @try with @timeout and @retry in different branches",
			input: `build: @try {
    main: @timeout(BUILD_TIMEOUT) {
        echo "Starting build process"
        npm run build:production
        npm run test:all
        echo "Build successful"
    }
    error: @retry(attempts = 3, delay = 2s) {
        echo "Build failed, cleaning up..."
        npm run clean
        echo "Retrying build..."
    }
    finally: echo "Build process completed"
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "build"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "try"},
				{types.LBRACE, "{"},
				// Main branch with @timeout
				{types.IDENTIFIER, "main"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "BUILD_TIMEOUT"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Starting build process"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run build:production"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run test:all"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Build successful"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Error branch with @retry
				{types.IDENTIFIER, "error"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "attempts"},
				{types.EQUALS, "="},
				{types.NUMBER, "3"},
				{types.COMMA, ","},
				{types.IDENTIFIER, "delay"},
				{types.EQUALS, "="},
				{types.DURATION, "2s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Build failed, cleaning up..."},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "npm run clean"},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Retrying build..."},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				// Finally branch simple syntax
				{types.IDENTIFIER, "finally"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Build process completed"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 7: The original failing case - shell after pattern",
			description: "The case that was failing - shell content after exiting pattern decorator",
			input: `test: @retry(attempts=3) {
		@when(ENV) {
			development: echo "Dev environment"
		}
		echo "Always execute"
	}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "test"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "attempts"},
				{types.EQUALS, "="},
				{types.NUMBER, "3"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "when"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "ENV"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.IDENTIFIER, "development"},
				{types.COLON, ":"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Dev environment"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Always execute"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""}, // This was failing before
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
		{
			name:        "Scenario 8: Performance test case - deeply nested",
			description: "From performance tests: deeply nested decorators",
			input: `complex1: @retry(attempts=RETRIES, delay=1s) {
	@timeout(duration=TIMEOUT) {
		@parallel {
			echo "Task 1"
			echo "Task 2"
			echo "Task 3"
		}
	}
}`,
			expected: []tokenExpectation{
				{types.IDENTIFIER, "complex1"},
				{types.COLON, ":"},
				{types.AT, "@"},
				{types.IDENTIFIER, "retry"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "attempts"},
				{types.EQUALS, "="},
				{types.IDENTIFIER, "RETRIES"},
				{types.COMMA, ","},
				{types.IDENTIFIER, "delay"},
				{types.EQUALS, "="},
				{types.DURATION, "1s"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "timeout"},
				{types.LPAREN, "("},
				{types.IDENTIFIER, "duration"},
				{types.EQUALS, "="},
				{types.IDENTIFIER, "TIMEOUT"},
				{types.RPAREN, ")"},
				{types.LBRACE, "{"},
				{types.AT, "@"},
				{types.IDENTIFIER, "parallel"},
				{types.LBRACE, "{"},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Task 1"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Task 2"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.SHELL_TEXT, "echo "},
				{types.STRING_START, "\""},
				{types.STRING_TEXT, "Task 3"},
				{types.STRING_END, "\""},
				{types.SHELL_END, ""},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.RBRACE, "}"},
				{types.EOF, ""},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Description: %s", scenario.description)
			t.Logf("Input:\n%s", scenario.input)

			// Use the existing assertTokens helper
			assertTokens(t, scenario.name, scenario.input, scenario.expected)
		})
	}
}

// TestModeTransitionEdgeCases tests specific edge cases for proper mode handling
func TestModeTransitionEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name        string
		input       string
		description string
		shouldFail  bool // Mark cases that should fail with current implementation
	}{
		{
			name: "Edge case 1: Empty pattern branches",
			input: `deploy: @when(ENV) {
  prod:
  dev: echo "something"
}`,
			description: "Pattern with empty branch should handle gracefully",
			shouldFail:  true, // Likely to fail - empty branches not well defined
		},
		{
			name: "Edge case 2: Nested patterns (not valid per spec)",
			input: `deploy: @when(OUTER) {
  prod: @when(INNER) {
    staging: echo "nested"
  }
}`,
			description: "Nested pattern decorators - should this be valid?",
			shouldFail:  true, // Need to check if this is valid syntax
		},
		{
			name: "Edge case 3: Single brace in shell content",
			input: `test: @timeout(30s) {
  echo "This { should be fine"
  echo "And this } too"
}`,
			description: "Braces inside shell content should not affect mode transitions",
			shouldFail:  false,
		},
	}

	for _, edge := range edgeCases {
		t.Run(edge.name, func(t *testing.T) {
			t.Logf("Description: %s", edge.description)
			t.Logf("Input:\n%s", edge.input)

			if edge.shouldFail {
				t.Logf("Expected to fail with current implementation")
			}

			lexer := New(strings.NewReader(edge.input))
			tokens := lexer.TokenizeToSlice()

			t.Logf("Actual tokens:")
			for i, tok := range tokens {
				t.Logf("  [%d] %s: %q", i, tok.Type, tok.Value)
			}

			// Don't fail the test, just document the behavior
			if !edge.shouldFail {
				// Basic validation - should not have ILLEGAL tokens for valid cases
				for _, tok := range tokens {
					if tok.Type == types.ILLEGAL {
						t.Errorf("Found ILLEGAL token: %q - this should parse correctly", tok.Value)
					}
				}
			}
		})
	}
}
