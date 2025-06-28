package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

// Diff helper function
func showDiff(t *testing.T, expected, actual interface{}, path string) {
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("%s mismatch (-expected +actual):\n%s", path, diff)
	}
}

// Helper functions to convert elements recursively
func convertElementsToComparable(elements []CommandElement) []interface{} {
	result := make([]interface{}, len(elements))
	for i, elem := range elements {
		if elem.IsDecorator() {
			decorator := elem.(*DecoratorElement)
			result[i] = map[string]interface{}{
				"Type":          "decorator",
				"DecoratorName": decorator.Name,
				"DecoratorType": decorator.Type,
				"Args":          convertElementsToComparable(decorator.Args),
			}
		} else {
			result[i] = map[string]interface{}{
				"Type": "text",
				"Text": elem.String(),
			}
		}
	}
	return result
}

func convertExpectedElementsToComparable(elements []ExpectedElement) []interface{} {
	result := make([]interface{}, len(elements))
	for i, elem := range elements {
		if elem.Type == "decorator" {
			result[i] = map[string]interface{}{
				"Type":          "decorator",
				"DecoratorName": elem.DecoratorName,
				"DecoratorType": elem.DecoratorType,
				"Args":          convertExpectedElementsToComparable(elem.Args),
			}
		} else {
			result[i] = map[string]interface{}{
				"Type": "text",
				"Text": elem.Text,
			}
		}
	}
	return result
}

func verifyCommand(t *testing.T, actual Command, expected ExpectedCommand, index int) {
	prefix := fmt.Sprintf("Command[%d]", index)

	// Create comparable structures
	actualComparable := map[string]interface{}{
		"Name":    actual.Name,
		"IsWatch": actual.IsWatch,
		"IsStop":  actual.IsStop,
		"IsBlock": actual.IsBlock,
	}

	expectedComparable := map[string]interface{}{
		"Name":    expected.Name,
		"IsWatch": expected.IsWatch,
		"IsStop":  expected.IsStop,
		"IsBlock": expected.IsBlock,
	}

	if !expected.IsBlock {
		actualComparable["Command"] = actual.Command
		actualComparable["Elements"] = convertElementsToComparable(actual.Elements)

		expectedComparable["Command"] = expected.Command
		expectedComparable["Elements"] = convertExpectedElementsToComparable(expected.Elements)
	} else {
		// Convert block statements
		actualBlock := make([]interface{}, len(actual.Block))
		for i, stmt := range actual.Block {
			actualBlock[i] = map[string]interface{}{
				"IsDecorated":   stmt.IsDecorated,
				"Decorator":     stmt.Decorator,
				"DecoratorType": stmt.DecoratorType,
				"Command":       stmt.Command,
				"Elements":      convertElementsToComparable(stmt.Elements),
			}
		}

		expectedBlock := make([]interface{}, len(expected.Block))
		for i, stmt := range expected.Block {
			expectedBlock[i] = map[string]interface{}{
				"IsDecorated":   stmt.IsDecorated,
				"Decorator":     stmt.Decorator,
				"DecoratorType": stmt.DecoratorType,
				"Command":       stmt.Command,
				"Elements":      convertExpectedElementsToComparable(stmt.Elements),
			}
		}

		actualComparable["Block"] = actualBlock
		expectedComparable["Block"] = expectedBlock
	}

	// Show diff
	showDiff(t, expectedComparable, actualComparable, prefix)
}

func verifyDefinition(t *testing.T, actual Definition, expected ExpectedDefinition, index int) {
	prefix := fmt.Sprintf("Definition[%d]", index)

	actualComparable := map[string]interface{}{
		"Name":  actual.Name,
		"Value": actual.Value,
	}

	expectedComparable := map[string]interface{}{
		"Name":  expected.Name,
		"Value": expected.Value,
	}

	showDiff(t, expectedComparable, actualComparable, prefix)
}

func runTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		// Enable debug for failing cases
		debug := tc.WantErr || strings.Contains(tc.Name, "debug") || strings.Contains(tc.Name, "complex")
		result, err := Parse(tc.Input, debug)

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
		{
			Name: "debug backup - DATE assignment with semicolon",
			Input: `backup-debug2: {
        @sh(DATE=$(date +%Y%m%d-%H%M%S); echo "test")
      }`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("backup-debug2",
						DecoratedStatement("sh", "function", "DATE=$(date +%Y%m%d-%H%M%S); echo \"test\"",
							Decorator("sh", "function", Text("DATE=$(date +%Y%m%d-%H%M%S); echo \"test\"")))),
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

// Test cases specifically targeting the failing shell command structure
func TestComplexShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple shell command substitution",
			Input: `test-simple: @sh(echo "$(date)");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-simple",
						DecoratedStatement("sh", "function", "echo \"$(date)\"",
							Decorator("sh", "function", Text("echo \"$(date)\"")))),
				},
			},
		},
		{
			Name:  "shell command with test and command substitution",
			Input: `test-condition: @sh(if [ "$(echo test)" = "test" ]; then echo ok; fi);`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-condition",
						DecoratedStatement("sh", "function", "if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi",
							Decorator("sh", "function", Text("if [ \"$(echo test)\" = \"test\" ]; then echo ok; fi")))),
				},
			},
		},
		{
			Name:  "command with @var and shell substitution",
			Input: `test-mixed: @sh(cd @var(SRC) && echo "files: $(ls | wc -l)");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-mixed",
						DecoratedStatement("sh", "function", "cd @var(SRC) && echo \"files: $(ls | wc -l)\"",
							Decorator("sh", "function",
								Text("cd "), Var("SRC"), Text(" && echo \"files: $(ls | wc -l)\"")))),
				},
			},
		},
		{
			Name: "the actual failing command from commands.cli",
			Input: `test-quick: {
    echo "⚡ Running quick checks...";
    @sh(if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; fi);
}`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-quick",
						Statement("echo \"⚡ Running quick checks...\"", Text("echo \"⚡ Running quick checks...\"")),
						DecoratedStatement("sh", "function",
							"if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; fi",
							// Fixed the typo: changed "gofmut" to "gofumpt"
							Decorator("sh", "function", Text("if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; fi")))),
				},
			},
		},
		{
			Name:  "simplified version of failing command",
			Input: `test-format: @sh(if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "issues"; fi);`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-format",
						DecoratedStatement("sh", "function", "if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi",
							Decorator("sh", "function", Text("if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"issues\"; fi")))),
				},
			},
		},
		{
			Name:  "even simpler - just the command substitution in quotes",
			Input: `test-basic: @sh("$(gofumpt -l . | wc -l)");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-basic",
						DecoratedStatement("sh", "function", "\"$(gofumpt -l . | wc -l)\"",
							Decorator("sh", "function", Text("\"$(gofumpt -l . | wc -l)\"")))),
				},
			},
		},
		{
			Name:  "debug - minimal parentheses in quotes",
			Input: `test-debug: @sh("()");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-debug",
						DecoratedStatement("sh", "function", "\"()\"",
							Decorator("sh", "function", Text("\"()\"")))),
				},
			},
		},
		{
			Name:  "debug - single command substitution",
			Input: `test-debug2: @sh($(echo test));`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-debug2",
						DecoratedStatement("sh", "function", "$(echo test)",
							Decorator("sh", "function", Text("$(echo test)")))),
				},
			},
		},
		// Test case for backup command with complex shell substitution and @var()
		{
			Name: "backup command with shell substitution and @var",
			Input: `backup: {
        echo "Creating backup...";
        # Shell command substitution uses regular $() syntax in @sh()
        @sh(DATE=$(date +%Y%m%d-%H%M%S); echo "Backup timestamp: $DATE");
        @sh((which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo "No database")
      }`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("backup",
						Statement("echo \"Creating backup...\"", Text("echo \"Creating backup...\"")),
						DecoratedStatement("sh", "function",
							"DATE=$(date +%Y%m%d-%H%M%S); echo \"Backup timestamp: $DATE\"",
							Decorator("sh", "function", Text("DATE=$(date +%Y%m%d-%H%M%S); echo \"Backup timestamp: $DATE\""))),
						DecoratedStatement("sh", "function",
							"(which kubectl && kubectl exec deployment/database -n @var(KUBE_NAMESPACE) -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo \"No database\"",
							Decorator("sh", "function",
								Text("(which kubectl && kubectl exec deployment/database -n "),
								Var("KUBE_NAMESPACE"),
								Text(" -- pg_dump myapp > backup-$(date +%Y%m%d-%H%M%S).sql) || echo \"No database\"")))),
				},
			},
		},
		// Add this test case at the end of the existing testCases slice:
		{
			Name: "exact command from real commands.cli file",
			Input: `test-quick: {
    echo "⚡ Running quick checks...";
    echo "🔍 Checking Go formatting...";
    @sh(if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofmt -l .; exit 1; fi; fi);
    echo "🔍 Checking Nix formatting...";
    @sh(if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo "❌ Run 'dev format' to fix"; exit 1); else echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; fi);
    dev lint;
    echo "✅ Quick checks passed!";
}`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-quick",
						Statement("echo \"⚡ Running quick checks...\"", Text("echo \"⚡ Running quick checks...\"")),
						Statement("echo \"🔍 Checking Go formatting...\"", Text("echo \"🔍 Checking Go formatting...\"")),
						DecoratedStatement("sh", "function",
							"if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofmt -l .; exit 1; fi; fi",
							Decorator("sh", "function", Text("if command -v gofumpt >/dev/null 2>&1; then if [ \"$(gofumpt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofumpt -l .; exit 1; fi; else if [ \"$(gofmt -l . | wc -l)\" -gt 0 ]; then echo \"❌ Go formatting issues:\"; gofmt -l .; exit 1; fi; fi"))),
						Statement("echo \"🔍 Checking Nix formatting...\"", Text("echo \"🔍 Checking Nix formatting...\"")),
						DecoratedStatement("sh", "function",
							"if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo \"❌ Run 'dev format' to fix\"; exit 1); else echo \"⚠️  nixpkgs-fmt not available, skipping Nix format check\"; fi",
							Decorator("sh", "function", Text("if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo \"❌ Run 'dev format' to fix\"; exit 1); else echo \"⚠️  nixpkgs-fmt not available, skipping Nix format check\"; fi"))),
						Statement("dev lint", Text("dev lint")),
						Statement("echo \"✅ Quick checks passed!\"", Text("echo \"✅ Quick checks passed!\""))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test cases for edge cases in quote and parentheses handling
func TestQuoteAndParenthesesEdgeCases(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "escaped quotes in shell command",
			Input: `test-escaped: @sh(echo "He said \"hello\" to me");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-escaped",
						DecoratedStatement("sh", "function", "echo \"He said \\\"hello\\\" to me\"",
							Decorator("sh", "function", Text("echo \"He said \\\"hello\\\" to me\"")))),
				},
			},
		},
		{
			Name:  "mixed quotes with parentheses",
			Input: `test-mixed-quotes: @sh(echo 'test "$(date)" done' && echo "test '$(whoami)' done");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-mixed-quotes",
						DecoratedStatement("sh", "function", "echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\"",
							Decorator("sh", "function", Text("echo 'test \"$(date)\" done' && echo \"test '$(whoami)' done\"")))),
				},
			},
		},
		{
			Name:  "backticks with parentheses",
			Input: "test-backticks: @sh(echo `date` and $(whoami));",
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-backticks",
						DecoratedStatement("sh", "function", "echo `date` and $(whoami)",
							Decorator("sh", "function", Text("echo `date` and $(whoami)")))),
				},
			},
		},
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}

// Test cases for @var() within shell commands
func TestVarInShellCommands(t *testing.T) {
	testCases := []TestCase{
		{
			Name:  "simple @var in shell command",
			Input: `test-var: @sh(cd @var(DIR));`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-var",
						DecoratedStatement("sh", "function", "cd @var(DIR)",
							Decorator("sh", "function", Text("cd "), Var("DIR")))),
				},
			},
		},
		{
			Name:  "@var with shell command substitution",
			Input: `test-var-cmd: @sh(cd @var(DIR) && echo "$(pwd)");`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-var-cmd",
						DecoratedStatement("sh", "function", "cd @var(DIR) && echo \"$(pwd)\"",
							Decorator("sh", "function", Text("cd "), Var("DIR"), Text(" && echo \"$(pwd)\"")))),
				},
			},
		},
		{
			Name:  "multiple @var with complex shell",
			Input: `test-multi-var: @sh(if [ -d @var(SRC) ] && [ "$(ls @var(SRC) | wc -l)" -gt 0 ]; then echo "Source dir has files"; fi);`,
			Expected: struct {
				Definitions []ExpectedDefinition
				Commands    []ExpectedCommand
			}{
				Commands: []ExpectedCommand{
					BlockCommand("test-multi-var",
						DecoratedStatement("sh", "function", "if [ -d @var(SRC) ] && [ \"$(ls @var(SRC) | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi",
							Decorator("sh", "function",
								Text("if [ -d "), Var("SRC"), Text(" ] && [ \"$(ls "), Var("SRC"), Text(" | wc -l)\" -gt 0 ]; then echo \"Source dir has files\"; fi")))),
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
			Input: "services: { @parallel { server; client } }",
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
			ErrorSubstr: "syntax error",
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
  @parallel {
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
