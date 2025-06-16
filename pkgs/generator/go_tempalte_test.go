package generator

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	devcmdParser "github.com/aledsdavies/devcmd/pkgs/parser"
)

func TestPreprocessCommands(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedData func(*TemplateData) bool
		expectError  bool
	}{
		{
			name:  "simple command",
			input: "build: go build ./...;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "build" &&
					data.Commands[0].FunctionName == "runBuild" &&
					data.Commands[0].Type == "regular" &&
					data.Commands[0].ShellCommand == "go build ./..." &&
					!data.HasProcessMgmt
			},
		},
		{
			name:  "watch command",
			input: "watch server: npm start;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					data.Commands[0].Type == "watch-only" &&
					data.Commands[0].IsBackground &&
					data.Commands[0].WatchCommand == "npm start" &&
					data.HasProcessMgmt
			},
		},
		{
			name:  "stop command",
			input: "stop server: pkill node;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					data.Commands[0].Type == "stop-only" &&
					data.Commands[0].StopCommand == "pkill node" &&
					!data.HasProcessMgmt // stop alone doesn't need process mgmt
			},
		},
		{
			name:  "hyphenated command name",
			input: "check-deps: which go;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "check-deps" &&
					data.Commands[0].FunctionName == "runCheckDeps"
			},
		},
		{
			name:  "watch-stop pair",
			input: "watch server: npm start;\nstop server: pkill node;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					data.Commands[0].Type == "watch-stop" &&
					data.Commands[0].IsBackground &&
					data.Commands[0].WatchCommand == "npm start" &&
					data.Commands[0].StopCommand == "pkill node" &&
					data.HasProcessMgmt
			},
		},
		{
			name:  "block command with @sh decorator",
			input: "cleanup: @sh(find . -name \"*.tmp\" -exec rm {} \\;);",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "cleanup" &&
					data.Commands[0].Type == "regular" &&
					strings.Contains(data.Commands[0].ShellCommand, "find . -name \"*.tmp\" -exec rm {} \\;")
			},
		},
		{
			name:  "block command with @parallel decorator",
			input: "services: { @parallel: { server; client; database } }",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "services" &&
					data.Commands[0].Type == "regular" &&
					strings.Contains(data.Commands[0].ShellCommand, "server &") &&
					strings.Contains(data.Commands[0].ShellCommand, "client &") &&
					strings.Contains(data.Commands[0].ShellCommand, "database &") &&
					strings.Contains(data.Commands[0].ShellCommand, "wait")
			},
		},
		{
			name:        "unsupported decorator",
			input:       "test: @unsupported(echo hello);",
			expectError: true,
		},
		{
			name:        "invalid @parallel usage",
			input:       "test: @parallel(echo hello);",
			expectError: true,
		},
		{
			name:        "invalid @sh usage with block",
			input:       "test: { @sh: { echo hello; echo world } }",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Preprocess commands
			data, err := PreprocessCommands(cf)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("PreprocessCommands error: %v", err)
			}

			if !tt.expectedData(data) {
				t.Errorf("Data validation failed for %s", tt.name)
				t.Logf("Commands: %+v", data.Commands)
				t.Logf("HasProcessMgmt: %v", data.HasProcessMgmt)
			}
		})
	}
}

func TestSanitizeFunctionName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"build", "runBuild"},
		{"check-deps", "runCheckDeps"},
		{"run-all", "runRunAll"},
		{"test_coverage", "runTestCoverage"},
		{"api-server-dev", "runApiServerDev"},
		{"", "runCommand"},
		{"kebab-case-command", "runKebabCaseCommand"},
		{"snake_case_command", "runSnakeCaseCommand"},
		{"CamelCase", "runCamelcase"},
		{"123-numeric", "run123Numeric"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFunctionName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFunctionName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildShellCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     devcmdParser.Command
		expected  string
		expectErr bool
	}{
		{
			name: "simple command",
			input: devcmdParser.Command{
				Command: "echo hello",
			},
			expected: "echo hello",
		},
		{
			name: "block command with regular statements",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "npm install", IsDecorated: false},
					{Command: "echo done", IsDecorated: false},
				},
			},
			expected: "npm install; echo done",
		},
		{
			name: "block command with @sh decorator",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{
						IsDecorated:   true,
						Decorator:     "sh",
						DecoratorType: "function",
						Command:       "find . -name \"*.tmp\" -exec rm {} \\;",
					},
				},
			},
			expected: "find . -name \"*.tmp\" -exec rm {} \\;",
		},
		{
			name: "block command with @parallel decorator",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{
						IsDecorated:   true,
						Decorator:     "parallel",
						DecoratorType: "block",
						DecoratedBlock: []devcmdParser.BlockStatement{
							{Command: "server", IsDecorated: false},
							{Command: "client", IsDecorated: false},
						},
					},
				},
			},
			expected: "server &; client &; wait",
		},
		{
			name: "mixed block with regular and decorated commands",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "echo starting", IsDecorated: false},
					{
						IsDecorated:   true,
						Decorator:     "parallel",
						DecoratorType: "block",
						DecoratedBlock: []devcmdParser.BlockStatement{
							{Command: "task1", IsDecorated: false},
							{Command: "task2", IsDecorated: false},
						},
					},
					{Command: "echo done", IsDecorated: false},
				},
			},
			expected: "echo starting; task1 &; task2 &; wait; echo done",
		},
		{
			name: "block command with unsupported decorator",
			input: devcmdParser.Command{
				Name:    "test",
				Line:    1,
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{
						IsDecorated:   true,
						Decorator:     "unsupported",
						DecoratorType: "function",
						Command:       "echo hello",
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildShellCommand(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("buildShellCommand() error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("buildShellCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildDecoratedStatement(t *testing.T) {
	tests := []struct {
		name      string
		input     devcmdParser.BlockStatement
		expected  string
		expectErr bool
	}{
		{
			name: "@sh function decorator",
			input: devcmdParser.BlockStatement{
				IsDecorated:   true,
				Decorator:     "sh",
				DecoratorType: "function",
				Command:       "find . -name \"*.tmp\" -exec rm {} \\;",
			},
			expected: "find . -name \"*.tmp\" -exec rm {} \\;",
		},
		{
			name: "@sh simple decorator",
			input: devcmdParser.BlockStatement{
				IsDecorated:   true,
				Decorator:     "sh",
				DecoratorType: "simple",
				Command:       "echo hello",
			},
			expected: "echo hello",
		},
		{
			name: "@parallel block decorator",
			input: devcmdParser.BlockStatement{
				IsDecorated:   true,
				Decorator:     "parallel",
				DecoratorType: "block",
				DecoratedBlock: []devcmdParser.BlockStatement{
					{Command: "server", IsDecorated: false},
					{Command: "client", IsDecorated: false},
					{Command: "worker", IsDecorated: false},
				},
			},
			expected: "server &; client &; worker &; wait",
		},
		{
			name: "unsupported decorator",
			input: devcmdParser.BlockStatement{
				IsDecorated:   true,
				Decorator:     "unknown",
				DecoratorType: "simple",
				Command:       "echo test",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildDecoratedStatement(tt.input, "test", 1)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("buildDecoratedStatement() error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("buildDecoratedStatement() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateDecorators(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid @sh decorator",
			input:       "cleanup: @sh(find . -name \"*.tmp\" -delete);",
			expectError: false,
		},
		{
			name:        "valid @parallel decorator",
			input:       "services: { @parallel: { server; client } }",
			expectError: false,
		},
		{
			name:        "unsupported decorator",
			input:       "test: @unsupported(echo hello);",
			expectError: true,
			errorSubstr: "unsupported decorator '@unsupported'",
		},
		{
			name:        "invalid @parallel usage - function syntax",
			input:       "test: @parallel(echo hello);",
			expectError: true,
			errorSubstr: "@parallel decorator must be used with block syntax",
		},
		{
			name:        "invalid @sh usage - block syntax",
			input:       "test: { @sh: { echo hello; echo world } }",
			expectError: true,
			errorSubstr: "@sh decorator must be used with function or simple syntax",
		},
		{
			name:        "multiple valid decorators",
			input:       "complex: { @sh(echo start); @parallel: { task1; task2 } }",
			expectError: false,
		},
		{
			name:        "nested decorator validation",
			input:       "nested: { @parallel: { @sh(echo task1); @sh(echo task2) } }",
			expectError: false,
		},
		{
			name:        "nested unsupported decorator",
			input:       "nested: { @parallel: { @invalid(echo task1); echo task2 } }",
			expectError: true,
			errorSubstr: "unsupported decorator '@invalid'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Test validation through PreprocessCommands
			_, err = PreprocessCommands(cf)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateGo_BasicCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "simple command",
			input: "build: go build ./...;",
			expectedInCode: []string{
				"func (c *CLI) runBuild(args []string)",
				"go build ./...",
				`case "build":`,
				"c.runBuild(args)",
				"// Regular command",
			},
			notInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"syscall",        // Should not import syscall for regular commands
				"logs <process>", // Should not have global logs command
			},
		},
		{
			name:  "command with POSIX find using @sh",
			input: "cleanup: @sh(find . -name \"*.tmp\" -exec rm {} \\;);",
			expectedInCode: []string{
				"func (c *CLI) runCleanup(args []string)",
				"find . -name \"*.tmp\" -exec rm {} \\;",
				`case "cleanup":`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name:  "hyphenated command name",
			input: "check-deps: which go || echo missing;",
			expectedInCode: []string{
				"func (c *CLI) runCheckDeps(args []string)",
				`case "check-deps":`, // Case should use original name
				"c.runCheckDeps(args)",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name:  "block command with @parallel",
			input: "services: { @parallel: { server; client; database } }",
			expectedInCode: []string{
				"func (c *CLI) runServices(args []string)",
				"server &; client &; database &; wait",
				`case "services":`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry", // Regular commands don't need process management
				"showLogsFor",
				"logs <process>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notExpected := range tt.notInCode {
				if strings.Contains(generated, notExpected) {
					t.Errorf("Generated code contains unwanted content: %q", notExpected)
				}
			}
		})
	}
}

func TestGenerateGo_WatchStopCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "simple watch command",
			input: "watch server: npm start;",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"func (c *CLI) runServer(args []string)",
				"npm start",
				`case "server":`,
				"syscall", // Watch commands should include syscall
				"start|stop|logs",
				"showLogsFor",
			},
			notInCode: []string{
				"logs <process>", // Should not have global logs command
			},
		},
		{
			name:  "simple stop command",
			input: "stop server: pkill node;",
			expectedInCode: []string{
				"func (c *CLI) runServer(args []string)",
				"pkill node",
			},
			notInCode: []string{
				"ProcessRegistry", // No watch commands means no process management
				"syscall",         // Stop-only commands don't need syscall
			},
		},
		{
			name:  "watch and stop pair",
			input: "watch api: go run main.go;\nstop api: pkill -f main.go;",
			expectedInCode: []string{
				"ProcessRegistry", // Should have ProcessRegistry due to watch command
				"func (c *CLI) runApi(args []string)",
				"go run main.go",
				"pkill -f main.go",
				"syscall", // Watch/stop pairs need syscall
				"start|stop|logs",
				"showLogsFor",
			},
			notInCode: []string{
				"logs <process>", // Should not have global logs command
			},
		},
		{
			name:  "watch command with @sh decorator",
			input: "watch cleanup: @sh(find . -name \"*.tmp\" -exec rm {} \\;);",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"find . -name \"*.tmp\" -exec rm {} \\;",
				"syscall",
				"showLogsFor",
			},
			notInCode: []string{
				"logs <process>",
			},
		},
		{
			name:  "watch command with @parallel block",
			input: "watch dev: { @parallel: { npm start; go run ./api } }",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"npm start &; go run ./api &; wait",
				"syscall",
				"showLogsFor",
			},
			notInCode: []string{
				"logs <process>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notInCode := range tt.notInCode {
				if strings.Contains(generated, notInCode) {
					t.Errorf("Generated code contains unwanted content: %q", notInCode)
				}
			}
		})
	}
}

func TestGenerateGo_DecoratorHandling(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "@sh function decorator",
			input: "cleanup: @sh(find . -name \"*.tmp\" -exec rm {} \\;);",
			expectedInCode: []string{
				"func (c *CLI) runCleanup(args []string)",
				"find . -name \"*.tmp\" -exec rm {} \\;",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name:  "@parallel block decorator",
			input: "services: { @parallel: { server; client; worker } }",
			expectedInCode: []string{
				"func (c *CLI) runServices(args []string)",
				"server &; client &; worker &; wait",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name:  "mixed decorators in block",
			input: "complex: { echo starting; @parallel: { task1; task2 }; @sh(echo \"done\") }",
			expectedInCode: []string{
				"func (c *CLI) runComplex(args []string)",
				"echo starting",
				"task1 &; task2 &; wait",
				"echo \"done\"",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name:  "watch command with decorators",
			input: "watch dev: { echo starting; @parallel: { server; client } }",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"echo starting; server &; client &; wait",
				"syscall",
				"showLogsFor",
			},
			notInCode: []string{
				"logs <process>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notInCode := range tt.notInCode {
				if strings.Contains(generated, notInCode) {
					t.Errorf("Generated code contains unwanted content: %q", notInCode)
				}
			}
		})
	}
}

func TestGenerateGo_DecoratorValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "unsupported decorator",
			input:       "test: @invalid(echo hello);",
			expectError: true,
			errorSubstr: "unsupported decorator '@invalid'",
		},
		{
			name:        "invalid @parallel usage",
			input:       "test: @parallel(echo hello);",
			expectError: true,
			errorSubstr: "@parallel decorator must be used with block syntax",
		},
		{
			name:        "invalid @sh block usage",
			input:       "test: { @sh: { echo hello; echo world } }",
			expectError: true,
			errorSubstr: "@sh decorator must be used with function or simple syntax",
		},
		{
			name:        "valid decorators should not error",
			input:       "test: { @sh(echo hello); @parallel: { task1; task2 } }",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code (this should trigger validation)
			_, err = GenerateGo(cf)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateGo_VariableHandling(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name: "commands with variables",
			input: `def SRC = ./src;
def PORT = 8080;
build: cd $(SRC) && go build;
start: go run $(SRC) --port=$(PORT);`,
			expectedInCode: []string{
				"func (c *CLI) runBuild(args []string)",
				"func (c *CLI) runStart(args []string)",
				"cd ./src && go build",
				"go run ./src --port=8080",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name:  "variables with @sh decorator",
			input: `cleanup: @sh(find . -name "*.tmp" -exec rm {} \\;);`,
			expectedInCode: []string{
				"find . -name \"*.tmp\" -exec rm {} \\;",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				"showLogsFor",
				"logs <process>",
			},
		},
		{
			name: "variables with @parallel decorator",
			input: `def CMD1 = server --port=8080;
def CMD2 = client --host=localhost;
services: { @parallel: { $(CMD1); $(CMD2) } }`,
			expectedInCode: []string{
				"server --port=8080 &; client --host=localhost &; wait",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Expand variables
			err = cf.ExpandVariables()
			if err != nil {
				t.Fatalf("ExpandVariables error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected content
			for _, expected := range tt.expectedInCode {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected content: %q", expected)
				}
			}

			// Check that unwanted content is not present
			for _, notInCode := range tt.notInCode {
				if strings.Contains(generated, notInCode) {
					t.Errorf("Generated code contains unwanted content: %q", notInCode)
				}
			}
		})
	}
}

func TestGenerateGo_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       *devcmdParser.CommandFile
		template    *string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil command file",
			input:       nil,
			expectError: true,
			errorMsg:    "command file cannot be nil",
		},
		{
			name:        "empty template string",
			input:       &devcmdParser.CommandFile{},
			template:    stringPtr(""),
			expectError: true,
			errorMsg:    "template string cannot be empty",
		},
		{
			name:        "whitespace-only template",
			input:       &devcmdParser.CommandFile{},
			template:    stringPtr("   \n\t  "),
			expectError: true,
			errorMsg:    "template string cannot be empty",
		},
		{
			name: "invalid template syntax",
			input: &devcmdParser.CommandFile{
				Commands: []devcmdParser.Command{
					{Name: "test", Command: "echo test"},
				},
			},
			template:    stringPtr("{{.InvalidField"),
			expectError: true,
			errorMsg:    "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.template != nil {
				_, err = GenerateGoWithTemplate(tt.input, *tt.template)
			} else {
				_, err = GenerateGo(tt.input)
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestBasicDevExample_NoSyscall(t *testing.T) {
	// Test that basic development commands don't include syscall imports
	basicDevCommands := `
def SRC = ./src;
def BUILD_DIR = ./build;

build: {
  echo "Building project...";
  mkdir -p $(BUILD_DIR);
  cd $(SRC) && make;
}

test: {
  echo "Running tests...";
  cd $(SRC) && make test;
  echo "Tests complete";
}

clean: @sh(find . -name "*.tmp" -delete);

parallel-tasks: {
  @parallel: {
    echo "Task 1";
    echo "Task 2";
    echo "Task 3"
  };
  echo "All tasks complete";
}
`

	// Parse the input
	cf, err := devcmdParser.Parse(basicDevCommands, false)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Expand variables
	err = cf.ExpandVariables()
	if err != nil {
		t.Fatalf("ExpandVariables error: %v", err)
	}

	// Generate Go code
	generated, err := GenerateGo(cf)
	if err != nil {
		t.Fatalf("GenerateGo error: %v", err)
	}

	// Verify generated code is valid Go - this is the main compile check
	if !isValidGoCode(t, generated) {
		t.Errorf("Generated code is not valid Go")
		t.Logf("Generated code:\n%s", generated)
		return
	}

	// These should be present (basic functionality)
	expectedContent := []string{
		"func (c *CLI) runBuild(args []string)",
		"func (c *CLI) runTest(args []string)",
		"func (c *CLI) runClean(args []string)",
		"func (c *CLI) runParallelTasks(args []string)",
		`"fmt"`,
		`"os"`,
		`"os/exec"`,
		// Variable expansions
		"./src",
		"./build",
		// Regular commands should work with variables
		"cd ./src && make",
		// @sh decorators should be converted (no variables inside)
		"find . -name \"*.tmp\" -delete",
		// @parallel should create background processes
		"echo \"Task 1\" &; echo \"Task 2\" &; echo \"Task 3\" &; wait",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected content: %q", expected)
		}
	}

	// These should NOT be present (no watch commands)
	unwantedContent := []string{
		`"syscall"`,
		`"encoding/json"`,
		`"os/signal"`,
		`"time"`,
		"ProcessRegistry",
		"runInBackground",
		"gracefulStop",
		"showLogsFor",
		"logs <process>",
	}

	for _, unwanted := range unwantedContent {
		if strings.Contains(generated, unwanted) {
			t.Errorf("Generated code contains unwanted content: %q", unwanted)
		}
	}
}

// Helper function to check if generated code is valid Go - THE KEY FUNCTION
func isValidGoCode(t *testing.T, code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
	if err != nil {
		t.Logf("Go parsing error: %v", err)
		return false
	}
	return true
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Integration test for complete file with all features
func TestCompleteIntegration(t *testing.T) {
	complexInput := `
def SRC = ./src;
def PORT = 8080;

# Regular commands
build: cd $(SRC) && go build;
test: cd $(SRC) && go test -v ./...;

# Parallel execution
parallel-build: {
  echo "Starting parallel build...";
  @parallel: {
    @sh(cd frontend && npm run build);
    @sh(cd backend && go build);
    @sh(cd worker && python setup.py build)
  };
  echo "All builds complete";
}

# Watch commands for development
watch dev: {
  echo "Starting development environment...";
  @parallel: {
    @sh(cd frontend && npm start);
    cd backend && go run main.go --port=$(PORT);
    @sh(cd worker && python worker.py)
  }
}

stop dev: {
  echo "Stopping development environment...";
  @sh(pkill -f "npm start" || true);
  @sh(pkill -f "go run main.go" || true);
  @sh(pkill -f "python worker.py" || true);
  echo "Development environment stopped";
}

# Complex cleanup
cleanup: {
  echo "Cleaning up...";
  @parallel: {
    @sh(find ./build -name "*.tmp" -delete);
    @sh(find ./logs -name "*.log" -mtime +7 -delete);
    rm -rf ./cache
  };
  echo "Cleanup complete";
}
`

	// Parse the input
	cf, err := devcmdParser.Parse(complexInput, false)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Expand variables
	err = cf.ExpandVariables()
	if err != nil {
		t.Fatalf("ExpandVariables error: %v", err)
	}

	// Generate Go code
	generated, err := GenerateGo(cf)
	if err != nil {
		t.Fatalf("GenerateGo error: %v", err)
	}

	// Verify generated code is valid Go
	if !isValidGoCode(t, generated) {
		t.Errorf("Generated code is not valid Go")
		t.Logf("Generated code:\n%s", generated)
		return
	}

	// Should include process management due to watch commands
	expectedFeatures := []string{
		"ProcessRegistry",
		"runInBackground",
		"syscall",
		"encoding/json",
		"showLogsFor",
		"start|stop|logs",
	}

	for _, expected := range expectedFeatures {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected feature: %q", expected)
		}
	}

	// Should handle all command types correctly
	expectedCommands := []string{
		"func (c *CLI) runBuild(args []string)",
		"func (c *CLI) runTest(args []string)",
		"func (c *CLI) runParallelBuild(args []string)",
		"func (c *CLI) runDev(args []string)",
		"func (c *CLI) runCleanup(args []string)",
	}

	for _, expected := range expectedCommands {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected command function: %q", expected)
		}
	}

	// Should properly expand variables
	expectedExpansions := []string{
		"cd ./src && go build",
		"cd ./src && go test -v ./...",
		"go run main.go --port=8080",
	}

	for _, expected := range expectedExpansions {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected variable expansion: %q", expected)
		}
	}

	// Should NOT have global logs command
	unwantedContent := []string{
		"logs <process>",
	}

	for _, unwanted := range unwantedContent {
		if strings.Contains(generated, unwanted) {
			t.Errorf("Generated code contains unwanted content: %q", unwanted)
		}
	}
}
