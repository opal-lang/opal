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
					(data.Commands[0].Type == "watch-only" || data.Commands[0].Type == "watch") &&
					data.Commands[0].IsBackground &&
					data.HasProcessMgmt
			},
		},
		{
			name:  "stop command",
			input: "stop server: pkill node;",
			expectedData: func(data *TemplateData) bool {
				return len(data.Commands) == 1 &&
					data.Commands[0].Name == "server" &&
					(data.Commands[0].Type == "stop-only" || data.Commands[0].Type == "stop") &&
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
					(data.Commands[0].Type == "watch-stop" || strings.Contains(data.Commands[0].Type, "watch")) &&
					data.Commands[0].IsBackground &&
					data.HasProcessMgmt
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
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
		name     string
		input    devcmdParser.Command
		expected string
	}{
		{
			name: "simple command",
			input: devcmdParser.Command{
				Command: "echo hello",
			},
			expected: "echo hello",
		},
		{
			name: "block command",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "npm install", Background: false},
					{Command: "npm start", Background: true},
					{Command: "echo done", Background: false},
				},
			},
			expected: "npm install; npm start &; echo done",
		},
		{
			name: "block with all background",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "server", Background: true},
					{Command: "client", Background: true},
				},
			},
			expected: "server &; client &",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShellCommand(tt.input)
			if result != tt.expected {
				t.Errorf("buildShellCommand() = %q, want %q", result, tt.expected)
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
				`go build ./...`,
				`case "build":`,
				"c.runBuild(args)",
				"// Regular command",
			},
			notInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				"syscall", // Should not import syscall for regular commands
			},
		},
		{
			name:  "command with POSIX parentheses",
			input: "check: (which go && echo \"found\") || echo \"not found\";",
			expectedInCode: []string{
				"func (c *CLI) runCheck(args []string)",
				`(which go && echo "found") || echo "not found"`,
				`case "check":`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "command with watch/stop keywords in text",
			input: "monitor: watch -n 1 \"ps aux\" && echo \"stop with Ctrl+C\";",
			expectedInCode: []string{
				"func (c *CLI) runMonitor(args []string)",
				`watch -n 1 "ps aux" && echo "stop with Ctrl+C"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
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
			},
		},
		{
			name:  "command with POSIX find and braces",
			input: "cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
			expectedInCode: []string{
				"func (c *CLI) runCleanup(args []string)",
				`find . -name "*.tmp" -exec rm {} \;`,
				`case "cleanup":`,
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
			cf, err := devcmdParser.Parse(tt.input)
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
				`npm start`,
				`case "server":`,
				"syscall", // Watch commands should include syscall
			},
		},
		{
			name:  "simple stop command",
			input: "stop server: pkill node;",
			expectedInCode: []string{
				"func (c *CLI) runServer(args []string)",
				`pkill node`,
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
			},
		},
		{
			name:  "watch command with parentheses",
			input: "watch dev: (cd src && npm start);",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				`(cd src && npm start)`,
				"syscall",
			},
		},
		{
			name:  "watch command with POSIX find and braces",
			input: "watch cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
			expectedInCode: []string{
				"ProcessRegistry",
				"runInBackground",
				`find . -name "*.tmp" -exec rm {} \;`,
				"syscall",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
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

func TestGenerateGo_BlockCommands(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "simple block command",
			input: "setup: { npm install; go mod tidy; echo done }",
			expectedInCode: []string{
				"func (c *CLI) runSetup(args []string)",
				"npm install; go mod tidy; echo done",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "block with background processes",
			input: "run-all: { server &; client &; monitor }",
			expectedInCode: []string{
				"func (c *CLI) runRunAll(args []string)",
				"server &; client &; monitor",
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "watch block command",
			input: "watch services: { server &; worker &; echo \"started\" }",
			expectedInCode: []string{
				"ProcessRegistry",
				"server &; worker &; echo \"started\"",
				"runInBackground",
				"syscall",
			},
		},
		{
			name:  "block with parentheses and complex syntax",
			input: "parallel: { (task1 && echo \"done1\") &; (task2 || echo \"failed2\") }",
			expectedInCode: []string{
				`(task1 && echo "done1") &; (task2 || echo "failed2")`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name:  "block with POSIX find and braces",
			input: "cleanup: { find . -name \"*.tmp\" -exec rm {} \\;; echo \"cleanup done\" }",
			expectedInCode: []string{
				`find . -name "*.tmp" -exec rm {} \;`,
				`echo "cleanup done"`,
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
			cf, err := devcmdParser.Parse(tt.input)
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
			},
		},
		{
			name: "variables with parentheses",
			input: `def CHECK = (which go || echo "missing");
validate: $(CHECK) && echo "ok";`,
			expectedInCode: []string{
				`(which go || echo "missing") && echo "ok"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
			},
		},
		{
			name: "variables with POSIX find and braces",
			input: `def PATTERN = "*.tmp";
cleanup: find . -name $(PATTERN) -exec rm {} \;;`,
			expectedInCode: []string{
				`find . -name "*.tmp" -exec rm {} \;`,
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
			cf, err := devcmdParser.Parse(tt.input)
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

func TestBasicDevExample_NoSyscall(t *testing.T) {
	// This tests the specific case mentioned by the user - basicDev shouldn't get syscall
	basicDevCommands := `
# Basic development commands
def SRC = ./src;
def BUILD_DIR = ./build;

build: {
  echo "Building project...";
  mkdir -p $(BUILD_DIR);
  (cd $(SRC) && make) || echo "No Makefile found"
}

test: {
  echo "Running tests...";
  (cd $(SRC) && make test) || go test ./... || npm test || echo "No tests found"
}

clean: {
  echo "Cleaning build artifacts...";
  rm -rf $(BUILD_DIR);
  find . -name "*.tmp" -delete;
  echo "Clean complete"
}

lint: {
  echo "Running linters...";
  (which golangci-lint && golangci-lint run) || echo "No Go linter";
  (which eslint && eslint .) || echo "No JS linter";
  echo "Linting complete"
}

deps: {
  echo "Installing dependencies...";
  (test -f go.mod && go mod download) || echo "No Go modules";
  (test -f package.json && npm install) || echo "No NPM packages";
  (test -f requirements.txt && pip install -r requirements.txt) || echo "No Python packages";
  echo "Dependencies installed"
}
`

	// Parse the input
	cf, err := devcmdParser.Parse(basicDevCommands)
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
		"func (c *CLI) runLint(args []string)",
		"func (c *CLI) runDeps(args []string)",
		`"fmt"`,
		`"os"`,
		`"os/exec"`,
		// Variable expansions
		"./src",
		"./build",
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
	}

	for _, unwanted := range unwantedContent {
		if strings.Contains(generated, unwanted) {
			t.Errorf("Generated code contains unwanted content: %q", unwanted)
		}
	}
}

// Add these test functions to your go_template_test.go file

func TestGenerateGo_DollarSyntaxHandling(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "escaped shell command substitution",
			input: "date: echo \\$(date);",
			expectedInCode: []string{
				"func (c *CLI) runDate(args []string)",
				`echo $(date)`, // Should be unescaped in the generated shell command
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`\$(date)`, // Should not contain the escaped version
			},
		},
		{
			name:  "mixed devcmd and shell variables",
			input: "def DIR = /tmp;\ninfo: echo \"Dir: $(DIR), User: \\$USER, Time: \\$(date)\";",
			expectedInCode: []string{
				"func (c *CLI) runInfo(args []string)",
				`echo "Dir: /tmp, User: $USER, Time: $(date)"`, // Variables expanded, escapes resolved
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`$(DIR)`, // Should not contain unexpanded devcmd variable
				`\\$`,    // Should not contain escaped syntax
			},
		},
		{
			name:  "docker command with mixed syntax",
			input: "def IMAGE = node:18;\ndocker: docker run $(IMAGE) -e NODE_ENV=\\$NODE_ENV -e BUILD_TIME=\\$(date);",
			expectedInCode: []string{
				"func (c *CLI) runDocker(args []string)",
				`docker run node:18 -e NODE_ENV=$NODE_ENV -e BUILD_TIME=$(date)`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`$(IMAGE)`, // Should be expanded
				`\\$`,      // Should not contain escape syntax
			},
		},
		{
			name:  "complex shell operations",
			input: "count: echo \"Go files: \\$(find . -name '*.go' | wc -l)\";",
			expectedInCode: []string{
				"func (c *CLI) runCount(args []string)",
				`echo "Go files: $(find . -name '*.go' | wc -l)"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`\\$(find`, // Should not contain escaped version
			},
		},
		{
			name:  "arithmetic expansion",
			input: "math: echo \"Result: \\$((2 + 3))\";",
			expectedInCode: []string{
				"func (c *CLI) runMath(args []string)",
				`echo "Result: $((2 + 3))"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`\\$((`, // Should not contain escaped version
			},
		},
		{
			name:  "parameter expansion",
			input: "home: echo \"Home: \\${HOME:-/tmp}\";",
			expectedInCode: []string{
				"func (c *CLI) runHome(args []string)",
				`echo "Home: ${HOME:-/tmp}"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`\\${`, // Should not contain escaped version
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
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

func TestGenerateGo_DollarSyntaxInBlocks(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "block with mixed dollar syntax",
			input: "def PORT = 8080;\nsetup: { echo \"Port: $(PORT)\"; echo \"Time: \\$(date)\"; echo \"PID: \\$\\$\" }",
			expectedInCode: []string{
				"func (c *CLI) runSetup(args []string)",
				`echo "Port: 8080"; echo "Time: $(date)"; echo "PID: $$"`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`$(PORT)`, // Should be expanded
				`\\$(`,    // Should not contain escaped version
				`\\$\\$`,  // Should not contain double-escaped version
			},
		},
		{
			name:  "watch block with shell command substitution",
			input: "watch dev: { echo \"Started: \\$(date)\"; npm start &; echo \"Background PID: \\$!\" }",
			expectedInCode: []string{
				"ProcessRegistry", // Watch command should include process management
				"runInBackground",
				`echo "Started: $(date)"; npm start &; echo "Background PID: $!"`,
				"syscall",
			},
			notInCode: []string{
				`\\$(date)`, // Should not contain escaped version
				`\\$!`,      // Should not contain escaped version
			},
		},
		{
			name:  "complex docker setup with variables",
			input: "def IMAGE = myapp:latest;\ndocker: { docker build -t $(IMAGE) .; echo \"Image ID: \\$(docker images -q $(IMAGE))\"; docker run -d $(IMAGE) }",
			expectedInCode: []string{
				"func (c *CLI) runDocker(args []string)",
				`docker build -t myapp:latest .; echo "Image ID: $(docker images -q myapp:latest)"; docker run -d myapp:latest`,
			},
			notInCode: []string{
				"syscall",
				"ProcessRegistry",
				`$(IMAGE)`, // Should be expanded
				`\\$(`,     // Should not contain escaped version
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
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

func TestGenerateGo_DollarSyntaxEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedInCode []string
		notInCode      []string
	}{
		{
			name:  "multiple dollar signs in sequence",
			input: "special: echo \"\\$\\$PID: \\$\\$, Time: \\$(date)\";",
			expectedInCode: []string{
				`echo "$$PID: $$, Time: $(date)"`,
			},
			notInCode: []string{
				`\\$\\$`, // Should not contain escaped version
				`\\$(`,   // Should not contain escaped version
			},
		},
		{
			name:  "dollar signs in different quote contexts",
			input: "quotes: echo 'Cost: \\$10' && echo \"Command: \\$(date)\" && echo Price:\\$5;",
			expectedInCode: []string{
				`echo 'Cost: $10' && echo "Command: $(date)" && echo Price:$5`,
			},
			notInCode: []string{
				`\\$10`,     // Should not contain escaped version
				`\\$(`,      // Should not contain escaped version
				`Price:\\$`, // Should not contain escaped version
			},
		},
		{
			name:  "mixed with find command and braces",
			input: "cleanup: find . -name \"*.log\" -exec sh -c 'echo \"Removing: \\$1\" && rm \"\\$1\"' _ {} \\;;",
			expectedInCode: []string{
				`find . -name "*.log" -exec sh -c 'echo "Removing: $1" && rm "$1"' _ {} \;`,
			},
			notInCode: []string{
				`\\$1`, // Should not contain escaped version
			},
		},
		{
			name:  "environment variable operations",
			input: "env: export PATH=\\$PATH:/usr/local/bin && echo \\$PATH && echo \"Node: \\$(which node)\";",
			expectedInCode: []string{
				`export PATH=$PATH:/usr/local/bin && echo $PATH && echo "Node: $(which node)"`,
			},
			notInCode: []string{
				`\\$PATH`, // Should not contain escaped version
				`\\$(`,    // Should not contain escaped version
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
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

func TestBuildShellCommand_DollarSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    devcmdParser.Command
		expected string
	}{
		{
			name: "simple command with escaped dollar",
			input: devcmdParser.Command{
				Command: "echo $(date)",
			},
			expected: "echo $(date)",
		},
		{
			name: "block with mixed dollar syntax",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "echo $HOME", Background: false},
					{Command: "echo $(date)", Background: false},
					{Command: "echo $$", Background: false},
				},
			},
			expected: "echo $HOME; echo $(date); echo $$",
		},
		{
			name: "block with background processes and dollar syntax",
			input: devcmdParser.Command{
				IsBlock: true,
				Block: []devcmdParser.BlockStatement{
					{Command: "echo $(date)", Background: true},
					{Command: "echo $USER", Background: false},
				},
			},
			expected: "echo $(date) &; echo $USER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShellCommand(tt.input)
			if result != tt.expected {
				t.Errorf("buildShellCommand() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Integration test to verify the complete flow from parsing to generation
func TestDollarSyntaxIntegration(t *testing.T) {
	// This is a comprehensive example that combines all dollar syntax variants
	complexInput := `
def SRC = ./src;
def PORT = 8080;
def IMAGE = myapp:latest;

# Regular command with mixed syntax
info: echo "Source: $(SRC), User: \$USER, Time: \$(date)";

# Block command with various dollar uses
setup: {
  echo "Building in $(SRC)";
  export NODE_ENV=\$NODE_ENV;
  echo "Environment: \$NODE_ENV";
  echo "Build time: \$(date)"
}

# Watch command with background processes
watch dev: {
  echo "Starting development server on port $(PORT)";
  cd $(SRC) && npm start &;
  echo "Server PID: \$!";
  echo "Monitor with: ps aux | grep \$(echo node)"
}

# Stop command with shell operations
stop dev: {
  echo "Stopping development server";
  pkill -f "npm start" || echo "No npm processes";
  echo "Stopped at: \$(date)"
}

# Docker command with complex shell operations
docker: {
  docker build -t $(IMAGE) .;
  echo "Image ID: \$(docker images -q $(IMAGE))";
  docker run -d -p $(PORT):3000 -e NODE_ENV=\$NODE_ENV $(IMAGE);
  echo "Container: \$(docker ps -q -f ancestor=$(IMAGE))"
}
`

	// Parse the input
	cf, err := devcmdParser.Parse(complexInput)
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

	// Check that key transformations occurred correctly
	expectedTransformations := []string{
		// Variable expansions
		`echo "Source: ./src, User: $USER, Time: $(date)"`,
		`echo "Starting development server on port 8080"`,
		`docker build -t myapp:latest .`,
		`docker run -d -p 8080:3000`,

		// Shell syntax preservation
		`export NODE_ENV=$NODE_ENV`,
		`echo "Environment: $NODE_ENV"`,
		`echo "Build time: $(date)"`,
		`echo "Server PID: $!"`,
		`echo "Monitor with: ps aux | grep $(echo node)"`,
		`echo "Image ID: $(docker images -q myapp:latest)"`,
		`echo "Container: $(docker ps -q -f ancestor=myapp:latest)"`,

		// Process management for watch commands
		`ProcessRegistry`,
		`runInBackground`,
		`syscall`,
	}

	for _, expected := range expectedTransformations {
		if !strings.Contains(generated, expected) {
			t.Errorf("Generated code missing expected transformation: %q", expected)
		}
	}

	// Check that escape sequences are not present in final output
	unwantedEscapes := []string{
		`\$USER`,
		`\$(date)`,
		`\$!`,
		`\$(echo`,
		`\$(docker`,
		`$(SRC)`,   // Should be expanded
		`$(PORT)`,  // Should be expanded
		`$(IMAGE)`, // Should be expanded
	}

	for _, unwanted := range unwantedEscapes {
		if strings.Contains(generated, unwanted) {
			t.Errorf("Generated code contains unwanted escape sequence: %q", unwanted)
		}
	}
}

func TestImportHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name:  "regular commands only - minimal imports",
			input: "build: go build;\ntest: go test;\nclean: rm -rf dist;",
			shouldHave: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
			},
			shouldNotHave: []string{
				`"syscall"`,
				`"encoding/json"`,
				`"os/signal"`,
				`"time"`,
				"ProcessRegistry",
			},
		},
		{
			name:  "watch commands - full imports",
			input: "watch server: npm start;",
			shouldHave: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
				`"syscall"`,
				"ProcessRegistry",
			},
			shouldNotHave: []string{}, // All imports should be present
		},
		{
			name:  "mixed commands - full imports due to watch",
			input: "build: go build;\nwatch dev: npm start;",
			shouldHave: []string{
				`"fmt"`,
				`"os"`,
				`"os/exec"`,
				`"syscall"`,
				"ProcessRegistry",
			},
			shouldNotHave: []string{}, // All imports should be present due to watch command
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go - MAIN COMPILE CHECK
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Check expected imports/features
			for _, expected := range tt.shouldHave {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected import/feature: %q", expected)
				}
			}

			// Check that unwanted imports/features are not present
			for _, notExpected := range tt.shouldNotHave {
				if strings.Contains(generated, notExpected) {
					t.Errorf("Generated code contains unwanted import/feature: %q", notExpected)
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

func TestGenerateGo_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty command file",
			input: "",
		},
		{
			name:  "only definitions",
			input: "def VAR = value;",
		},
		{
			name:  "command with special characters",
			input: `special: echo "quotes" && echo 'single' && echo \$escaped;`,
		},
		{
			name:  "command with unicode",
			input: "unicode: echo \"Hello 世界\";",
		},
		{
			name:  "command with POSIX find and braces",
			input: "cleanup: find . -name \"*.tmp\" -exec rm {} \\;;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				t.Fatalf("GenerateGo error: %v", err)
			}

			// Verify generated code is valid Go - MAIN COMPILE CHECK
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated code is not valid Go")
				t.Logf("Generated code:\n%s", generated)
				return
			}

			// Basic structure should always be present
			expectedStructure := []string{
				"package main",
				"func main()",
				"cli := NewCLI()",
				"cli.Execute()",
			}

			for _, expected := range expectedStructure {
				if !strings.Contains(generated, expected) {
					t.Errorf("Generated code missing expected structure: %q", expected)
				}
			}
		})
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

// Benchmark tests for performance
func BenchmarkGenerateGo_SimpleCommand(b *testing.B) {
	cf := &devcmdParser.CommandFile{
		Commands: []devcmdParser.Command{
			{Name: "build", Command: "go build ./..."},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateGo(cf)
		if err != nil {
			b.Fatalf("GenerateGo error: %v", err)
		}
	}
}

func BenchmarkPreprocessCommands(b *testing.B) {
	cf := &devcmdParser.CommandFile{
		Commands: []devcmdParser.Command{
			{Name: "build", Command: "go build"},
			{Name: "test", Command: "go test"},
			{Name: "watch-server", Command: "npm start", IsWatch: true},
			{Name: "stop-server", Command: "pkill node", IsStop: true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := PreprocessCommands(cf)
		if err != nil {
			b.Fatalf("PreprocessCommands error: %v", err)
		}
	}
}
