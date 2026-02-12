package parser

import (
	"testing"
)

// TestVarShellFunIntegration tests parsing a file with mixed var, shell, and fun statements
func TestVarShellFunIntegration(t *testing.T) {
	input := `var env = "production"
var replicas = 3

echo "Starting deployment"
kubectl get pods

fun deploy(service String) {
  echo "Deploying service"
  kubectl apply -f deployment.yaml
}

echo "Deployment complete"
kubectl get deployments`

	tree := Parse([]byte(input))

	// Should parse without errors
	if len(tree.Errors) > 0 {
		t.Errorf("unexpected parse errors:")
		for _, err := range tree.Errors {
			t.Errorf("  %s at line %d: %s", err.Context, err.Position.Line, err.Message)
		}
	}

	// Should have events
	if len(tree.Events) == 0 {
		t.Fatal("expected events but got none")
	}

	// Count node types to verify structure
	var varCount, shellCount, funCount int
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen {
			switch NodeKind(evt.Data) {
			case NodeVarDecl:
				varCount++
			case NodeShellCommand:
				shellCount++
			case NodeFunction:
				funCount++
			}
		}
	}

	// Verify we got the expected structure
	// 2 vars: env, replicas
	// 6 shell commands: 2 before function, 2 inside function, 2 after function
	// 1 function: deploy
	if varCount != 2 {
		t.Errorf("expected 2 var declarations, got %d", varCount)
	}
	if shellCount != 6 {
		t.Errorf("expected 6 shell commands (2 top-level + 2 in function + 2 top-level), got %d", shellCount)
	}
	if funCount != 1 {
		t.Errorf("expected 1 function, got %d", funCount)
	}

	t.Logf("Successfully parsed: %d vars, %d shell commands, %d functions", varCount, shellCount, funCount)
}
