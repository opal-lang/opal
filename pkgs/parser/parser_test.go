package parser

import (
	"fmt"
	"strings"
	"testing"
)

// Test helper types for cleaner test definitions
type ExpectedCommand struct {
	Name     string
	IsWatch  bool
	IsStop   bool
	IsBlock  bool
	Command  string
	Elements []ExpectedElement
	Block    []ExpectedBlockStatement
}

type ExpectedBlockStatement struct {
	IsDecorated   bool
	Decorator     string
	DecoratorType string
	Command       string
	Elements      []ExpectedElement
}

type ExpectedElement struct {
	Type string // "text" or "decorator"
	Text string // for text elements

	// For decorator elements
	DecoratorName string
	DecoratorType string // "function", "simple", "block"
	Args          []ExpectedElement
}

type ExpectedDefinition struct {
	Name  string
	Value string
}

type TestCase struct {
	Name        string
	Input       string
	WantErr     bool
	ErrorSubstr string
	Expected    struct {
		Definitions []ExpectedDefinition
		Commands    []ExpectedCommand
	}
}

// Helper functions for creating expected elements
func Text(text string) ExpectedElement {
	return ExpectedElement{
		Type: "text",
		Text: text,
	}
}

func Var(name string) ExpectedElement {
	return ExpectedElement{
		Type:          "decorator",
		DecoratorName: "var",
		DecoratorType: "function",
		Args:          []ExpectedElement{Text(name)},
	}
}

func Decorator(name, dtype string, args ...ExpectedElement) ExpectedElement {
	return ExpectedElement{
		Type:          "decorator",
		DecoratorName: name,
		DecoratorType: dtype,
		Args:          args,
	}
}

func SimpleCommand(name, command string, elements ...ExpectedElement) ExpectedCommand {
	return ExpectedCommand{
		Name:     name,
		Command:  command,
		Elements: elements,
	}
}

func WatchCommand(name, command string, elements ...ExpectedElement) ExpectedCommand {
	return ExpectedCommand{
		Name:     name,
		IsWatch:  true,
		Command:  command,
		Elements: elements,
	}
}

func StopCommand(name, command string, elements ...ExpectedElement) ExpectedCommand {
	return ExpectedCommand{
		Name:     name,
		IsStop:   true,
		Command:  command,
		Elements: elements,
	}
}

func BlockCommand(name string, statements ...ExpectedBlockStatement) ExpectedCommand {
	return ExpectedCommand{
		Name:    name,
		IsBlock: true,
		Block:   statements,
	}
}

func WatchBlockCommand(name string, statements ...ExpectedBlockStatement) ExpectedCommand {
	return ExpectedCommand{
		Name:    name,
		IsWatch: true,
		IsBlock: true,
		Block:   statements,
	}
}

func Statement(command string, elements ...ExpectedElement) ExpectedBlockStatement {
	return ExpectedBlockStatement{
		Command:  command,
		Elements: elements,
	}
}

func DecoratedStatement(decorator, decoratorType, command string, elements ...ExpectedElement) ExpectedBlockStatement {
	return ExpectedBlockStatement{
		IsDecorated:   true,
		Decorator:     decorator,
		DecoratorType: decoratorType,
		Command:       command,
		Elements:      elements,
	}
}

// For block decorators, we expect the elements to contain the decorator
func BlockDecoratedStatement(decorator, decoratorType, command string) ExpectedBlockStatement {
	return ExpectedBlockStatement{
		IsDecorated:   true,
		Decorator:     decorator,
		DecoratorType: decoratorType,
		Command:       command,
		Elements:      []ExpectedElement{Decorator(decorator, decoratorType)},
	}
}

func Def(name, value string) ExpectedDefinition {
	return ExpectedDefinition{Name: name, Value: value}
}

// Helper function to format elements for diff output
func formatElements(elements []CommandElement) string {
	var parts []string
	for i, elem := range elements {
		if elem.IsDecorator() {
			decorator := elem.(*DecoratorElement)
			if len(decorator.Args) == 0 {
				parts = append(parts, fmt.Sprintf("[%d] DECORATOR: @%s()", i, decorator.Name))
			} else {
				argStrs := make([]string, len(decorator.Args))
				for j, arg := range decorator.Args {
					argStrs[j] = arg.String()
				}
				parts = append(parts, fmt.Sprintf("[%d] DECORATOR: @%s(%s)", i, decorator.Name, strings.Join(argStrs, "")))
			}
		} else {
			parts = append(parts, fmt.Sprintf("[%d] TEXT: %q", i, elem.String()))
		}
	}
	return strings.Join(parts, "\n")
}

func formatExpectedElements(elements []ExpectedElement) string {
	var parts []string
	for i, elem := range elements {
		if elem.Type == "decorator" {
			if len(elem.Args) == 0 {
				parts = append(parts, fmt.Sprintf("[%d] DECORATOR: @%s()", i, elem.DecoratorName))
			} else {
				var argStrs []string
				for _, arg := range elem.Args {
					if arg.Type == "decorator" {
						argStrs = append(argStrs, fmt.Sprintf("@%s(%s)", arg.DecoratorName, arg.Text))
					} else {
						argStrs = append(argStrs, arg.Text)
					}
				}
				parts = append(parts, fmt.Sprintf("[%d] DECORATOR: @%s(%s)", i, elem.DecoratorName, strings.Join(argStrs, "")))
			}
		} else {
			parts = append(parts, fmt.Sprintf("[%d] TEXT: %q", i, elem.Text))
		}
	}
	return strings.Join(parts, "\n")
}

// Enhanced diff output for elements
func showElementsDiff(t *testing.T, actual []CommandElement, expected []ExpectedElement, path string) {
	if len(actual) != len(expected) {
		t.Errorf("%s: expected %d elements, got %d", path, len(expected), len(actual))
		t.Errorf("EXPECTED:\n%s", formatExpectedElements(expected))
		t.Errorf("ACTUAL:\n%s", formatElements(actual))
		return
	}

	// If counts match, check individual elements
	verifyElements(t, actual, expected, path)
}

// Verification functions
func verifyElements(t *testing.T, actual []CommandElement, expected []ExpectedElement, path string) {
	for i, expectedElem := range expected {
		if i >= len(actual) {
			t.Errorf("%s[%d]: missing element, expected %s", path, i, expectedElem.Type)
			continue
		}

		actualElem := actual[i]
		elemPath := fmt.Sprintf("%s[%d]", path, i)

		switch expectedElem.Type {
		case "text":
			if actualElem.IsDecorator() {
				t.Errorf("%s: expected text %q, got decorator %s", elemPath, expectedElem.Text, actualElem.String())
				continue
			}

			textElem, ok := actualElem.(*TextElement)
			if !ok {
				t.Errorf("%s: expected TextElement, got %T", elemPath, actualElem)
				continue
			}

			if textElem.Text != expectedElem.Text {
				t.Errorf("%s: expected text %q, got %q", elemPath, expectedElem.Text, textElem.Text)
			}

		case "decorator":
			if !actualElem.IsDecorator() {
				t.Errorf("%s: expected decorator %s, got text %q", elemPath, expectedElem.DecoratorName, actualElem.String())
				continue
			}

			decorator, ok := actualElem.(*DecoratorElement)
			if !ok {
				t.Errorf("%s: expected DecoratorElement, got %T", elemPath, actualElem)
				continue
			}

			if decorator.Name != expectedElem.DecoratorName {
				t.Errorf("%s: expected decorator name %q, got %q", elemPath, expectedElem.DecoratorName, decorator.Name)
			}

			if decorator.Type != expectedElem.DecoratorType {
				t.Errorf("%s: expected decorator type %q, got %q", elemPath, expectedElem.DecoratorType, decorator.Type)
			}

			// Recursively verify decorator arguments with diff
			if len(expectedElem.Args) > 0 || len(decorator.Args) > 0 {
				showElementsDiff(t, decorator.Args, expectedElem.Args, fmt.Sprintf("%s.Args", elemPath))
			}

		default:
			t.Errorf("%s: unknown expected element type %q", elemPath, expectedElem.Type)
		}
	}
}

func verifyCommand(t *testing.T, actual Command, expected ExpectedCommand, index int) {
	prefix := fmt.Sprintf("Command[%d]", index)

	if actual.Name != expected.Name {
		t.Errorf("%s: expected name %q, got %q", prefix, expected.Name, actual.Name)
	}

	if actual.IsWatch != expected.IsWatch {
		t.Errorf("%s: expected IsWatch %v, got %v", prefix, expected.IsWatch, actual.IsWatch)
	}

	if actual.IsStop != expected.IsStop {
		t.Errorf("%s: expected IsStop %v, got %v", prefix, expected.IsStop, actual.IsStop)
	}

	if actual.IsBlock != expected.IsBlock {
		t.Errorf("%s: expected IsBlock %v, got %v", prefix, expected.IsBlock, actual.IsBlock)
	}

	if !expected.IsBlock {
		// Simple command verification
		if actual.Command != expected.Command {
			t.Errorf("%s: expected command %q, got %q", prefix, expected.Command, actual.Command)
		}

		if len(expected.Elements) > 0 || len(actual.Elements) > 0 {
			showElementsDiff(t, actual.Elements, expected.Elements, fmt.Sprintf("%s.Elements", prefix))
		}
	} else {
		// Block command verification
		if len(actual.Block) != len(expected.Block) {
			t.Errorf("%s: expected %d block statements, got %d", prefix, len(expected.Block), len(actual.Block))
			return
		}

		for i, expectedStmt := range expected.Block {
			actualStmt := actual.Block[i]
			stmtPrefix := fmt.Sprintf("%s.Block[%d]", prefix, i)

			if actualStmt.IsDecorated != expectedStmt.IsDecorated {
				t.Errorf("%s: expected IsDecorated %v, got %v", stmtPrefix, expectedStmt.IsDecorated, actualStmt.IsDecorated)
			}

			if expectedStmt.IsDecorated {
				if actualStmt.Decorator != expectedStmt.Decorator {
					t.Errorf("%s: expected decorator %q, got %q", stmtPrefix, expectedStmt.Decorator, actualStmt.Decorator)
				}

				if actualStmt.DecoratorType != expectedStmt.DecoratorType {
					t.Errorf("%s: expected decorator type %q, got %q", stmtPrefix, expectedStmt.DecoratorType, actualStmt.DecoratorType)
				}
			}

			if actualStmt.Command != expectedStmt.Command {
				t.Errorf("%s: expected command %q, got %q", stmtPrefix, expectedStmt.Command, actualStmt.Command)
			}

			if len(expectedStmt.Elements) > 0 || len(actualStmt.Elements) > 0 {
				showElementsDiff(t, actualStmt.Elements, expectedStmt.Elements, fmt.Sprintf("%s.Elements", stmtPrefix))
			}
		}
	}
}

func verifyDefinition(t *testing.T, actual Definition, expected ExpectedDefinition, index int) {
	prefix := fmt.Sprintf("Definition[%d]", index)

	if actual.Name != expected.Name {
		t.Errorf("%s: expected name %q, got %q", prefix, expected.Name, actual.Name)
	}

	if actual.Value != expected.Value {
		t.Errorf("%s: expected value %q, got %q", prefix, expected.Value, actual.Value)
	}
}

func runTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		// Using debug=false for typical test runs unless a specific test needs it
		result, err := Parse(tc.Input, false)

		// Check error expectations
		if tc.WantErr {
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tc.ErrorSubstr != "" && !strings.Contains(err.Error(), tc.ErrorSubstr) {
				t.Errorf("expected error containing %q, got %q", tc.ErrorSubstr, err.Error())
			}
			return
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify definitions
		if len(result.Definitions) != len(tc.Expected.Definitions) {
			t.Errorf("expected %d definitions, got %d", len(tc.Expected.Definitions), len(result.Definitions))
		} else {
			for i, expectedDef := range tc.Expected.Definitions {
				verifyDefinition(t, result.Definitions[i], expectedDef, i)
			}
		}

		// Verify commands
		if len(result.Commands) != len(tc.Expected.Commands) {
			t.Errorf("expected %d commands, got %d", len(tc.Expected.Commands), len(result.Commands))
		} else {
			for i, expectedCmd := range tc.Expected.Commands {
				verifyCommand(t, result.Commands[i], expectedCmd, i)
			}
		}
	})
}

// Main test functions with updated expectations for the new parser
func TestBasicCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple command",
			Input: "build: echo hello;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("build", "echo hello", Text("echo hello")),
				},
			},
		},
		{
			Name:  "command with special characters",
			Input: "run: echo 'Hello, World!';",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("run", "echo 'Hello, World!'", Text("echo 'Hello, World!'")),
				},
			},
		},
		{
			Name:  "empty command",
			Input: "noop: ;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("noop", ""),
				},
			},
		},
		{
			Name:  "command with parentheses",
			Input: "check: (echo test);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("check", "(echo test)", Text("(echo test)")),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestVarDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var() reference",
			Input: "build: cd @var(SRC);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("build", "cd @var(SRC)", Text("cd "), Var("SRC")),
				},
			},
		},
		{
			Name:  "multiple @var() references",
			Input: "deploy: docker build -t @var(IMAGE):@var(TAG);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("deploy", "docker build -t @var(IMAGE):@var(TAG)",
						Text("docker build -t "), Var("IMAGE"), Text(":"), Var("TAG")),
				},
			},
		},
		{
			Name:  "@var() in quoted string",
			Input: "echo: echo \"Building @var(PROJECT) version @var(VERSION)\";",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("echo", "echo \"Building @var(PROJECT) version @var(VERSION)\"",
						Text("echo \"Building "), Var("PROJECT"), Text(" version "), Var("VERSION"), Text("\"")),
				},
			},
		},
		{
			Name:  "mixed @var() and shell variables",
			Input: "info: echo \"Project: @var(NAME), User: $USER\";",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("info", "echo \"Project: @var(NAME), User: $USER\"",
						Text("echo \"Project: "), Var("NAME"), Text(", User: $USER\"")),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestNestedDecorators(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "@sh() with @var()",
			Input: "build: @sh(cd @var(SRC));",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("build",
						DecoratedStatement("sh", "function", "cd @var(SRC)",
							Decorator("sh", "function", Text("cd "), Var("SRC")))),
				},
			},
		},
		{
			Name:  "@sh() with multiple @var()",
			Input: "server: @sh(go run @var(MAIN_FILE) --port=@var(PORT));",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("server",
						DecoratedStatement("sh", "function", "go run @var(MAIN_FILE) --port=@var(PORT)",
							Decorator("sh", "function",
								Text("go run "),
								Var("MAIN_FILE"),
								Text(" --port="),
								Var("PORT")))),
				},
			},
		},
		{
			Name:  "complex @sh() with parentheses and @var()",
			Input: "check: @sh((cd @var(SRC) && make) || echo \"failed\");",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("check",
						DecoratedStatement("sh", "function", "(cd @var(SRC) && make) || echo \"failed\"",
							Decorator("sh", "function",
								Text("(cd "), Var("SRC"),
								Text(" && make) || echo \"failed\"")))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestBlockCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "empty block",
			Input: "setup: { }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("setup"),
				},
			},
		},
		{
			Name:  "single statement block",
			Input: "setup: { npm install }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("setup", Statement("npm install", Text("npm install"))),
				},
			},
		},
		{
			Name:  "multiple statements",
			Input: "setup: { npm install; go mod tidy; echo done }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("setup",
						Statement("npm install", Text("npm install")),
						Statement("go mod tidy", Text("go mod tidy")),
						Statement("echo done", Text("echo done"))),
				},
			},
		},
		{
			Name:  "block with @var() references",
			Input: "build: { cd @var(SRC); make @var(TARGET) }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("build",
						Statement("cd @var(SRC)", Text("cd "), Var("SRC")),
						Statement("make @var(TARGET)", Text("make "), Var("TARGET"))),
				},
			},
		},
		{
			Name:  "block with decorators",
			Input: "services: { @parallel: { server; client } }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("services",
						BlockDecoratedStatement("parallel", "block", "")),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestWatchStopCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple watch command",
			Input: "watch server: npm start;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					WatchCommand("server", "npm start", Text("npm start")),
				},
			},
		},
		{
			Name:  "simple stop command",
			Input: "stop server: pkill node;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					StopCommand("server", "pkill node", Text("pkill node")),
				},
			},
		},
		{
			Name:  "watch command with @var()",
			Input: "watch server: go run @var(MAIN_FILE) --port=@var(PORT);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					WatchCommand("server", "go run @var(MAIN_FILE) --port=@var(PORT)",
						Text("go run "), Var("MAIN_FILE"), Text(" --port="), Var("PORT")),
				},
			},
		},
		{
			Name:  "watch block command",
			Input: "watch dev: { npm start; go run main.go }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					WatchBlockCommand("dev",
						Statement("npm start", Text("npm start")),
						Statement("go run main.go", Text("go run main.go"))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestContinuationLines(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple continuation",
			Input: "build: echo hello \\\nworld;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("build", "echo hello world", Text("echo hello world")),
				},
			},
		},
		{
			Name:  "continuation with @var()",
			Input: "build: cd @var(DIR) \\\n&& make;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("build", "cd @var(DIR) && make", Text("cd "), Var("DIR"), Text(" && make")),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestDefinitions(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple definition",
			Input: "def SRC = ./src;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("SRC", "./src"),
				},
			},
		},
		{
			Name:  "definition with complex value",
			Input: "def CMD = go test -v ./...;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("CMD", "go test -v ./..."),
				},
			},
		},
		{
			Name:  "empty definition",
			Input: "def EMPTY = ;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("EMPTY", ""),
				},
			},
		},
		{
			Name:  "multiple definitions",
			Input: "def SRC = ./src;\ndef BIN = ./bin;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("SRC", "./src"),
					Def("BIN", "./bin"),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestErrorHandling(t *testing.T) {
	testCases := []TestCase{
		{
			Name:        "duplicate command",
			Input:       "build: echo hello;\nbuild: echo world;",
			WantErr:     true,
			ErrorSubstr: "duplicate command",
		},
		{
			Name:        "duplicate definition",
			Input:       "def VAR = value1;\ndef VAR = value2;",
			WantErr:     true,
			ErrorSubstr: "duplicate definition",
		},
		{
			Name:        "syntax error in command",
			Input:       "build echo hello;", // Missing colon
			WantErr:     true,
			ErrorSubstr: "missing ':'",
		},
		{
			Name:        "missing semicolon in definition",
			Input:       "def VAR = value\nbuild: echo hello;",
			WantErr:     true,
			ErrorSubstr: "missing ';'",
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

func TestCompleteFile(t *testing.T) {
	input := `
# Development commands
def SRC = ./src;
def BIN = ./bin;

# Build commands
build: cd @var(SRC) && make all;

# Run commands with parallel execution
watch server: {
  cd @var(SRC);
  @parallel: {
    ./server --port=8080;
    ./worker --queue=jobs
  };
}

stop server: pkill -f "server|worker";

# POSIX shell commands with braces using @sh()
cleanup: @sh(find . -name "*.tmp" -exec rm {} \;);
`

	tc := TestCase{
		Name:  "complete file",
		Input: input,
		Expected: struct {
			Definitions []ExpectedDefinition
			Commands    []ExpectedCommand
		}{
			Definitions: []ExpectedDefinition{
				Def("SRC", "./src"),
				Def("BIN", "./bin"),
			},
			Commands: []ExpectedCommand{
				SimpleCommand("build", "cd @var(SRC) && make all",
					Text("cd "), Var("SRC"), Text(" && make all")),
				WatchBlockCommand("server",
					Statement("cd @var(SRC)", Text("cd "), Var("SRC")),
					BlockDecoratedStatement("parallel", "block", "")),
				StopCommand("server", "pkill -f \"server|worker\"",
					Text("pkill -f \"server|worker\"")),
				BlockCommand("cleanup",
					DecoratedStatement("sh", "function", "find . -name \"*.tmp\" -exec rm {} \\;",
						Decorator("sh", "function",
							Text("find . -name \"*.tmp\" -exec rm {} \\;")))),
			},
		},
	}

	runTestCase(t, tc)
}

// Additional comprehensive test cases from original file with semantic expectations
func TestAdvancedScenarios(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "command with escaped semicolon in @sh()",
			Input: "clean: @sh(find . -name \"*.log\" -exec rm {} \\;);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("clean",
						DecoratedStatement("sh", "function", "find . -name \"*.log\" -exec rm {} \\;",
							Decorator("sh", "function",
								Text("find . -name \"*.log\" -exec rm {} \\;")))),
				},
			},
		},
		{
			Name:  "command with nested parentheses in @sh()",
			Input: "complex: @sh((test -f config && (source config && run)) || default);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("complex",
						DecoratedStatement("sh", "function", "(test -f config && (source config && run)) || default",
							Decorator("sh", "function",
								Text("(test -f config && (source config && run)) || default")))),
				},
			},
		},
		{
			Name:  "test command with braces (not @sh)",
			Input: "check-files: test -f {} && echo \"File exists\" || echo \"Missing\";",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("check-files", "test -f {} && echo \"File exists\" || echo \"Missing\"",
						Text("test -f {} && echo \"File exists\" || echo \"Missing\"")),
				},
			},
		},
		{
			Name:  "command containing watch/stop keywords",
			Input: "manage: watch -n 5 \"systemctl status app || systemctl stop app\";",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("manage", "watch -n 5 \"systemctl status app || systemctl stop app\"",
						Text("watch -n 5 \"systemctl status app || systemctl stop app\"")),
				},
			},
		},
		{
			Name:  "simple decorator",
			Input: "deploy: { @retry: docker push myapp:latest }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("deploy",
						DecoratedStatement("retry", "simple", "docker push myapp:latest",
							Text("docker push myapp:latest"))),
				},
			},
		},
		{
			Name:  "mixed decorators and regular commands",
			Input: "complex: { echo \"starting\"; @parallel: { task1; task2 }; @retry: flaky-command; echo \"done\" }",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("complex",
						Statement("echo \"starting\"", Text("echo \"starting\"")),
						BlockDecoratedStatement("parallel", "block", ""),
						DecoratedStatement("retry", "simple", "flaky-command",
							Text("flaky-command")),
						Statement("echo \"done\"", Text("echo \"done\""))),
				},
			},
		},
		{
			Name:  "multiline block",
			Input: "setup: {\n  npm install;\n  go mod tidy;\n  echo done\n}",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("setup",
						Statement("npm install", Text("npm install")),
						Statement("go mod tidy", Text("go mod tidy")),
						Statement("echo done", Text("echo done"))),
				},
			},
		},
		{
			Name:  "watch block with parentheses and keywords",
			Input: "watch monitor: {\n(watch -n 1 \"ps aux\");\necho \"stop monitoring with Ctrl+C\"\n}",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					WatchBlockCommand("monitor",
						Statement("(watch -n 1 \"ps aux\")",
							Text("(watch -n 1 \"ps aux\")")),
						Statement("echo \"stop monitoring with Ctrl+C\"",
							Text("echo \"stop monitoring with Ctrl+C\""))),
				},
			},
		},
		{
			Name:  "definition with parentheses",
			Input: "def CHECK_CMD = (which go && echo \"found\");",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("CHECK_CMD", "(which go && echo \"found\")"),
				},
			},
		},
		{
			Name:  "definition with numbers and decimals",
			Input: "def VERSION = 1.5;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("VERSION", "1.5"),
				},
			},
		},
		{
			Name:  "definition with special characters",
			Input: "def PATH = /usr/local/bin:$PATH;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Definitions: []ExpectedDefinition{
					Def("PATH", "/usr/local/bin:$PATH"),
				},
			},
		},
		{
			Name:  "multiple continuations",
			Input: "build: echo hello \\\nworld \\\nuniverse;",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("build", "echo hello world universe",
						Text("echo hello world universe")),
				},
			},
		},
		{
			Name:  "continuation with parentheses",
			Input: "check: (which go \\\n|| echo \"not found\");",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("check", "(which go || echo \"not found\")",
						Text("(which go || echo \"not found\")")),
				},
			},
		},
		{
			Name:  "complex continuation with parentheses",
			Input: "setup: (cd src && \\\nmake clean) \\\n|| echo \"failed\";",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("setup", "(cd src && make clean) || echo \"failed\"",
						Text("(cd src && make clean) || echo \"failed\"")),
				},
			},
		},
		{
			Name:  "continuation with @sh()",
			Input: "cleanup: @sh(find . -name \"*.tmp\" \\\n-delete);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("cleanup",
						DecoratedStatement("sh", "function", "find . -name \"*.tmp\" \\\n-delete",
							Decorator("sh", "function",
								Text("find . -name \"*.tmp\" \\\n-delete")))),
				},
			},
		},
		{
			Name:  "@var() with complex names",
			Input: "build: make @var(BUILD_TARGET_RELEASE) @var(EXTRA_FLAGS);",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					SimpleCommand("build", "make @var(BUILD_TARGET_RELEASE) @var(EXTRA_FLAGS)",
						Text("make "), Var("BUILD_TARGET_RELEASE"), Text(" "), Var("EXTRA_FLAGS")),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}
