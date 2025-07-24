package decorators

import (
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// ValidateParameterType validates that a parameter value matches the expected type
// Allows both literal values and identifiers (which can resolve at runtime)
func ValidateParameterType(paramName string, paramValue ast.Expression, expectedType ast.ExpressionType, decoratorName string) error {
	switch expectedType {
	case ast.StringType:
		switch paramValue.(type) {
		case *ast.StringLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type string", decoratorName, paramName)
		}
	case ast.NumberType:
		switch paramValue.(type) {
		case *ast.NumberLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type number", decoratorName, paramName)
		}
	case ast.DurationType:
		switch paramValue.(type) {
		case *ast.DurationLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type duration", decoratorName, paramName)
		}
	case ast.BooleanType:
		switch paramValue.(type) {
		case *ast.BooleanLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type boolean", decoratorName, paramName)
		}
	default:
		return fmt.Errorf("@%s '%s' parameter has unsupported type %v", decoratorName, paramName, expectedType)
	}
}

// ValidateRequiredParameter validates that a required parameter exists and has the correct type
func ValidateRequiredParameter(params []ast.NamedParameter, paramName string, expectedType ast.ExpressionType, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s requires '%s' parameter", decoratorName, paramName)
	}
	return ValidateParameterType(paramName, param.Value, expectedType, decoratorName)
}

// ValidateOptionalParameter validates that an optional parameter (if present) has the correct type
func ValidateOptionalParameter(params []ast.NamedParameter, paramName string, expectedType ast.ExpressionType, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter not provided, which is fine
	}
	return ValidateParameterType(paramName, param.Value, expectedType, decoratorName)
}
