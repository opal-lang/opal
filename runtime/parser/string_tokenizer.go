package parser

import "github.com/aledsdavies/opal/core/decorator"

// StringPart represents a part of an interpolated string using byte offsets (zero allocation)
type StringPart struct {
	Start         int  // Byte offset in content (start of this part)
	End           int  // Byte offset in content (end of this part)
	IsLiteral     bool // true = literal text, false = decorator
	PropertyStart int  // For @var.name, byte offset of "name" (or -1 if no property)
	PropertyEnd   int  // End of property name
}

// TokenizeString splits string content into literal and decorator parts
// content should be WITHOUT quotes (the string between the quote characters)
// quoteType is '"', '\”, or '`'
// Returns slice of StringParts with byte offsets into content (zero allocation for content)
func TokenizeString(content []byte, quoteType byte) []StringPart {
	// Single quotes have no interpolation
	if quoteType == '\'' {
		if len(content) == 0 {
			return nil
		}
		return []StringPart{{Start: 0, End: len(content), IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}}
	}

	// Fast path: no @ symbols means no interpolation
	hasAt := false
	for i := 0; i < len(content); i++ {
		if content[i] == '@' {
			hasAt = true
			break
		}
	}
	if !hasAt {
		if len(content) == 0 {
			return nil
		}
		return []StringPart{{Start: 0, End: len(content), IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}}
	}

	// First pass: count parts to pre-allocate exact size
	partCount := countStringParts(content)

	// Second pass: fill parts with byte offsets
	parts := make([]StringPart, 0, partCount)
	pos := 0
	literalStart := 0 // Track start of current literal section

	for pos < len(content) {
		// Find next @ symbol
		atPos := findNextAt(content, pos)

		// No more @ symbols, rest is literal
		if atPos == -1 {
			break
		}

		// Try to parse decorator after @
		decoratorStart := atPos + 1
		if decoratorStart >= len(content) {
			// @ at end of string, treat as literal (will be added after loop)
			break
		}

		// Read decorator name (identifier)
		decoratorEnd := readIdentifier(content, decoratorStart)

		if decoratorEnd == decoratorStart {
			// No identifier after @, treat @ as literal (continue literal section)
			pos = decoratorStart
			continue
		}

		// Check if it's a registered value decorator (need to convert to string for registry lookup)
		decoratorName := string(content[decoratorStart:decoratorEnd])

		// Use new decorator registry to check if this is a value decorator
		isValueDecorator := false
		if entry, ok := decorator.Global().Lookup(decoratorName); ok {
			// Check if decorator has RoleProvider (value decorator)
			for _, role := range entry.Roles {
				if role == decorator.RoleProvider {
					isValueDecorator = true
					break
				}
			}
		}

		if !isValueDecorator {
			// Not a value decorator (either unregistered or execution decorator)
			// Treat as literal including any .property that follows
			// Skip over .property if present (even though it's not a decorator)
			if decoratorEnd < len(content) && content[decoratorEnd] == '.' {
				propStart := decoratorEnd + 1
				propEnd := readIdentifier(content, propStart)
				if propEnd > propStart {
					pos = propEnd
				} else {
					pos = decoratorEnd
				}
			} else {
				pos = decoratorEnd
			}
			// Continue literal section (don't add part yet)
			continue
		}

		// It's a registered value decorator - finalize any preceding literal
		if atPos > literalStart {
			parts = append(parts, StringPart{
				Start:         literalStart,
				End:           atPos,
				IsLiteral:     true,
				PropertyStart: -1,
				PropertyEnd:   -1,
			})
		}

		// Check for property access: @var.name
		propStart := -1
		propEnd := -1
		if decoratorEnd < len(content) && content[decoratorEnd] == '.' {
			// Read property name after dot
			propStart = decoratorEnd + 1
			propEnd = readIdentifier(content, propStart)
			if propEnd > propStart {
				pos = propEnd
			} else {
				pos = decoratorEnd
				propStart = -1
				propEnd = -1
			}
		} else {
			pos = decoratorEnd
		}

		parts = append(parts, StringPart{
			Start:         decoratorStart,
			End:           decoratorEnd,
			IsLiteral:     false,
			PropertyStart: propStart,
			PropertyEnd:   propEnd,
		})

		// Next literal section starts after this decorator
		literalStart = pos
	}

	// Add any remaining literal content
	if literalStart < len(content) {
		parts = append(parts, StringPart{
			Start:         literalStart,
			End:           len(content),
			IsLiteral:     true,
			PropertyStart: -1,
			PropertyEnd:   -1,
		})
	}

	return parts
}

// countStringParts counts how many parts the string will have (first pass)
func countStringParts(content []byte) int {
	count := 0
	pos := 0

	for pos < len(content) {
		atPos := findNextAt(content, pos)
		if atPos == -1 {
			if pos < len(content) {
				count++ // Final literal part
			}
			break
		}

		if atPos > pos {
			count++ // Literal before @
		}

		decoratorStart := atPos + 1
		if decoratorStart >= len(content) {
			count++ // @ at end
			break
		}

		decoratorEnd := readIdentifier(content, decoratorStart)
		if decoratorEnd == decoratorStart {
			count++ // @ with no identifier
			pos = decoratorStart
			continue
		}

		// Check for property access (consume it whether registered or not)
		if decoratorEnd < len(content) && content[decoratorEnd] == '.' {
			propStart := decoratorEnd + 1
			propEnd := readIdentifier(content, propStart)
			if propEnd > propStart {
				pos = propEnd
			} else {
				pos = decoratorEnd
			}
		} else {
			pos = decoratorEnd
		}

		count++ // Decorator or literal part (both count as 1)
	}

	return count
}

// findNextAt finds the next @ symbol starting from pos, returns -1 if not found
func findNextAt(content []byte, pos int) int {
	for i := pos; i < len(content); i++ {
		if content[i] == '@' {
			return i
		}
	}
	return -1
}

// readIdentifier reads an identifier starting at pos, returns end position
func readIdentifier(content []byte, pos int) int {
	end := pos
	for end < len(content) && isIdentifierChar(content[end]) {
		end++
	}
	return end
}

// isIdentifierChar checks if a character can be part of an identifier
func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}
