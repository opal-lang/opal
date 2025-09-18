package transform_test

import (
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/ir"
	"github.com/aledsdavies/devcmd/core/transform"
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
			node, err := transform.TransformCommand(tt.input)

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

			node, err := transform.TransformCommand(cmd)
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

// TestTransformActionDecorator tests action decorator transformation
func TestTransformActionDecorator(t *testing.T) {
	tests := []struct {
		name     string
		input    *ast.ActionDecorator
		wantName string
		wantArgs int
	}{
		{
			name: "simple action without args",
			input: &ast.ActionDecorator{
				Name: "confirm",
				Args: []ast.NamedParameter{},
			},
			wantName: "confirm",
			wantArgs: 0,
		},
		{
			name: "action with string arg",
			input: &ast.ActionDecorator{
				Name: "log",
				Args: []ast.NamedParameter{
					{Name: "", Value: ast.Str("Building project...")},
				},
			},
			wantName: "log",
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ast.CommandDecl{
				Name: "test",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{tt.input},
				},
			}

			node, err := transform.TransformCommand(cmd)
			require.NoError(t, err)

			seq, ok := node.(ir.CommandSeq)
			require.True(t, ok)
			require.Len(t, seq.Steps, 1)

			step := seq.Steps[0]
			require.Len(t, step.Chain, 1)

			element := step.Chain[0]
			assert.Equal(t, ir.ElementKindAction, element.Kind)
			assert.Equal(t, tt.wantName, element.Name)
			assert.Len(t, element.Args, tt.wantArgs)
		})
	}
}

// TestTransformBlockDecorator tests block decorator transformation
func TestTransformBlockDecorator(t *testing.T) {
	tests := []struct {
		name       string
		input      *ast.BlockDecorator
		wantKind   string
		wantParams int
		wantInner  int
	}{
		{
			name: "retry block with simple content",
			input: &ast.BlockDecorator{
				Name: "retry",
				Args: []ast.NamedParameter{
					{Name: "times", Value: ast.Num("3")},
				},
				Content: []ast.CommandContent{
					ast.Shell(ast.Text("npm install")),
				},
			},
			wantKind:   "retry",
			wantParams: 1,
			wantInner:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ast.CommandDecl{
				Name: "test",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{tt.input},
				},
			}

			node, err := transform.TransformCommand(cmd)
			require.NoError(t, err)

			wrapper, ok := node.(ir.Wrapper)
			require.True(t, ok, "expected Wrapper, got %T", node)

			assert.Equal(t, tt.wantKind, wrapper.Kind)
			assert.Len(t, wrapper.Params, tt.wantParams)
			assert.Len(t, wrapper.Inner.Steps, tt.wantInner)
		})
	}
}

// TestTransformExpressions tests parameter expression transformation
func TestTransformExpressions(t *testing.T) {
	tests := []struct {
		name    string
		input   ast.Expression
		want    interface{}
		wantErr bool
	}{
		{
			name:  "string literal",
			input: ast.Str("hello world"),
			want:  "hello world",
		},
		{
			name:  "number literal int",
			input: ast.Num("42"),
			want:  42,
		},
		{
			name:  "number literal float",
			input: ast.Num("3.14"),
			want:  3.14,
		},
		{
			name:  "boolean true",
			input: ast.Bool(true),
			want:  true,
		},
		{
			name:  "boolean false",
			input: ast.Bool(false),
			want:  false,
		},
		{
			name:  "identifier",
			input: ast.Id("BUILD_DIR"),
			want:  "BUILD_DIR",
		},
		{
			name:    "invalid number",
			input:   ast.Num("not-a-number"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a block decorator with this expression as parameter
			block := &ast.BlockDecorator{
				Name: "test",
				Args: []ast.NamedParameter{
					{Name: "value", Value: tt.input},
				},
				Content: []ast.CommandContent{
					ast.Shell(ast.Text("echo test")),
				},
			}

			cmd := &ast.CommandDecl{
				Name: "test",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{block},
				},
			}

			node, err := transform.TransformCommand(cmd)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			wrapper, ok := node.(ir.Wrapper)
			require.True(t, ok)

			// Check parameter was transformed correctly
			value, exists := wrapper.Params["value"]
			require.True(t, exists)
			assert.Equal(t, tt.want, value)
		})
	}
}

// TestTransformComplexStructures tests complex nested structures
func TestTransformComplexStructures(t *testing.T) {
	t.Run("shell with mixed content", func(t *testing.T) {
		shell := ast.Shell(
			ast.Text("docker build -t "),
			ast.At("var", ast.UnnamedParam(ast.Id("IMAGE_NAME"))),
			ast.Text(":"),
			ast.At("env", ast.UnnamedParam(ast.Id("VERSION"))),
			ast.Text(" ."),
		)

		cmd := &ast.CommandDecl{
			Name: "test",
			Body: ast.CommandBody{Content: []ast.CommandContent{shell}},
		}

		node, err := transform.TransformCommand(cmd)
		require.NoError(t, err)

		seq := node.(ir.CommandSeq)
		require.Len(t, seq.Steps, 1)

		element := seq.Steps[0].Chain[0]
		assert.Equal(t, ir.ElementKindShell, element.Kind)
		require.NotNil(t, element.Content)

		parts := element.Content.Parts
		assert.Len(t, parts, 5)

		// Check structure: text, decorator, text, decorator, text
		assert.Equal(t, ir.PartKindLiteral, parts[0].Kind)
		assert.Equal(t, "docker build -t ", parts[0].Text)

		assert.Equal(t, ir.PartKindDecorator, parts[1].Kind)
		assert.Equal(t, "var", parts[1].DecoratorName)

		assert.Equal(t, ir.PartKindLiteral, parts[2].Kind)
		assert.Equal(t, ":", parts[2].Text)

		assert.Equal(t, ir.PartKindDecorator, parts[3].Kind)
		assert.Equal(t, "env", parts[3].DecoratorName)

		assert.Equal(t, ir.PartKindLiteral, parts[4].Kind)
		assert.Equal(t, " .", parts[4].Text)
	})

	t.Run("multiple commands in sequence", func(t *testing.T) {
		cmd := &ast.CommandDecl{
			Name: "test",
			Body: ast.CommandBody{
				Content: []ast.CommandContent{
					ast.Shell(ast.Text("npm install")),
					ast.Shell(ast.Text("npm test")),
					&ast.ActionDecorator{Name: "confirm", Args: []ast.NamedParameter{}},
					ast.Shell(ast.Text("npm run build")),
				},
			},
		}

		node, err := transform.TransformCommand(cmd)
		require.NoError(t, err)

		seq := node.(ir.CommandSeq)
		assert.Len(t, seq.Steps, 4)

		// Check each step type
		assert.Equal(t, ir.ElementKindShell, seq.Steps[0].Chain[0].Kind)
		assert.Equal(t, ir.ElementKindShell, seq.Steps[1].Chain[0].Kind)
		assert.Equal(t, ir.ElementKindAction, seq.Steps[2].Chain[0].Kind)
		assert.Equal(t, ir.ElementKindShell, seq.Steps[3].Chain[0].Kind)
	})
}
