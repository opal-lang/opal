package planner

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/opal-lang/opal/core/types"
)

// ExprKind identifies the type of expression.
type ExprKind int

const (
	ExprLiteral       ExprKind = iota // Literal value (string, int, bool)
	ExprVarRef                        // Variable reference (@var.X)
	ExprDecoratorRef                  // Decorator reference (@env.HOME, @aws.secret.key)
	ExprEnumMemberRef                 // Enum member reference (Type.Member)
	ExprBinaryOp                      // Binary operation (==, !=, &&, ||)
	ExprTypeCast                      // Type cast (expr as Type, expr as Type?)
)

// durationLiteral preserves duration typing for literals (e.g., 5m, 30s).
type durationLiteral string

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

	// For ExprEnumMemberRef - enum type and member
	EnumName   string
	EnumMember string

	// For ExprBinaryOp - operator and operands
	Op    string  // "==", "!=", "&&", "||", "<", ">", "<=", ">="
	Left  *ExprIR // Left operand
	Right *ExprIR // Right operand

	// For ExprTypeCast - target type and optionality
	TypeName string // Target type name (String, Int, Object, etc.)
	Optional bool   // True for Type? casts
}

// DecoratorRef is a structured decorator reference.
// Represents @decorator.selector.path with optional arguments.
type DecoratorRef struct {
	Name     string    // Decorator name: "env", "aws", "var", "shell", etc.
	Selector []string  // Property path: ["HOME"], ["secret", "api_key"]
	Args     []*ExprIR // For parameterized decorators: @retry(3, "1s")
	ArgNames []string  // Argument keys aligned with Args (canonical names; empty means positional source)
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
// Named CommandExpr to distinguish expression-level command parts from
// statement-level command IR types.
type CommandExpr struct {
	Parts []*ExprIR // Sequence of literals and expression references
}

// ValueLookup is a function that looks up a value by name.
// Returns (value, true) if found, (nil, false) if not found.
// Used by EvaluateExpr to look up variable and decorator values.
type ValueLookup func(name string) (any, bool)

// EvaluateExpr evaluates an expression using a lookup function for values.
// Used during the resolution phase to evaluate conditions like `@var.ENV == "prod"`.
//
// The getValue function looks up values by name:
//   - For ExprVarRef, looks up by variable name (e.g., "ENV")
//   - For ExprDecoratorRef, looks up by decorator key (e.g., "env.HOME")
//
// For ExprBinaryOp, recursively evaluates operands and applies the operator.
func EvaluateExpr(expr *ExprIR, getValue ValueLookup) (any, error) {
	if expr == nil {
		return nil, &EvalError{Message: "missing expression"}
	}

	switch expr.Kind {
	case ExprLiteral:
		// Handle array literals: []*ExprIR → []any (evaluated)
		if arr, ok := expr.Value.([]*ExprIR); ok {
			return evaluateExprArray(arr, getValue)
		}
		// Handle object literals: map[string]*ExprIR → map[string]any (evaluated)
		if obj, ok := expr.Value.(map[string]*ExprIR); ok {
			return evaluateExprObject(obj, getValue)
		}
		// Primitive literal (string, int, bool)
		return expr.Value, nil

	case ExprVarRef:
		val, ok := getValue(expr.VarName)
		if !ok {
			return nil, &EvalError{
				Message: "undefined variable",
				VarName: expr.VarName,
				Span:    expr.Span,
			}
		}
		return val, nil

	case ExprDecoratorRef:
		// Decorator refs should be resolved before evaluation.
		// The key is the decorator path (e.g., "env.HOME").
		key := decoratorKey(expr.Decorator)
		val, ok := getValue(key)
		if !ok {
			return nil, &EvalError{
				Message: "unresolved decorator",
				VarName: key,
				Span:    expr.Span,
			}
		}
		return val, nil

	case ExprEnumMemberRef:
		key := enumMemberRefKey(expr.EnumName, expr.EnumMember)
		val, ok := getValue(key)
		if !ok {
			return nil, &EvalError{
				Message: "unresolved enum member",
				VarName: key,
				Span:    expr.Span,
			}
		}
		return val, nil

	case ExprBinaryOp:
		return evaluateBinaryOp(expr, getValue)

	case ExprTypeCast:
		return evaluateTypeCast(expr, getValue)

	default:
		return nil, &EvalError{
			Message: "unknown expression kind",
			Span:    expr.Span,
		}
	}
}

// evaluateBinaryOp evaluates a binary operation.
func evaluateBinaryOp(expr *ExprIR, getValue ValueLookup) (any, error) {
	if expr.Left == nil || expr.Right == nil {
		return nil, &EvalError{Message: "incomplete binary expression", Span: expr.Span}
	}

	left, err := EvaluateExpr(expr.Left, getValue)
	if err != nil {
		return nil, err
	}

	// Short-circuit evaluation for && and ||
	switch expr.Op {
	case "&&":
		if !IsTruthy(left) {
			return false, nil
		}
		right, err := EvaluateExpr(expr.Right, getValue)
		if err != nil {
			return nil, err
		}
		return IsTruthy(right), nil

	case "||":
		if IsTruthy(left) {
			return true, nil
		}
		right, err := EvaluateExpr(expr.Right, getValue)
		if err != nil {
			return nil, err
		}
		return IsTruthy(right), nil
	}

	// Non-short-circuit operators need both operands
	right, err := EvaluateExpr(expr.Right, getValue)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case "+":
		if a, ok := left.(string); ok {
			if b, ok := right.(string); ok {
				return a + b, nil
			}
		}
		if aInt, bInt, ok := toInt64Pair(left, right); ok {
			return aInt + bInt, nil
		}
		if aFloat, bFloat, ok := toFloat64Pair(left, right); ok {
			return aFloat + bFloat, nil
		}
		return nil, &EvalError{Message: "cannot add values", Span: expr.Span}

	case "-":
		if aInt, bInt, ok := toInt64Pair(left, right); ok {
			return aInt - bInt, nil
		}
		if aFloat, bFloat, ok := toFloat64Pair(left, right); ok {
			return aFloat - bFloat, nil
		}
		return nil, &EvalError{Message: "cannot subtract non-numeric values", Span: expr.Span}

	case "*":
		if aInt, bInt, ok := toInt64Pair(left, right); ok {
			return aInt * bInt, nil
		}
		if aFloat, bFloat, ok := toFloat64Pair(left, right); ok {
			return aFloat * bFloat, nil
		}
		return nil, &EvalError{Message: "cannot multiply non-numeric values", Span: expr.Span}

	case "/":
		if aInt, bInt, ok := toInt64Pair(left, right); ok {
			if bInt == 0 {
				return nil, &EvalError{Message: "division by zero", Span: expr.Span}
			}
			return aInt / bInt, nil
		}
		if aFloat, bFloat, ok := toFloat64Pair(left, right); ok {
			if bFloat == 0 {
				return nil, &EvalError{Message: "division by zero", Span: expr.Span}
			}
			return aFloat / bFloat, nil
		}
		return nil, &EvalError{Message: "cannot divide non-numeric values", Span: expr.Span}

	case "%":
		if aInt, bInt, ok := toInt64Pair(left, right); ok {
			if bInt == 0 {
				return nil, &EvalError{Message: "division by zero", Span: expr.Span}
			}
			return aInt % bInt, nil
		}
		return nil, &EvalError{Message: "cannot modulo non-integer values", Span: expr.Span}

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

type castTypeSpec struct {
	kind     types.ParamType
	optional bool
}

func evaluateTypeCast(expr *ExprIR, getValue ValueLookup) (any, error) {
	if expr.Left == nil {
		return nil, &EvalError{Message: "incomplete type cast", Span: expr.Span}
	}

	spec, err := parseCastType(expr.TypeName, expr.Optional)
	if err != nil {
		return nil, &EvalError{Message: err.Error(), Span: expr.Span}
	}

	value, err := EvaluateExpr(expr.Left, getValue)
	if err != nil {
		return nil, err
	}

	castValue, err := castValueAsType(value, spec)
	if err != nil {
		return nil, &EvalError{Message: err.Error(), Span: expr.Span}
	}

	return castValue, nil
}

func parseCastType(typeName string, optional bool) (castTypeSpec, error) {
	switch strings.ToLower(typeName) {
	case "string":
		return castTypeSpec{kind: types.TypeString, optional: optional}, nil
	case "int", "integer":
		return castTypeSpec{kind: types.TypeInt, optional: optional}, nil
	case "float":
		return castTypeSpec{kind: types.TypeFloat, optional: optional}, nil
	case "bool", "boolean":
		return castTypeSpec{kind: types.TypeBool, optional: optional}, nil
	case "duration":
		return castTypeSpec{kind: types.TypeDuration, optional: optional}, nil
	case "array":
		return castTypeSpec{kind: types.TypeArray, optional: optional}, nil
	case "map", "object":
		return castTypeSpec{kind: types.TypeObject, optional: optional}, nil
	default:
		return castTypeSpec{}, fmt.Errorf("unsupported cast target %q", typeName)
	}
}

func castTypeLabel(spec castTypeSpec) string {
	label := functionTypeLabel(spec.kind)
	if spec.optional {
		return label + " or none"
	}
	return label
}

func castValueAsType(value any, spec castTypeSpec) (any, error) {
	if value == nil {
		if spec.optional {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot cast none to %s", functionTypeLabel(spec.kind))
	}

	switch spec.kind {
	case types.TypeString:
		return literalToString(value), nil

	case types.TypeInt:
		if intVal, ok := toInt64Strict(value); ok {
			return intVal, nil
		}
		return nil, fmt.Errorf("cannot cast %T to %s", value, castTypeLabel(spec))

	case types.TypeFloat:
		if floatVal, ok := toFloat64Strict(value); ok {
			return floatVal, nil
		}
		return nil, fmt.Errorf("cannot cast %T to %s", value, castTypeLabel(spec))

	case types.TypeBool:
		if boolVal, ok := toBoolStrict(value); ok {
			return boolVal, nil
		}
		return nil, fmt.Errorf("cannot cast %T to %s", value, castTypeLabel(spec))

	case types.TypeDuration:
		if durationVal, ok := toDurationStrict(value); ok {
			return durationVal, nil
		}
		return nil, fmt.Errorf("cannot cast %T to %s", value, castTypeLabel(spec))

	case types.TypeArray:
		if arrayVal, ok := toAnyArray(value); ok {
			return arrayVal, nil
		}
		return nil, fmt.Errorf("cannot cast %T to %s", value, castTypeLabel(spec))

	case types.TypeObject:
		if objectVal, ok := toAnyObject(value); ok {
			return objectVal, nil
		}
		return nil, fmt.Errorf("cannot cast %T to %s", value, castTypeLabel(spec))

	default:
		return nil, fmt.Errorf("cannot cast to unsupported type %q", spec.kind)
	}
}

func toInt64Strict(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		maxInt64 := uint64(^uint64(0) >> 1)
		if uint64(v) > maxInt64 {
			return 0, false
		}
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		maxInt64 := uint64(^uint64(0) >> 1)
		if v > maxInt64 {
			return 0, false
		}
		return int64(v), true
	case float32:
		f := float64(v)
		if f != float64(int64(f)) {
			return 0, false
		}
		return int64(f), true
	case float64:
		if v != float64(int64(v)) {
			return 0, false
		}
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func toFloat64Strict(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func toBoolStrict(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		if lower == "true" {
			return true, true
		}
		if lower == "false" {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

func toDurationStrict(value any) (types.Duration, bool) {
	switch v := value.(type) {
	case types.Duration:
		return v, true
	case time.Duration:
		parsed, err := types.ParseDuration(v.String())
		if err != nil {
			return types.Duration{}, false
		}
		return parsed, true
	case durationLiteral:
		parsed, err := types.ParseDuration(string(v))
		if err != nil {
			return types.Duration{}, false
		}
		return parsed, true
	case string:
		parsed, err := types.ParseDuration(v)
		if err != nil {
			return types.Duration{}, false
		}
		return parsed, true
	default:
		return types.Duration{}, false
	}
}

func toAnyArray(value any) ([]any, bool) {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil, false
	}
	kind := v.Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		return nil, false
	}

	result := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}

	return result, true
}

func toAnyObject(value any) (map[string]any, bool) {
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.Kind() != reflect.Map {
		return nil, false
	}
	if v.Type().Key().Kind() != reflect.String {
		return nil, false
	}

	result := make(map[string]any, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		result[iter.Key().String()] = iter.Value().Interface()
	}

	return result, true
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
	if aDur, ok := a.(durationLiteral); ok {
		if bDur, ok := b.(durationLiteral); ok {
			return string(aDur) == string(bDur)
		}
		if bStr, ok := b.(string); ok {
			return string(aDur) == bStr
		}
	}
	if bDur, ok := b.(durationLiteral); ok {
		if aStr, ok := a.(string); ok {
			return aStr == string(bDur)
		}
	}

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

// evaluateExprArray evaluates an array literal where elements are *ExprIR.
// This handles arrays like ["a", @var.x, 1+2] by evaluating each element.
func evaluateExprArray(arr []*ExprIR, getValue ValueLookup) (any, error) {
	result := make([]any, len(arr))
	for i, expr := range arr {
		val, err := EvaluateExpr(expr, getValue)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

// evaluateExprObject evaluates an object literal where field values are *ExprIR.
// This handles objects like {name: @var.x, count: 1+2} by evaluating each field.
func evaluateExprObject(obj map[string]*ExprIR, getValue ValueLookup) (any, error) {
	result := make(map[string]any, len(obj))
	for key, expr := range obj {
		val, err := EvaluateExpr(expr, getValue)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
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

func enumMemberRefKey(enumName, member string) string {
	if enumName == "" {
		return member
	}
	if member == "" {
		return enumName
	}
	return enumName + "." + member
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

	case ExprEnumMemberRef:
		key := enumMemberRefKey(expr.EnumName, expr.EnumMember)
		if value, ok := displayIDs[key]; ok {
			return value
		}
		return key

	case ExprTypeCast:
		if expr.Left == nil {
			return "<unsupported>"
		}
		return RenderExpr(expr.Left, displayIDs)

	default:
		// Binary ops shouldn't be rendered (they're for conditions)
		return "<unsupported>"
	}
}

// RenderCommand renders a command expression to a string with DisplayID placeholders.
// Concatenates all parts, rendering each according to its type.
// Trims leading and trailing whitespace from the result.
func RenderCommand(cmd *CommandExpr, displayIDs map[string]string) string {
	if cmd == nil || len(cmd.Parts) == 0 {
		return ""
	}

	var result string
	for _, part := range cmd.Parts {
		result += RenderExpr(part, displayIDs)
	}
	return strings.TrimSpace(result)
}

// literalToString converts a literal value to its string representation.
func literalToString(v any) string {
	switch val := v.(type) {
	case nil:
		return "none"
	case string:
		return val
	case durationLiteral:
		return string(val)
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
