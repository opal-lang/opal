package lexer

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/types"
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin" // Import for decorator registration
)

// TestStringInterpolationModeTransitions tests that the lexer correctly handles
// mode transitions when parsing decorators inside interpolated strings, then
// returns to the proper mode for parsing subsequent line-start decorators.
//
// This is the root cause bug: @var() inside strings breaks subsequent block decorators
func TestStringInterpolationModeTransitions(t *testing.T) {
	// FIRST: Test what we ACTUALLY get vs what we WANT
	t.Run("debug_actual_vs_expected", func(t *testing.T) {
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

		// What we WANT for decorators inside command blocks
		wanted := []types.TokenType{
			types.IDENTIFIER,   // setup
			types.COLON,        // :
			types.LBRACE,       // {
			types.AT,           // @ (NOT SHELL_TEXT!)
			types.IDENTIFIER,   // log
			types.LPAREN,       // (
			types.STRING_START, // "
			types.STRING_TEXT,  // test
			types.STRING_END,   // "
			types.RPAREN,       // )
			types.AT,           // @ (NOT SHELL_TEXT!)
			types.IDENTIFIER,   // parallel
			types.LBRACE,       // {
			types.SHELL_TEXT,   // echo
			types.STRING_START, // "
			types.STRING_TEXT,  // done
			types.STRING_END,   // "
			types.SHELL_END,    // end
			types.RBRACE,       // }
			types.RBRACE,       // }
			types.EOF,
		}

		t.Logf("\n=== WANTED TOKENS (CORRECT BEHAVIOR) ===")
		for i, tokenType := range wanted {
			t.Logf("[%d] %s", i, tokenType.String())
		}

		// This will fail until we fix the lexer mode transitions
		if len(tokens) >= 7 && tokens[7].Type == types.SHELL_TEXT && tokens[7].Value == "@log(" {
			t.Errorf("❌ BUG CONFIRMED: @log parsed as SHELL_TEXT instead of AT+IDENTIFIER+LPAREN")
			t.Log("   The lexer is in LanguageMode inside command blocks instead of CommandMode")
		}
	})

	tests := []struct {
		name   string
		input  string
		tokens []types.TokenType // Expected token sequence
	}{
		{
			name: "simple @parallel without string interpolation should work",
			input: `setup: @parallel {
    echo "test"
}`,
			tokens: []types.TokenType{
				types.IDENTIFIER,   // setup
				types.COLON,        // :
				types.AT,           // @
				types.IDENTIFIER,   // parallel
				types.LBRACE,       // {
				types.SHELL_TEXT,   // echo
				types.STRING_START, // "
				types.STRING_TEXT,  // test
				types.STRING_END,   // "
				types.SHELL_END,    // End of shell content
				types.RBRACE,       // }
				types.EOF,
			},
		},
		{
			name: "@var() in string followed by @parallel should work (ROOT CAUSE BUG)",
			input: `var PROJECT = "test"

setup: {
    @log("Setting up @var(PROJECT) development...")
    @parallel {
        echo "first"
        echo "second"
    }
}`,
			tokens: []types.TokenType{
				types.VAR,          // var
				types.IDENTIFIER,   // PROJECT
				types.EQUALS,       // =
				types.STRING_START, // " (start of simple string)
				types.STRING_TEXT,  // "test"
				types.STRING_END,   // " (end of simple string)
				types.IDENTIFIER,   // setup
				types.COLON,        // :
				types.LBRACE,       // {
				// *** CORRECTED: Inside command blocks, decorators should be parsed properly ***
				types.AT,           // @
				types.IDENTIFIER,   // log
				types.LPAREN,       // (
				types.STRING_START, // " (start of interpolated string)
				types.STRING_TEXT,  // "Setting up "
				types.AT,           // @ (within string)
				types.IDENTIFIER,   // var
				types.LPAREN,       // (
				types.IDENTIFIER,   // PROJECT
				types.RPAREN,       // )
				types.STRING_TEXT,  // " development..."
				types.STRING_END,   // " (end of interpolated string)
				types.RPAREN,       // )
				// *** CRITICAL: After string interpolation, lexer should return to command mode ***
				types.AT,           // @ (start of @parallel decorator)
				types.IDENTIFIER,   // parallel
				types.LBRACE,       // {
				types.SHELL_TEXT,   // echo
				types.STRING_START, // " (start of string)
				types.STRING_TEXT,  // "first"
				types.STRING_END,   // " (end of string)
				types.SHELL_END,    // End of first command
				types.SHELL_TEXT,   // echo
				types.STRING_START, // " (start of string)
				types.STRING_TEXT,  // "second"
				types.STRING_END,   // " (end of string)
				types.SHELL_END,    // End of second command
				types.RBRACE,       // }
				types.RBRACE,       // }
				types.EOF,
			},
		},
		{
			name: "@env() in string followed by @workdir should work",
			input: `build: {
    @log("Building in @env(NODE_ENV) mode...")
    @workdir("src") {
        npm run build
    }
}`,
			tokens: []types.TokenType{
				types.IDENTIFIER,   // build
				types.COLON,        // :
				types.LBRACE,       // {
				types.AT,           // @
				types.IDENTIFIER,   // log
				types.LPAREN,       // (
				types.STRING_START, // "Building in
				types.STRING_TEXT,  // Building in
				types.AT,           // @
				types.IDENTIFIER,   // env
				types.LPAREN,       // (
				types.IDENTIFIER,   // NODE_ENV
				types.RPAREN,       // )
				types.STRING_TEXT,  // mode...
				types.STRING_END,   // "
				types.RPAREN,       // )
				types.AT,           // @ (CRITICAL TRANSITION POINT)
				types.IDENTIFIER,   // workdir
				types.LPAREN,       // (
				types.STRING_START, // "
				types.STRING_TEXT,  // src
				types.STRING_END,   // "
				types.RPAREN,       // )
				types.LBRACE,       // {
				types.SHELL_TEXT,   // npm run build
				types.SHELL_END,    // End of shell content
				types.RBRACE,       // }
				types.RBRACE,       // }
				types.EOF,
			},
		},
		{
			name: "multiple @var() in same string should work",
			input: `deploy: {
    @log("Deploying @var(APP) version @var(VERSION) to @var(ENV)")
    @parallel {
        echo "deployment complete"
    }
}`,
			tokens: []types.TokenType{
				types.IDENTIFIER,   // deploy
				types.COLON,        // :
				types.LBRACE,       // {
				types.AT,           // @
				types.IDENTIFIER,   // log
				types.LPAREN,       // (
				types.STRING_START, // "Deploying
				types.STRING_TEXT,  // Deploying
				types.AT,           // @
				types.IDENTIFIER,   // var
				types.LPAREN,       // (
				types.IDENTIFIER,   // APP
				types.RPAREN,       // )
				types.STRING_TEXT,  // version
				types.AT,           // @
				types.IDENTIFIER,   // var
				types.LPAREN,       // (
				types.IDENTIFIER,   // VERSION
				types.RPAREN,       // )
				types.STRING_TEXT,  // to
				types.AT,           // @
				types.IDENTIFIER,   // var
				types.LPAREN,       // (
				types.IDENTIFIER,   // ENV
				types.RPAREN,       // )
				types.STRING_END,   // "
				types.RPAREN,       // )
				types.AT,           // @ (AFTER COMPLEX STRING INTERPOLATION)
				types.IDENTIFIER,   // parallel
				types.LBRACE,       // {
				types.SHELL_TEXT,   // echo
				types.STRING_START, // "
				types.STRING_TEXT,  // deployment complete
				types.STRING_END,   // "
				types.SHELL_END,    // End of shell content
				types.RBRACE,       // }
				types.RBRACE,       // }
				types.EOF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(strings.NewReader(tt.input))
			var tokens []types.Token

			for {
				tok := lexer.NextToken()
				tokens = append(tokens, tok)
				if tok.Type == types.EOF {
					break
				}
			}

			// Extract token types
			var gotTypes []types.TokenType
			for _, tok := range tokens {
				gotTypes = append(gotTypes, tok.Type)
			}

			// Compare lengths first
			if len(gotTypes) != len(tt.tokens) {
				t.Errorf("Token count mismatch: got %d, want %d", len(gotTypes), len(tt.tokens))
				t.Logf("Got tokens:")
				for i, tok := range tokens {
					t.Logf("  [%d] %s: %q", i, tok.Type.String(), tok.Value)
				}
				t.Logf("Expected tokens:")
				for i, tokenType := range tt.tokens {
					t.Logf("  [%d] %s", i, tokenType.String())
				}
				return
			}

			// Compare token by token
			for i, expected := range tt.tokens {
				if gotTypes[i] != expected {
					t.Errorf("Token mismatch at position %d: got %s, want %s", i, gotTypes[i].String(), expected.String())
					t.Logf("Context around position %d:", i)
					start := maxInt(0, i-3)
					end := minInt(len(tokens), i+4)
					for j := start; j < end; j++ {
						marker := "   "
						if j == i {
							marker = ">>>"
						}
						t.Logf("  %s[%d] %s: %q", marker, j, tokens[j].Type.String(), tokens[j].Value)
					}
					return
				}
			}
		})
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
