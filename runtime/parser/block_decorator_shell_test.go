package parser

import (
	"strings"
	"testing"
)

// TestBlockDecoratorFollowedByShell tests the specific issue where a block decorator
// is followed by shell commands in the same command block
func TestBlockDecoratorFollowedByShell(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldPass  bool
		description string
	}{
		{
			name: "parallel_then_shell_command",
			input: `test: {
    @parallel {
        @cmd(build)
    }
    echo "done"
}`,
			shouldPass:  true,
			description: "Block decorator followed by shell command should work",
		},
		{
			name: "timeout_then_shell_command",
			input: `test: {
    @timeout(30s) {
        npm test
    }
    echo "tests completed"
}`,
			shouldPass:  true,
			description: "Timeout block followed by shell command should work",
		},
		{
			name: "commands_cli_exact_pattern",
			input: `setup: {
    @log("Starting...")
    @parallel {
        @cmd(deps)
    }
    go work sync
    @log("Done!")
}`,
			shouldPass:  true,
			description: "Exact commands.cli pattern should work",
		},
		{
			name: "multiple_shell_after_block",
			input: `deploy: {
    @parallel {
        @cmd(build)
        @cmd(test)
    }
    kubectl apply -f deployment.yaml
    kubectl rollout status deployment/app
}`,
			shouldPass:  true,
			description: "Multiple shell commands after block decorator should work",
		},
		{
			name: "action_block_shell_action_pattern",
			input: `complex: {
    @log("Step 1")
    @parallel {
        @cmd(task1)
        @cmd(task2)
    }
    echo "Parallel done"
    @log("Step 2")
}`,
			shouldPass:  true,
			description: "Mixed action/block/shell/action pattern should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			program, err := Parse(reader)

			if tt.shouldPass {
				if err != nil {
					t.Errorf("Expected %s to pass but got error: %v", tt.description, err)
					t.Logf("Input was:\n%s", tt.input)

					// Log the specific error details for debugging
					if strings.Contains(err.Error(), "unexpected") {
						t.Logf("🔍 Parser context issue detected!")
					}
				} else if program == nil {
					t.Errorf("Expected non-nil program for %s", tt.description)
				} else {
					t.Logf("✅ %s", tt.description)
				}
			} else {
				if err == nil {
					t.Errorf("Expected %s to fail but it passed", tt.description)
				} else {
					t.Logf("✅ Correctly failed: %s - %v", tt.description, err)
				}
			}
		})
	}
}

// TestMinimalReproduction creates the smallest possible reproduction of the issue
func TestMinimalReproduction(t *testing.T) {
	input := `test: {
    @parallel {
        @cmd(x)
    }
    echo done
}`

	reader := strings.NewReader(input)
	program, err := Parse(reader)

	if err != nil {
		t.Logf("🔍 MINIMAL REPRODUCTION OF BUG:")
		t.Logf("Error: %v", err)
		t.Logf("Input:\n%s", input)

		// This documents the exact bug for fixing
		if strings.Contains(err.Error(), "unexpected") {
			t.Logf("💡 Context switching issue after block decorator")
		}
	} else if program != nil {
		t.Log("✅ Bug has been fixed!")
	} else {
		t.Error("Unexpected nil program with no error")
	}
}
