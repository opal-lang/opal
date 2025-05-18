package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	defRegex = regexp.MustCompile(`^def\s+([A-Za-z0-9_]+)\s*=\s*(.*)$`)
	cmdRegex = regexp.MustCompile(`^([A-Za-z0-9_-]+):\s*(.*)$`)
	comRegex = regexp.MustCompile(`^#.*$`)
)

// Parse parses a command file content into a CommandFile structure
func Parse(content string) (*CommandFile, error) {
	lines := strings.Split(content, "\n")
	result := &CommandFile{
		Lines:       lines,
		Definitions: []Definition{},
		Commands:    []Command{},
	}

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || comRegex.MatchString(line) {
			// Skip empty lines and comments
			continue
		}

		// Try to match definition
		if matches := defRegex.FindStringSubmatch(line); matches != nil {
			result.Definitions = append(result.Definitions, Definition{
				Name:  matches[1],
				Value: matches[2],
				Line:  lineNum + 1,
			})
			continue
		}

		// Try to match command
		if matches := cmdRegex.FindStringSubmatch(line); matches != nil {
			result.Commands = append(result.Commands, Command{
				Name:    matches[1],
				Command: matches[2],
				Line:    lineNum + 1,
			})
			continue
		}

		// No pattern matched - error
		return nil, fmt.Errorf("line %d: invalid syntax: %s", lineNum+1, line)
	}

	// Verify no duplicate definitions
	defs := make(map[string]int)
	for _, def := range result.Definitions {
		if line, exists := defs[def.Name]; exists {
			return nil, fmt.Errorf("duplicate definition of '%s' at lines %d and %d",
				def.Name, line, def.Line)
		}
		defs[def.Name] = def.Line
	}

	// Verify no duplicate commands
	cmds := make(map[string]int)
	for _, cmd := range result.Commands {
		if line, exists := cmds[cmd.Name]; exists {
			return nil, fmt.Errorf("duplicate command '%s' at lines %d and %d",
				cmd.Name, line, cmd.Line)
		}
		cmds[cmd.Name] = cmd.Line
	}

	return result, nil
}
