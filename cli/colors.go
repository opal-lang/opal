package main

import (
	"os"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
)

// Colorize wraps text in ANSI color codes if color is enabled
func Colorize(text, color string, useColor bool) string {
	if !useColor {
		return text
	}
	return color + text + ColorReset
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
