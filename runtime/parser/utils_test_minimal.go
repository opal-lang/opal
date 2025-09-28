package parser

import (
	"fmt"
	"testing"

	// Import builtins to register decorators
	_ "github.com/aledsdavies/opal/runtime/decorators/builtin"
)

// init registers any test-specific decorators not in decorators
func init() {
	registerTestOnlyDecorators()
}

// registerTestOnlyDecorators - no test decorators needed (we have real ones)
func registerTestOnlyDecorators() {
	// No test decorators to register - we use real decorators from builtin package
}

// Test utility types and functions
type (
	ExpectedProgram  []interface{}
	ExpectedVariable struct {
		Name  string
		Value ExpectedExpression
	}
)

type ExpectedExpression interface {
	IsExpectedExpression() bool
}

type ExpectedCommand struct {
	Name string
	Body ExpectedCommandBody
}

type ExpectedCommandBody []ExpectedCommandContent

type ExpectedCommandContent interface {
	IsExpectedCommandContent() bool
}

type ExpectedShellContent []ExpectedShellPart

func (s ExpectedShellContent) IsExpectedCommandContent() bool { return true }

type ExpectedShellChain struct {
	Elements []ExpectedShellChainElement
}

func (s ExpectedShellChain) IsExpectedCommandContent() bool { return true }

type ExpectedShellPart interface {
	IsExpectedShellPart() bool
}

type ExpectedShellChainElement struct {
	Content  ExpectedShellContent
	Operator string
	Target   string
}

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

type ExpectedPatternContent struct {
	Decorator ExpectedDecorator
	Patterns  []ExpectedPatternBranch
}

func (p ExpectedPatternContent) IsExpectedCommandContent() bool { return true }

type ExpectedDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedFunctionDecorator struct {
	Name string
	Args []ExpectedExpression
}

func (f ExpectedFunctionDecorator) IsExpectedCommandContent() bool { return true }

type ExpectedPatternBranch struct {
	Pattern ExpectedPattern
	Content []ExpectedCommandContent
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

// String literal expressions
type ExpectedStringLiteral struct {
	Value string
}

func (s ExpectedStringLiteral) IsExpectedExpression() bool { return true }
func (s ExpectedStringLiteral) IsExpectedShellPart() bool  { return true }

type ExpectedStringWithParts struct {
	Parts []ExpectedShellPart
}

func (s ExpectedStringWithParts) IsExpectedExpression() bool { return true }

// Other expression types
type ExpectedIdentifier struct {
	Name string
}

func (i ExpectedIdentifier) IsExpectedExpression() bool { return true }

type ExpectedNumber struct {
	Value interface{}
}

func (n ExpectedNumber) IsExpectedExpression() bool { return true }

type ExpectedBoolean struct {
	Value bool
}

func (b ExpectedBoolean) IsExpectedExpression() bool { return true }

type ExpectedDuration struct {
	Value string
}

func (d ExpectedDuration) IsExpectedExpression() bool { return true }

type ExpectedNamedParameter struct {
	Name  string
	Value ExpectedExpression
}

func (n ExpectedNamedParameter) IsExpectedExpression() bool { return true }

type ExpectedValueDecorator struct {
	Name string
	Args []ExpectedExpression
}

func (v ExpectedValueDecorator) IsExpectedShellPart() bool { return true }

// Utility functions to build test cases
func Program(items ...interface{}) ExpectedProgram {
	return ExpectedProgram(items)
}

func Var(name string, value interface{}) ExpectedVariable {
	return ExpectedVariable{Name: name, Value: toExpression(value)}
}

func Str(value string) ExpectedExpression {
	return ExpectedStringLiteral{Value: value}
}

func Id(name string) ExpectedExpression {
	return ExpectedIdentifier{Name: name}
}

func Num(value interface{}) ExpectedExpression {
	return ExpectedNumber{Value: value}
}

func Bool(value bool) ExpectedExpression {
	return ExpectedBoolean{Value: value}
}

func Dur(value string) ExpectedExpression {
	return ExpectedDuration{Value: value}
}

func DurationExpr(value string) ExpectedExpression {
	return ExpectedDuration{Value: value}
}

func Duration(value string) ExpectedExpression {
	return ExpectedDuration{Value: value}
}

func Named(name string, value ExpectedExpression) ExpectedExpression {
	return ExpectedNamedParameter{Name: name, Value: value}
}

func BooleanExpr(value bool) ExpectedExpression {
	return ExpectedBoolean{Value: value}
}

func Cmd(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{Name: name, Body: toCommandBody(body)}
}

func CmdBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{Name: name, Body: toMultipleCommandContent(content...)}
}

func Simple(parts ...interface{}) ExpectedCommandBody {
	return toMultipleCommandContent(parts...)
}

func Text(text string) ExpectedShellPart {
	return ExpectedStringLiteral{Value: text}
}

func StrPart(value string) ExpectedShellPart {
	return ExpectedStringLiteral{Value: value}
}

func StrWithParts(parts ...interface{}) ExpectedExpression {
	return ExpectedStringWithParts{Parts: toShellParts(parts...)}
}

func At(name string, args ...interface{}) ExpectedShellPart {
	return ExpectedValueDecorator{Name: name, Args: toExpressionSlice(args)}
}

func Decorator(name string, args ...interface{}) ExpectedDecorator {
	return ExpectedDecorator{Name: name, Args: toExpressionSlice(args)}
}

func Shell(parts ...interface{}) ExpectedCommandContent {
	return ExpectedShellContent(toShellParts(parts...))
}

func Chain(elements ...interface{}) ExpectedCommandContent {
	var chainElements []ExpectedShellChainElement
	for _, elem := range elements {
		if chainElem, ok := elem.(ExpectedShellChainElement); ok {
			chainElements = append(chainElements, chainElem)
		}
	}
	return ExpectedShellChain{Elements: chainElements}
}

func ChainElement(content interface{}, operator string, target ...string) ExpectedShellChainElement {
	var targetStr string
	if len(target) > 0 {
		targetStr = target[0]
	}

	var shellContent ExpectedShellContent
	if sc, ok := content.(ExpectedShellContent); ok {
		shellContent = sc
	} else {
		shellContent = ExpectedShellContent{toSingleShellPart(content)}
	}

	return ExpectedShellChainElement{
		Content:  shellContent,
		Operator: operator,
		Target:   targetStr,
	}
}

func Pipe(content interface{}) ExpectedShellChainElement {
	return ChainElement(content, "|")
}

func And(content interface{}) ExpectedShellChainElement {
	return ChainElement(content, "&&")
}

func Or(content interface{}) ExpectedShellChainElement {
	return ChainElement(content, "||")
}

func Append(content interface{}, target string) ExpectedShellChainElement {
	return ChainElement(content, ">>", target)
}

func BlockDecorator(name string, args ...interface{}) ExpectedCommandContent {
	return ExpectedBlockDecorator{Name: name, Args: toExpressionSlice(args)}
}

// Additional utility functions for command types
func Watch(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{Name: name, Body: toCommandBody(body)}
}

func Stop(name string, body interface{}) ExpectedCommand {
	return ExpectedCommand{Name: name, Body: toCommandBody(body)}
}

func WatchBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{Name: name, Body: toMultipleCommandContent(content...)}
}

func StopBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{Name: name, Body: toMultipleCommandContent(content...)}
}

func DecoratedShell(decorator ExpectedDecorator, shellParts ...ExpectedShellPart) ExpectedCommandContent {
	shellContent := ExpectedShellContent(shellParts)
	return ExpectedBlockDecorator{
		Name:    decorator.Name,
		Args:    decorator.Args,
		Content: []ExpectedCommandContent{shellContent},
	}
}

func PatternDecoratorWithBranches(name string, arg ExpectedExpression, branches ...ExpectedPatternBranch) ExpectedCommandContent {
	return ExpectedPatternDecorator{
		Name:     name,
		Args:     []ExpectedExpression{arg},
		Patterns: branches,
	}
}

func Branch(pattern string, content ...ExpectedCommandContent) ExpectedPatternBranch {
	return ExpectedPatternBranch{
		Pattern: ExpectedIdentifierPattern{Name: pattern},
		Content: content,
	}
}

// Helper functions
func toExpression(v interface{}) ExpectedExpression {
	switch val := v.(type) {
	case string:
		return ExpectedStringLiteral{Value: val}
	case int:
		return ExpectedNumber{Value: val}
	case bool:
		return ExpectedBoolean{Value: val}
	case ExpectedExpression:
		return val
	default:
		return ExpectedStringLiteral{Value: fmt.Sprintf("%v", val)}
	}
}

func toExpressionSlice(args []interface{}) []ExpectedExpression {
	var result []ExpectedExpression
	for _, arg := range args {
		result = append(result, toExpression(arg))
	}
	return result
}

func toCommandBody(v interface{}) ExpectedCommandBody {
	switch val := v.(type) {
	case ExpectedCommandBody:
		return val
	case []interface{}:
		return toMultipleCommandContent(val...)
	default:
		return ExpectedCommandBody{toSingleCommandContent(val)}
	}
}

func toMultipleCommandContent(items ...interface{}) []ExpectedCommandContent {
	var result []ExpectedCommandContent
	for _, item := range items {
		result = append(result, toSingleCommandContent(item))
	}
	return result
}

func toSingleCommandContent(item interface{}) ExpectedCommandContent {
	if content, ok := item.(ExpectedCommandContent); ok {
		return content
	}
	// Default to shell content
	return ExpectedShellContent{toSingleShellPart(item)}
}

func toShellParts(items ...interface{}) []ExpectedShellPart {
	var result []ExpectedShellPart
	for _, item := range items {
		result = append(result, toSingleShellPart(item))
	}
	return result
}

func toSingleShellPart(item interface{}) ExpectedShellPart {
	switch val := item.(type) {
	case ExpectedShellPart:
		return val
	case string:
		return ExpectedStringLiteral{Value: val}
	default:
		return ExpectedStringLiteral{Value: fmt.Sprintf("%v", val)}
	}
}

// Test case runner
type TestCase struct {
	Name        string
	Input       string
	Expected    interface{}
	WantErr     bool
	ErrorSubstr string
}

func RunTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		// Implementation would parse input and compare with expected
		// This is a stub for now
	})
}
