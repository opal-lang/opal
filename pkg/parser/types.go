package parser

// Definition represents a variable definition in the command file
type Definition struct {
	Name  string
	Value string
	Line  int
}

// Command represents a command definition in the command file
type Command struct {
	Name    string
	Command string
	Line    int
}

// CommandFile represents the parsed command file
type CommandFile struct {
	Definitions []Definition
	Commands    []Command
	Lines       []string // Original file lines for error reporting
}

// ExpandVariables expands variable references in commands
func (cf *CommandFile) ExpandVariables() {
	// Create lookup map for variables
	vars := make(map[string]string)
	for _, def := range cf.Definitions {
		vars[def.Name] = def.Value
	}

	// Expand variables in commands
	for i := range cf.Commands {
		cmd := &cf.Commands[i]
		cmd.Command = expandVariables(cmd.Command, vars)
	}
}

// expandVariables replaces ${name} in a string with its value
func expandVariables(s string, vars map[string]string) string {
	var result []byte
	var varName []byte
	inVar := false

	for i := 0; i < len(s); i++ {
		if !inVar {
			// Look for variable start
			if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
				inVar = true
				varName = varName[:0] // Reset var name
				i++                   // Skip the '{'
			} else {
				result = append(result, s[i])
			}
		} else {
			// In a variable reference
			if s[i] == '}' {
				// End of variable reference
				name := string(varName)
				if value, ok := vars[name]; ok {
					result = append(result, value...)
				} else {
					// Keep the original reference if variable not found
					result = append(result, '$', '{')
					result = append(result, varName...)
					result = append(result, '}')
				}
				inVar = false
			} else {
				varName = append(varName, s[i])
			}
		}
	}

	return string(result)
}
