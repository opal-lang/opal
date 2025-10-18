package main

import (
	"os"

	"github.com/aledsdavies/opal/core/planfmt/formatter"
)

// Re-export color constants from formatter package for convenience
const (
	ColorReset  = formatter.ColorReset
	ColorRed    = formatter.ColorRed
	ColorGreen  = formatter.ColorGreen
	ColorYellow = formatter.ColorYellow
	ColorBlue   = formatter.ColorBlue
	ColorCyan   = formatter.ColorCyan
	ColorGray   = formatter.ColorGray
)

// Colorize wraps text in ANSI color codes if color is enabled
// This is a convenience wrapper around formatter.Colorize
func Colorize(text, color string, useColor bool) string {
	return formatter.Colorize(text, color, useColor)
}

// ShouldUseColor determines if color output should be used
// Respects --no-color flag and NO_COLOR environment variable
func ShouldUseColor(noColorFlag bool) bool {
	if noColorFlag {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Check if stdout is a terminal
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
