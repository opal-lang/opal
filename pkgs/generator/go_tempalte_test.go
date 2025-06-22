package generator

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	devcmdParser "github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestTemplateEdgeCases tests various edge cases that might cause template failures
func TestTemplateEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldFail  bool
		description string
	}{
		{
			name:        "empty_file",
			input:       "",
			shouldFail:  false,
			description: "Completely empty commands file",
		},
		{
			name:        "only_comments",
			input:       "# Just comments\n# Nothing else",
			shouldFail:  false,
			description: "File with only comments",
		},
		{
			name:        "only_variables",
			input:       "def PORT = 8080;\ndef HOST = localhost;",
			shouldFail:  false,
			description: "File with only variable definitions",
		},
		{
			name:        "single_simple_command",
			input:       "test: echo hello;",
			shouldFail:  false,
			description: "Single simple command",
		},
		{
			name:        "command_with_special_chars",
			input:       "test: echo 'Hello \\$USER, welcome to \\$(pwd)!';",
			shouldFail:  false,
			description: "Command with escaped shell special characters",
		},
		{
			name:        "command_with_quotes",
			input:       `test: echo "Hello \"world\"";`,
			shouldFail:  false,
			description: "Command with escaped quotes",
		},
		{
			name:        "multiple_simple_commands",
			input:       "build: go build;\ntest: go test;\nclean: rm -rf dist;",
			shouldFail:  false,
			description: "Multiple simple commands",
		},
		{
			name:        "single_watch_command",
			input:       "watch server: npm start;",
			shouldFail:  false,
			description: "Single watch command only",
		},
		{
			name:        "single_stop_command",
			input:       "stop server: pkill npm;",
			shouldFail:  false,
			description: "Single stop command only",
		},
		{
			name:        "watch_stop_pair",
			input:       "watch server: npm start;\nstop server: pkill npm;",
			shouldFail:  false,
			description: "Watch and stop command pair",
		},
		{
			name:        "multiple_watch_commands",
			input:       "watch frontend: npm start;\nwatch backend: go run main.go;\nwatch db: docker run postgres;",
			shouldFail:  false,
			description: "Multiple independent watch commands",
		},
		{
			name:        "mixed_regular_and_watch",
			input:       "build: go build;\nwatch server: npm start;\ntest: go test;",
			shouldFail:  false,
			description: "Mix of regular and watch commands",
		},
		{
			name:        "command_with_variables",
			input:       "def PORT = 8080;\nserver: go run main.go --port=$(PORT);",
			shouldFail:  false,
			description: "Command using variables",
		},
		{
			name:        "block_command_simple",
			input:       "build: { echo starting; go build; echo done }",
			shouldFail:  false,
			description: "Simple block command",
		},
		{
			name:        "block_command_with_sh",
			input:       "cleanup: { @sh(find . -name '*.tmp' -delete); echo cleaned }",
			shouldFail:  false,
			description: "Block command with @sh decorator",
		},
		{
			name:        "block_command_with_parallel",
			input:       "services: { @parallel: { server; client; worker } }",
			shouldFail:  false,
			description: "Block command with @parallel decorator",
		},
		{
			name:        "complex_nested_blocks",
			input:       "deploy: { echo starting; @parallel: { @sh(docker build .); @sh(npm run build) }; echo done }",
			shouldFail:  false,
			description: "Complex nested block with multiple decorators",
		},
		{
			name:        "command_name_edge_cases",
			input:       "build-all: echo ok;\ntest_unit: echo ok;\nserver-dev: echo ok;\napi_v2: echo ok;",
			shouldFail:  false,
			description: "Various command name formats",
		},
		{
			name:        "long_command_line",
			input:       "test: go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout=5m ./... && go tool cover -html=coverage.out -o coverage.html;",
			shouldFail:  false,
			description: "Very long command line",
		},
		{
			name:        "commands_with_pipes_and_redirects",
			input:       "logs: tail -f app.log | grep ERROR > errors.log;\nbackup: tar czf backup.tar.gz . 2>/dev/null;",
			shouldFail:  false,
			description: "Commands with pipes and redirects",
		},
		{
			name:        "multiline_commands",
			input:       "build: { echo 'Building...'; go build -o bin/app; echo 'Done' }",
			shouldFail:  false,
			description: "Multi-line block commands",
		},
		{
			name: "real_world_commands",
			input: `
def PORT = 8080;
def BUILD_DIR = ./dist;

# Regular commands
build: {
    echo "Building application...";
    mkdir -p $(BUILD_DIR);
    go build -o $(BUILD_DIR)/app .;
    echo "Build complete"
}

test: {
    echo "Running tests...";
    go test -v ./...;
    echo "Tests complete"
}

# Watch commands
watch server: {
    echo "Starting server on port $(PORT)...";
    @sh(cd cmd/server && go run main.go --port=$(PORT))
}

stop server: {
    echo "Stopping server...";
    @sh(pkill -f "go run.*cmd/server" || true)
}

# Complex parallel command
deploy: {
    echo "Deploying application...";
    @parallel: {
        @sh(docker build -t myapp .);
        @sh(npm run build);
        go test ./...
    };
    echo "Deployment ready"
}
`,
			shouldFail:  false,
			description: "Real-world complex commands file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			cf, err := devcmdParser.Parse(tt.input, false)
			if err != nil {
				if tt.shouldFail {
					t.Logf("Expected parse failure for %s: %v", tt.description, err)
					return
				}
				t.Fatalf("Parse error for %s: %v", tt.description, err)
			}

			// Expand variables
			err = cf.ExpandVariables()
			if err != nil {
				if tt.shouldFail {
					t.Logf("Expected variable expansion failure for %s: %v", tt.description, err)
					return
				}
				t.Fatalf("ExpandVariables error for %s: %v", tt.description, err)
			}

			// Generate Go code
			generated, err := GenerateGo(cf)
			if err != nil {
				if tt.shouldFail {
					t.Logf("Expected generation failure for %s: %v", tt.description, err)
					return
				}
				t.Errorf("GenerateGo error for %s: %v", tt.description, err)
				t.Logf("Input was: %s", tt.input)
				return
			}

			// Check if generated code is valid Go
			if !isValidGoCode(t, generated) {
				t.Errorf("Generated invalid Go code for %s", tt.description)
				t.Logf("Generated code:\n%s", generated)
				t.Logf("Input was: %s", tt.input)
				return
			}

			// Try actual compilation
			if err := testActualCompilation(t, generated); err != nil {
				t.Errorf("Generated code failed compilation for %s: %v", tt.description, err)
				t.Logf("Generated code:\n%s", generated)
				t.Logf("Input was: %s", tt.input)
				return
			}

			t.Logf("✅ %s: %s", tt.name, tt.description)
		})
	}
}

// TestMasterTemplateAssembly tests the master template assembly process
func TestMasterTemplateAssembly(t *testing.T) {
	registry := NewTemplateRegistry()

	// Test with minimal data
	minimalData := &TemplateData{
		PackageName:    "main",
		Imports:        []string{"fmt", "os"},
		HasProcessMgmt: false,
		Commands:       []TemplateCommand{},
	}

	// Test with watch command data
	watchData := &TemplateData{
		PackageName:    "main",
		Imports:        []string{"fmt", "os", "os/exec", "syscall", "time"},
		HasProcessMgmt: true,
		Commands: []TemplateCommand{
			{
				Name:            "server",
				FunctionName:    "runServer",
				GoCase:          "server",
				Type:            "watch-only",
				WatchCommand:    "npm start",
				IsBackground:    true,
				HelpDescription: "server start|stop|logs",
			},
		},
	}

	// Test both scenarios
	testCases := []struct {
		name string
		data *TemplateData
	}{
		{"minimal", minimalData},
		{"with_watch", watchData},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allTemplates := registry.GetAllTemplates()

			// Parse the complete template
			tmpl, err := template.New("go-cli").Parse(allTemplates)
			if err != nil {
				t.Fatalf("Failed to parse complete template: %v", err)
			}

			// Execute the main template
			var buf strings.Builder
			err = tmpl.ExecuteTemplate(&buf, "main", tc.data)
			if err != nil {
				t.Fatalf("Failed to execute main template: %v", err)
			}

			result := buf.String()

			// Check basic structure
			if !strings.Contains(result, "package main") {
				t.Error("Generated code missing package declaration")
			}

			if !strings.Contains(result, "func main()") {
				t.Error("Generated code missing main function")
			}

			// Check imports
			if tc.data.HasProcessMgmt {
				if !strings.Contains(result, "syscall") {
					t.Error("Watch commands should include syscall import")
				}
			}

			// Validate Go syntax
			if !isValidGoCode(t, result) {
				t.Errorf("Generated invalid Go code for %s", tc.name)
				t.Logf("Generated code:\n%s", result)
			}
		})
	}
}

// TestImportsTemplateSpecifically focuses on the imports template that's causing issues
func TestImportsTemplateSpecifically(t *testing.T) {
	registry := NewTemplateRegistry()
	importsTemplate, exists := registry.GetTemplate("imports")
	if !exists {
		t.Fatal("Imports template not found")
	}

	testCases := []struct {
		name    string
		imports []string
	}{
		{"empty_imports", []string{}},
		{"single_import", []string{"fmt"}},
		{"multiple_imports", []string{"fmt", "os", "os/exec"}},
		{"many_imports", []string{"fmt", "os", "os/exec", "strings", "syscall", "time", "encoding/json"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := &TemplateData{
				Imports: tc.imports,
			}

			tmpl, err := template.New("imports").Parse(importsTemplate)
			if err != nil {
				t.Fatalf("Failed to parse imports template: %v", err)
			}

			var buf strings.Builder
			err = tmpl.Execute(&buf, data)
			if err != nil {
				t.Fatalf("Failed to execute imports template: %v", err)
			}

			result := buf.String()
			t.Logf("Imports result for %s:\n%s", tc.name, result)

			// Check for malformed imports
			if len(tc.imports) == 0 {
				// Empty imports should generate valid but minimal import block
				expected := "import (\n)\n"
				if !strings.Contains(result, expected) {
					t.Errorf("Empty imports generated unexpected result: %q", result)
				}
			} else {
				// Should contain import block
				if !strings.Contains(result, "import (") {
					t.Error("Missing import block start")
				}
				if !strings.Contains(result, ")") {
					t.Error("Missing import block end")
				}

				// Should contain all specified imports
				for _, imp := range tc.imports {
					expected := fmt.Sprintf(`"%s"`, imp)
					if !strings.Contains(result, expected) {
						t.Errorf("Missing import %s in result", imp)
					}
				}
			}

			// Create a minimal Go file with just the imports to test
			testCode := fmt.Sprintf("package main\n%s\nfunc main() {}", result)
			if !isValidGoCode(t, testCode) {
				t.Errorf("Generated imports create invalid Go code: %s", testCode)
			}
		})
	}
}

// TestTemplateExecutionOrder tests the order of template execution
func TestTemplateExecutionOrder(t *testing.T) {
	// This test helps identify if template execution order causes issues
	registry := NewTemplateRegistry()

	data := &TemplateData{
		PackageName:    "main",
		Imports:        []string{"fmt", "os"},
		HasProcessMgmt: false,
		Commands: []TemplateCommand{
			{
				Name:            "test",
				FunctionName:    "runTest",
				GoCase:          "test",
				Type:            "regular",
				ShellCommand:    "echo test",
				HelpDescription: "test",
			},
		},
	}

	// Get all templates and examine the master template
	allTemplates := registry.GetAllTemplates()
	t.Logf("Complete template length: %d", len(allTemplates))

	masterTemplate := registry.GetMasterTemplate()
	t.Logf("Master template: %s", masterTemplate)

	// Parse step by step
	tmpl, err := template.New("go-cli").Parse(allTemplates)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// List all defined templates
	for _, definedTmpl := range tmpl.Templates() {
		t.Logf("Defined template: %s", definedTmpl.Name())
	}

	// Execute
	var buf strings.Builder
	err = tmpl.ExecuteTemplate(&buf, "main", data)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result := buf.String()
	lines := strings.Split(result, "\n")
	t.Logf("Generated %d lines of code", len(lines))

	// Show first 20 lines for debugging
	for i, line := range lines {
		if i >= 20 {
			break
		}
		t.Logf("Line %d: %s", i+1, line)
	}
}

// TestRealWorldScenario tests a real-world commands.cli scenario
func TestRealWorldScenario(t *testing.T) {
	// This is similar to your actual commands.cli
	realWorldInput := `
def SERVER_PORT = 8080;
def FRONTEND_PORT = 4200;
def BUILD_DIR = ./dist;

watch server: {
    echo "🚀 Starting Go server on port $(SERVER_PORT)...";
    @sh(cd cmd/ailuvia && go run main.go)
}

stop server: {
    echo "🛑 Stopping Go server...";
    @sh(pkill -f "go run.*cmd/ailuvia" || true)
}

watch frontend: {
    echo "⚡ Starting Angular dev server on port $(FRONTEND_PORT)...";
    @sh(cd frontend && ng serve --port=$(FRONTEND_PORT))
}

stop frontend: {
    echo "🛑 Stopping Angular dev server...";
    @sh(pkill -f "ng serve" || true)
}

build: {
    echo "🔨 Building Go server...";
    mkdir -p $(BUILD_DIR);
    @sh(cd cmd/ailuvia && go build -o ../../$(BUILD_DIR)/server .)
}

test: {
    echo "🧪 Running Go tests...";
    go test -v ./...
}
`

	t.Run("real_world_full_test", func(t *testing.T) {
		// Parse
		cf, err := devcmdParser.Parse(realWorldInput, false)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		// Expand variables
		err = cf.ExpandVariables()
		if err != nil {
			t.Fatalf("ExpandVariables error: %v", err)
		}

		// Generate
		generated, err := GenerateGo(cf)
		if err != nil {
			t.Fatalf("GenerateGo error: %v", err)
		}

		// Validate
		if !isValidGoCode(t, generated) {
			t.Errorf("Generated invalid Go code")

			// Show the problematic code
			lines := strings.Split(generated, "\n")
			for i, line := range lines {
				t.Logf("Line %d: %s", i+1, line)
				if i >= 50 { // Show first 50 lines
					t.Logf("... (truncated)")
					break
				}
			}
		}

		// Test compilation
		if err := testActualCompilation(t, generated); err != nil {
			t.Errorf("Compilation failed: %v", err)
		}

		t.Logf("✅ Real world scenario passed")
	})
}

// Helper functions (keeping existing ones from the original test file)

func isValidGoCode(t *testing.T, code string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "generated.go", code, parser.ParseComments)
	if err != nil {
		t.Logf("Go parsing error: %v", err)
		return false
	}
	return true
}

func testActualCompilation(t *testing.T, code string) error {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "devcmd_test_*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			t.Logf("Warning: failed to clean up temp directory %s: %v", tmpDir, removeErr)
		}
	}()

	// Write the generated code to a temporary file
	tmpFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpFile, []byte(code), 0o644); err != nil {
		return err
	}

	// Initialize go.mod in the temporary directory
	cmd := exec.Command("go", "mod", "init", "testmodule")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Try to compile the code
	cmd = exec.Command("go", "build", "-o", "/dev/null", tmpFile)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Compilation output: %s", string(output))
		return err
	}

	return nil
}
