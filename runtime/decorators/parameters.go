package decorators

import (
	"fmt"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ResolvePositionalParameters converts positional parameters to named parameters
// based on their position in the parameter schema. This implements Kotlin-style
// parameter handling where positional parameters are mapped to schema parameters
// by position, and named parameters are preserved as-is.
//
// Rules:
// 1. Positional parameters (Name == "") are mapped to schema parameters by position
// 2. Named parameters are preserved as-is
// 3. Positional parameters must come before named parameters (Kotlin rule)
// 4. Cannot have more positional parameters than schema parameters
func ResolvePositionalParameters(params []ast.NamedParameter, schema []ParameterSchema) ([]ast.NamedParameter, error) {
	if len(params) == 0 {
		return []ast.NamedParameter{}, nil
	}

	resolved := make([]ast.NamedParameter, len(params))
	positionalCount := 0
	foundNamed := false

	// First pass: identify positional vs named parameters and validate order
	for i, param := range params {
		isPositional := param.Name == "" && param.NameToken == nil

		if isPositional {
			if foundNamed {
				return nil, fmt.Errorf("positional parameters must come before named parameters")
			}
			positionalCount++
		} else {
			foundNamed = true
		}

		// Copy the parameter for now
		resolved[i] = param
	}

	// Validate we don't have too many positional parameters
	if positionalCount > len(schema) {
		return nil, fmt.Errorf("too many positional parameters: got %d, schema has %d parameters", positionalCount, len(schema))
	}

	// Second pass: resolve positional parameters to named ones
	schemaIndex := 0
	for i, param := range resolved {
		isPositional := param.Name == "" && param.NameToken == nil

		if isPositional {
			if schemaIndex >= len(schema) {
				return nil, fmt.Errorf("internal error: positional parameter index out of bounds")
			}

			// Map this positional parameter to the corresponding schema parameter
			resolved[i].Name = schema[schemaIndex].Name
			schemaIndex++
		}
	}

	return resolved, nil
}
