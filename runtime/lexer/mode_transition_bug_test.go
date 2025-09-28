package lexer

import (
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/types"
	_ "github.com/aledsdavies/opal/runtime/decorators/builtin" // Import for decorator registration
)

// TestModeTransitionBug reproduces the exact bug: lexer switches to LanguageMode
// inside command blocks instead of staying in CommandMode to parse decorators
func TestModeTransitionBug(t *testing.T) {
	t.Run("simple_decorator_in_command_block", func(t *testing.T) {
		input := `setup: {
    @log("test")
    @parallel {
        echo "done"
    }
}`

		lexer := New(strings.NewReader(input))
		var tokens []types.Token
		for {
			tok := lexer.NextToken()
			tokens = append(tokens, tok)
			if tok.Type == types.EOF {
				break
			}
		}

		t.Logf("=== ACTUAL TOKENS (CURRENT LEXER BEHAVIOR) ===")
		for i, tok := range tokens {
			t.Logf("[%d] %s: %q", i, tok.Type.String(), tok.Value)
		}

		// Check the critical failure point: @log should be AT + IDENTIFIER + LPAREN
		// NOT SHELL_TEXT("@log(")
		if len(tokens) >= 8 {
			// After "setup: {" we expect "@log" to be parsed as separate tokens
			if tokens[7].Type == types.SHELL_TEXT && strings.Contains(tokens[7].Value, "@log") {
				t.Errorf("❌ LEXER BUG: @log parsed as SHELL_TEXT %q instead of AT+IDENTIFIER+LPAREN", tokens[7].Value)
				t.Log("   Root cause: lexer switches to LanguageMode inside command blocks")
				t.Log("   Expected: AT(@) + IDENTIFIER(log) + LPAREN")
				t.Log("   Actual:   SHELL_TEXT(@log()")
			}

			// Also check @parallel
			found_parallel_bug := false
			for i, tok := range tokens {
				if tok.Type == types.SHELL_TEXT && strings.Contains(tok.Value, "@parallel") {
					t.Errorf("❌ LEXER BUG: @parallel parsed as SHELL_TEXT %q at position %d", tok.Value, i)
					found_parallel_bug = true
					break
				}
			}

			if !found_parallel_bug {
				// Good! Check if we have proper AT tokens
				at_count := 0
				for _, tok := range tokens {
					if tok.Type == types.AT {
						at_count++
					}
				}
				if at_count >= 2 {
					t.Log("✅ Decorators properly parsed as AT tokens")
				}
			}
		}
	})

	t.Run("string_interpolation_breaks_mode", func(t *testing.T) {
		input := `var PROJECT = "test"

setup: {
    @log("Setting up @var(PROJECT)...")
    @parallel {
        echo "done"  
    }
}`

		lexer := New(strings.NewReader(input))
		var tokens []types.Token
		for {
			tok := lexer.NextToken()
			tokens = append(tokens, tok)
			if tok.Type == types.EOF {
				break
			}
		}

		t.Logf("=== STRING INTERPOLATION TEST ===")

		// Find where @log appears
		log_as_shell := false
		parallel_as_shell := false

		for i, tok := range tokens {
			t.Logf("[%d] %s: %q", i, tok.Type.String(), tok.Value)
			if tok.Type == types.SHELL_TEXT && strings.Contains(tok.Value, "@log") {
				log_as_shell = true
			}
			if tok.Type == types.SHELL_TEXT && strings.Contains(tok.Value, "@parallel") {
				parallel_as_shell = true
			}
		}

		if log_as_shell {
			t.Errorf("❌ BUG: @log with string interpolation parsed as SHELL_TEXT")
		}

		if parallel_as_shell {
			t.Errorf("❌ BUG: @parallel after string interpolation parsed as SHELL_TEXT")
		}

		if !log_as_shell && !parallel_as_shell {
			t.Log("✅ String interpolation doesn't break decorator parsing")
		}
	})
}
