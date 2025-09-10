package decorators

import (
	"fmt"
	"strconv"
	"time"
)

// ================================================================================================
// DECORATOR PARAMETER TYPES - Self-contained parameter abstraction for decorators
// ================================================================================================

// DecoratorParam represents a parameter passed to a decorator
type DecoratorParam struct {
	Name  string      `json:"name"`  // Parameter name (empty for positional)
	Value interface{} `json:"value"` // Parameter value
}

// ================================================================================================
// PARAMETER EXTRACTION HELPERS
// ================================================================================================

// ExtractString extracts a string parameter by name, with optional default value
func ExtractString(params []DecoratorParam, name string, defaultVal string) (string, error) {
	for _, param := range params {
		if param.Name == name {
			if str, ok := param.Value.(string); ok {
				return str, nil
			}
			return fmt.Sprintf("%v", param.Value), nil // Convert to string
		}
	}
	return defaultVal, nil
}

// ExtractInt extracts an integer parameter by name, with optional default value
func ExtractInt(params []DecoratorParam, name string, defaultVal int) (int, error) {
	for _, param := range params {
		if param.Name == name {
			switch v := param.Value.(type) {
			case int:
				return v, nil
			case int64:
				return int(v), nil
			case float64:
				return int(v), nil
			case string:
				if val, err := strconv.Atoi(v); err == nil {
					return val, nil
				}
				return 0, fmt.Errorf("parameter '%s': cannot convert '%s' to integer", name, v)
			default:
				return 0, fmt.Errorf("parameter '%s': expected integer, got %T", name, v)
			}
		}
	}
	return defaultVal, nil
}

// ExtractBool extracts a boolean parameter by name, with optional default value
func ExtractBool(params []DecoratorParam, name string, defaultVal bool) (bool, error) {
	for _, param := range params {
		if param.Name == name {
			switch v := param.Value.(type) {
			case bool:
				return v, nil
			case string:
				if val, err := strconv.ParseBool(v); err == nil {
					return val, nil
				}
				return false, fmt.Errorf("parameter '%s': cannot convert '%s' to boolean", name, v)
			default:
				return false, fmt.Errorf("parameter '%s': expected boolean, got %T", name, v)
			}
		}
	}
	return defaultVal, nil
}

// ExtractDuration extracts a duration parameter by name, with optional default value
func ExtractDuration(params []DecoratorParam, name string, defaultVal time.Duration) (time.Duration, error) {
	for _, param := range params {
		if param.Name == name {
			switch v := param.Value.(type) {
			case time.Duration:
				return v, nil
			case string:
				if val, err := time.ParseDuration(v); err == nil {
					return val, nil
				}
				return 0, fmt.Errorf("parameter '%s': cannot parse '%s' as duration", name, v)
			case int64:
				return time.Duration(v), nil // Assume nanoseconds
			case float64:
				return time.Duration(v), nil // Assume nanoseconds
			default:
				return 0, fmt.Errorf("parameter '%s': expected duration, got %T", name, v)
			}
		}
	}
	return defaultVal, nil
}

// ExtractFloat extracts a float64 parameter by name, with optional default value
func ExtractFloat(params []DecoratorParam, name string, defaultVal float64) (float64, error) {
	for _, param := range params {
		if param.Name == name {
			switch v := param.Value.(type) {
			case float64:
				return v, nil
			case float32:
				return float64(v), nil
			case int:
				return float64(v), nil
			case int64:
				return float64(v), nil
			case string:
				if val, err := strconv.ParseFloat(v, 64); err == nil {
					return val, nil
				}
				return 0, fmt.Errorf("parameter '%s': cannot convert '%s' to float", name, v)
			default:
				return 0, fmt.Errorf("parameter '%s': expected float, got %T", name, v)
			}
		}
	}
	return defaultVal, nil
}

// ExtractPositional extracts a positional parameter by index (0-based)
func ExtractPositional(params []DecoratorParam, index int) (interface{}, error) {
	positionalCount := 0
	for _, param := range params {
		if param.Name == "" { // Positional parameter
			if positionalCount == index {
				return param.Value, nil
			}
			positionalCount++
		}
	}
	return nil, fmt.Errorf("positional parameter at index %d not found", index)
}

// ExtractPositionalString extracts a positional string parameter by index
func ExtractPositionalString(params []DecoratorParam, index int, defaultVal string) (string, error) {
	value, err := ExtractPositional(params, index)
	if err != nil {
		return defaultVal, nil // Return default if not found
	}

	if str, ok := value.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", value), nil // Convert to string
}

// HasParam checks if a parameter with the given name exists
func HasParam(params []DecoratorParam, name string) bool {
	for _, param := range params {
		if param.Name == name {
			return true
		}
	}
	return false
}

// ParamCount returns the total number of parameters
func ParamCount(params []DecoratorParam) int {
	return len(params)
}

// PositionalCount returns the number of positional parameters
func PositionalCount(params []DecoratorParam) int {
	count := 0
	for _, param := range params {
		if param.Name == "" {
			count++
		}
	}
	return count
}

// NamedCount returns the number of named parameters
func NamedCount(params []DecoratorParam) int {
	count := 0
	for _, param := range params {
		if param.Name != "" {
			count++
		}
	}
	return count
}
