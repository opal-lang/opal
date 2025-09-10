package parser

import (
	"strings"
	"testing"

	_ "github.com/aledsdavies/devcmd/runtime/decorators/builtin" // Register decorators for tests
)

// TestRealWorldBugs contains all tests for bugs found in actual usage of devcmd
// These tests ensure we don't regress on issues found in real .cli files
func TestRealWorldBugs(t *testing.T) {
	t.Run("commands_cli_setup_exact", func(t *testing.T) {
		// Extract the exact setup command that fails (lines 265-277 from commands.cli)
		input := `var PROJECT = "devcmd"

# Quick setup for new contributors
setup: {
    @log("🔧 Setting up @var(PROJECT) development environment...")
    @log("📦 Downloading Go dependencies for all modules...")
    @parallel {
        @cmd(core-deps)
        @cmd(codegen-deps)
        @cmd(runtime-deps)
        @cmd(testing-deps)
        @cmd(cli-deps)
    }
    go work sync
    @log("✅ Setup complete! Run 'dev ci' to verify everything works.")
}`

		reader := strings.NewReader(input)
		program, err := Parse(reader)

		if err != nil {
			t.Errorf("commands.cli setup command should parse but got error: %v", err)
			t.Logf("This is the EXACT code from commands.cli lines 265-277")

			// Log specific error location
			if strings.Contains(err.Error(), "unexpected LBRACE") {
				t.Logf("ERROR: Parser incorrectly treating @parallel { as unexpected")
			}
		} else if program == nil {
			t.Error("Expected non-nil program")
		} else {
			t.Logf("✅ commands.cli setup command parses correctly")
		}
	})

	t.Run("commands_cli_build_workdir", func(t *testing.T) {
		// Line 358 from commands.cli - @workdir with inline block
		input := `var PROJECT = "devcmd"
var VERSION = "1.0"
var BUILD_TIME = "2024"

build: {
    @log("🔨 Building binary...")
    @workdir("cli") { go build -ldflags="-s -w -X main.Version=@var(VERSION) -X main.BuildTime=@var(BUILD_TIME)" -o ../@var(PROJECT) ./main.go }
    @log("✅ Built: ./@var(PROJECT)")
}`

		reader := strings.NewReader(input)
		program, err := Parse(reader)

		if err != nil {
			t.Errorf("@workdir with inline block should parse but got error: %v", err)
			t.Logf("This tests the pattern from commands.cli line 358")
		} else if program == nil {
			t.Error("Expected non-nil program")
		} else {
			t.Logf("✅ @workdir with inline block parses correctly")
		}
	})

	t.Run("progressive_complexity", func(t *testing.T) {
		// Test increasingly complex patterns to identify where parsing breaks
		tests := []struct {
			name        string
			input       string
			description string
		}{
			{
				name: "single_log",
				input: `test: {
    @log("test")
}`,
				description: "Single @log decorator",
			},
			{
				name: "two_logs",
				input: `test: {
    @log("first")
    @log("second")
}`,
				description: "Two @log decorators",
			},
			{
				name: "log_then_parallel",
				input: `test: {
    @log("starting")
    @parallel {
        @cmd(test)
    }
}`,
				description: "@log followed by @parallel block",
			},
			{
				name: "two_logs_then_parallel",
				input: `test: {
    @log("first")
    @log("second")
    @parallel {
        @cmd(test)
    }
}`,
				description: "Two @logs then @parallel",
			},
			{
				name: "log_parallel_shell",
				input: `test: {
    @log("starting")
    @parallel {
        @cmd(test)
    }
    echo "done"
}`,
				description: "@log, @parallel, then shell command",
			},
			{
				name: "exact_pattern_with_vars",
				input: `var PROJECT = "test"
test: {
    @log("Setting up @var(PROJECT)")
    @log("Downloading...")
    @parallel {
        @cmd(deps)
    }
    go work sync
    @log("Complete!")
}`,
				description: "Exact commands.cli pattern with variables",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader := strings.NewReader(tt.input)
				program, err := Parse(reader)
				if err != nil {
					t.Errorf("%s should parse but got error: %v", tt.description, err)
					t.Logf("⚠️  Parsing breaks at: %s", tt.name)
					return
				}

				if program == nil {
					t.Error("Expected non-nil program")
					return
				}

				t.Logf("✅ %s", tt.description)
			})
		}
	})

	t.Run("action_then_block_decorator_patterns", func(t *testing.T) {
		// Test various combinations of action decorators followed by block decorators
		tests := []struct {
			name  string
			input string
		}{
			{
				name: "log_then_parallel",
				input: `test: {
    @log("Starting parallel tasks...")
    @parallel {
        @cmd(task1)
        @cmd(task2)
    }
}`,
			},
			{
				name: "log_then_timeout",
				input: `test: {
    @log("Running with timeout...")
    @timeout(30s) {
        npm test
    }
}`,
			},
			{
				name: "log_then_workdir",
				input: `test: {
    @log("Changing directory...")
    @workdir("subdir") {
        go build
    }
}`,
			},
			{
				name: "cmd_then_parallel",
				input: `test: {
    @cmd(setup)
    @parallel {
        @cmd(test1)
        @cmd(test2)
    }
}`,
			},
			{
				name: "multiple_actions_then_block",
				input: `test: {
    @log("Step 1")
    @cmd(prepare)
    @log("Step 2")
    @parallel {
        @cmd(run1)
        @cmd(run2)
    }
    @log("Done")
}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader := strings.NewReader(tt.input)
				program, err := Parse(reader)

				if err != nil {
					t.Errorf("%s pattern should parse but got error: %v", tt.name, err)
					if strings.Contains(err.Error(), "unexpected LBRACE") {
						t.Logf("⚠️  Block decorator after action decorator not recognized")
					}
				} else if program == nil {
					t.Error("Expected non-nil program")
				} else {
					t.Logf("✅ %s pattern works", tt.name)
				}
			})
		}
	})
}

// TestMinimalBugReproduction provides the absolute minimal case that reproduces the bug
func TestMinimalBugReproduction(t *testing.T) {
	input := `test: {
    @log("test")
    @parallel {
        @cmd(test)
    }
}`

	reader := strings.NewReader(input)
	program, err := Parse(reader)

	if err != nil {
		t.Errorf("Minimal bug reproduction failed: %v", err)
		t.Log("This is the simplest case that reproduces the @parallel bug")
		t.Log("Pattern: action decorator (@log) followed by block decorator (@parallel)")
	} else if program == nil {
		t.Error("Expected non-nil program")
	} else {
		t.Log("✅ Bug appears to be fixed!")
	}
}

// TestStringInterpolationImpact tests if @var in strings affects subsequent parsing
func TestStringInterpolationImpact(t *testing.T) {
	t.Run("comparison", func(t *testing.T) {
		tests := []struct {
			name        string
			input       string
			shouldPass  bool
			description string
		}{
			{
				name: "log_plain_then_parallel",
				input: `test: {
    @log("Starting tasks...")
    @parallel {
        @cmd(task1)
    }
}`,
				shouldPass:  true,
				description: "@log without @var + @parallel",
			},
			{
				name: "log_with_var_then_parallel",
				input: `var PROJECT = "test"
test: {
    @log("Starting @var(PROJECT) tasks...")
    @parallel {
        @cmd(task1)
    }
}`,
				shouldPass:  true,
				description: "@log WITH @var + @parallel (BUG HERE)",
			},
			{
				name: "double_log_with_var_then_parallel",
				input: `var PROJECT = "test"
test: {
    @log("First: @var(PROJECT)")
    @log("Second: @var(PROJECT)")
    @parallel {
        @cmd(task1)
    }
}`,
				shouldPass:  true,
				description: "Two @logs with @var + @parallel",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader := strings.NewReader(tt.input)
				program, err := Parse(reader)

				if tt.shouldPass {
					if err != nil {
						t.Errorf("%s should pass but got error: %v", tt.description, err)
						if strings.Contains(err.Error(), "unexpected LBRACE") {
							t.Log("⚠️  STRING INTERPOLATION BUG: @var in @log breaks subsequent @parallel")
						}
					} else if program == nil {
						t.Error("Expected non-nil program")
					} else {
						t.Logf("✅ %s works", tt.description)
					}
				}
			})
		}
	})
}
