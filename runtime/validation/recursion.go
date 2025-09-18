package validation

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/core/ast"
)

// RecursionError represents a recursion detection error with cycle information
type RecursionError struct {
	Command string   // The command where recursion was detected
	Cycle   []string // The cycle path (e.g., ["build", "test", "build"])
	Message string
}

func (e *RecursionError) Error() string {
	return e.Message
}

// ValidateNoRecursion validates that no commands have recursive @cmd() references
// This prevents infinite loops at both plan generation and execution time
func ValidateNoRecursion(program *ast.Program) error {
	// Build a map of command name -> command declaration for quick lookup
	commands := make(map[string]*ast.CommandDecl)
	for i := range program.Commands {
		commands[program.Commands[i].Name] = &program.Commands[i]
	}

	// Check each command for recursive references
	for _, command := range program.Commands {
		if err := detectRecursion(command.Name, commands, []string{}, make(map[string]bool)); err != nil {
			return err
		}
	}

	return nil
}

// detectRecursion performs depth-first search to detect cycles in @cmd() references
func detectRecursion(commandName string, commands map[string]*ast.CommandDecl, path []string, visiting map[string]bool) error {
	// Check if we're currently visiting this command (back edge = cycle)
	if visiting[commandName] {
		// Found a cycle - build the cycle path
		cycleStart := -1
		for i, cmd := range path {
			if cmd == commandName {
				cycleStart = i
				break
			}
		}

		var cycle []string
		if cycleStart >= 0 {
			cycle = append(path[cycleStart:], commandName)
		} else {
			// Shouldn't happen, but fallback
			cycle = append(path, commandName)
		}

		return &RecursionError{
			Command: commandName,
			Cycle:   cycle,
			Message: fmt.Sprintf("Recursive command reference detected: %s",
				strings.Join(cycle, " -> ")),
		}
	}

	// Get the command declaration
	command, exists := commands[commandName]
	if !exists {
		// Command not found - this will be caught elsewhere as a different error
		return nil
	}

	// Mark this command as being visited
	visiting[commandName] = true
	newPath := append(path, commandName)

	// Find all @cmd() references in this command
	cmdRefs := findCmdReferences(command)

	// Recursively check each referenced command
	for _, refName := range cmdRefs {
		if err := detectRecursion(refName, commands, newPath, visiting); err != nil {
			return err
		}
	}

	// Unmark this command (backtrack)
	delete(visiting, commandName)

	return nil
}

// findCmdReferences finds all @cmd(name) references in a command declaration
func findCmdReferences(command *ast.CommandDecl) []string {
	var refs []string

	// Walk through the command body and find @cmd decorators
	for _, content := range command.Body.Content {
		refs = append(refs, findCmdReferencesInContent(content)...)
	}

	return refs
}

// findCmdReferencesInContent recursively finds @cmd references in command content
func findCmdReferencesInContent(content ast.CommandContent) []string {
	var refs []string

	switch c := content.(type) {
	case *ast.ShellContent:
		// Shell content can contain @cmd decorators in mixed content
		for _, part := range c.Parts {
			if decorator, ok := part.(*ast.ActionDecorator); ok {
				if decorator.Name == "cmd" {
					// Extract the command name from the first argument
					if len(decorator.Args) > 0 {
						if cmdName := extractStringArg(decorator.Args[0]); cmdName != "" {
							refs = append(refs, cmdName)
						}
					}
				}
			}
		}

	case *ast.ShellChain:
		// Shell chains contain multiple elements that might have decorators
		for _, element := range c.Elements {
			if element.Content != nil {
				refs = append(refs, findCmdReferencesInShellContent(element.Content)...)
			}
		}

	case *ast.ActionDecorator:
		// Direct @cmd decorator
		if c.Name == "cmd" && len(c.Args) > 0 {
			if cmdName := extractStringArg(c.Args[0]); cmdName != "" {
				refs = append(refs, cmdName)
			}
		}

	case *ast.BlockDecorator:
		// Block decorators can contain inner commands
		for _, innerContent := range c.Content {
			refs = append(refs, findCmdReferencesInContent(innerContent)...)
		}

	case *ast.PatternDecorator:
		// Pattern decorators can have branches with commands
		for _, pattern := range c.Patterns {
			for _, branchContent := range pattern.Commands {
				refs = append(refs, findCmdReferencesInContent(branchContent)...)
			}
		}
	}

	return refs
}

// findCmdReferencesInShellContent finds @cmd references in shell content parts
func findCmdReferencesInShellContent(shellContent *ast.ShellContent) []string {
	var refs []string

	for _, part := range shellContent.Parts {
		if decorator, ok := part.(*ast.ActionDecorator); ok {
			if decorator.Name == "cmd" {
				// Extract the command name from the first argument
				if len(decorator.Args) > 0 {
					if cmdName := extractStringArg(decorator.Args[0]); cmdName != "" {
						refs = append(refs, cmdName)
					}
				}
			}
		}
	}

	return refs
}

// extractStringArg extracts a string value from a decorator argument using safe AST methods
func extractStringArg(arg ast.NamedParameter) string {
	if arg.Value != nil {
		switch val := arg.Value.(type) {
		case *ast.StringLiteral:
			// Use StringLiteral.String() which properly handles .Parts[] system
			return val.String()
		case *ast.Identifier:
			// Handle unquoted identifiers like @cmd(build)
			return val.Name
		}
	}
	return ""
}
