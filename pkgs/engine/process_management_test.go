package engine

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestProcessManagementGeneration tests the generation of process management commands
func TestProcessManagementGeneration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected struct {
			hasProcessGroup bool
			identifier      string
			hasDefaultRun   bool
			hasRunSubcmd    bool
			hasStopSubcmd   bool
			hasStatusSubcmd bool
			hasLogsSubcmd   bool
		}
	}{
		{
			name: "watch command only",
			input: `
var PROJECT = "test-app"
watch web: echo "Starting @var(PROJECT) web server"
			`,
			expected: struct {
				hasProcessGroup bool
				identifier      string
				hasDefaultRun   bool
				hasRunSubcmd    bool
				hasStopSubcmd   bool
				hasStatusSubcmd bool
				hasLogsSubcmd   bool
			}{
				hasProcessGroup: true,
				identifier:      "web",
				hasDefaultRun:   true,
				hasRunSubcmd:    true,
				hasStopSubcmd:   true, // Generated with default logic
				hasStatusSubcmd: true, // Auto-generated
				hasLogsSubcmd:   true, // Auto-generated
			},
		},
		{
			name: "watch and stop commands",
			input: `
var PROJECT = "test-app"
watch api: echo "Starting @var(PROJECT) API server"
stop api: echo "Gracefully shutting down @var(PROJECT) API"
			`,
			expected: struct {
				hasProcessGroup bool
				identifier      string
				hasDefaultRun   bool
				hasRunSubcmd    bool
				hasStopSubcmd   bool
				hasStatusSubcmd bool
				hasLogsSubcmd   bool
			}{
				hasProcessGroup: true,
				identifier:      "api",
				hasDefaultRun:   true,
				hasRunSubcmd:    true,
				hasStopSubcmd:   true, // Uses custom stop logic
				hasStatusSubcmd: true, // Auto-generated
				hasLogsSubcmd:   true, // Auto-generated
			},
		},
		{
			name: "mixed regular and process commands",
			input: `
var PROJECT = "test-app"
build: echo "Building @var(PROJECT)"
watch dev: echo "Starting @var(PROJECT) development"
test: echo "Running tests for @var(PROJECT)"
stop dev: echo "Stopping @var(PROJECT) development"
			`,
			expected: struct {
				hasProcessGroup bool
				identifier      string
				hasDefaultRun   bool
				hasRunSubcmd    bool
				hasStopSubcmd   bool
				hasStatusSubcmd bool
				hasLogsSubcmd   bool
			}{
				hasProcessGroup: true,
				identifier:      "dev",
				hasDefaultRun:   true,
				hasRunSubcmd:    true,
				hasStopSubcmd:   true,
				hasStatusSubcmd: true,
				hasLogsSubcmd:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Create engine and generate code
			engine := New(program)
			result, err := engine.GenerateCode(program)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			generatedCode := result.String()

			if tt.expected.hasProcessGroup {
				// Check that process management structure is generated
				expectedMainCmd := tt.expected.identifier + "Cmd := &cobra.Command{"
				if !strings.Contains(generatedCode, expectedMainCmd) {
					t.Errorf("Expected main process command %s not found in generated code", expectedMainCmd)
				}

				if tt.expected.hasDefaultRun {
					// Check that main command has Run function for default behavior
					expectedDefaultRun := "Run:   " + toCamelCase(tt.expected.identifier) + "Run, // Default action is to run"
					if !strings.Contains(generatedCode, expectedDefaultRun) {
						t.Errorf("Expected default run behavior not found: %s", expectedDefaultRun)
					}
				}

				if tt.expected.hasRunSubcmd {
					// Check that explicit run subcommand exists
					expectedRunSubcmd := toCamelCase(tt.expected.identifier) + "RunCmd := &cobra.Command{"
					if !strings.Contains(generatedCode, expectedRunSubcmd) {
						t.Errorf("Expected run subcommand not found: %s", expectedRunSubcmd)
					}
				}

				if tt.expected.hasStopSubcmd {
					// Check that stop subcommand exists
					expectedStopSubcmd := toCamelCase(tt.expected.identifier) + "StopCmd := &cobra.Command{"
					if !strings.Contains(generatedCode, expectedStopSubcmd) {
						t.Errorf("Expected stop subcommand not found: %s", expectedStopSubcmd)
					}
				}

				if tt.expected.hasStatusSubcmd {
					// Check that status subcommand exists
					expectedStatusSubcmd := toCamelCase(tt.expected.identifier) + "StatusCmd := &cobra.Command{"
					if !strings.Contains(generatedCode, expectedStatusSubcmd) {
						t.Errorf("Expected status subcommand not found: %s", expectedStatusSubcmd)
					}
				}

				if tt.expected.hasLogsSubcmd {
					// Check that logs subcommand exists
					expectedLogsSubcmd := toCamelCase(tt.expected.identifier) + "LogsCmd := &cobra.Command{"
					if !strings.Contains(generatedCode, expectedLogsSubcmd) {
						t.Errorf("Expected logs subcommand not found: %s", expectedLogsSubcmd)
					}
				}

				// Check that all subcommands are added to main command
				expectedAddCommands := []string{
					tt.expected.identifier + "Cmd.AddCommand(" + toCamelCase(tt.expected.identifier) + "RunCmd)",
					tt.expected.identifier + "Cmd.AddCommand(" + toCamelCase(tt.expected.identifier) + "StopCmd)",
					tt.expected.identifier + "Cmd.AddCommand(" + toCamelCase(tt.expected.identifier) + "StatusCmd)",
					tt.expected.identifier + "Cmd.AddCommand(" + toCamelCase(tt.expected.identifier) + "LogsCmd)",
				}

				for _, expectedAdd := range expectedAddCommands {
					if !strings.Contains(generatedCode, expectedAdd) {
						t.Errorf("Expected subcommand addition not found: %s", expectedAdd)
					}
				}

				// Check that main command is added to root
				expectedAddToRoot := "rootCmd.AddCommand(" + toCamelCase(tt.expected.identifier) + "Cmd)"
				if !strings.Contains(generatedCode, expectedAddToRoot) {
					t.Errorf("Expected main command addition to root not found: %s", expectedAddToRoot)
				}
			}
		})
	}
}

// TestCommandAnalysis tests the command grouping logic
func TestCommandAnalysis(t *testing.T) {
	input := `
build: echo "Building"
watch web: echo "Starting web server"
test: echo "Testing"
stop web: echo "Stopping web server"
watch api: echo "Starting API"
	`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	engine := New(program)
	groups := engine.analyzeCommands(program.Commands)

	// Should have 2 regular commands: build, test
	if len(groups.RegularCommands) != 2 {
		t.Errorf("Expected 2 regular commands, got %d", len(groups.RegularCommands))
	}

	// Should have 2 process groups: web, api
	if len(groups.ProcessGroups) != 2 {
		t.Errorf("Expected 2 process groups, got %d", len(groups.ProcessGroups))
	}

	// Check that process groups are correctly formed
	processGroupMap := make(map[string]ProcessGroup)
	for _, group := range groups.ProcessGroups {
		processGroupMap[group.Identifier] = group
	}

	// Web group should have both watch and stop
	webGroup, hasWeb := processGroupMap["web"]
	if !hasWeb {
		t.Error("Expected web process group not found")
	} else {
		if webGroup.WatchCommand == nil {
			t.Error("Web group missing watch command")
		}
		if webGroup.StopCommand == nil {
			t.Error("Web group missing stop command")
		}
		if webGroup.WatchCommand != nil && webGroup.WatchCommand.Type != ast.WatchCommand {
			t.Error("Web watch command has wrong type")
		}
		if webGroup.StopCommand != nil && webGroup.StopCommand.Type != ast.StopCommand {
			t.Error("Web stop command has wrong type")
		}
	}

	// API group should have only watch (no stop)
	apiGroup, hasApi := processGroupMap["api"]
	if !hasApi {
		t.Error("Expected api process group not found")
	} else {
		if apiGroup.WatchCommand == nil {
			t.Error("API group missing watch command")
		}
		if apiGroup.StopCommand != nil {
			t.Error("API group should not have stop command")
		}
	}
}

// TestProcessManagementDefaultStopLogic tests default stop command generation
func TestProcessManagementDefaultStopLogic(t *testing.T) {
	input := `
watch web: echo "Starting web server"
	`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	generatedCode := result.String()

	// Should contain default pkill logic since no custom stop command provided
	expectedDefaultStop := `pkill -f 'web'`
	if !strings.Contains(generatedCode, expectedDefaultStop) {
		t.Errorf("Expected default stop logic not found: %s", expectedDefaultStop)
	}
}

// TestProcessManagementCustomStopLogic tests custom stop command generation
func TestProcessManagementCustomStopLogic(t *testing.T) {
	input := `
var PID_FILE = "/tmp/web.pid"
watch web: echo "Starting web server"
stop web: echo "Gracefully stopping" && kill -TERM @var(PID_FILE)
	`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	engine := New(program)
	result, err := engine.GenerateCode(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	generatedCode := result.String()

	// Should contain custom stop logic with resolved variable value, not default pkill
	// Look for the pattern in the plan output section which correctly resolves variables
	expectedCustomStop := `echo \"Gracefully stopping\" && kill -TERM /tmp/web.pid`
	if !strings.Contains(generatedCode, expectedCustomStop) {
		t.Errorf("Expected custom stop logic not found: %s", expectedCustomStop)
	}

	// Should NOT contain default pkill logic
	unexpectedDefaultStop := `pkill -f 'web'`
	if strings.Contains(generatedCode, unexpectedDefaultStop) {
		t.Errorf("Unexpected default stop logic found when custom stop provided: %s", unexpectedDefaultStop)
	}
}

// TestProcessManagementInterpreterMode tests that watch/stop commands work in interpreter mode
func TestProcessManagementInterpreterMode(t *testing.T) {
	input := `
var PROJECT = "test-app"
watch dev: echo "Starting @var(PROJECT) development server"
stop dev: echo "Stopping @var(PROJECT) development"
build: echo "Building @var(PROJECT)"
	`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	engine := New(program)

	// Test that regular commands still work
	buildCmd := findCommandByName(program.Commands, "build")
	if buildCmd == nil {
		t.Fatal("Build command not found")
	}

	result, err := engine.ExecuteCommand(buildCmd)
	if err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected build command to succeed, got status: %s", result.Status)
	}

	// Test that watch commands work
	watchCmd := findCommandByName(program.Commands, "dev")
	if watchCmd == nil {
		t.Fatal("Watch dev command not found")
	}

	if watchCmd.Type != ast.WatchCommand {
		t.Error("Dev command should be WatchCommand type")
	}

	result, err = engine.ExecuteCommand(watchCmd)
	if err != nil {
		t.Errorf("Failed to execute watch command: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected watch command to succeed, got status: %s", result.Status)
	}

	// Test that stop commands work
	stopCmd := findCommandByName(program.Commands, "dev")
	if stopCmd == nil {
		t.Fatal("Stop dev command not found")
	}

	// Find the actual stop command (both watch and stop have same name "dev")
	var actualStopCmd *ast.CommandDecl
	for i, cmd := range program.Commands {
		if cmd.Name == "dev" && cmd.Type == ast.StopCommand {
			actualStopCmd = &program.Commands[i]
			break
		}
	}

	if actualStopCmd == nil {
		t.Fatal("Stop command with StopCommand type not found")
	}

	result, err = engine.ExecuteCommand(actualStopCmd)
	if err != nil {
		t.Errorf("Failed to execute stop command: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected stop command to succeed, got status: %s", result.Status)
	}
}

// Helper function to find a command by name
func findCommandByName(commands []ast.CommandDecl, name string) *ast.CommandDecl {
	for i, cmd := range commands {
		if cmd.Name == name {
			return &commands[i]
		}
	}
	return nil
}
