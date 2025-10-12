package parser

import "time"

// ParserOpt represents a parser configuration option
type ParserOpt func(*ParserConfig)

// TelemetryMode controls telemetry collection (production-safe)
type TelemetryMode int

const (
	TelemetryOff    TelemetryMode = iota // Zero overhead (default)
	TelemetryBasic                       // Parse counts only
	TelemetryTiming                      // Parse counts + timing per phase
)

// DebugLevel controls debug tracing (development only)
type DebugLevel int

const (
	DebugOff      DebugLevel = iota // No debug info (default)
	DebugPaths                      // Method call tracing
	DebugDetailed                   // Event-level tracing
)

// ParserConfig holds parser configuration
type ParserConfig struct {
	telemetry TelemetryMode
	debug     DebugLevel
}

// WithTelemetryBasic enables basic telemetry (parse counts only)
func WithTelemetryBasic() ParserOpt {
	return func(c *ParserConfig) {
		c.telemetry = TelemetryBasic
	}
}

// WithTelemetryTiming enables timing telemetry (counts + timing per phase)
func WithTelemetryTiming() ParserOpt {
	return func(c *ParserConfig) {
		c.telemetry = TelemetryTiming
	}
}

// WithDebugPaths enables debug path tracing (development only)
func WithDebugPaths() ParserOpt {
	return func(c *ParserConfig) {
		c.debug = DebugPaths
	}
}

// WithDebugDetailed enables detailed debug tracing (development only)
func WithDebugDetailed() ParserOpt {
	return func(c *ParserConfig) {
		c.debug = DebugDetailed
	}
}

// ParseTelemetry holds parser performance metrics (production-safe)
type ParseTelemetry struct {
	LexTime    time.Duration // Time spent lexing
	ParseTime  time.Duration // Time spent parsing
	TotalTime  time.Duration // Total parse time
	TokenCount int           // Number of tokens
	EventCount int           // Number of events
	ErrorCount int           // Number of parse errors
}

// DebugEvent holds debug tracing information (development only)
type DebugEvent struct {
	Timestamp time.Time
	Event     string // "enter_fnDecl", "exit_fnDecl", etc.
	TokenPos  int    // Current token position
	Context   string // Additional context
}
