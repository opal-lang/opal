package parser

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/ast"
	"github.com/google/go-cmp/cmp"

	// Import builtins to register decorators
	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin"
)

// init registers any test-specific decorators not in decorators
func init() {
	registerTestOnlyDecorators()
}

// Test-only decorator implementations

// DebounceDecorator - test implementation
type DebounceDecorator struct{}

func (d *DebounceDecorator) Name() string { return "debounce" }
func (d *DebounceDecorator) Description() string {
	return "Debounces command execution with specified delay"
}

func (d *DebounceDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "delay", Type: decorators.ArgTypeDuration, Required: true, Description: "Debounce delay"},
	}
}

func (d *DebounceDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{Code: "@debounce(500ms) { echo hello }", Description: "Debounce with 500ms delay"},
	}
}

func (d *DebounceDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

// BlockDecorator interface methods
func (d *DebounceDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	// Test implementation - just execute the inner content
	return ctx.ExecSequential(inner.Steps)
}

func (d *DebounceDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: "debounce",
		Children:    []plan.ExecutionStep{inner},
	}
}

// CwdDecorator - test implementation
type CwdDecorator struct{}

func (c *CwdDecorator) Name() string        { return "cwd" }
func (c *CwdDecorator) Description() string { return "Changes working directory for command execution" }
func (c *CwdDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "directory", Type: decorators.ArgTypeString, Required: true, Description: "Working directory"},
	}
}

func (c *CwdDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{Code: "@cwd(/path/to/dir) { commands }", Description: "Change directory for command execution"},
	}
}

func (c *CwdDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

// BlockDecorator interface methods
func (c *CwdDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	// Get directory parameter
	dir, err := decorators.ExtractString(args, "directory", "")
	if err != nil {
		return decorators.CommandResult{Stderr: err.Error(), ExitCode: 1}
	}

	// Change working directory in context and execute inner commands
	newCtx := ctx.WithWorkDir(dir)
	return newCtx.ExecSequential(inner.Steps)
}

func (c *CwdDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	dir, _ := decorators.ExtractString(args, "directory", "")
	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: fmt.Sprintf("change directory to %s", dir),
		Children:    []plan.ExecutionStep{inner},
	}
}

// WatchFilesDecorator - test implementation
type WatchFilesDecorator struct{}

func (w *WatchFilesDecorator) Name() string { return "watch-files" }
func (w *WatchFilesDecorator) Description() string {
	return "Watches files for changes and executes commands"
}

func (w *WatchFilesDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "pattern", Type: decorators.ArgTypeString, Required: true, Description: "File pattern to watch"},
	}
}

func (w *WatchFilesDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{Code: "@watch-files(*.go) { commands }", Description: "Watch Go files for changes"},
	}
}

func (w *WatchFilesDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

// BlockDecorator interface methods
func (w *WatchFilesDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	// Get pattern parameter
	pattern, err := decorators.ExtractString(args, "pattern", "")
	if err != nil {
		return decorators.CommandResult{Stderr: err.Error(), ExitCode: 1}
	}

	// Simulate file watching (in real implementation would set up file watcher)
	if ctx.DryRun {
		return decorators.CommandResult{Stdout: fmt.Sprintf("Would watch files matching: %s", pattern)}
	}
	return ctx.ExecSequential(inner.Steps)
}

func (w *WatchFilesDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	pattern, _ := decorators.ExtractString(args, "pattern", "")
	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: fmt.Sprintf("watch files matching %s", pattern),
		Children:    []plan.ExecutionStep{inner},
	}
}

// OffsetDecorator - test implementation
type OffsetDecorator struct{}

func (o *OffsetDecorator) Name() string { return "offset" }
func (o *OffsetDecorator) Description() string {
	return "Test decorator - applies numeric offset to command execution"
}

func (o *OffsetDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "value", Type: decorators.ArgTypeInt, Required: true, Description: "Offset value"},
	}
}

func (o *OffsetDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{Code: "@offset(5) { commands }", Description: "Apply numeric offset to execution"},
	}
}

func (o *OffsetDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

// BlockDecorator interface methods
func (o *OffsetDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	// Get offset value parameter
	offset, err := decorators.ExtractInt(args, "value", 0)
	if err != nil {
		return decorators.CommandResult{Stderr: err.Error(), ExitCode: 1}
	}

	// Apply offset (in real implementation would modify execution behavior)
	if ctx.DryRun {
		return decorators.CommandResult{Stdout: fmt.Sprintf("Would apply offset: %d", offset)}
	}
	return ctx.ExecSequential(inner.Steps)
}

func (o *OffsetDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	offset, _ := decorators.ExtractInt(args, "value", 0)
	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: fmt.Sprintf("apply offset %d", offset),
		Children:    []plan.ExecutionStep{inner},
	}
}

// FactorDecorator - test implementation
type FactorDecorator struct{}

func (f *FactorDecorator) Name() string { return "factor" }
func (f *FactorDecorator) Description() string {
	return "Test decorator - applies scaling factor to command execution"
}

func (f *FactorDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "multiplier", Type: decorators.ArgTypeFloat, Required: true, Description: "Scaling factor"},
	}
}

func (f *FactorDecorator) Examples() []decorators.Example {
	return []decorators.Example{
		{Code: "@factor(2.5) { commands }", Description: "Apply scaling factor to execution"},
	}
}

func (f *FactorDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

// BlockDecorator interface methods
func (f *FactorDecorator) WrapCommands(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner decorators.CommandSeq) decorators.CommandResult {
	// Get multiplier parameter
	multiplier, err := decorators.ExtractFloat(args, "multiplier", 1.0)
	if err != nil {
		return decorators.CommandResult{Stderr: err.Error(), ExitCode: 1}
	}

	// Apply scaling factor (in real implementation would modify execution behavior)
	if ctx.DryRun {
		return decorators.CommandResult{Stdout: fmt.Sprintf("Would apply scaling factor: %.2f", multiplier)}
	}
	return ctx.ExecSequential(inner.Steps)
}

func (f *FactorDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep {
	multiplier, _ := decorators.ExtractFloat(args, "multiplier", 1.0)
	return plan.ExecutionStep{
		Type:        plan.StepSequence,
		Description: fmt.Sprintf("apply scaling factor %.2f", multiplier),
		Children:    []plan.ExecutionStep{inner},
	}
}

// registerTestOnlyDecorators registers decorators that are only used for testing
func registerTestOnlyDecorators() {
	// Register test decorators with the new decorator registry
	decorators.RegisterBlock(&DebounceDecorator{})
	decorators.RegisterBlock(&CwdDecorator{})
	decorators.RegisterBlock(&WatchFilesDecorator{})
	decorators.RegisterBlock(&OffsetDecorator{})
	decorators.RegisterBlock(&FactorDecorator{})

	// Note: @try is already registered in the main decorators package
	// Note: @when is already registered in the main decorators package
}

// Test helper types for the new CST structure
type ExpectedProgram struct {
	Variables []ExpectedVariable
	Commands  []ExpectedCommand
}

type ExpectedVariable struct {
	Name  string
	Value ExpectedExpression
}

type ExpectedCommand struct {
	Name string
	Type ast.CommandType
	Body ExpectedCommandBody
}

type ExpectedCommandBody struct {
	Content []ExpectedCommandContent // Updated to match AST structure
}

type ExpectedCommandContent interface {
	IsExpectedCommandContent() bool
}

type ExpectedShellContent struct {
	Parts []ExpectedShellPart
}

func (s ExpectedShellContent) IsExpectedCommandContent() bool { return true }

// ExpectedShellChain represents a sequence of shell commands connected by operators
type ExpectedShellChain struct {
	Elements []ExpectedShellChainElement
}

func (s ExpectedShellChain) IsExpectedCommandContent() bool { return true }

// ExpectedShellChainElement represents a single element in a shell chain
type ExpectedShellChainElement struct {
	Content  ExpectedShellContent // The shell command content
	Operator string               // The operator following this element ("&&", "||", "|", ">>", "")
	Target   string               // For ">>" operator, the target file (optional)
}

// ExpectedBlockDecoratorContent removed - use ExpectedBlockDecorator instead

// New expected types for the refactored AST structure
type ExpectedBlockDecorator struct {
	Name    string
	Args    []ExpectedExpression
	Content []ExpectedCommandContent
}

func (d ExpectedBlockDecorator) IsExpectedCommandContent() bool { return true }

type ExpectedPatternDecorator struct {
	Name     string
	Args     []ExpectedExpression
	Patterns []ExpectedPatternBranch
}

func (d ExpectedPatternDecorator) IsExpectedCommandContent() bool { return true }

// ExpectedFunctionDecorator moved to avoid duplication

type ExpectedPatternContent struct {
	Decorator ExpectedDecorator
	Branches  []ExpectedPatternBranch
}

func (p ExpectedPatternContent) IsExpectedCommandContent() bool { return true }

type ExpectedPatternBranch struct {
	Pattern  ExpectedPattern
	Commands []ExpectedCommandContent // Updated to match AST structure
}

type ExpectedPattern interface {
	IsExpectedPattern() bool
}

type ExpectedIdentifierPattern struct {
	Name string
}

func (i ExpectedIdentifierPattern) IsExpectedPattern() bool { return true }

type ExpectedWildcardPattern struct{}

func (w ExpectedWildcardPattern) IsExpectedPattern() bool { return true }

type ExpectedShellPart struct {
	Type              string
	Text              string
	FunctionDecorator *ExpectedFunctionDecorator
	ActionDecorator   *ExpectedFunctionDecorator
}

type ExpectedDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedFunctionDecorator struct {
	Name string
	Args []ExpectedExpression
}

func (f ExpectedFunctionDecorator) IsExpectedCommandContent() bool { return true }

type ExpectedExpression struct {
	Type  string
	Value string
	// For string interpolation (unified string system)
	Parts []ExpectedShellPart `json:"parts,omitempty"`
	// For function decorators
	Name string               `json:"name,omitempty"`
	Args []ExpectedExpression `json:"args,omitempty"`
}

// Test case structure
type TestCase struct {
	Name        string
	Input       string
	WantErr     bool
	ErrorSubstr string
	Expected    ExpectedProgram
}

// DSL for building expected test results using natural language
// NOTE: This DSL creates Expected* structures for parser testing, not actual AST nodes.
// For creating actual AST nodes, use the functions in pkgs/ast/builder.go

// ExpectedProgram creates an expected program for parser tests
func Program(items ...interface{}) ExpectedProgram {
	var variables []ExpectedVariable
	var commands []ExpectedCommand

	for _, item := range items {
		switch v := item.(type) {
		case ExpectedVariable:
			variables = append(variables, v)
		case ExpectedCommand:
			commands = append(commands, v)
		}
	}

	return ExpectedProgram{
		Variables: variables,
		Commands:  commands,
	}
}

// Var creates a variable declaration: var NAME = VALUE
func Var(name string, value interface{}) ExpectedVariable {
	return ExpectedVariable{
		Name:  name,
		Value: toExpression(value),
	}
}

// Explicit type constructors for test expressions

// Str creates a string literal expression
func Str(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "string",
		Value: value,
	}
}

// Id creates an identifier expression
func Id(name string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "identifier",
		Value: name,
	}
}

// Num creates a number expression
func Num(value interface{}) ExpectedExpression {
	switch v := value.(type) {
	case int:
		return ExpectedExpression{
			Type:  "number",
			Value: strconv.Itoa(v),
		}
	case int64:
		return ExpectedExpression{
			Type:  "number",
			Value: strconv.FormatInt(v, 10),
		}
	case float64:
		return ExpectedExpression{
			Type:  "number",
			Value: strconv.FormatFloat(v, 'g', -1, 64),
		}
	case float32:
		return ExpectedExpression{
			Type:  "number",
			Value: strconv.FormatFloat(float64(v), 'g', -1, 32),
		}
	default:
		return ExpectedExpression{
			Type:  "number",
			Value: fmt.Sprintf("%v", v),
		}
	}
}

// Bool creates a boolean expression
func Bool(value bool) ExpectedExpression {
	return ExpectedExpression{
		Type:  "boolean",
		Value: strconv.FormatBool(value),
	}
}

// Dur creates a duration expression
func Dur(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "duration",
		Value: value,
	}
}

// Legacy aliases for backwards compatibility
func DurationExpr(value string) ExpectedExpression {
	return Dur(value)
}

func Duration(value string) ExpectedExpression {
	return Dur(value)
}

// Named creates a named parameter expression for decorators
func Named(name string, value ExpectedExpression) ExpectedExpression {
	return ExpectedExpression{
		Type:  value.Type,  // Use the actual underlying type
		Value: value.Value, // Use the actual underlying value
		Name:  name,        // Preserve the parameter name
		Args:  []ExpectedExpression{value},
	}
}

func BooleanExpr(value bool) ExpectedExpression {
	return Bool(value)
}

// Cmd creates a simple command: NAME: BODY
// This applies syntax sugar for simple shell commands with or without function decorators
func Cmd(name string, body interface{}) ExpectedCommand {
	cmdBody := toCommandBody(body)

	return ExpectedCommand{
		Name: name,
		Type: ast.Command,
		Body: cmdBody,
	}
}

// Watch creates a watch command: watch NAME: BODY
// This applies syntax sugar for simple shell commands with or without function decorators
func Watch(name string, body interface{}) ExpectedCommand {
	cmdBody := toCommandBody(body)

	return ExpectedCommand{
		Name: name,
		Type: ast.WatchCommand,
		Body: cmdBody,
	}
}

// WatchBlock creates a watch command with explicit block syntax
func WatchBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.WatchCommand,
		Body: ExpectedCommandBody{
			Content: toMultipleCommandContent(content...),
		},
	}
}

// Stop creates a stop command: stop NAME: BODY
// This applies syntax sugar for simple shell commands with or without function decorators
func Stop(name string, body interface{}) ExpectedCommand {
	cmdBody := toCommandBody(body)

	return ExpectedCommand{
		Name: name,
		Type: ast.StopCommand,
		Body: cmdBody,
	}
}

// StopBlock creates a stop command with explicit block syntax
func StopBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.StopCommand,
		Body: ExpectedCommandBody{
			Content: toMultipleCommandContent(content...),
		},
	}
}

// CmdBlock creates a command with explicit block syntax: NAME: { content }
func CmdBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.Command,
		Body: ExpectedCommandBody{
			Content: toMultipleCommandContent(content...),
		},
	}
}

// Simple creates a simple command body (single line)
// This enforces that simple commands cannot contain BLOCK decorators (per syntax sugar rules)
// Function decorators (@var) are allowed and get syntax sugar
func Simple(parts ...interface{}) ExpectedCommandBody {
	shellParts := toShellParts(parts...)

	// Validate that simple commands don't contain BLOCK decorators
	// Function decorators are allowed in simple commands
	for _, part := range shellParts {
		if part.Type == "value_decorator" {
			if part.FunctionDecorator != nil && !decorators.IsDecorator(part.FunctionDecorator.Name) {
				// Instead of panic, return an error body
				return ExpectedCommandBody{
					Content: []ExpectedCommandContent{
						ExpectedShellContent{
							Parts: []ExpectedShellPart{
								Text("ERROR: Simple() command bodies cannot contain block decorators. Per spec: 'Block decorators require explicit braces' - use Block() instead"),
							},
						},
					},
				}
			}
		}
	}

	return ExpectedCommandBody{
		Content: []ExpectedCommandContent{
			ExpectedShellContent{
				Parts: shellParts,
			},
		},
	}
}

// Text creates a text part
func Text(text string) ExpectedShellPart {
	return ExpectedShellPart{
		Type: "text",
		Text: text,
	}
}

// StrPart creates a string literal part within shell content
func StrPart(value string) ExpectedShellPart {
	return ExpectedShellPart{
		Type: "string",
		Text: value, // Use Text field for the string value
	}
}

// StrWithParts creates a string expression with interpolated parts for decorator arguments
// This represents a StringLiteral with Parts[] for string interpolation
func StrWithParts(parts ...interface{}) ExpectedExpression {
	var expectedParts []ExpectedShellPart
	for _, part := range parts {
		switch p := part.(type) {
		case ExpectedShellPart:
			expectedParts = append(expectedParts, p)
		case string:
			expectedParts = append(expectedParts, StrPart(p))
		default:
			// Convert other types to shell parts
			expectedParts = append(expectedParts, ExpectedShellPart{
				Type: "unknown",
				Text: fmt.Sprintf("%v", p),
			})
		}
	}

	return ExpectedExpression{
		Type:  "string_with_parts",
		Parts: expectedParts,
	}
}

// At creates a function decorator within shell content: @var(NAME)
// Only valid for function decorators like @var()
func At(name string, args ...interface{}) ExpectedShellPart {
	// Validate that this is a function decorator
	if !decorators.IsDecorator(name) {
		// Instead of panic, return an error shell part
		return ExpectedShellPart{
			Type: "text",
			Text: fmt.Sprintf("ERROR: At() can only be used with function decorators, but '%s' is not a function decorator", name),
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	// Determine the correct decorator type
	var decoratorType string
	var funcDecorator *ExpectedFunctionDecorator
	var actionDecorator *ExpectedFunctionDecorator

	funcDecorator = &ExpectedFunctionDecorator{
		Name: name,
		Args: decoratorArgs,
	}

	if decorators.IsValueDecorator(name) {
		decoratorType = "value_decorator"
	} else if decorators.IsActionDecorator(name) {
		decoratorType = "action_decorator"
		actionDecorator = funcDecorator
		funcDecorator = nil
	} else {
		// Fallback to value_decorator for unknown types
		decoratorType = "value_decorator"
	}

	return ExpectedShellPart{
		Type:              decoratorType,
		FunctionDecorator: funcDecorator,
		ActionDecorator:   actionDecorator,
	}
}

// Decorator creates a block decorator: @timeout(30s)
// Only valid for block decorators that require explicit braces
func Decorator(name string, args ...interface{}) ExpectedDecorator {
	// Check if decorator exists in the new registry system
	if _, err := decorators.GetBlock(name); err != nil {
		// Return error decorator name to make test failures clear
		return ExpectedDecorator{
			Name: fmt.Sprintf("ERROR_NOT_BLOCK_DECORATOR_%s", name),
			Args: []ExpectedExpression{},
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	return ExpectedDecorator{
		Name: name,
		Args: decoratorArgs,
	}
}

// PatternDecorator creates a pattern decorator: @when(VAR) or @try
// Only valid for pattern decorators that handle pattern matching
func PatternDecorator(name string, args ...interface{}) ExpectedDecorator {
	// Check if decorator exists in the new registry system
	if _, err := decorators.GetPattern(name); err != nil {
		// Return error decorator name to make test failures clear
		return ExpectedDecorator{
			Name: fmt.Sprintf("ERROR_NOT_PATTERN_DECORATOR_%s", name),
			Args: []ExpectedExpression{},
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	return ExpectedDecorator{
		Name: name,
		Args: decoratorArgs,
	}
}

func PatternDecoratorWithBranches(name string, firstArg interface{}, branches ...ExpectedPatternBranch) ExpectedPatternDecorator {
	// Check if decorator exists in the new registry system
	if _, err := decorators.GetPattern(name); err != nil {
		panic(fmt.Sprintf("Pattern decorator %s not found in registry: %v", name, err))
	}

	var decoratorArgs []ExpectedExpression
	if firstArg != nil {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, firstArg))
	}

	return ExpectedPatternDecorator{
		Name:     name,
		Args:     decoratorArgs,
		Patterns: branches,
	}
}

// Pattern creates a pattern content with branches: @when(VAR) { pattern: command }
func Pattern(decorator ExpectedDecorator, branches ...ExpectedPatternBranch) ExpectedPatternContent {
	return ExpectedPatternContent{
		Decorator: decorator,
		Branches:  branches,
	}
}

// Branch creates a pattern branch: pattern: command or pattern: { commands }
// **UPDATED**: Now supports multiple commands per branch
func Branch(pattern interface{}, commands ...interface{}) ExpectedPatternBranch {
	var patternObj ExpectedPattern

	switch p := pattern.(type) {
	case string:
		if p == "default" {
			patternObj = ExpectedWildcardPattern{}
		} else {
			patternObj = ExpectedIdentifierPattern{Name: p}
		}
	case ExpectedPattern:
		patternObj = p
	default:
		patternObj = ExpectedIdentifierPattern{Name: fmt.Sprintf("%v", p)}
	}

	// Convert commands to array of CommandContent
	var commandArray []ExpectedCommandContent
	for _, cmd := range commands {
		commandArray = append(commandArray, toSingleCommandContent(cmd))
	}

	// If no commands provided, create empty shell content
	if len(commandArray) == 0 {
		commandArray = []ExpectedCommandContent{
			ExpectedShellContent{Parts: []ExpectedShellPart{}},
		}
	}

	return ExpectedPatternBranch{
		Pattern:  patternObj,
		Commands: commandArray,
	}
}

// Wildcard creates a wildcard pattern: *
func Wildcard() ExpectedPattern {
	return ExpectedWildcardPattern{}
}

// PatternId creates an identifier pattern: production, main, etc.
func PatternId(name string) ExpectedPattern {
	return ExpectedIdentifierPattern{Name: name}
}

// Shell creates a shell content item
func Shell(parts ...interface{}) ExpectedCommandContent {
	return ExpectedShellContent{
		Parts: toShellParts(parts...),
	}
}

// Chain creates a shell chain with operators
func Chain(elements ...interface{}) ExpectedCommandContent {
	if len(elements) == 0 {
		return ExpectedShellChain{Elements: []ExpectedShellChainElement{}}
	}

	var chainElements []ExpectedShellChainElement

	for i, elem := range elements {
		var content ExpectedShellContent

		switch e := elem.(type) {
		case ExpectedShellChainElement:
			content = e.Content
			// For chain elements, the operator applies to the PREVIOUS element
			if i > 0 {
				chainElements[i-1].Operator = e.Operator
				chainElements[i-1].Target = e.Target
			}
		case ExpectedShellContent:
			content = e
		default:
			// Convert other types to shell content first
			content = Shell(e).(ExpectedShellContent)
		}

		chainElements = append(chainElements, ExpectedShellChainElement{
			Content:  content,
			Operator: "", // Will be set by next element if any
		})
	}

	return ExpectedShellChain{
		Elements: chainElements,
	}
}

// ChainElement creates a shell chain element with an operator
func ChainElement(content interface{}, operator string, target ...string) ExpectedShellChainElement {
	var shellContent ExpectedShellContent
	switch c := content.(type) {
	case ExpectedShellContent:
		shellContent = c
	default:
		shellContent = Shell(c).(ExpectedShellContent)
	}

	elem := ExpectedShellChainElement{
		Content:  shellContent,
		Operator: operator,
	}

	if len(target) > 0 {
		elem.Target = target[0]
	}

	return elem
}

// Pipe creates a pipe operator chain element: cmd1 | cmd2
func Pipe(content interface{}) ExpectedShellChainElement {
	return ChainElement(content, "|")
}

// And creates a logical AND chain element: cmd1 && cmd2
func And(content interface{}) ExpectedShellChainElement {
	return ChainElement(content, "&&")
}

// Or creates a logical OR chain element: cmd1 || cmd2
func Or(content interface{}) ExpectedShellChainElement {
	return ChainElement(content, "||")
}

// Append creates an append redirect chain element: cmd >> file
func Append(content interface{}, target string) ExpectedShellChainElement {
	return ChainElement(content, ">>", target)
}

// DecoratedShell creates decorated shell content: @timeout(30s) npm run build
func DecoratedShell(decorator ExpectedDecorator, parts ...interface{}) ExpectedCommandContent {
	// Determine decorator type and create appropriate expected structure
	if decorators.IsBlockDecorator(decorator.Name) {
		return ExpectedBlockDecorator{
			Name:    decorator.Name,
			Args:    decorator.Args,
			Content: []ExpectedCommandContent{Shell(parts...)},
		}
	} else if decorators.IsPatternDecorator(decorator.Name) {
		return ExpectedPatternDecorator{
			Name:     decorator.Name,
			Args:     decorator.Args,
			Patterns: []ExpectedPatternBranch{}, // Would need to be populated based on content
		}
	} else {
		// For unknown decorators, assume block decorator
		return ExpectedBlockDecorator{
			Name:    decorator.Name,
			Args:    decorator.Args,
			Content: []ExpectedCommandContent{Shell(parts...)},
		}
	}
}

// BlockDecorator creates a block decorator with multiple commands in its content
func BlockDecorator(name string, args ...interface{}) ExpectedCommandContent {
	// Split args into decorator args and content
	var decoratorArgs []ExpectedExpression
	var content []ExpectedCommandContent

	for _, arg := range args {
		switch v := arg.(type) {
		case ExpectedExpression:
			decoratorArgs = append(decoratorArgs, v)
		case ExpectedCommandContent:
			content = append(content, v)
		case string:
			// If it's a string, treat it as shell content
			content = append(content, Shell(Text(v)))
		default:
			// Try to convert to expression (for decorator args like timeouts)
			if len(content) == 0 {
				decoratorArgs = append(decoratorArgs, toExpression(arg))
			} else {
				// If we already have content, this should be shell content
				content = append(content, Shell(Text(fmt.Sprintf("%v", arg))))
			}
		}
	}

	return ExpectedBlockDecorator{
		Name:    name,
		Args:    decoratorArgs,
		Content: content,
	}
}

// toDecoratorArgument converts arguments for decorator parameters
// Since function decorators are no longer allowed in decorator arguments,
// this simply converts to the appropriate expression type
func toDecoratorArgument(decoratorName string, arg interface{}) ExpectedExpression {
	// For all decorators, use the default conversion
	// Variable references should be direct identifiers now
	return toExpression(arg)
}

// Helper conversion functions
func toExpression(v interface{}) ExpectedExpression {
	switch val := v.(type) {
	case ExpectedExpression:
		// Already an explicit expression, return as-is
		return val
	case string:
		// For backwards compatibility, treat plain strings as string literals
		// Users should use Id() for identifiers and Str() for string literals
		return Str(val)
	case int:
		return Num(val)
	case int64:
		return Num(val)
	case float64:
		return Num(val)
	case float32:
		return Num(val)
	case bool:
		return Bool(val)
	case ExpectedFunctionDecorator:
		return ExpectedExpression{
			Type: "value_decorator",
			Name: val.Name,
			Args: val.Args,
		}
	default:
		// Try to convert to string and handle as string literal
		str := fmt.Sprintf("%v", val)
		return Str(str)
	}
}

func toCommandBody(v interface{}) ExpectedCommandBody {
	switch val := v.(type) {
	case ExpectedCommandBody:
		return val
	case string:
		// Empty string should create empty shell content
		if val == "" {
			return ExpectedCommandBody{
				Content: []ExpectedCommandContent{
					ExpectedShellContent{Parts: []ExpectedShellPart{}},
				},
			}
		}
		// Simple string becomes simple command body (gets syntax sugar)
		return Simple(Text(val))
	case ExpectedShellContent:
		// Shell content that doesn't explicitly specify block structure
		// Check if it contains BLOCK decorators - if so, it needs explicit blocks
		// Function decorators are allowed and get syntax sugar
		for _, part := range val.Parts {
			if part.Type == "value_decorator" {
				if part.FunctionDecorator != nil && !decorators.IsDecorator(part.FunctionDecorator.Name) {
					// Instead of panic, return an error body
					return ExpectedCommandBody{
						Content: []ExpectedCommandContent{
							ExpectedShellContent{
								Parts: []ExpectedShellPart{
									Text("ERROR: Shell content with block decorators requires explicit block syntax"),
								},
							},
						},
					}
				}
			}
		}
		return ExpectedCommandBody{
			Content: []ExpectedCommandContent{val},
		}
	case ExpectedBlockDecorator:
		// Block decorators ALWAYS require explicit blocks per spec
		return ExpectedCommandBody{
			Content: []ExpectedCommandContent{val},
		}
	case ExpectedPatternContent:
		// Pattern content ALWAYS requires explicit blocks per spec
		return ExpectedCommandBody{
			Content: []ExpectedCommandContent{val},
		}
	case ExpectedPatternDecorator:
		// Pattern decorators ALWAYS require explicit blocks per spec
		return ExpectedCommandBody{
			Content: []ExpectedCommandContent{val},
		}
	case ExpectedShellChain:
		// Shell chains should be preserved as-is in command body
		return ExpectedCommandBody{
			Content: []ExpectedCommandContent{val},
		}
	default:
		return ExpectedCommandBody{
			Content: []ExpectedCommandContent{
				ExpectedShellContent{
					Parts: []ExpectedShellPart{},
				},
			},
		}
	}
}

// toMultipleCommandContent converts variadic args to array of command content
func toMultipleCommandContent(items ...interface{}) []ExpectedCommandContent {
	if len(items) == 0 {
		return []ExpectedCommandContent{}
	}

	var contentItems []ExpectedCommandContent

	i := 0
	for i < len(items) {
		item := items[i]

		// Check if this is already a CommandContent
		if content, ok := item.(ExpectedCommandContent); ok {
			contentItems = append(contentItems, content)
			i++
			continue
		}

		// Check if this is a decorator followed by content
		if decorator, ok := item.(ExpectedDecorator); ok {
			// Look for content after the decorator
			if i+1 < len(items) {
				nextItem := items[i+1]

				// If next item is also a decorator, this decorator has no content
				if _, isDecorator := nextItem.(ExpectedDecorator); isDecorator {
					// Decorator with no content - create empty shell content
					contentItems = append(contentItems, ExpectedBlockDecorator{
						Name:    decorator.Name,
						Args:    decorator.Args,
						Content: []ExpectedCommandContent{ExpectedShellContent{Parts: []ExpectedShellPart{}}},
					})
					i++
					continue
				}

				// If next item is CommandContent, use it directly
				if content, ok := nextItem.(ExpectedCommandContent); ok {
					contentItems = append(contentItems, ExpectedBlockDecorator{
						Name:    decorator.Name,
						Args:    decorator.Args,
						Content: []ExpectedCommandContent{content},
					})
					i += 2
					continue
				}

				// Otherwise, convert next item to shell content
				shellContent := toSingleCommandContent(nextItem)
				contentItems = append(contentItems, ExpectedBlockDecorator{
					Name:    decorator.Name,
					Args:    decorator.Args,
					Content: []ExpectedCommandContent{shellContent},
				})
				i += 2
				continue
			} else {
				// Decorator at end with no content
				contentItems = append(contentItems, ExpectedBlockDecorator{
					Name:    decorator.Name,
					Args:    decorator.Args,
					Content: []ExpectedCommandContent{ExpectedShellContent{Parts: []ExpectedShellPart{}}},
				})
				i++
				continue
			}
		}

		// Convert other items to shell content
		contentItems = append(contentItems, toSingleCommandContent(item))
		i++
	}

	return contentItems
}

// toSingleCommandContent converts a single item to CommandContent
func toSingleCommandContent(item interface{}) ExpectedCommandContent {
	switch val := item.(type) {
	case ExpectedCommandContent:
		return val
	case string:
		return ExpectedShellContent{
			Parts: []ExpectedShellPart{Text(val)},
		}
	case ExpectedShellPart:
		return ExpectedShellContent{
			Parts: []ExpectedShellPart{val},
		}
	case []ExpectedShellPart:
		return ExpectedShellContent{
			Parts: val,
		}
	default:
		return ExpectedShellContent{
			Parts: []ExpectedShellPart{Text(fmt.Sprintf("%v", val))},
		}
	}
}

func toShellParts(items ...interface{}) []ExpectedShellPart {
	var parts []ExpectedShellPart
	for _, item := range items {
		switch v := item.(type) {
		case ExpectedShellPart:
			parts = append(parts, v)
		case string:
			parts = append(parts, Text(v))
		case ExpectedFunctionDecorator:
			// Validate that function decorators are only used inline
			if !decorators.IsDecorator(v.Name) {
				// Instead of panic, create an error text part
				parts = append(parts, Text(fmt.Sprintf("ERROR: '%s' is not a function decorator and cannot be used inline in shell content", v.Name)))
			} else {
				parts = append(parts, ExpectedShellPart{
					Type:              "value_decorator",
					FunctionDecorator: &v,
				})
			}
		default:
			parts = append(parts, Text(fmt.Sprintf("%v", v)))
		}
	}
	return parts
}

// flattenVariables collects all variables from individual and grouped declarations
func flattenVariables(program *ast.Program) []ast.VariableDecl {
	var allVariables []ast.VariableDecl

	// Add individual variables
	allVariables = append(allVariables, program.Variables...)

	// Add variables from groups
	for _, group := range program.VarGroups {
		allVariables = append(allVariables, group.Variables...)
	}

	return allVariables
}

// Comparison helpers for the new CST structure
func expressionToComparable(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return map[string]interface{}{
			"Type":  "string",
			"Value": e.String(),
		}
	case *ast.NumberLiteral:
		return map[string]interface{}{
			"Type":  "number",
			"Value": e.Value,
		}
	case *ast.DurationLiteral:
		return map[string]interface{}{
			"Type":  "duration",
			"Value": e.Value,
		}
	case *ast.BooleanLiteral:
		return map[string]interface{}{
			"Type":  "boolean",
			"Value": strconv.FormatBool(e.Value),
		}
	case *ast.Identifier:
		return map[string]interface{}{
			"Type":  "identifier",
			"Value": e.Name,
		}
	default:
		return map[string]interface{}{
			"Type":  "unknown",
			"Value": expr.String(),
		}
	}
}

// namedParameterToComparable converts a NamedParameter to a comparable format
func namedParameterToComparable(param ast.NamedParameter) interface{} {
	result := map[string]interface{}{
		"Name": param.Name,
	}

	// Add the value using the existing expression logic
	if param.Value != nil {
		result["Value"] = expressionToComparable(param.Value)
	}

	return result
}

// Helper function to convert expected decorator arguments to NamedParameter format using registry
func expectedArgsToNamedParams(decoratorName string, args []ExpectedExpression) []interface{} {
	result := make([]interface{}, len(args))

	// Look up the decorator in the registry to get parameter names
	// Try value, action, block, and pattern decorators
	var schema []decorators.ParameterSchema
	if valueDecorator, err := decorators.GetValue(decoratorName); err == nil {
		schema = valueDecorator.ParameterSchema()
	} else if actionDecorator, err := decorators.GetAction(decoratorName); err == nil {
		schema = actionDecorator.ParameterSchema()
	} else if blockDecorator, err := decorators.GetBlock(decoratorName); err == nil {
		schema = blockDecorator.ParameterSchema()
	} else if patternDecorator, err := decorators.GetPattern(decoratorName); err == nil {
		schema = patternDecorator.ParameterSchema()
	} else {
		panic(fmt.Sprintf("Decorator %s not found in registry - all decorators used in tests must be registered", decoratorName))
	}

	for i, arg := range args {
		var paramName string

		// If the argument already has a name (from Named() helper), use it
		if arg.Name != "" {
			paramName = arg.Name
		} else {
			// Map positional arguments to parameter names from schema
			if i >= len(schema) {
				panic(fmt.Sprintf("Decorator %s expects at most %d parameters, but test provided %d parameters", decoratorName, len(schema), len(args)))
			}
			paramName = schema[i].Name
		}

		result[i] = map[string]interface{}{
			"Name": paramName,
			"Value": map[string]interface{}{
				"Type":  arg.Type,
				"Value": arg.Value,
			},
		}
	}

	return result
}

func shellPartToComparable(part ast.ShellPart) interface{} {
	switch p := part.(type) {
	case *ast.TextPart:
		return map[string]interface{}{
			"Type": "text",
			"Text": p.Text,
		}
	case *ast.TextStringPart:
		return map[string]interface{}{
			"Type":  "string",
			"Value": p.Text,
		}
	case *ast.ValueDecorator:
		args := make([]interface{}, len(p.Args))
		for i, arg := range p.Args {
			args[i] = namedParameterToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "value_decorator",
			"ValueDecorator": map[string]interface{}{
				"Name": p.Name,
				"Args": args,
			},
		}
	case *ast.ActionDecorator:
		args := make([]interface{}, len(p.Args))
		for i, arg := range p.Args {
			args[i] = namedParameterToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "action_decorator",
			"ActionDecorator": map[string]interface{}{
				"Name": p.Name,
				"Args": args,
			},
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
			"Text": part.String(),
		}
	}
}

func expectedShellPartToComparable(part ExpectedShellPart) interface{} {
	result := map[string]interface{}{
		"Type": part.Type,
	}

	switch part.Type {
	case "text":
		result["Text"] = part.Text
	case "string":
		result["Value"] = part.Text // String content stored in Text field
	case "value_decorator":
		if part.FunctionDecorator != nil {
			args := expectedArgsToNamedParams(part.FunctionDecorator.Name, part.FunctionDecorator.Args)
			result["ValueDecorator"] = map[string]interface{}{
				"Name": part.FunctionDecorator.Name,
				"Args": args,
			}
		}
	case "action_decorator":
		if part.ActionDecorator != nil {
			args := expectedArgsToNamedParams(part.ActionDecorator.Name, part.ActionDecorator.Args)
			result["ActionDecorator"] = map[string]interface{}{
				"Name": part.ActionDecorator.Name,
				"Args": args,
			}
		}
	}

	return result
}

func patternToComparable(pattern ast.Pattern) interface{} {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return map[string]interface{}{
			"Type": "identifier",
			"Name": p.Name,
		}
	case *ast.WildcardPattern:
		return map[string]interface{}{
			"Type": "wildcard",
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func expectedPatternToComparable(pattern ExpectedPattern) interface{} {
	switch p := pattern.(type) {
	case ExpectedIdentifierPattern:
		return map[string]interface{}{
			"Type": "identifier",
			"Name": p.Name,
		}
	case ExpectedWildcardPattern:
		return map[string]interface{}{
			"Type": "wildcard",
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func commandContentToComparable(content ast.CommandContent) interface{} {
	switch c := content.(type) {
	case *ast.ShellContent:
		parts := make([]interface{}, len(c.Parts))
		for i, part := range c.Parts {
			parts[i] = shellPartToComparable(part)
		}
		return map[string]interface{}{
			"Type":  "shell",
			"Parts": parts,
		}
	case *ast.ShellChain:
		// Preserve ShellChain structure with proper operator semantics
		elements := make([]interface{}, len(c.Elements))
		for i, element := range c.Elements {
			elementMap := map[string]interface{}{
				"Content": map[string]interface{}{
					"Type": "shell",
					"Parts": func() []interface{} {
						parts := make([]interface{}, len(element.Content.Parts))
						for j, part := range element.Content.Parts {
							parts[j] = shellPartToComparable(part)
						}
						return parts
					}(),
				},
				"Operator": element.Operator,
			}
			if element.Target != "" {
				elementMap["Target"] = element.Target
			}
			elements[i] = elementMap
		}
		return map[string]interface{}{
			"Type":     "chain",
			"Elements": elements,
		}
	case *ast.ActionDecorator:
		args := make([]interface{}, len(c.Args))
		for i, arg := range c.Args {
			args[i] = namedParameterToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "action_decorator",
			"Name": c.Name,
			"Args": args,
		}
	// ast.BlockDecoratorContent removed - functionality moved to BlockDecorator
	case *ast.BlockDecorator:
		args := make([]interface{}, len(c.Args))
		for i, arg := range c.Args {
			args[i] = namedParameterToComparable(arg)
		}
		contentArray := make([]interface{}, len(c.Content))
		for i, content := range c.Content {
			contentArray[i] = commandContentToComparable(content)
		}
		return map[string]interface{}{
			"Type":    "block_decorator",
			"Name":    c.Name,
			"Args":    args,
			"Content": contentArray,
		}
	case *ast.PatternDecorator:
		args := make([]interface{}, len(c.Args))
		for i, arg := range c.Args {
			args[i] = namedParameterToComparable(arg)
		}
		patterns := make([]interface{}, len(c.Patterns))
		for i, pattern := range c.Patterns {
			commandArray := make([]interface{}, len(pattern.Commands))
			for j, cmd := range pattern.Commands {
				commandArray[j] = commandContentToComparable(cmd)
			}
			patterns[i] = map[string]interface{}{
				"Pattern":  patternToComparable(pattern.Pattern),
				"Commands": commandArray,
			}
		}
		return map[string]interface{}{
			"Type":     "pattern_decorator",
			"Name":     c.Name,
			"Args":     args,
			"Patterns": patterns,
		}
	case *ast.PatternContent:
		// Simplified PatternContent now just has Pattern string and Commands
		commandArray := make([]interface{}, len(c.Commands))
		for i, cmd := range c.Commands {
			commandArray[i] = commandContentToComparable(cmd)
		}
		return map[string]interface{}{
			"Type":     "pattern_content",
			"Pattern":  c.Pattern,
			"Commands": commandArray,
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

// Duplicate helper functions removed - already defined above

func expectedCommandContentToComparable(content ExpectedCommandContent) interface{} {
	switch c := content.(type) {
	case ExpectedShellContent:
		parts := make([]interface{}, len(c.Parts))
		for i, part := range c.Parts {
			parts[i] = expectedShellPartToComparable(part)
		}
		return map[string]interface{}{
			"Type":  "shell",
			"Parts": parts,
		}
	case ExpectedShellChain:
		elements := make([]interface{}, len(c.Elements))
		for i, element := range c.Elements {
			elementMap := map[string]interface{}{
				"Content":  expectedCommandContentToComparable(element.Content),
				"Operator": element.Operator,
			}
			if element.Target != "" {
				elementMap["Target"] = element.Target
			}
			elements[i] = elementMap
		}
		return map[string]interface{}{
			"Type":     "chain",
			"Elements": elements,
		}
	case ExpectedBlockDecorator:
		args := expectedArgsToNamedParams(c.Name, c.Args)
		content := make([]interface{}, len(c.Content))
		for i, cont := range c.Content {
			content[i] = expectedCommandContentToComparable(cont)
		}
		return map[string]interface{}{
			"Type":    "block_decorator",
			"Name":    c.Name,
			"Args":    args,
			"Content": content,
		}
	case ExpectedPatternDecorator:
		args := expectedArgsToNamedParams(c.Name, c.Args)
		patterns := make([]interface{}, len(c.Patterns))
		for i, pattern := range c.Patterns {
			// Convert pattern branch to comparable format
			patterns[i] = map[string]interface{}{
				"Pattern":  expectedPatternToComparable(pattern.Pattern),
				"Commands": expectedCommandContentArrayToComparable(pattern.Commands),
			}
		}
		return map[string]interface{}{
			"Type":     "pattern_decorator",
			"Name":     c.Name,
			"Args":     args,
			"Patterns": patterns,
		}
	case ExpectedFunctionDecorator:
		args := expectedArgsToNamedParams(c.Name, c.Args)
		return map[string]interface{}{
			"Type": "value_decorator",
			"Name": c.Name,
			"Args": args,
		}
	case ExpectedPatternContent:
		// Use the new simplified structure
		branches := make([]interface{}, len(c.Branches))
		for i, branch := range c.Branches {
			commandArray := make([]interface{}, len(branch.Commands))
			for j, cmd := range branch.Commands {
				commandArray[j] = expectedCommandContentToComparable(cmd)
			}
			branches[i] = map[string]interface{}{
				"Pattern":  expectedPatternToComparable(branch.Pattern),
				"Commands": commandArray,
			}
		}

		return map[string]interface{}{
			"Type":     "pattern",
			"Branches": branches,
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func commandBodyToComparable(body ast.CommandBody) interface{} {
	contentArray := make([]interface{}, len(body.Content))
	for i, content := range body.Content {
		contentArray[i] = commandContentToComparable(content)
	}

	return map[string]interface{}{
		"Content": contentArray,
	}
}

func expectedCommandBodyToComparable(body ExpectedCommandBody) interface{} {
	contentArray := make([]interface{}, len(body.Content))
	for i, content := range body.Content {
		contentArray[i] = expectedCommandContentToComparable(content)
	}

	return map[string]interface{}{
		"Content": contentArray,
	}
}

func expectedCommandContentArrayToComparable(contents []ExpectedCommandContent) []interface{} {
	result := make([]interface{}, len(contents))
	for i, content := range contents {
		result[i] = expectedCommandContentToComparable(content)
	}
	return result
}

func RunTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		program, err := Parse(strings.NewReader(tc.Input))

		if tc.WantErr {
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tc.ErrorSubstr != "" && !strings.Contains(err.Error(), tc.ErrorSubstr) {
				t.Errorf("expected error containing %q, got %q", tc.ErrorSubstr, err.Error())
			}
			return
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Flatten all variables (individual and grouped) for comparison
		allVariables := flattenVariables(program)

		// Verify variables
		if len(allVariables) != len(tc.Expected.Variables) {
			t.Errorf("expected %d variables, got %d", len(tc.Expected.Variables), len(allVariables))
		} else {
			for i, expectedVar := range tc.Expected.Variables {
				actualVar := allVariables[i]

				actualComparable := map[string]interface{}{
					"Name":  actualVar.Name,
					"Value": expressionToComparable(actualVar.Value),
				}

				expectedComparable := map[string]interface{}{
					"Name": expectedVar.Name,
					"Value": map[string]interface{}{
						"Type":  expectedVar.Value.Type,
						"Value": expectedVar.Value.Value,
					},
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Variable[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}

		// Verify commands
		if len(program.Commands) != len(tc.Expected.Commands) {
			t.Errorf("expected %d commands, got %d", len(tc.Expected.Commands), len(program.Commands))
		} else {
			for i, expectedCmd := range tc.Expected.Commands {
				actualCmd := program.Commands[i]

				actualComparable := map[string]interface{}{
					"Name": actualCmd.Name,
					"Type": actualCmd.Type,
					"Body": commandBodyToComparable(actualCmd.Body),
				}

				expectedComparable := map[string]interface{}{
					"Name": expectedCmd.Name,
					"Type": expectedCmd.Type,
					"Body": expectedCommandBodyToComparable(expectedCmd.Body),
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Command[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}
	})
}
