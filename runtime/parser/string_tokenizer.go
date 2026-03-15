package parser

// StringPart represents a part of an interpolated string using byte offsets (zero allocation)
type StringPart struct {
	Start         int  // Byte offset in content (start of this part)
	End           int  // Byte offset in content (end of this part)
	IsLiteral     bool // true = literal text, false = decorator
	PropertyStart int  // For @var.name, byte offset of "name" (or -1 if no property)
	PropertyEnd   int  // End of property name
}

type parsedStringDecorator struct {
	start         int
	end           int
	propertyStart int
	propertyEnd   int
	nextPos       int
	isValue       bool
}

// TokenizeString splits string content into literal and decorator parts.
// content should be WITHOUT quotes (the string between the quote characters).
// quoteType is '"', '\”, or '`'.
// Returns slice of StringParts with byte offsets into content (zero allocation for content).
func TokenizeString(content []byte, quoteType byte) []StringPart {
	if quoteType == '\'' {
		if len(content) == 0 {
			return nil
		}
		return []StringPart{{Start: 0, End: len(content), IsLiteral: true, PropertyStart: -1, PropertyEnd: -1}}
	}

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

	partCount := countStringParts(content)
	parts := make([]StringPart, 0, partCount)
	pos := 0
	literalStart := 0

	for pos < len(content) {
		atPos := findNextAt(content, pos)
		if atPos == -1 {
			break
		}

		parsed, ok := parseStringDecorator(content, atPos)
		if !ok {
			pos = atPos + 1
			continue
		}

		if !parsed.isValue {
			pos = parsed.nextPos
			continue
		}

		if atPos > literalStart {
			parts = append(parts, StringPart{
				Start:         literalStart,
				End:           atPos,
				IsLiteral:     true,
				PropertyStart: -1,
				PropertyEnd:   -1,
			})
		}

		parts = append(parts, StringPart{
			Start:         parsed.start,
			End:           parsed.end,
			IsLiteral:     false,
			PropertyStart: parsed.propertyStart,
			PropertyEnd:   parsed.propertyEnd,
		})

		pos = parsed.nextPos
		literalStart = pos
	}

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
				count++
			}
			break
		}

		if atPos > pos {
			count++
		}

		parsed, ok := parseStringDecorator(content, atPos)
		if !ok {
			count++
			pos = atPos + 1
			continue
		}

		count++
		pos = parsed.nextPos
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

func hasMalformedBracedInterpolation(content []byte) bool {
	for pos := 0; pos < len(content)-1; pos++ {
		if content[pos] != '@' || content[pos+1] != '{' {
			continue
		}
		if _, ok := parseStringDecorator(content, pos); !ok {
			return true
		}
	}
	return false
}

func parseStringDecorator(content []byte, atPos int) (parsedStringDecorator, bool) {
	decoratorStart := atPos + 1
	if decoratorStart >= len(content) {
		return parsedStringDecorator{}, false
	}

	braced := false
	if content[decoratorStart] == '{' {
		braced = true
		decoratorStart++
		if decoratorStart >= len(content) {
			return parsedStringDecorator{}, false
		}
	}

	segmentStarts := []int{}
	segmentEnds := []int{}
	segmentStart := decoratorStart
	segmentEnd := readIdentifier(content, segmentStart)
	if segmentEnd == segmentStart {
		return parsedStringDecorator{}, false
	}
	segmentStarts = append(segmentStarts, segmentStart)
	segmentEnds = append(segmentEnds, segmentEnd)
	nextPos := segmentEnd

	for nextPos < len(content) && content[nextPos] == '.' {
		segmentStart = nextPos + 1
		segmentEnd = readIdentifier(content, segmentStart)
		if segmentEnd == segmentStart {
			break
		}
		segmentStarts = append(segmentStarts, segmentStart)
		segmentEnds = append(segmentEnds, segmentEnd)
		nextPos = segmentEnd
	}

	pathEnd := segmentEnds[0]
	longestValueSegments := 0
	for i := range segmentEnds {
		candidateEnd := segmentEnds[i]
		if isValueDecorator(content[decoratorStart:candidateEnd]) {
			pathEnd = candidateEnd
			longestValueSegments = i + 1
		}
	}

	propertyStart := -1
	propertyEnd := -1
	if longestValueSegments > 0 && longestValueSegments < len(segmentStarts) {
		propertyStart = segmentStarts[longestValueSegments]
		propertyEnd = segmentEnds[longestValueSegments]
		if braced {
			propertyEnd = segmentEnds[len(segmentEnds)-1]
		}
		nextPos = propertyEnd
	}

	if braced {
		if nextPos >= len(content) || content[nextPos] != '}' {
			return parsedStringDecorator{}, false
		}
		nextPos++
	}

	return parsedStringDecorator{
		start:         decoratorStart,
		end:           pathEnd,
		propertyStart: propertyStart,
		propertyEnd:   propertyEnd,
		nextPos:       nextPos,
		isValue:       longestValueSegments > 0,
	}, true
}

func isValueDecorator(name []byte) bool {
	return isPluginValueDecorator(string(name))
}
