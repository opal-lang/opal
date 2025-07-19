package lexer

import (
	"strings"
	"testing"
)

func TestStateMachineTransitions(t *testing.T) {
	tests := []struct {
		name   string
		tokens []struct {
			typ   TokenType
			value string
		}
		wantStates []LexerState
		wantError  bool
	}{
		{
			name: "simple command",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello"},
				{EOF, ""},
			},
			wantStates: []LexerState{
				StateTopLevel,       // build
				StateAfterColon,     // :
				StateCommandContent, // echo hello
				StateTopLevel,       // EOF
			},
		},
		{
			name: "command with block",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo hello"},
				{RBRACE, "}"},
			},
			wantStates: []LexerState{
				StateTopLevel,       // build
				StateAfterColon,     // :
				StateCommandContent, // {
				StateCommandContent, // echo hello
				StateTopLevel,       // }
			},
		},
		{
			name: "variable declaration followed by command",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{VAR, "var"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"},
				{IDENTIFIER, "server"},
				{COLON, ":"},
			},
			wantStates: []LexerState{
				StateVarDecl,    // var
				StateVarDecl,    // PORT
				StateVarValue,   // =
				StateVarValue,   // 8080
				StateTopLevel,   // server (transition back from VarValue)
				StateAfterColon, // :
			},
		},
		{
			name: "variable declaration with function decorator",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{VAR, "var"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"},
				{IDENTIFIER, "server"},
				{COLON, ":"},
				{SHELL_TEXT, "go run main.go --port=@var(PORT)"},
			},
			wantStates: []LexerState{
				StateVarDecl,        // var
				StateVarDecl,        // PORT
				StateVarValue,       // =
				StateVarValue,       // 8080
				StateTopLevel,       // server (critical transition from VarValue to TopLevel)
				StateAfterColon,     // :
				StateCommandContent, // go run main.go --port=@var(PORT)
			},
		},
		{
			name: "decorator with command",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "30s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm run build"},
				{RBRACE, "}"},
			},
			wantStates: []LexerState{
				StateTopLevel,       // build
				StateAfterColon,     // :
				StateDecorator,      // @
				StateDecorator,      // timeout
				StateDecoratorArgs,  // (
				StateDecoratorArgs,  // 30s
				StateAfterDecorator, // )
				StateCommandContent, // {
				StateCommandContent, // npm run build
				StateTopLevel,       // }
			},
		},
		{
			name: "pattern decorator @when",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{SHELL_TEXT, "echo prod"},
				{IDENTIFIER, "dev"},
				{COLON, ":"},
				{SHELL_TEXT, "echo dev"},
				{RBRACE, "}"},
			},
			wantStates: []LexerState{
				StateTopLevel,          // deploy
				StateAfterColon,        // :
				StateDecorator,         // @
				StateDecorator,         // when
				StateDecoratorArgs,     // (
				StateDecoratorArgs,     // ENV
				StateAfterDecorator,    // )
				StatePatternBlock,      // {
				StatePatternBlock,      // prod
				StateAfterPatternColon, // :
				StateCommandContent,    // echo prod
				StatePatternBlock,      // dev (newline brings us back to pattern)
				StateAfterPatternColon, // :
				StateCommandContent,    // echo dev
				StateTopLevel,          // }
			},
		},
		{
			name: "nested decorators in pattern",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "when"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "60s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "deploy prod"},
				{RBRACE, "}"},
				{RBRACE, "}"},
			},
			wantStates: []LexerState{
				StateTopLevel,          // deploy
				StateAfterColon,        // :
				StateDecorator,         // @
				StateDecorator,         // when
				StateDecoratorArgs,     // (
				StateDecoratorArgs,     // ENV
				StateAfterDecorator,    // )
				StatePatternBlock,      // {
				StatePatternBlock,      // prod
				StateAfterPatternColon, // :
				StateDecorator,         // @
				StateDecorator,         // timeout
				StateDecoratorArgs,     // (
				StateDecoratorArgs,     // 60s
				StateAfterDecorator,    // )
				StateCommandContent,    // { (NOT PatternBlock!)
				StateCommandContent,    // deploy prod
				StatePatternBlock,      // } (back to pattern)
				StateTopLevel,          // } (exit pattern)
			},
		},
		{
			name: "try pattern with nested decorators",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "try"},
				{LBRACE, "{"},
				{IDENTIFIER, "main"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "retry"},
				{LPAREN, "("},
				{NUMBER, "3"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{SHELL_TEXT, "npm test"},
				{RBRACE, "}"},
				{RBRACE, "}"},
			},
			wantStates: []LexerState{
				StateTopLevel,          // test
				StateAfterColon,        // :
				StateDecorator,         // @
				StateAfterDecorator,    // try (no args)
				StatePatternBlock,      // {
				StatePatternBlock,      // main
				StateAfterPatternColon, // :
				StateDecorator,         // @
				StateDecorator,         // retry
				StateDecoratorArgs,     // (
				StateDecoratorArgs,     // 3
				StateAfterDecorator,    // )
				StateCommandContent,    // {
				StateCommandContent,    // npm test
				StatePatternBlock,      // }
				StateTopLevel,          // }
			},
		},
		{
			name: "decorator without args directly to shell",
			tokens: []struct {
				typ   TokenType
				value string
			}{
				{IDENTIFIER, "prod"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "parallel"},
				{SHELL_TEXT, "echo test"},
			},
			wantStates: []LexerState{
				StateTopLevel,       // prod
				StateAfterColon,     // :
				StateDecorator,      // @
				StateAfterDecorator, // parallel (no args)
				StateCommandContent, // echo test
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()

			for i, tok := range tt.tokens {
				state, err := sm.HandleToken(tok.typ, tok.value)

				if tt.wantError && err == nil {
					t.Errorf("Expected error at token %d, got none", i)
				}
				if !tt.wantError && err != nil {
					t.Errorf("Unexpected error at token %d: %v", i, err)
				}

				if i < len(tt.wantStates) {
					if state != tt.wantStates[i] {
						t.Errorf("Token %d (%s): want state %s, got %s",
							i, tok.typ, tt.wantStates[i], state)
					}
				}
			}
		})
	}
}

func TestStateMachineInvalidTransitions(t *testing.T) {
	tests := []struct {
		name      string
		fromState LexerState
		token     TokenType
		wantError bool
	}{
		{
			name:      "var not at top level",
			fromState: StateCommandContent,
			token:     VAR,
			wantError: true,
		},
		{
			name:      "@ in variable declaration",
			fromState: StateVarDecl,
			token:     AT,
			wantError: true,
		},
		{
			name:      "@ at top level",
			fromState: StateTopLevel,
			token:     AT,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()
			sm.current = tt.fromState

			_, err := sm.HandleToken(tt.token, "")

			if tt.wantError && err == nil {
				t.Errorf("Expected error for %s in state %s, got none",
					tt.token, tt.fromState)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error for %s in state %s: %v",
					tt.token, tt.fromState, err)
			}
		})
	}
}

func TestStateMachineContextStack(t *testing.T) {
	sm := NewStateMachine()

	// Test push/pop
	ctx1 := Context{Type: ContextCommand, State: StateTopLevel}
	sm.PushContext(ctx1)

	if sm.CurrentContext() == nil {
		t.Error("Expected context, got nil")
	}

	ctx2 := Context{Type: ContextBlock, State: StateCommandContent, BraceLevel: 1}
	sm.PushContext(ctx2)

	// Pop and verify
	popped, err := sm.PopContext()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if popped.Type != ContextBlock {
		t.Errorf("Expected ContextBlock, got %v", popped.Type)
	}

	// Pop again
	popped, err = sm.PopContext()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if popped.Type != ContextCommand {
		t.Errorf("Expected ContextCommand, got %v", popped.Type)
	}

	// Pop from empty should error
	_, err = sm.PopContext()
	if err == nil {
		t.Error("Expected error from empty pop, got nil")
	}
}

func TestStateMachineGetMode(t *testing.T) {
	tests := []struct {
		state    LexerState
		wantMode LexerMode
	}{
		{StateTopLevel, LanguageMode},
		{StateAfterColon, LanguageMode},
		{StateCommandContent, CommandMode},
		{StatePatternBlock, PatternMode},
		{StateAfterPatternColon, PatternMode},
		{StateDecorator, LanguageMode},
		{StateVarDecl, LanguageMode},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			sm := NewStateMachine()
			sm.current = tt.state

			mode := sm.GetMode()
			if mode != tt.wantMode {
				t.Errorf("State %s: want mode %d, got %d",
					tt.state, tt.wantMode, mode)
			}
		})
	}
}

func TestIsInPatternContext(t *testing.T) {
	sm := NewStateMachine()

	// Initially not in pattern
	if sm.IsInPatternContext() {
		t.Error("Expected not in pattern context initially")
	}

	// Push pattern context
	sm.PushContext(Context{
		Type:      ContextPatternBlock,
		State:     StateAfterDecorator,
		IsPattern: true,
	})

	if !sm.IsInPatternContext() {
		t.Error("Expected to be in pattern context after push")
	}

	// Push command context on top
	sm.PushContext(Context{
		Type:  ContextBlock,
		State: StateCommandContent,
	})

	// Should still be in pattern context (nested)
	if !sm.IsInPatternContext() {
		t.Error("Expected to still be in pattern context when nested")
	}

	// Set current state to pattern
	sm.current = StatePatternBlock
	if !sm.IsInPatternContext() {
		t.Error("Expected to be in pattern context with pattern state")
	}
}

func TestBraceBalancing(t *testing.T) {
	sm := NewStateMachine()

	// Test negative brace level detection
	sm.braceLevel = 0
	_, err := sm.HandleToken(RBRACE, "}")
	if err == nil || !strings.Contains(err.Error(), "unmatched") {
		t.Error("Expected error for unmatched closing brace")
	}

	// Test closeContextsUpTo
	sm = NewStateMachine()
	sm.braceLevel = 3
	sm.PushContext(Context{Type: ContextBlock, BraceLevel: 1})
	sm.PushContext(Context{Type: ContextBlock, BraceLevel: 2})
	sm.PushContext(Context{Type: ContextBlock, BraceLevel: 3})

	err = sm.closeContextsUpTo(1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(sm.contextStack) != 1 {
		t.Errorf("Expected 1 context after closing to level 1, got %d", len(sm.contextStack))
	}
}

func TestDebugMode(t *testing.T) {
	sm := NewStateMachine()
	sm.SetDebug(true)

	// This should print debug output
	if err := sm.Transition(StateAfterColon); err != nil {
		t.Errorf("Transition failed: %v", err)
	}

	// Just verify it doesn't panic
}

func TestEmptyPatternBranch(t *testing.T) {
	// Test: command: @when(ENV) { prod: dev: echo dev }
	// This tests an empty pattern branch followed by a non-empty one
	sm := NewStateMachine()

	tokens := []struct {
		typ   TokenType
		value string
	}{
		{IDENTIFIER, "deploy"}, // Missing command declaration
		{COLON, ":"},           // Command colon
		{AT, "@"},              // Now @ is valid (StateAfterColon)
		{IDENTIFIER, "when"},
		{LPAREN, "("},
		{IDENTIFIER, "ENV"},
		{RPAREN, ")"},
		{LBRACE, "{"}, // Enter pattern block
		{IDENTIFIER, "prod"},
		{COLON, ":"}, // Empty branch
		{IDENTIFIER, "dev"},
		{COLON, ":"},
		{SHELL_TEXT, "echo dev"},
		{RBRACE, "}"},
	}

	expectedStates := []LexerState{
		StateTopLevel,          // deploy
		StateAfterColon,        // :
		StateDecorator,         // @
		StateDecorator,         // when
		StateDecoratorArgs,     // (
		StateDecoratorArgs,     // ENV
		StateAfterDecorator,    // )
		StatePatternBlock,      // { (pattern block because when is pattern decorator)
		StatePatternBlock,      // prod
		StateAfterPatternColon, // : (in pattern context)
		StatePatternBlock,      // dev (back to pattern after empty branch)
		StateAfterPatternColon, // :
		StateCommandContent,    // echo dev (shell content in pattern branch)
		StateTopLevel,          // } (exit pattern)
	}

	for i, tok := range tokens {
		state, err := sm.HandleToken(tok.typ, tok.value)
		if err != nil {
			t.Errorf("Token %d (%s=%s): unexpected error: %v", i, tok.typ, tok.value, err)
		}

		if i < len(expectedStates) {
			if state != expectedStates[i] {
				t.Errorf("Token %d (%s=%s): expected state %s, got %s",
					i, tok.typ, tok.value, expectedStates[i], state)
			}
		}

		// Specific checks for pattern transitions
		if tok.typ == COLON && tok.value == ":" && i == 9 { // After "prod:"
			if state != StateAfterPatternColon {
				t.Errorf("After prod:, expected StateAfterPatternColon, got %s", state)
			}
		}

		if tok.typ == IDENTIFIER && tok.value == "dev" && i == 10 { // "dev" identifier
			if state != StatePatternBlock {
				t.Errorf("At dev identifier, expected StatePatternBlock, got %s", state)
			}
		}
	}
}

func TestTransitionTableGeneration(t *testing.T) {
	// Verify the generated transition table has expected properties
	transitions := generateTransitionMap()

	// Every state should have at least one valid transition
	for state := StateTopLevel; state <= StateVarValue; state++ {
		if _, exists := transitions[state]; !exists {
			t.Errorf("State %s has no transitions defined", state)
		}
	}

	// Self-loops should be allowed for states that need them
	selfLoopStates := []LexerState{
		StateTopLevel,
		StateCommandContent,
		StatePatternBlock,
		StateDecorator,
		StateDecoratorArgs,
		StateVarDecl,
		StateVarValue,
	}

	for _, state := range selfLoopStates {
		if allowed, exists := transitions[state]; exists {
			if !allowed[state] {
				t.Errorf("State %s should allow self-loop", state)
			}
		}
	}
}
