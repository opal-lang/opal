package ast

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/types"
)

// Node represents any node in the AST
type Node interface {
	String() string
	Position() Position
	TokenRange() TokenRange
	SemanticTokens() []types.Token
}

// Position represents source location information
type Position struct {
	Line   int
	Column int
	Offset int // Byte offset in source
}

// TokenRange represents the span of tokens for this AST node
type TokenRange struct {
	Start types.Token
	End   types.Token
	All   []types.Token
}

// Program represents the root of the CST (entire opal file)
// Preserves concrete syntax for LSP, Tree-sitter, and formatting tools
type Program struct {
	Variables []VariableDecl
	VarGroups []VarGroup // Grouped variable declarations: var ( ... )
	Commands  []CommandDecl
	Pos       Position
	Tokens    TokenRange
}

func (p *Program) String() string {
	var parts []string
	for _, v := range p.Variables {
		parts = append(parts, v.String())
	}
	for _, g := range p.VarGroups {
		parts = append(parts, g.String())
	}
	for _, c := range p.Commands {
		parts = append(parts, c.String())
	}
	return strings.Join(parts, "\n")
}

func (p *Program) Position() Position {
	return p.Pos
}

func (p *Program) TokenRange() TokenRange {
	return p.Tokens
}

func (p *Program) SemanticTokens() []types.Token {
	return p.Tokens.All
}

// VariableDecl represents variable declarations (both individual and grouped)
type VariableDecl struct {
	Name   string
	Value  Expression
	Pos    Position
	Tokens TokenRange

	// LSP-specific information
	NameToken  types.Token
	ValueToken types.Token
}

func (v *VariableDecl) String() string {
	return fmt.Sprintf("var %s = %s", v.Name, v.Value.String())
}

func (v *VariableDecl) Position() Position {
	return v.Pos
}

func (v *VariableDecl) TokenRange() TokenRange {
	return v.Tokens
}

func (v *VariableDecl) SemanticTokens() []types.Token {
	tokens := []types.Token{v.NameToken, v.ValueToken}
	for _, token := range v.Tokens.All {
		if token.Type == types.IDENTIFIER && token.Value == v.Name {
			token.Semantic = types.SemVariable
		}
	}
	return tokens
}

// VarGroup represents grouped variable declarations: var ( NAME = value; ANOTHER = value )
// Preserves the concrete syntax for formatting and LSP features
type VarGroup struct {
	Variables []VariableDecl
	Pos       Position
	Tokens    TokenRange

	// Concrete syntax tokens for precise formatting
	VarToken   types.Token // The "var" keyword
	OpenParen  types.Token // The "(" token
	CloseParen types.Token // The ")" token
}

func (g *VarGroup) String() string {
	var parts []string
	parts = append(parts, "var (")
	for _, v := range g.Variables {
		parts = append(parts, fmt.Sprintf("  %s = %s", v.Name, v.Value.String()))
	}
	parts = append(parts, ")")
	return strings.Join(parts, "\n")
}

func (g *VarGroup) Position() Position {
	return g.Pos
}

func (g *VarGroup) TokenRange() TokenRange {
	return g.Tokens
}

func (g *VarGroup) SemanticTokens() []types.Token {
	var tokens []types.Token

	// Add structural tokens with proper semantics
	varToken := g.VarToken
	varToken.Semantic = types.SemKeyword
	tokens = append(tokens, varToken)

	tokens = append(tokens, g.OpenParen)

	// Add variable tokens
	for _, v := range g.Variables {
		tokens = append(tokens, v.SemanticTokens()...)
	}

	tokens = append(tokens, g.CloseParen)

	return tokens
}

// NamedParameter represents a named parameter in decorator arguments
// Supports both named syntax (name = value) and positional (resolved by parser)
type NamedParameter struct {
	Name   string     // Parameter name (e.g., "concurrency", "failOnFirstError")
	Value  Expression // Parameter value
	Pos    Position
	Tokens TokenRange

	// Concrete syntax tokens for LSP support
	NameToken   *types.Token // The parameter name token (nil for positional args)
	EqualsToken *types.Token // The "=" token (nil for positional args)
}

func (n NamedParameter) String() string {
	if n.NameToken != nil {
		return fmt.Sprintf("%s = %s", n.Name, n.Value.String())
	}
	return n.Value.String() // Positional argument
}

func (n NamedParameter) Position() Position {
	return n.Pos
}

func (n NamedParameter) TokenRange() TokenRange {
	return n.Tokens
}

func (n NamedParameter) SemanticTokens() []types.Token {
	var tokens []types.Token
	if n.NameToken != nil {
		nameToken := *n.NameToken
		nameToken.Semantic = types.SemParameter
		tokens = append(tokens, nameToken)
	}
	if n.EqualsToken != nil {
		tokens = append(tokens, *n.EqualsToken)
	}
	tokens = append(tokens, n.Value.SemanticTokens()...)
	return tokens
}

// IsNamed returns true if this parameter was specified with a name
func (n NamedParameter) IsNamed() bool {
	return n.NameToken != nil
}

// Helper functions for working with named parameters

// FindParameter searches for a parameter by name in the slice
func FindParameter(params []NamedParameter, name string) *NamedParameter {
	for i := range params {
		if params[i].Name == name {
			return &params[i]
		}
	}
	return nil
}

// GetStringParam retrieves a string parameter value with default fallback
func GetStringParam(params []NamedParameter, name string, defaultValue string) string {
	if param := FindParameter(params, name); param != nil {
		if str, ok := param.Value.(*StringLiteral); ok {
			return str.String()
		}
	}
	return defaultValue
}

// GetIntParam retrieves an integer parameter value with default fallback
func GetIntParam(params []NamedParameter, name string, defaultValue int) int {
	if param := FindParameter(params, name); param != nil {
		if num, ok := param.Value.(*NumberLiteral); ok {
			if val, err := strconv.Atoi(num.Value); err == nil {
				return val
			}
		}
	}
	return defaultValue
}

// GetBoolParam retrieves a boolean parameter value with default fallback
func GetBoolParam(params []NamedParameter, name string, defaultValue bool) bool {
	if param := FindParameter(params, name); param != nil {
		if b, ok := param.Value.(*BooleanLiteral); ok {
			return b.Value
		}
	}
	return defaultValue
}

// GetDurationParam retrieves a duration parameter value with default fallback
func GetDurationParam(params []NamedParameter, name string, defaultValue time.Duration) time.Duration {
	if param := FindParameter(params, name); param != nil {
		if dur, ok := param.Value.(*DurationLiteral); ok {
			if d, err := time.ParseDuration(dur.Value); err == nil {
				return d
			}
		}
	}
	return defaultValue
}

// Expression represents any expression (literals, identifiers, etc.)
type Expression interface {
	Node
	IsExpression() bool
	GetType() types.ExpressionType
}

// Use types from the shared types package
type ExpressionType = types.ExpressionType

const (
	StringType     = types.StringType
	NumberType     = types.NumberType
	DurationType   = types.DurationType
	IdentifierType = types.IdentifierType
	BooleanType    = types.BooleanType
)

// StringLiteral represents string values with support for embedded value decorators
type StringLiteral struct {
	Parts       []StringPart // Always use parts, even for simple strings
	Raw         string       // Original raw string
	Pos         Position
	Tokens      TokenRange
	StringToken types.Token
}

func (s *StringLiteral) String() string {
	var result strings.Builder
	for _, part := range s.Parts {
		result.WriteString(part.String())
	}
	return result.String()
}

// StringPart represents a part of string content (text or inline value decorator)
type StringPart interface {
	Node
	IsStringPart() bool
	IsShellPart() bool
}

// TextStringPart represents plain text within a string literal
type TextStringPart struct {
	Text   string
	Pos    Position
	Tokens TokenRange
}

func (t *TextStringPart) String() string {
	return t.Text
}

func (t *TextStringPart) Position() Position {
	return t.Pos
}

func (t *TextStringPart) TokenRange() TokenRange {
	return t.Tokens
}

func (t *TextStringPart) SemanticTokens() []types.Token {
	tokens := make([]types.Token, len(t.Tokens.All))
	copy(tokens, t.Tokens.All)

	// Mark all tokens as string content
	for i := range tokens {
		tokens[i].Semantic = types.SemString
	}

	return tokens
}

func (t *TextStringPart) IsStringPart() bool {
	return true
}

func (t *TextStringPart) IsShellPart() bool {
	return true
}

func (s *StringLiteral) Position() Position {
	return s.Pos
}

func (s *StringLiteral) TokenRange() TokenRange {
	return s.Tokens
}

func (s *StringLiteral) SemanticTokens() []types.Token {
	return []types.Token{s.StringToken}
}

func (s *StringLiteral) IsExpression() bool {
	return true
}

func (s *StringLiteral) GetType() ExpressionType {
	return StringType
}

// NumberLiteral represents numeric values
type NumberLiteral struct {
	Value  string
	Pos    Position
	Tokens TokenRange
	Token  types.Token
}

func (n *NumberLiteral) String() string {
	return n.Value
}

func (n *NumberLiteral) Position() Position {
	return n.Pos
}

func (n *NumberLiteral) TokenRange() TokenRange {
	return n.Tokens
}

func (n *NumberLiteral) SemanticTokens() []types.Token {
	return []types.Token{n.Token}
}

func (n *NumberLiteral) IsExpression() bool {
	return true
}

func (n *NumberLiteral) GetType() ExpressionType {
	return NumberType
}

// DurationLiteral represents duration values like 30s, 5m
type DurationLiteral struct {
	Value  string
	Pos    Position
	Tokens TokenRange
	Token  types.Token
}

func (d *DurationLiteral) String() string {
	return d.Value
}

func (d *DurationLiteral) Position() Position {
	return d.Pos
}

func (d *DurationLiteral) TokenRange() TokenRange {
	return d.Tokens
}

func (d *DurationLiteral) SemanticTokens() []types.Token {
	return []types.Token{d.Token}
}

func (d *DurationLiteral) IsExpression() bool {
	return true
}

func (d *DurationLiteral) GetType() ExpressionType {
	return DurationType
}

// BooleanLiteral represents boolean values (true/false)
type BooleanLiteral struct {
	Value  bool   // The boolean value
	Raw    string // The raw string ("true" or "false")
	Pos    Position
	Tokens TokenRange
	Token  types.Token
}

func (b *BooleanLiteral) String() string {
	return b.Raw
}

func (b *BooleanLiteral) Position() Position {
	return b.Pos
}

func (b *BooleanLiteral) TokenRange() TokenRange {
	return b.Tokens
}

func (b *BooleanLiteral) SemanticTokens() []types.Token {
	token := b.Token
	token.Semantic = types.SemBoolean
	return []types.Token{token}
}

func (b *BooleanLiteral) IsExpression() bool {
	return true
}

func (b *BooleanLiteral) GetType() ExpressionType {
	return BooleanType
}

// Identifier represents identifiers
type Identifier struct {
	Name   string
	Pos    Position
	Tokens TokenRange
	Token  types.Token
}

func (i *Identifier) String() string {
	return i.Name
}

func (i *Identifier) Position() Position {
	return i.Pos
}

func (i *Identifier) TokenRange() TokenRange {
	return i.Tokens
}

func (i *Identifier) SemanticTokens() []types.Token {
	return []types.Token{i.Token}
}

func (i *Identifier) IsExpression() bool {
	return true
}

func (i *Identifier) GetType() ExpressionType {
	return IdentifierType
}

// CommandDecl represents command definitions with concrete syntax preservation
type CommandDecl struct {
	Name   string
	Type   CommandType
	Body   CommandBody
	Pos    Position
	Tokens TokenRange

	// Concrete syntax tokens for precise formatting and LSP
	TypeToken  *types.Token // The watch/stop keyword (nil for regular commands)
	NameToken  types.Token  // The command name token
	ColonToken types.Token  // The ":" token
}

func (c *CommandDecl) String() string {
	typeStr := ""
	switch c.Type {
	case WatchCommand:
		typeStr = "watch "
	case StopCommand:
		typeStr = "stop "
	case Command:
		typeStr = ""
	}

	return fmt.Sprintf("%s%s: %s", typeStr, c.Name, c.Body.String())
}

func (c *CommandDecl) Position() Position {
	return c.Pos
}

func (c *CommandDecl) TokenRange() TokenRange {
	return c.Tokens
}

func (c *CommandDecl) SemanticTokens() []types.Token {
	var tokens []types.Token

	if c.TypeToken != nil && c.TypeToken.Type != types.ILLEGAL {
		typeToken := *c.TypeToken
		typeToken.Semantic = types.SemKeyword
		tokens = append(tokens, typeToken)
	}

	nameToken := c.NameToken
	nameToken.Semantic = types.SemCommand
	tokens = append(tokens, nameToken)

	tokens = append(tokens, c.Body.SemanticTokens()...)

	return tokens
}

// CommandType represents the type of command
type CommandType int

const (
	Command CommandType = iota
	WatchCommand
	StopCommand
)

func (ct CommandType) String() string {
	switch ct {
	case Command:
		return "command"
	case WatchCommand:
		return "watch"
	case StopCommand:
		return "stop"
	default:
		return "unknown"
	}
}

// CommandBody represents the unified body of a command with concrete syntax preservation
// Now supports multiple content items for complex command structures
type CommandBody struct {
	Content []CommandContent // Multiple content items within the command body
	Pos     Position
	Tokens  TokenRange

	// Concrete syntax tokens for precise formatting
	OpenBrace  *types.Token // The "{" token (nil for simple commands)
	CloseBrace *types.Token // The "}" token (nil for simple commands)
}

func (b *CommandBody) String() string {
	var parts []string
	for _, content := range b.Content {
		parts = append(parts, content.String())
	}

	contentStr := strings.Join(parts, " ")

	return contentStr
}

func (b *CommandBody) Position() Position {
	return b.Pos
}

func (b *CommandBody) TokenRange() TokenRange {
	return b.Tokens
}

func (b *CommandBody) SemanticTokens() []types.Token {
	var tokens []types.Token

	if b.OpenBrace != nil {
		tokens = append(tokens, *b.OpenBrace)
	}

	for _, content := range b.Content {
		tokens = append(tokens, content.SemanticTokens()...)
	}

	if b.CloseBrace != nil {
		tokens = append(tokens, *b.CloseBrace)
	}

	return tokens
}

// CommandContent represents the content within a command body
type CommandContent interface {
	Node
	IsCommandContent() bool
}

// ShellContent represents shell command content with potential inline decorators
// This supports mixed content like: echo "Building on port @var(PORT)"
type ShellContent struct {
	Parts  []ShellPart // Mixed content: text and inline decorators
	Pos    Position
	Tokens TokenRange
}

func (s *ShellContent) String() string {
	var parts []string
	for _, part := range s.Parts {
		parts = append(parts, part.String())
	}
	return strings.Join(parts, "")
}

func (s *ShellContent) Position() Position {
	return s.Pos
}

func (s *ShellContent) TokenRange() TokenRange {
	return s.Tokens
}

func (s *ShellContent) SemanticTokens() []types.Token {
	var tokens []types.Token
	for _, part := range s.Parts {
		tokens = append(tokens, part.SemanticTokens()...)
	}
	return tokens
}

func (s *ShellContent) IsCommandContent() bool {
	return true
}

// ShellPart represents a part of shell content (text or inline decorator)
type ShellPart interface {
	Node
	IsShellPart() bool
}

// TextPart represents plain text within shell content
type TextPart struct {
	Text   string
	Pos    Position
	Tokens TokenRange
}

func (t *TextPart) String() string {
	return t.Text
}

func (t *TextPart) Position() Position {
	return t.Pos
}

func (t *TextPart) TokenRange() TokenRange {
	return t.Tokens
}

func (t *TextPart) SemanticTokens() []types.Token {
	tokens := make([]types.Token, len(t.Tokens.All))
	copy(tokens, t.Tokens.All)

	// Mark all tokens as shell content
	for i := range tokens {
		if tokens[i].Semantic != types.SemCommand {
			tokens[i].Semantic = types.SemShellText
		}
	}

	return tokens
}

func (t *TextPart) IsShellPart() bool {
	return true
}

// BlockDecoratorContent removed - functionality moved to BlockDecorator.Content

// BlockDecorator represents block decorators like @parallel, @timeout, @retry
// This handles cases like: @parallel { cmd1; cmd2 } or @timeout(30s) { npm start }
type BlockDecorator struct {
	Name    string           // Decorator name: "parallel", "timeout", "retry"
	Args    []NamedParameter // Arguments within parentheses
	Content []CommandContent // The commands inside the decorator block
	Pos     Position
	Tokens  TokenRange

	// LSP support
	AtToken   types.Token
	NameToken types.Token
}

func (d *BlockDecorator) String() string {
	result := "@" + d.Name
	if len(d.Args) > 0 {
		var argStrs []string
		for _, arg := range d.Args {
			argStrs = append(argStrs, arg.String())
		}
		result += "(" + strings.Join(argStrs, ", ") + ")"
	}
	if len(d.Content) > 0 {
		var contentStrs []string
		for _, content := range d.Content {
			contentStrs = append(contentStrs, content.String())
		}
		result += " { " + strings.Join(contentStrs, "; ") + " }"
	}
	return result
}

func (d *BlockDecorator) Position() Position {
	return d.Pos
}

func (d *BlockDecorator) TokenRange() TokenRange {
	return d.Tokens
}

func (d *BlockDecorator) SemanticTokens() []types.Token {
	var tokens []types.Token

	// Add @ token
	atToken := d.AtToken
	atToken.Semantic = types.SemOperator
	tokens = append(tokens, atToken)

	// Add name token
	nameToken := d.NameToken
	nameToken.Semantic = types.SemKeyword
	tokens = append(tokens, nameToken)

	// Add argument tokens
	for _, arg := range d.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	// Add content tokens
	for _, content := range d.Content {
		tokens = append(tokens, content.SemanticTokens()...)
	}

	return tokens
}

func (d *BlockDecorator) IsCommandContent() bool {
	return true
}

// BlockDecorators can be used in shell context per specification
// Examples: setup: @parallel { ... }, server: @timeout(30s) { ... }
func (d *BlockDecorator) IsShellPart() bool {
	return true
}

// PatternDecorator represents pattern decorators like @when, @try
// This handles cases like: @when(MODE) { production: deploy.sh; staging: deploy-staging.sh }
type PatternDecorator struct {
	Name     string           // Decorator name: "when", "try"
	Args     []NamedParameter // Arguments within parentheses (e.g., variable for @when)
	Patterns []PatternBranch  // Pattern branches inside the decorator block
	Pos      Position
	Tokens   TokenRange

	// LSP support
	AtToken   types.Token
	NameToken types.Token
}

func (d *PatternDecorator) String() string {
	result := "@" + d.Name
	if len(d.Args) > 0 {
		var argStrs []string
		for _, arg := range d.Args {
			argStrs = append(argStrs, arg.String())
		}
		result += "(" + strings.Join(argStrs, ", ") + ")"
	}
	if len(d.Patterns) > 0 {
		result += " { "
		var patternStrs []string
		for _, pattern := range d.Patterns {
			patternStrs = append(patternStrs, pattern.String())
		}
		result += strings.Join(patternStrs, "; ")
		result += " }"
	}
	return result
}

func (d *PatternDecorator) Position() Position {
	return d.Pos
}

func (d *PatternDecorator) TokenRange() TokenRange {
	return d.Tokens
}

func (d *PatternDecorator) SemanticTokens() []types.Token {
	var tokens []types.Token

	// Add @ token
	atToken := d.AtToken
	atToken.Semantic = types.SemOperator
	tokens = append(tokens, atToken)

	// Add name token
	nameToken := d.NameToken
	nameToken.Semantic = types.SemKeyword
	tokens = append(tokens, nameToken)

	// Add argument tokens
	for _, arg := range d.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	// Add pattern tokens
	for _, pattern := range d.Patterns {
		tokens = append(tokens, pattern.SemanticTokens()...)
	}

	return tokens
}

func (d *PatternDecorator) IsCommandContent() bool {
	return true
}

// PatternContent represents a simple pattern with commands
// Simplified to just Pattern and Commands
type PatternContent struct {
	Pattern  string           // The pattern string (e.g., "production", "main", "*")
	Commands []CommandContent // The commands to execute for this pattern
	Pos      Position
	Tokens   TokenRange
}

func (p *PatternContent) String() string {
	var parts []string
	for _, cmd := range p.Commands {
		parts = append(parts, cmd.String())
	}
	return p.Pattern + ": " + strings.Join(parts, "; ")
}

func (p *PatternContent) Position() Position {
	return p.Pos
}

func (p *PatternContent) TokenRange() TokenRange {
	return p.Tokens
}

func (p *PatternContent) SemanticTokens() []types.Token {
	var tokens []types.Token
	for _, cmd := range p.Commands {
		tokens = append(tokens, cmd.SemanticTokens()...)
	}
	return tokens
}

func (p *PatternContent) IsCommandContent() bool {
	return true
}

// PatternBranch represents a single pattern branch in pattern-matching decorators
// Examples: "production: deploy.sh", "main: npm start", "*: default.sh"
// Supports multiple commands per pattern when using newlines
type PatternBranch struct {
	Pattern  Pattern          // The pattern identifier or wildcard
	Commands []CommandContent // The commands to execute for this pattern (supports multiple)
	Pos      Position
	Tokens   TokenRange

	// Concrete syntax tokens for precise formatting and LSP
	ColonToken types.Token // The ":" token separating pattern from command
}

func (b *PatternBranch) String() string {
	var commandStrs []string
	for _, cmd := range b.Commands {
		commandStrs = append(commandStrs, cmd.String())
	}
	return fmt.Sprintf("%s: %s", b.Pattern.String(), strings.Join(commandStrs, "\n"))
}

func (b *PatternBranch) Position() Position {
	return b.Pos
}

func (b *PatternBranch) TokenRange() TokenRange {
	return b.Tokens
}

func (b *PatternBranch) SemanticTokens() []types.Token {
	var tokens []types.Token

	tokens = append(tokens, b.Pattern.SemanticTokens()...)

	colonToken := b.ColonToken
	colonToken.Semantic = types.SemOperator
	tokens = append(tokens, colonToken)

	for _, command := range b.Commands {
		tokens = append(tokens, command.SemanticTokens()...)
	}

	return tokens
}

// Pattern represents a pattern in pattern-matching decorators
type Pattern interface {
	Node
	IsPattern() bool
	GetPatternType() PatternType
}

// PatternType represents the type of pattern
type PatternType int

const (
	IdentifierPatternType PatternType = iota // Named patterns like "production", "main"
	WildcardPatternType                      // Wildcard pattern "*"
)

func (pt PatternType) String() string {
	switch pt {
	case IdentifierPatternType:
		return "identifier"
	case WildcardPatternType:
		return "wildcard"
	default:
		return "unknown"
	}
}

// IdentifierPattern represents named patterns like "production", "main", "error"
type IdentifierPattern struct {
	Name   string
	Pos    Position
	Tokens TokenRange
	Token  types.Token
}

func (i *IdentifierPattern) String() string {
	return i.Name
}

func (i *IdentifierPattern) Position() Position {
	return i.Pos
}

func (i *IdentifierPattern) TokenRange() TokenRange {
	return i.Tokens
}

func (i *IdentifierPattern) SemanticTokens() []types.Token {
	token := i.Token
	token.Semantic = types.SemPattern
	return []types.Token{token}
}

func (i *IdentifierPattern) IsPattern() bool {
	return true
}

func (i *IdentifierPattern) GetPatternType() PatternType {
	return IdentifierPatternType
}

// WildcardPattern represents the wildcard pattern "*"
type WildcardPattern struct {
	Pos    Position
	Tokens TokenRange
	Token  types.Token
}

func (w *WildcardPattern) String() string {
	return "*"
}

func (w *WildcardPattern) Position() Position {
	return w.Pos
}

func (w *WildcardPattern) TokenRange() TokenRange {
	return w.Tokens
}

func (w *WildcardPattern) SemanticTokens() []types.Token {
	token := w.Token
	token.Semantic = types.SemPattern
	return []types.Token{token}
}

func (w *WildcardPattern) IsPattern() bool {
	return true
}

func (w *WildcardPattern) GetPatternType() PatternType {
	return WildcardPatternType
}

// Decorator types: BlockDecorator, PatternDecorator, ValueDecorator, ActionDecorator

// ValueDecorator represents inline decorators that provide values for shell interpolation
// Examples: @var(NAME), @env(VAR) - these appear WITHIN shell content and return values
type ValueDecorator struct {
	Name   string
	Args   []NamedParameter
	Pos    Position
	Tokens TokenRange

	// Concrete syntax tokens for precise formatting and LSP
	AtToken    types.Token  // The "@" symbol
	NameToken  types.Token  // The decorator name token
	OpenParen  *types.Token // The "(" token (nil if no args)
	CloseParen *types.Token // The ")" token (nil if no args)
}

func (v *ValueDecorator) String() string {
	name := fmt.Sprintf("@%s", v.Name)

	if len(v.Args) > 0 {
		var argStrs []string
		for _, arg := range v.Args {
			argStrs = append(argStrs, arg.String())
		}
		name += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	return name
}

func (v *ValueDecorator) Position() Position {
	return v.Pos
}

func (v *ValueDecorator) TokenRange() TokenRange {
	return v.Tokens
}

func (v *ValueDecorator) SemanticTokens() []types.Token {
	var tokens []types.Token

	// @ token as operator
	atToken := v.AtToken
	atToken.Semantic = types.SemOperator
	tokens = append(tokens, atToken)

	// Value decorator name
	nameToken := v.NameToken
	nameToken.Semantic = types.SemVariable
	tokens = append(tokens, nameToken)

	// Add parentheses if present
	if v.OpenParen != nil {
		openParen := *v.OpenParen
		openParen.Semantic = types.SemOperator
		tokens = append(tokens, openParen)
	}

	// Add argument tokens
	for _, arg := range v.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	if v.CloseParen != nil {
		closeParen := *v.CloseParen
		closeParen.Semantic = types.SemOperator
		tokens = append(tokens, closeParen)
	}

	return tokens
}

func (v *ValueDecorator) IsExpression() bool {
	return true
}

func (v *ValueDecorator) GetType() ExpressionType {
	return IdentifierType
}

func (v *ValueDecorator) IsShellPart() bool {
	return true
}

func (v *ValueDecorator) IsStringPart() bool {
	return true
}

// ActionDecorator represents standalone decorators that execute commands
// Examples: @cmd(helper) - these appear as standalone CommandContent
type ActionDecorator struct {
	Name   string
	Args   []NamedParameter
	Pos    Position
	Tokens TokenRange

	// Concrete syntax tokens for precise formatting and LSP
	AtToken    types.Token  // The "@" symbol
	NameToken  types.Token  // The decorator name token
	OpenParen  *types.Token // The "(" token (nil if no args)
	CloseParen *types.Token // The ")" token (nil if no args)
}

func (a *ActionDecorator) String() string {
	name := fmt.Sprintf("@%s", a.Name)

	if len(a.Args) > 0 {
		var argStrs []string
		for _, arg := range a.Args {
			argStrs = append(argStrs, arg.String())
		}
		name += fmt.Sprintf("(%s)", strings.Join(argStrs, ", "))
	}

	return name
}

func (a *ActionDecorator) Position() Position {
	return a.Pos
}

func (a *ActionDecorator) TokenRange() TokenRange {
	return a.Tokens
}

func (a *ActionDecorator) SemanticTokens() []types.Token {
	var tokens []types.Token

	// @ token as operator
	atToken := a.AtToken
	atToken.Semantic = types.SemOperator
	tokens = append(tokens, atToken)

	// Action decorator name
	nameToken := a.NameToken
	nameToken.Semantic = types.SemDecorator
	tokens = append(tokens, nameToken)

	// Add parentheses if present
	if a.OpenParen != nil {
		openParen := *a.OpenParen
		openParen.Semantic = types.SemOperator
		tokens = append(tokens, openParen)
	}

	// Add argument tokens
	for _, arg := range a.Args {
		tokens = append(tokens, arg.SemanticTokens()...)
	}

	if a.CloseParen != nil {
		closeParen := *a.CloseParen
		closeParen.Semantic = types.SemOperator
		tokens = append(tokens, closeParen)
	}

	return tokens
}

func (a *ActionDecorator) IsCommandContent() bool {
	return true
}

func (a *ActionDecorator) IsShellPart() bool {
	return true
}

// Shell Chain Structures for Command Operators (&&, ||, |, >>)

// ShellChain represents a sequence of shell commands connected by operators
type ShellChain struct {
	Elements []ShellChainElement
	Pos      Position
	Tokens   TokenRange
}

func (c *ShellChain) String() string {
	var parts []string
	for i, element := range c.Elements {
		if i > 0 && element.Operator != "" {
			parts = append(parts, " "+element.Operator+" ")
		}
		parts = append(parts, element.String())
	}
	return strings.Join(parts, "")
}

func (c *ShellChain) Position() Position {
	return c.Pos
}

func (c *ShellChain) TokenRange() TokenRange {
	return c.Tokens
}

func (c *ShellChain) SemanticTokens() []types.Token {
	var tokens []types.Token
	for _, element := range c.Elements {
		tokens = append(tokens, element.SemanticTokens()...)
	}
	return tokens
}

func (c *ShellChain) IsCommandContent() bool {
	return true
}

// ShellChainElement represents a single element in a shell chain
type ShellChainElement struct {
	Content  *ShellContent // The shell command content
	Operator string        // The operator following this element ("&&", "||", "|", ">>", "")
	Target   string        // For ">>" operator, the target file (optional)
	Pos      Position
	Tokens   TokenRange
}

func (e *ShellChainElement) String() string {
	result := e.Content.String()
	if e.Operator == ">>" && e.Target != "" {
		result += " " + e.Operator + " " + e.Target
	}
	return result
}

func (e *ShellChainElement) Position() Position {
	return e.Pos
}

func (e *ShellChainElement) TokenRange() TokenRange {
	return e.Tokens
}

func (e *ShellChainElement) SemanticTokens() []types.Token {
	var tokens []types.Token
	tokens = append(tokens, e.Content.SemanticTokens()...)
	// Note: Operator tokens are handled by the parser when creating the chain
	return tokens
}

func (e *ShellChainElement) IsCommandContent() bool {
	return true
}

// Utility functions for AST traversal and analysis

// Walk traverses the CST and calls fn for each node
func Walk(node Node, fn func(Node) bool) {
	if !fn(node) {
		return
	}

	switch n := node.(type) {
	case *Program:
		for _, v := range n.Variables {
			Walk(&v, fn)
		}
		for _, g := range n.VarGroups {
			Walk(&g, fn)
		}
		for _, c := range n.Commands {
			Walk(&c, fn)
		}
	case *VarGroup:
		for _, v := range n.Variables {
			Walk(&v, fn)
		}
	case *CommandDecl:
		Walk(&n.Body, fn)
	case *CommandBody:
		for _, content := range n.Content {
			Walk(content, fn)
		}
	case *ShellContent:
		for _, part := range n.Parts {
			Walk(part, fn)
		}
	case *ShellChain:
		for _, element := range n.Elements {
			Walk(&element, fn)
		}
	case *ShellChainElement:
		Walk(n.Content, fn)
	case *TextPart:
		// Leaf node - plain text
	// BlockDecoratorContent removed - content is now in BlockDecorator.Content directly
	case *BlockDecorator:
		for _, arg := range n.Args {
			Walk(arg.Value, fn)
		}
		for _, content := range n.Content {
			Walk(content, fn)
		}
	case *PatternDecorator:
		for _, arg := range n.Args {
			Walk(arg.Value, fn)
		}
		for _, pattern := range n.Patterns {
			Walk(&pattern, fn)
		}
	case *PatternContent:
		for _, cmd := range n.Commands {
			Walk(cmd, fn)
		}
	case *PatternBranch:
		Walk(n.Pattern, fn)
		for _, command := range n.Commands {
			Walk(command, fn)
		}
	case *IdentifierPattern:
		// Leaf node - pattern identifier
	case *WildcardPattern:
		// Leaf node - wildcard pattern
	// Decorator types handle their own walking
	case *ValueDecorator:
		for _, arg := range n.Args {
			Walk(arg.Value, fn)
		}
	case *ActionDecorator:
		for _, arg := range n.Args {
			Walk(arg.Value, fn)
		}
	}
}

// Helper functions for backward compatibility and convenience

// IsSimpleCommand checks if a command body represents a simple (non-decorated) command
func (b *CommandBody) IsSimpleCommand() bool {
	_, isShell := b.Content[0].(*ShellContent)
	return isShell
}

// GetShellText returns the shell text if this is a simple shell command
func (b *CommandBody) GetShellText() string {
	if len(b.Content) == 1 {
		if shell, ok := b.Content[0].(*ShellContent); ok {
			var textParts []string
			for _, part := range shell.Parts {
				if textPart, ok := part.(*TextPart); ok {
					textParts = append(textParts, textPart.Text)
				} else if valueDecorator, ok := part.(*ValueDecorator); ok {
					textParts = append(textParts, valueDecorator.String())
				}
			}
			return strings.Join(textParts, "")
		}
	}
	return ""
}

// GetInlineDecorators returns all inline value decorators within shell content
func (b *CommandBody) GetInlineDecorators() []*ValueDecorator {
	var decorators []*ValueDecorator

	for _, content := range b.Content {
		if shell, ok := content.(*ShellContent); ok {
			for _, part := range shell.Parts {
				if valueDecorator, ok := part.(*ValueDecorator); ok {
					decorators = append(decorators, valueDecorator)
				}
			}
		}
	}

	return decorators
}

// GetAllShellContent returns all shell content from the command body
func (b *CommandBody) GetAllShellContent() []*ShellContent {
	var shellContents []*ShellContent

	for _, content := range b.Content {
		if shell, ok := content.(*ShellContent); ok {
			shellContents = append(shellContents, shell)
		}
	}

	return shellContents
}

// GetAllBlockDecoratorContent removed - use GetAllBlockDecorators instead

// GetAllPatternContent returns all pattern content from the command body
func (b *CommandBody) GetAllPatternContent() []*PatternContent {
	var patternContents []*PatternContent

	for _, content := range b.Content {
		if pattern, ok := content.(*PatternContent); ok {
			patternContents = append(patternContents, pattern)
		}
	}

	return patternContents
}

// GetPatternDecorators returns all pattern decorators in the AST
func GetPatternDecorators(node Node) []*PatternDecorator {
	var patterns []*PatternDecorator

	Walk(node, func(n Node) bool {
		if pattern, ok := n.(*PatternDecorator); ok {
			patterns = append(patterns, pattern)
		}
		return true
	})

	return patterns
}

// FindPatternBranches finds all pattern branches for a specific decorator
func FindPatternBranches(node Node, decoratorName string) []*PatternBranch {
	var branches []*PatternBranch

	Walk(node, func(n Node) bool {
		if pattern, ok := n.(*PatternDecorator); ok && pattern.Name == decoratorName {
			for _, branch := range pattern.Patterns {
				branches = append(branches, &branch)
			}
		}
		return true
	})

	return branches
}

// ValidatePatternDecorator validates pattern-matching decorator content
func ValidatePatternDecorator(pattern *PatternDecorator) []error {
	var errors []error

	// Check for duplicate patterns
	seenPatterns := make(map[string]*PatternBranch)
	hasWildcard := false

	for _, branch := range pattern.Patterns {
		patternStr := branch.Pattern.String()

		if patternStr == "*" {
			if hasWildcard {
				errors = append(errors, fmt.Errorf("multiple wildcard patterns not allowed in @%s at line %d", pattern.Name, branch.Pos.Line))
			}
			hasWildcard = true
		} else {
			if existing, exists := seenPatterns[patternStr]; exists {
				errors = append(errors, fmt.Errorf("duplicate pattern '%s' in @%s at line %d (first occurrence at line %d)", patternStr, pattern.Name, branch.Pos.Line, existing.Pos.Line))
			}
			seenPatterns[patternStr] = &branch
		}
	}

	// Decorator-specific validation
	switch pattern.Name {
	case "try":
		if _, hasMain := seenPatterns["main"]; !hasMain {
			errors = append(errors, fmt.Errorf("@try decorator requires 'main' pattern at line %d", pattern.Pos.Line))
		}
	case "when":
		if len(pattern.Patterns) == 0 {
			errors = append(errors, fmt.Errorf("@when decorator requires at least one pattern at line %d", pattern.Pos.Line))
		}
	}

	return errors
}

// FindVariableReferences finds all @var() value decorator references in the AST
func FindVariableReferences(node Node) []*ValueDecorator {
	var refs []*ValueDecorator

	Walk(node, func(n Node) bool {
		if valueDecorator, ok := n.(*ValueDecorator); ok && valueDecorator.Name == "var" {
			refs = append(refs, valueDecorator)
		}
		return true
	})

	return refs
}

// FindDecorators removed - use specific decorator type finders (FindBlockDecorators, FindPatternDecorators, FindVariableReferences)

// ValidateVariableReferences checks that all @var() decorator references are defined
func ValidateVariableReferences(program *Program) []error {
	var errors []error

	// Collect defined variables from both individual and grouped declarations
	defined := make(map[string]bool)

	// Individual variables
	for _, varDecl := range program.Variables {
		defined[varDecl.Name] = true
	}

	// Grouped variables
	for _, varGroup := range program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			defined[varDecl.Name] = true
		}
	}

	// Check all @var() decorator references
	refs := FindVariableReferences(program)
	for _, ref := range refs {
		if len(ref.Args) > 0 {
			if identifier, ok := ref.Args[0].Value.(*Identifier); ok {
				if !defined[identifier.Name] {
					errors = append(errors, fmt.Errorf("undefined variable '%s' at line %d", identifier.Name, ref.Pos.Line))
				}
			}
		}
	}

	return errors
}

// ValidatePatternDecorators validates all pattern decorators in the program
func ValidatePatternDecorators(program *Program) []error {
	var errors []error

	patterns := GetPatternDecorators(program)
	for _, pattern := range patterns {
		patternErrors := ValidatePatternDecorator(pattern)
		errors = append(errors, patternErrors...)
	}

	return errors
}

// GetDefinitionForVariable finds the variable declaration for a given reference
func GetDefinitionForVariable(program *Program, varName string) *VariableDecl {
	// Search individual variables
	for _, varDecl := range program.Variables {
		if varDecl.Name == varName {
			return &varDecl
		}
	}

	// Search grouped variables
	for _, varGroup := range program.VarGroups {
		for _, varDecl := range varGroup.Variables {
			if varDecl.Name == varName {
				return &varDecl
			}
		}
	}

	return nil
}

// GetReferencesForVariable finds all @var() value decorator references to a specific variable
func GetReferencesForVariable(program *Program, varName string) []*ValueDecorator {
	var references []*ValueDecorator

	refs := FindVariableReferences(program)
	for _, ref := range refs {
		if len(ref.Args) > 0 {
			if identifier, ok := ref.Args[0].Value.(*Identifier); ok && identifier.Name == varName {
				references = append(references, ref)
			}
		}
	}

	return references
}

// GetPatternBranchForPattern finds a specific pattern branch within pattern content
// GetPatternBranchForPattern finds a pattern branch for a specific pattern name
func GetPatternBranchForPattern(patternDecorator *PatternDecorator, patternName string) *PatternBranch {
	for _, branch := range patternDecorator.Patterns {
		if branch.Pattern.String() == patternName {
			return &branch
		}
	}
	return nil
}

// IsPatternDecorator checks if a decorator is a pattern-matching decorator
func IsPatternDecorator(decoratorName string) bool {
	return decoratorName == "when" || decoratorName == "try"
}

// GetPatternDecoratorsByName finds pattern decorators for a specific decorator type
func GetPatternDecoratorsByName(node Node, decoratorName string) []*PatternDecorator {
	var patterns []*PatternDecorator

	Walk(node, func(n Node) bool {
		if pattern, ok := n.(*PatternDecorator); ok && pattern.Name == decoratorName {
			patterns = append(patterns, pattern)
		}
		return true
	})

	return patterns
}
