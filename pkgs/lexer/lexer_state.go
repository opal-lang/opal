package lexer

import (
	"fmt"
)

// LexerState represents the current parsing state
type LexerState int

const (
	// StateTopLevel is the initial state for parsing structural elements
	StateTopLevel LexerState = iota

	// StateAfterColon is after seeing ':' in a command declaration
	StateAfterColon

	// StateCommandContent is parsing shell command text
	StateCommandContent

	// StatePatternBlock is inside a pattern-matching block (@when/@try)
	StatePatternBlock

	// StateAfterPatternColon is after seeing ':' in a pattern branch
	StateAfterPatternColon

	// StateDecorator is parsing decorator (collapsed from Start/Name)
	StateDecorator

	// StateDecoratorArgs is parsing decorator arguments
	StateDecoratorArgs

	// StateAfterDecorator is after parsing a complete decorator
	StateAfterDecorator

	// StateVarDecl is parsing variable declarations
	StateVarDecl

	// StateVarValue is parsing variable value
	StateVarValue
)

//go:generate stringer -type=LexerState,ContextType

// String returns a human-readable state name
func (s LexerState) String() string {
	names := []string{
		"TopLevel",
		"AfterColon",
		"CommandContent",
		"PatternBlock",
		"AfterPatternColon",
		"Decorator",
		"DecoratorArgs",
		"AfterDecorator",
		"VarDecl",
		"VarValue",
	}
	if int(s) < len(names) {
		return names[s]
	}
	return fmt.Sprintf("Unknown(%d)", s)
}

// ContextType represents the type of parsing context
type ContextType int

const (
	ContextTop ContextType = iota
	ContextCommand
	ContextBlock
	ContextPatternBlock
	ContextDecorator
	ContextVar
)

// Context represents a parsing context that can be pushed/popped
type Context struct {
	Type       ContextType
	State      LexerState
	BraceLevel int
	IsPattern  bool   // true for @when/@try decorators
	Decorator  string // decorator name if in decorator context
}

// StateMachine manages lexer state transitions
type StateMachine struct {
	current      LexerState
	contextStack []Context
	braceLevel   int
	debug        bool
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		current:      StateTopLevel,
		contextStack: make([]Context, 0, 8),
		braceLevel:   0,
		debug:        false,
	}
}

// SetDebug enables/disables debug logging
func (sm *StateMachine) SetDebug(enabled bool) {
	sm.debug = enabled
}

// Current returns the current state
func (sm *StateMachine) Current() LexerState {
	return sm.current
}

// BraceLevel returns the current brace nesting level
func (sm *StateMachine) BraceLevel() int {
	return sm.braceLevel
}

// PushContext saves the current context before entering a new one
func (sm *StateMachine) PushContext(ctx Context) {
	sm.contextStack = append(sm.contextStack, ctx)
}

// PopContext restores the previous context
func (sm *StateMachine) PopContext() (Context, error) {
	if len(sm.contextStack) == 0 {
		return Context{}, fmt.Errorf("context stack underflow")
	}
	ctx := sm.contextStack[len(sm.contextStack)-1]
	sm.contextStack = sm.contextStack[:len(sm.contextStack)-1]
	return ctx, nil
}

// CurrentContext returns the current context without popping
func (sm *StateMachine) CurrentContext() *Context {
	if len(sm.contextStack) == 0 {
		return nil
	}
	return &sm.contextStack[len(sm.contextStack)-1]
}

// IsInPatternContext checks if we're inside any pattern block
func (sm *StateMachine) IsInPatternContext() bool {
	for _, ctx := range sm.contextStack {
		if ctx.Type == ContextPatternBlock {
			return true
		}
	}
	return sm.current == StatePatternBlock
}

// Transition attempts to transition to a new state
func (sm *StateMachine) Transition(to LexerState) error {
	if !sm.isValidTransition(sm.current, to) {
		return fmt.Errorf("invalid transition from %s to %s", sm.current, to)
	}

	if sm.debug {
		fmt.Printf("STATE: %s → %s\n", sm.current, to)
	}

	sm.current = to
	return nil
}

// closeContextsUpTo pops contexts until we reach the given brace level
func (sm *StateMachine) closeContextsUpTo(targetLevel int) error {
	for len(sm.contextStack) > 0 {
		ctx := sm.contextStack[len(sm.contextStack)-1]
		if ctx.BraceLevel <= targetLevel {
			break
		}
		if _, err := sm.PopContext(); err != nil {
			return err
		}
	}
	return nil
}

// HandleToken processes a token and updates state accordingly
func (sm *StateMachine) HandleToken(tokenType TokenType, value string) (LexerState, error) {
	switch tokenType {
	case VAR:
		return sm.handleVar()
	case COLON:
		return sm.handleColon()
	case AT:
		return sm.handleAt()
	case LBRACE:
		return sm.handleLBrace()
	case RBRACE:
		return sm.handleRBrace()
	case LPAREN:
		return sm.handleLParen()
	case RPAREN:
		return sm.handleRParen()
	case IDENTIFIER:
		return sm.handleIdentifier(value)
	case WHEN, TRY:
		// Pattern decorator keywords - treat like identifiers for state machine
		return sm.handleIdentifier(value)
	case EQUALS:
		return sm.handleEquals()
	// NEWLINE removed - handled as whitespace
	case SHELL_TEXT:
		return sm.handleShellText()
	case EOF:
		return sm.handleEOF()
	default:
		// Most tokens don't trigger state changes
		return sm.current, nil
	}
}

// Token-specific handlers

func (sm *StateMachine) handleVar() (LexerState, error) {
	if sm.current != StateTopLevel {
		return sm.current, fmt.Errorf("var keyword only allowed at top level")
	}
	sm.PushContext(Context{
		Type:  ContextVar,
		State: sm.current,
	})
	return StateVarDecl, sm.Transition(StateVarDecl)
}

func (sm *StateMachine) handleColon() (LexerState, error) {
	switch sm.current {
	case StateTopLevel:
		// Command declaration: "build:"
		return StateAfterColon, sm.Transition(StateAfterColon)
	case StatePatternBlock:
		// Pattern branch: "prod:"
		return StateAfterPatternColon, sm.Transition(StateAfterPatternColon)
	default:
		// Colon in other contexts (like shell content) doesn't change state
		return sm.current, nil
	}
}

func (sm *StateMachine) handleAt() (LexerState, error) {
	// @ can appear in many contexts to start a decorator
	switch sm.current {
	case StateAfterColon, StateAfterPatternColon, StateCommandContent, StateAfterDecorator:
		sm.PushContext(Context{
			Type:  ContextDecorator,
			State: sm.current,
		})
		return StateDecorator, sm.Transition(StateDecorator)
	default:
		return sm.current, fmt.Errorf("@ not allowed in %s", sm.current)
	}
}

func (sm *StateMachine) handleLBrace() (LexerState, error) {
	sm.braceLevel++

	switch sm.current {
	case StateAfterColon:
		// Command block: "build: {"
		sm.PushContext(Context{
			Type:       ContextBlock,
			State:      sm.current,
			BraceLevel: sm.braceLevel,
		})
		return StateCommandContent, sm.Transition(StateCommandContent)

	case StateAfterDecorator:
		// Decorator block
		ctx := sm.CurrentContext()
		if ctx != nil && ctx.Type == ContextDecorator {
			if ctx.IsPattern {
				// Pattern decorator block: "@when(VAR) {"
				sm.PushContext(Context{
					Type:       ContextPatternBlock,
					State:      sm.current,
					BraceLevel: sm.braceLevel,
					IsPattern:  true,
				})
				return StatePatternBlock, sm.Transition(StatePatternBlock)
			} else {
				// Regular decorator block: "@timeout(30s) {"
				sm.PushContext(Context{
					Type:       ContextBlock,
					State:      sm.current,
					BraceLevel: sm.braceLevel,
				})
				return StateCommandContent, sm.Transition(StateCommandContent)
			}
		}

	case StateAfterPatternColon:
		// Nested block in pattern: "prod: {"
		sm.PushContext(Context{
			Type:       ContextBlock,
			State:      sm.current,
			BraceLevel: sm.braceLevel,
		})
		return StateCommandContent, sm.Transition(StateCommandContent)

	default:
		// { in other contexts doesn't change state
		return sm.current, nil
	}

	// Add missing return for edge cases
	return sm.current, nil
}

func (sm *StateMachine) handleRBrace() (LexerState, error) {
	if sm.braceLevel <= 0 {
		return sm.current, fmt.Errorf("unmatched closing brace")
	}

	sm.braceLevel--

	// Find the context that this brace closes
	var targetContext *Context
	for i := len(sm.contextStack) - 1; i >= 0; i-- {
		ctx := &sm.contextStack[i]
		if ctx.BraceLevel == sm.braceLevel+1 {
			targetContext = ctx
			break
		}
	}

	// Close contexts that were opened at a higher brace level
	if err := sm.closeContextsUpTo(sm.braceLevel); err != nil {
		return sm.current, err
	}

	// Determine the state to return to based on the context we're closing
	if targetContext != nil {
		switch targetContext.Type {
		case ContextPatternBlock:
			// Closing a pattern block - check if we're returning to top level or staying in pattern
			if sm.braceLevel == 0 {
				return StateTopLevel, sm.Transition(StateTopLevel)
			} else {
				// Still nested in pattern
				return StatePatternBlock, sm.Transition(StatePatternBlock)
			}
		case ContextBlock:
			// Check if we're nested in a pattern context
			if sm.IsInPatternContext() && sm.braceLevel > 0 {
				return StatePatternBlock, sm.Transition(StatePatternBlock)
			} else if sm.braceLevel == 0 {
				return StateTopLevel, sm.Transition(StateTopLevel)
			} else {
				return StateCommandContent, sm.Transition(StateCommandContent)
			}
		}
	}

	// Fallback: return to appropriate state based on brace level
	if sm.braceLevel == 0 {
		return StateTopLevel, sm.Transition(StateTopLevel)
	} else if sm.IsInPatternContext() {
		return StatePatternBlock, sm.Transition(StatePatternBlock)
	} else {
		return StateCommandContent, sm.Transition(StateCommandContent)
	}
}

func (sm *StateMachine) handleLParen() (LexerState, error) {
	if sm.current == StateDecorator {
		// Start of decorator arguments
		return StateDecoratorArgs, sm.Transition(StateDecoratorArgs)
	}
	return sm.current, nil
}

func (sm *StateMachine) handleRParen() (LexerState, error) {
	if sm.current == StateDecoratorArgs {
		// End of decorator arguments
		return StateAfterDecorator, sm.Transition(StateAfterDecorator)
	}
	return sm.current, nil
}

func (sm *StateMachine) handleIdentifier(value string) (LexerState, error) {
	switch sm.current {
	case StateDecorator:
		// Decorator name after @
		ctx := sm.CurrentContext()
		if ctx != nil && ctx.Type == ContextDecorator {
			ctx.Decorator = value
			ctx.IsPattern = (value == "when" || value == "try")
		}
		// Check if this is a decorator without args
		if value == "try" || value == "parallel" {
			// These decorators have no args, go straight to AfterDecorator
			return StateAfterDecorator, sm.Transition(StateAfterDecorator)
		}
		// Stay in decorator state - expect args
		return sm.current, nil

	case StateAfterDecorator:
		// If we see an identifier after decorator, it's shell content
		return StateCommandContent, sm.Transition(StateCommandContent)

	case StateAfterPatternColon:
		// After pattern colon, identifier could be shell content or next pattern
		if sm.IsInPatternContext() {
			// In pattern context, identifier is likely the next pattern
			return StatePatternBlock, sm.Transition(StatePatternBlock)
		}
		// Otherwise it's shell content
		return StateCommandContent, sm.Transition(StateCommandContent)

	case StateCommandContent:
		// In command content, an identifier could be a new pattern (if in pattern context)
		// This happens when we finish executing a pattern branch command and see the next pattern
		if sm.IsInPatternContext() {
			// In pattern context, any identifier at command level could be a new pattern
			// We need to check if this makes sense as a pattern transition
			return StatePatternBlock, sm.Transition(StatePatternBlock)
		}
		// Otherwise stay in command content
		return sm.current, nil

	case StatePatternBlock:
		// Pattern identifier in pattern block
		return sm.current, nil

	case StateVarDecl:
		// Variable name in declaration
		return sm.current, nil

	case StateVarValue:
		// End of variable declaration - identifier indicates next statement
		if ctx, err := sm.PopContext(); err == nil && ctx.Type == ContextVar {
			// Clear the variable context and transition to top level
			if err := sm.Transition(StateTopLevel); err != nil {
				return sm.current, err
			}
			// Now handle this identifier as if we're at top level
			return sm.handleIdentifier(value)
		}
		return sm.current, nil

	default:
		return sm.current, nil
	}
}

func (sm *StateMachine) handleShellText() (LexerState, error) {
	switch sm.current {
	case StateAfterColon, StateAfterPatternColon:
		// Shell text after colon means we're entering command content
		return StateCommandContent, sm.Transition(StateCommandContent)
	case StateAfterDecorator:
		// Shell text after decorator means we're entering command content
		return StateCommandContent, sm.Transition(StateCommandContent)
	default:
		// In other states, shell text doesn't change state
		return sm.current, nil
	}
}

func (sm *StateMachine) handleEquals() (LexerState, error) {
	if sm.current == StateVarDecl {
		// After variable name, expect value
		return StateVarValue, sm.Transition(StateVarValue)
	}
	return sm.current, nil
}

func (sm *StateMachine) handleNewline() (LexerState, error) {
	switch sm.current {
	case StateCommandContent:
		// Newline in simple command ends it
		if sm.braceLevel == 0 {
			return StateTopLevel, sm.Transition(StateTopLevel)
		}
		// Inside braces, check the immediate context (not overall pattern context)
		ctx := sm.CurrentContext()
		if ctx != nil && ctx.Type == ContextPatternBlock {
			// We're directly in a pattern block, return to pattern mode
			return StatePatternBlock, sm.Transition(StatePatternBlock)
		}
		// Otherwise stay in command content (e.g., we're in a regular decorator block)
		return sm.current, nil

	case StateVarValue:
		// End of variable declaration
		if ctx, err := sm.PopContext(); err == nil && ctx.Type == ContextVar {
			return StateTopLevel, sm.Transition(StateTopLevel)
		}
		return sm.current, nil

	case StateAfterPatternColon:
		// Empty pattern branch - return to pattern block
		return StatePatternBlock, sm.Transition(StatePatternBlock)

	default:
		return sm.current, nil
	}
}

func (sm *StateMachine) handleEOF() (LexerState, error) {
	// EOF should close all contexts
	if err := sm.closeContextsUpTo(-1); err != nil {
		return sm.current, err
	}

	// Clear remaining context stack
	sm.contextStack = sm.contextStack[:0]
	sm.braceLevel = 0

	return StateTopLevel, sm.Transition(StateTopLevel)
}

// transitionTable defines valid state transitions
type transitionRule struct {
	from LexerState
	to   []LexerState
}

// generateTransitionMap creates the valid transitions map from rules
func generateTransitionMap() map[LexerState]map[LexerState]bool {
	rules := []transitionRule{
		{StateTopLevel, []LexerState{
			StateTopLevel,   // self-loop
			StateAfterColon, // saw ':'
			StateVarDecl,    // saw 'var'
		}},
		{StateAfterColon, []LexerState{
			StateCommandContent, // direct shell content or saw SHELL_TEXT
			StateDecorator,      // saw '@'
			StateTopLevel,       // empty command
		}},
		{StateCommandContent, []LexerState{
			StateCommandContent, // self-loop
			StateDecorator,      // saw '@' in command
			StateTopLevel,       // command ended
			StatePatternBlock,   // return to pattern after } OR saw pattern identifier
		}},
		{StatePatternBlock, []LexerState{
			StatePatternBlock,      // self-loop
			StateAfterPatternColon, // saw ':' after pattern
			StateTopLevel,          // pattern block ended
			StateDecorator,         // saw '@' in pattern
		}},
		{StateAfterPatternColon, []LexerState{
			StateCommandContent, // direct shell content or saw SHELL_TEXT
			StateDecorator,      // saw '@'
			StatePatternBlock,   // back to pattern (for newlines in patterns)
		}},
		{StateDecorator, []LexerState{
			StateDecorator,      // self-loop (parsing name)
			StateDecoratorArgs,  // saw '('
			StateAfterDecorator, // no args (like @try, @parallel)
		}},
		{StateDecoratorArgs, []LexerState{
			StateDecoratorArgs,  // self-loop
			StateAfterDecorator, // saw ')'
		}},
		{StateAfterDecorator, []LexerState{
			StateCommandContent, // saw '{' or shell content or SHELL_TEXT
			StatePatternBlock,   // saw '{' for pattern
			StateTopLevel,       // decorator with no block
			StateDecorator,      // chained decorator
		}},
		{StateVarDecl, []LexerState{
			StateVarDecl,  // self-loop
			StateVarValue, // saw '='
		}},
		{StateVarValue, []LexerState{
			StateVarValue, // self-loop
			StateTopLevel, // end of var
		}},
	}

	// Build map with self-loops included
	transitions := make(map[LexerState]map[LexerState]bool)
	for _, rule := range rules {
		allowed := make(map[LexerState]bool)
		for _, to := range rule.to {
			allowed[to] = true
		}
		transitions[rule.from] = allowed
	}

	return transitions
}

var validTransitions = generateTransitionMap()

// isValidTransition checks if a state transition is allowed
func (sm *StateMachine) isValidTransition(from, to LexerState) bool {
	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	return allowed[to]
}

// GetMode returns the appropriate lexer mode for the current state
func (sm *StateMachine) GetMode() LexerMode {
	switch sm.current {
	case StateCommandContent:
		return CommandMode
	case StatePatternBlock, StateAfterPatternColon:
		return PatternMode
	default:
		return LanguageMode
	}
}
