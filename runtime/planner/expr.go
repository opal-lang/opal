package planner

import "strconv"

// ExprKind identifies the type of expression.
type ExprKind int

const (
	ExprLiteral      ExprKind = iota // Literal value (string, int, bool)
	ExprVarRef                       // Variable reference (@var.X)
	ExprDecoratorRef                 // Decorator reference (@env.HOME, @aws.secret.key)
	ExprBinaryOp                     // Binary operation (==, !=, &&, ||)
)

// ExprIR is the unified expression representation.
// Used for both conditions (if @var.X == "prod") and command interpolation (echo @var.X).
// This unification prevents divergence between two separate expression systems.
type ExprIR struct {
	Kind ExprKind
	Span SourceSpan

	// For ExprLiteral - the actual value (string, int, bool, etc.)
	Value any

	// For ExprVarRef - the variable name (without @var. prefix)
	VarName string

	// For ExprDecoratorRef - structured decorator reference
	Decorator *DecoratorRef

	// For ExprBinaryOp - operator and operands
	Op    string  // "==", "!=", "&&", "||", "<", ">", "<=", ">="
	Left  *ExprIR // Left operand
	Right *ExprIR // Right operand
}

// DecoratorRef is a structured decorator reference.
// Represents @decorator.selector.path with optional arguments.
type DecoratorRef struct {
	Name     string    // Decorator name: "env", "aws", "var", "shell", etc.
	Selector []string  // Property path: ["HOME"], ["secret", "api_key"]
	Args     []*ExprIR // For parameterized decorators: @retry(3, "1s")
}

// SourceSpan identifies a range in source code for error messages.
type SourceSpan struct {
	File  string // Source file name (empty for inline/REPL)
	Start int    // Start position (byte offset)
	End   int    // End position (byte offset)
}

// CommandExpr represents a command with interpolated expressions.
// A command like `echo "Hello @var.NAME"` becomes:
//
//	Parts: [Literal("echo \"Hello "), VarRef("NAME"), Literal("\"")]
//
// Note: Named CommandExpr to avoid conflict with existing CommandIR in planner.go.
// Will be renamed to CommandIR when old planner is removed in MR #3.
type CommandExpr struct {
	Parts []*ExprIR // Sequence of literals and expression references
}

// EvaluateExpr evaluates an expression given a map of resolved values.
// Used during the resolution phase to evaluate conditions like `@var.ENV == "prod"`.
//
// The values map contains variable name → resolved value mappings.
// For ExprVarRef, looks up the variable in values.
// For ExprDecoratorRef, the caller must have already resolved and stored in values.
// For ExprBinaryOp, recursively evaluates operands and applies the operator.
func EvaluateExpr(expr *ExprIR, values map[string]any) (any, error) {
	switch expr.Kind {
	case ExprLiteral:
		return expr.Value, nil

	case ExprVarRef:
		val, ok := values[expr.VarName]
		if !ok {
			return nil, &EvalError{
				Message: "undefined variable",
				VarName: expr.VarName,
				Span:    expr.Span,
			}
		}
		return val, nil

	case ExprDecoratorRef:
		// Decorator refs should be resolved and stored in values before evaluation.
		// The key is the decorator path (e.g., "env.HOME").
		key := decoratorKey(expr.Decorator)
		val, ok := values[key]
		if !ok {
			return nil, &EvalError{
				Message: "unresolved decorator",
				VarName: key,
				Span:    expr.Span,
			}
		}
		return val, nil

	case ExprBinaryOp:
		return evaluateBinaryOp(expr, values)

	default:
		return nil, &EvalError{
			Message: "unknown expression kind",
			Span:    expr.Span,
		}
	}
}

// evaluateBinaryOp evaluates a binary operation.
func evaluateBinaryOp(expr *ExprIR, values map[string]any) (any, error) {
	left, err := EvaluateExpr(expr.Left, values)
	if err != nil {
		return nil, err
	}

	// Short-circuit evaluation for && and ||
	switch expr.Op {
	case "&&":
		if !IsTruthy(left) {
			return false, nil
		}
		right, err := EvaluateExpr(expr.Right, values)
		if err != nil {
			return nil, err
		}
		return IsTruthy(right), nil

	case "||":
		if IsTruthy(left) {
			return true, nil
		}
		right, err := EvaluateExpr(expr.Right, values)
		if err != nil {
			return nil, err
		}
		return IsTruthy(right), nil
	}

	// Non-short-circuit operators need both operands
	right, err := EvaluateExpr(expr.Right, values)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case "==":
		return compareEqual(left, right), nil
	case "!=":
		return !compareEqual(left, right), nil
	case "<":
		return compareLess(left, right)
	case ">":
		return compareLess(right, left) // a > b is b < a
	case "<=":
		less, err := compareLess(left, right)
		if err != nil {
			return nil, err
		}
		return less || compareEqual(left, right), nil
	case ">=":
		less, err := compareLess(right, left)
		if err != nil {
			return nil, err
		}
		return less || compareEqual(left, right), nil
	default:
		return nil, &EvalError{
			Message: "unknown operator: " + expr.Op,
			Span:    expr.Span,
		}
	}
}

// IsTruthy determines if a value is truthy.
// Follows common scripting language conventions:
// - nil is falsy
// - false is falsy
// - 0 (any numeric type) is falsy
// - "" (empty string) is falsy
// - Everything else is truthy
func IsTruthy(v any) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		// Unknown types are truthy (non-nil)
		return true
	}
}

// compareEqual compares two values for equality.
func compareEqual(a, b any) bool {
	if a == b {
		return true
	}

	// Try integer comparison first (preserves precision)
	if aInt, bInt, ok := toInt64Pair(a, b); ok {
		return aInt == bInt
	}

	// Fall back to float64 for mixed int/float
	if aFloat, bFloat, ok := toFloat64Pair(a, b); ok {
		return aFloat == bFloat
	}

	return false
}

// compareLess compares two values for less-than.
func compareLess(a, b any) (bool, error) {
	if aInt, bInt, ok := toInt64Pair(a, b); ok {
		return aInt < bInt, nil
	}

	if aFloat, bFloat, ok := toFloat64Pair(a, b); ok {
		return aFloat < bFloat, nil
	}

	return false, &EvalError{Message: "cannot compare non-numeric values with <"}
}

// toInt64Pair converts both values to int64 if both are integer types.
func toInt64Pair(a, b any) (int64, int64, bool) {
	var aInt, bInt int64
	switch v := a.(type) {
	case int:
		aInt = int64(v)
	case int64:
		aInt = v
	default:
		return 0, 0, false
	}
	switch v := b.(type) {
	case int:
		bInt = int64(v)
	case int64:
		bInt = v
	default:
		return 0, 0, false
	}
	return aInt, bInt, true
}

// toFloat64Pair converts both values to float64 if both are numeric.
func toFloat64Pair(a, b any) (float64, float64, bool) {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if aOk && bOk {
		return aFloat, bFloat, true
	}
	return 0, 0, false
}

// toFloat64 converts a value to float64 if possible.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

// decoratorKey builds a lookup key for a decorator reference.
// e.g., DecoratorRef{Name: "env", Selector: ["HOME"]} → "env.HOME"
func decoratorKey(d *DecoratorRef) string {
	if d == nil {
		return ""
	}
	key := d.Name
	for _, s := range d.Selector {
		key += "." + s
	}
	return key
}

// EvalError represents an error during expression evaluation.
type EvalError struct {
	Message string
	VarName string     // Variable or decorator name (if applicable)
	Span    SourceSpan // Source location (if available)
}

func (e *EvalError) Error() string {
	if e.VarName != "" {
		return e.Message + ": " + e.VarName
	}
	return e.Message
}

// RenderExpr renders an expression to a string with DisplayID placeholders.
// Used during the emit phase to build plan commands.
//
// For literals, returns the string representation.
// For var refs and decorator refs, looks up the DisplayID in the map.
// Binary ops are not supported (they're for conditions, not interpolation).
func RenderExpr(expr *ExprIR, displayIDs map[string]string) string {
	switch expr.Kind {
	case ExprLiteral:
		return literalToString(expr.Value)

	case ExprVarRef:
		if id, ok := displayIDs[expr.VarName]; ok {
			return id
		}
		return "<unresolved:" + expr.VarName + ">"

	case ExprDecoratorRef:
		key := decoratorKey(expr.Decorator)
		if id, ok := displayIDs[key]; ok {
			return id
		}
		return "<unresolved:" + key + ">"

	default:
		// Binary ops shouldn't be rendered (they're for conditions)
		return "<unsupported>"
	}
}

// RenderCommand renders a command expression to a string with DisplayID placeholders.
// Concatenates all parts, rendering each according to its type.
func RenderCommand(cmd *CommandExpr, displayIDs map[string]string) string {
	if cmd == nil || len(cmd.Parts) == 0 {
		return ""
	}

	var result string
	for _, part := range cmd.Parts {
		result += RenderExpr(part, displayIDs)
	}
	return result
}

// literalToString converts a literal value to its string representation.
func literalToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return "<unknown>"
	}
}
