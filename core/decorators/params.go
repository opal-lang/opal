package decorators

import (
	"fmt"
	"strconv"
	"time"
)

// ================================================================================================
// DECORATOR PARAMETER TYPES - Self-contained parameter abstraction for decorators
// ================================================================================================

// Param is the concrete implementation of Param interface
type Param struct {
	ParamName  string      `json:"name"`  // Parameter name (empty for positional)
	ParamValue interface{} `json:"value"` // Parameter value
}

// GetName returns the parameter name (empty for positional parameters)
func (p Param) GetName() string {
	return p.ParamName
}

// GetValue returns the raw parameter value
func (p Param) GetValue() any {
	return p.ParamValue
}

// AsString converts the parameter value to a string
func (p Param) AsString() string {
	if str, ok := p.ParamValue.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", p.ParamValue)
}

// AsBool converts the parameter value to a boolean
func (p Param) AsBool() bool {
	switch v := p.ParamValue.(type) {
	case bool:
		return v
	case string:
		val, _ := strconv.ParseBool(v)
		return val
	}
	return false
}

// AsInt converts the parameter value to an integer
func (p Param) AsInt() int {
	switch v := p.ParamValue.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		val, _ := strconv.Atoi(v)
		return val
	}
	return 0
}

// AsFloat converts the parameter value to a float64
func (p Param) AsFloat() float64 {
	switch v := p.ParamValue.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		val, _ := strconv.ParseFloat(v, 64)
		return val
	}
	return 0.0
}

// NewParam creates a new parameter with the given name and value
func NewParam(name string, value any) Param {
	return Param{ParamName: name, ParamValue: value}
}

// NewPositionalParam creates a new positional parameter with the given value
func NewPositionalParam(value any) Param {
	return Param{ParamName: "", ParamValue: value}
}

// ================================================================================================
// PARAMETER EXTRACTION HELPERS
// ================================================================================================

// ExtractString extracts a string parameter by name, with optional default value
func ExtractString(params []Param, name string, defaultVal string) (string, error) {
	for _, param := range params {
		if param.GetName() == name {
			return param.AsString(), nil
		}
	}
	return defaultVal, nil
}

// ExtractInt extracts an integer parameter by name, with optional default value
func ExtractInt(params []Param, name string, defaultVal int) (int, error) {
	for _, param := range params {
		if param.GetName() == name {
			return param.AsInt(), nil
		}
	}
	return defaultVal, nil
}

// ExtractBool extracts a boolean parameter by name, with optional default value
func ExtractBool(params []Param, name string, defaultVal bool) (bool, error) {
	for _, param := range params {
		if param.GetName() == name {
			switch v := param.GetValue().(type) {
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
func ExtractDuration(params []Param, name string, defaultVal time.Duration) (time.Duration, error) {
	for _, param := range params {
		if param.GetName() == name {
			switch v := param.GetValue().(type) {
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
func ExtractFloat(params []Param, name string, defaultVal float64) (float64, error) {
	for _, param := range params {
		if param.GetName() == name {
			switch v := param.GetValue().(type) {
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
func ExtractPositional(params []Param, index int) (interface{}, error) {
	positionalCount := 0
	for _, param := range params {
		if param.GetName() == "" { // Positional parameter
			if positionalCount == index {
				return param.GetValue(), nil
			}
			positionalCount++
		}
	}
	return nil, fmt.Errorf("positional parameter at index %d not found", index)
}

// ExtractPositionalString extracts a positional string parameter by index
func ExtractPositionalString(params []Param, index int, defaultVal string) (string, error) {
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
func HasParam(params []Param, name string) bool {
	for _, param := range params {
		if param.GetName() == name {
			return true
		}
	}
	return false
}

// ParamCount returns the total number of parameters
func ParamCount(params []Param) int {
	return len(params)
}

// PositionalCount returns the number of positional parameters
func PositionalCount(params []Param) int {
	count := 0
	for _, param := range params {
		if param.GetName() == "" {
			count++
		}
	}
	return count
}

// NamedCount returns the number of named parameters
func NamedCount(params []Param) int {
	count := 0
	for _, param := range params {
		if param.GetName() != "" {
			count++
		}
	}
	return count
}
