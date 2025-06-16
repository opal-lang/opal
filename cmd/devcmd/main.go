package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aledsdavies/devcmd/pkgs/generator"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

func main() {
	// Exit code constants
	const (
		ExitSuccess          = 0
		ExitInvalidArguments = 1
		ExitIOError          = 2
		ExitParseError       = 3
		ExitGenerationError  = 4
	)

	// Command line flags
	var templateFile string
	var outputFormat string
	var debug bool
	flag.StringVar(&templateFile, "template", "", "Custom template file for generation")
	flag.StringVar(&outputFormat, "format", "go", "Output format: 'go'")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <commands-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  -format string\n")
		fmt.Fprintf(os.Stderr, "        Output format: 'go' (default \"go\")\n")
		fmt.Fprintf(os.Stderr, "  -template string\n")
		fmt.Fprintf(os.Stderr, "        Custom template file for generation\n")
		fmt.Fprintf(os.Stderr, "  -debug\n")
		fmt.Fprintf(os.Stderr, "        Enable debug output\n")
		fmt.Fprintf(os.Stderr, "\nFormats:\n")
		fmt.Fprintf(os.Stderr, "  go       Generate standalone Go CLI executable\n")
		os.Exit(ExitInvalidArguments)
	}

	inputFile := flag.Arg(0)

	// Validate output format
	if outputFormat != "go" {
		fmt.Fprintf(os.Stderr, "Error: unsupported format '%s'. Use 'go'\n", outputFormat)
		os.Exit(ExitInvalidArguments)
	}

	// Read command file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(ExitIOError)
	}

	// Parse the command definitions with debug flag
	commandFile, err := parser.Parse(string(content), debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing commands: %v\n", err)
		os.Exit(ExitParseError)
	}

	// Expand variable references
	err = commandFile.ExpandVariables()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expanding variables: %v\n", err)
		os.Exit(ExitParseError)
	}

	// Generate output based on format
	var output string
	switch outputFormat {
	case "go":
		output, err = generateGo(commandFile, templateFile)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported format '%s'\n", outputFormat)
		os.Exit(ExitInvalidArguments)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating %s output: %v\n", outputFormat, err)
		os.Exit(ExitGenerationError)
	}

	// Output the result
	fmt.Print(output)
	os.Exit(ExitSuccess)
}

// generateGo generates Go CLI output
func generateGo(commandFile *parser.CommandFile, templateFile string) (string, error) {
	if templateFile != "" {
		templateContent, err := os.ReadFile(templateFile)
		if err != nil {
			return "", fmt.Errorf("error reading template file: %v", err)
		}
		return generator.GenerateGoWithTemplate(commandFile, string(templateContent))
	}
	return generator.GenerateGo(commandFile)
}
