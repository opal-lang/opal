package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aledsdavies/devcmd/pkg/generator"
	"github.com/aledsdavies/devcmd/pkg/parser"
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
	flag.StringVar(&templateFile, "template", "", "Custom template file for shell function generation")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <commands-file>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(ExitInvalidArguments)
	}

	inputFile := flag.Arg(0)

	// Read command file
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(ExitIOError)
	}

	// Parse the command definitions
	commandFile, err := parser.Parse(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing commands: %v\n", err)
		os.Exit(ExitParseError)
	}

	// Expand variable references
	commandFile.ExpandVariables()

	// Generate shell script
	var shellScript string
	if templateFile != "" {
		templateContent, err := ioutil.ReadFile(templateFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading template file: %v\n", err)
			os.Exit(ExitIOError)
		}

		shellScript, err = generator.GenerateWithTemplate(commandFile, string(templateContent))
	} else {
		shellScript, err = generator.Generate(commandFile)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating shell script: %v\n", err)
		os.Exit(ExitGenerationError)
	}

	// Output the result
	fmt.Print(shellScript)
	os.Exit(ExitSuccess)
}
