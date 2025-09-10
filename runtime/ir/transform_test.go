package ir_test

import (
	"testing"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/ast"
	"github.com/aledsdavies/devcmd/runtime/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransformCommand tests the main AST→IR transformation entry point
func TestTransformCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   *ast.CommandDecl
		want    string // expected node type
		wantErr bool
	}{
		{
			name: "simple shell command",
			input: &ast.CommandDecl{
				Name: "test",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{
						ast.Shell(ast.Text("echo hello")),
					},
				},
			},
			want: "CommandSeq",
		},
		{
			name: "empty command",
			input: &ast.CommandDecl{
				Name: "empty",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{},
				},
			},
			want: "CommandSeq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ir.TransformCommand(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, node.NodeType())
		})
	}
}

// TestTransformShellContent tests shell content transformation with structured parts
func TestTransformShellContent(t *testing.T) {
	tests := []struct {
		name          string
		input         *ast.ShellContent
		wantElements  int
		wantParts     int // expected number of ContentParts in the shell element
		wantText      string
		wantDecorator string
	}{
		{
			name: "simple text only",
			input: ast.Shell(
				ast.Text("echo hello"),
			),
			wantElements: 1,
			wantParts:    1,
			wantText:     "echo hello",
		},
		{
			name: "text with value decorator",
			input: ast.Shell(
				ast.Text("echo "),
				ast.At("var", ast.UnnamedParam(ast.Id("NAME"))),
			),
			wantElements:  1,
			wantParts:     2,
			wantText:      "echo ",
			wantDecorator: "var",
		},
		{
			name: "multiple value decorators",
			input: ast.Shell(
				ast.Text("kubectl --context="),
				ast.At("env", ast.UnnamedParam(ast.Id("KUBE_CONTEXT"))),
				ast.Text(" apply -f "),
				ast.At("var", ast.UnnamedParam(ast.Id("BUILD_DIR"))),
				ast.Text("/k8s.yaml"),
			),
			wantElements:  1,
			wantParts:     5,
			wantText:      "kubectl --context=",
			wantDecorator: "env",
		},
		{
			name: "only value decorators",
			input: ast.Shell(
				ast.At("var", ast.UnnamedParam(ast.Id("CMD"))),
				ast.At("var", ast.UnnamedParam(ast.Id("ARGS"))),
			),
			wantElements:  1,
			wantParts:     2,
			wantDecorator: "var",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal command to test transformation
			cmd := &ast.CommandDecl{
				Name: "test",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{tt.input},
				},
			}

			node, err := ir.TransformCommand(cmd)
			require.NoError(t, err)

			// Should be CommandSeq
			seq, ok := node.(ir.CommandSeq)
			require.True(t, ok, "expected CommandSeq, got %T", node)
			require.Len(t, seq.Steps, 1, "expected exactly 1 step")

			step := seq.Steps[0]
			assert.Len(t, step.Chain, tt.wantElements, "expected %d chain elements", tt.wantElements)

			if tt.wantElements > 0 {
				// Check the first (shell) element
				shellElem := step.Chain[0]
				assert.Equal(t, ir.ElementKindShell, shellElem.Kind)

				// Check structured content
				require.NotNil(t, shellElem.Content, "expected structured content")
				assert.Len(t, shellElem.Content.Parts, tt.wantParts, "expected %d content parts", tt.wantParts)

				if tt.wantParts > 0 {
					// Check first part
					firstPart := shellElem.Content.Parts[0]
					if tt.wantText != "" {
						assert.Equal(t, ir.PartKindLiteral, firstPart.Kind)
						assert.Equal(t, tt.wantText, firstPart.Text)
					}

					// Check for decorator parts
					if tt.wantDecorator != "" {
						var foundDecorator bool
						for _, part := range shellElem.Content.Parts {
							if part.Kind == ir.PartKindDecorator {
								assert.Equal(t, tt.wantDecorator, part.DecoratorName)
								foundDecorator = true
								break
							}
						}
						assert.True(t, foundDecorator, "expected to find decorator %s", tt.wantDecorator)
					}
				}
			}
		})
	}
}

// TestTransformShellChain tests shell chain transformation with operators
func TestTransformShellChain(t *testing.T) {
	// Create a shell chain: echo hello && echo world
	elements := []ir.ChainElement{
		{
			Kind: ir.ElementKindShell,
			Content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{Kind: ir.PartKindLiteral, Text: "echo hello"},
				},
			},
		},
		{
			Kind:   ir.ElementKindShell,
			OpNext: ir.ChainOpAnd,
			Content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{Kind: ir.PartKindLiteral, Text: "echo world"},
				},
			},
		},
	}

	// For now, create this manually since AST shell chain construction is complex
	// TODO: Add comprehensive shell chain tests when AST builder supports it

	t.Run("manual chain verification", func(t *testing.T) {
		assert.Len(t, elements, 2)
		assert.Equal(t, ir.ChainOpNone, elements[0].OpNext)
		assert.Equal(t, ir.ChainOpAnd, elements[1].OpNext)
	})
}

// TestElementContentResolve tests the structured content resolution
func TestElementContentResolve(t *testing.T) {
	// Create a test registry with value decorators
	registry := decorators.NewRegistry()

	// Mock value decorator for testing
	mockVar := &mockValueDecorator{name: "var"}
	registry.RegisterValue(mockVar)

	tests := []struct {
		name    string
		content *ir.ElementContent
		vars    map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "simple text only",
			content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{Kind: ir.PartKindLiteral, Text: "echo hello"},
				},
			},
			want: "echo hello",
		},
		{
			name: "text with var decorator",
			content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{Kind: ir.PartKindLiteral, Text: "echo "},
					{
						Kind:          ir.PartKindDecorator,
						DecoratorName: "var",
						DecoratorArgs: []decorators.DecoratorParam{
							{Value: "BUILD_DIR"},
						},
					},
				},
			},
			vars: map[string]string{"BUILD_DIR": "/build"},
			want: "echo /build",
		},
		{
			name: "multiple decorators",
			content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{Kind: ir.PartKindLiteral, Text: "docker build -t "},
					{
						Kind:          ir.PartKindDecorator,
						DecoratorName: "var",
						DecoratorArgs: []decorators.DecoratorParam{
							{Value: "IMAGE_NAME"},
						},
					},
					{Kind: ir.PartKindLiteral, Text: ":"},
					{
						Kind:          ir.PartKindDecorator,
						DecoratorName: "var",
						DecoratorArgs: []decorators.DecoratorParam{
							{Value: "IMAGE_TAG"},
						},
					},
				},
			},
			vars: map[string]string{
				"IMAGE_NAME": "myapp",
				"IMAGE_TAG":  "v1.0",
			},
			want: "docker build -t myapp:v1.0",
		},
		{
			name: "unknown decorator",
			content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{
						Kind:          ir.PartKindDecorator,
						DecoratorName: "unknown",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with test variables
			ctx := &decorators.Ctx{
				Vars: tt.vars,
			}

			result, err := tt.content.Resolve(ctx, registry)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

// TestChainElementGetResolvedText tests the convenience method
func TestChainElementGetResolvedText(t *testing.T) {
	registry := decorators.NewRegistry()
	mockVar := &mockValueDecorator{name: "var"}
	registry.RegisterValue(mockVar)

	t.Run("with structured content", func(t *testing.T) {
		elem := ir.ChainElement{
			Kind: ir.ElementKindShell,
			Content: &ir.ElementContent{
				Parts: []ir.ContentPart{
					{Kind: ir.PartKindLiteral, Text: "echo "},
					{
						Kind:          ir.PartKindDecorator,
						DecoratorName: "var",
						DecoratorArgs: []decorators.DecoratorParam{
							{Value: "MESSAGE"},
						},
					},
				},
			},
		}

		ctx := &decorators.Ctx{
			Vars: map[string]string{"MESSAGE": "hello world"},
		}

		result, err := elem.GetResolvedText(ctx, registry)
		require.NoError(t, err)
		assert.Equal(t, "echo hello world", result)
	})

	t.Run("missing structured content should error", func(t *testing.T) {
		elem := ir.ChainElement{
			Kind: ir.ElementKindShell,
			// Content is nil - this should error in clean greenfield code
		}

		ctx := &decorators.Ctx{}

		result, err := elem.GetResolvedText(ctx, registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing structured content")
		assert.Empty(t, result)
	})
}

// Mock value decorator for testing
type mockValueDecorator struct {
	name string
}

func (m *mockValueDecorator) Name() string                                  { return m.name }
func (m *mockValueDecorator) Description() string                           { return "Mock decorator" }
func (m *mockValueDecorator) ParameterSchema() []decorators.ParameterSchema { return nil }
func (m *mockValueDecorator) Examples() []decorators.Example                { return nil }
func (m *mockValueDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

func (m *mockValueDecorator) Render(ctx *decorators.Ctx, args []decorators.DecoratorParam) (string, error) {
	if len(args) == 0 {
		return "", nil
	}

	// Simple variable lookup
	varName, ok := args[0].Value.(string)
	if !ok {
		return "", nil
	}

	if value, exists := ctx.Vars[varName]; exists {
		return value, nil
	}

	return "", nil
}

func (m *mockValueDecorator) Describe(ctx *decorators.Ctx, args []decorators.DecoratorParam) plan.ExecutionStep {
	// Not needed for these tests
	return plan.ExecutionStep{}
}
