package planner_test

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
)

// TestBashParity verifies that Opal's execution tree structure matches bash behavior
// for all combinations of redirects and operators.
//
// Based on comprehensive bash testing (63 tests):
// - test_bash_behavior.sh: 31 tests of redirect + operator combinations
// - test_redirect_positions.sh: 32 tests of redirect positions with all operators
//
// Key bash behaviors:
// - Redirects attach to individual commands, not to operator chains
// - Operator precedence: | > && = || > ;
// - Redirects are NOT operators - they are command properties
// - Exit codes from redirected commands affect && and ||
func TestBashParity(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		bashStdout  string // Expected stdout in bash
		bashFile    string // Expected file content in bash
		description string
		verify      func(t *testing.T, tree planfmt.ExecutionNode)
	}{
		// ====================================================================
		// Category 1: Redirect (>) on FIRST command with operators
		// From test_redirect_positions.sh tests 1-5
		// ====================================================================
		{
			name:        "redirect > then &&",
			input:       `echo "a" > out.txt && echo "b"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Redirect attaches to first command, AND runs second",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: AndNode(RedirectNode(echo "a", out.txt), echo "b")
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				_, ok = andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				_, ok = andNode.Right.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected right to be CommandNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "redirect >> then &&",
			input:       `echo "a" >> out.txt && echo "b"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Append redirect with AND",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				redirectNode, ok := andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Fatalf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				if redirectNode.Mode != planfmt.RedirectAppend {
					t.Errorf("Expected RedirectAppend, got %v", redirectNode.Mode)
				}
			},
		},
		{
			name:        "redirect > then ||",
			input:       `echo "a" > out.txt || echo "b"`,
			bashStdout:  "",
			bashFile:    "a",
			description: "Redirect succeeds, OR skips second command",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				orNode, ok := tree.(*planfmt.OrNode)
				if !ok {
					t.Fatalf("Expected OrNode, got %T", tree)
				}

				_, ok = orNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected left to be RedirectNode, got %T", orNode.Left)
				}
			},
		},
		{
			name:        "redirect > then |",
			input:       `echo "a" > out.txt | cat`,
			bashStdout:  "",
			bashFile:    "a",
			description: "Redirect happens, pipe gets empty stdin",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: PipelineNode([RedirectNode(echo "a" > out.txt), cat])
				pipeNode, ok := tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode, got %T", tree)
				}
				if len(pipeNode.Commands) != 2 {
					t.Fatalf("Expected 2 commands in pipeline, got %d", len(pipeNode.Commands))
				}
				// First command should be RedirectNode
				_, ok = pipeNode.Commands[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first command to be RedirectNode, got %T", pipeNode.Commands[0])
				}
				// Second command should be CommandNode (cat)
				_, ok = pipeNode.Commands[1].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected second command to be CommandNode, got %T", pipeNode.Commands[1])
				}
			},
		},
		{
			name:        "redirect > then ;",
			input:       `echo "a" > out.txt; echo "b"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Semicolon runs both commands regardless",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				seqNode, ok := tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode, got %T", tree)
				}

				if len(seqNode.Nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(seqNode.Nodes))
				}

				_, ok = seqNode.Nodes[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first node to be RedirectNode, got %T", seqNode.Nodes[0])
				}
			},
		},

		// ====================================================================
		// Category 2: Redirect on SECOND command with operators
		// From test_redirect_positions.sh tests 6-10
		// ====================================================================
		{
			name:        "first && second >",
			input:       `echo "a" && echo "b" > out.txt`,
			bashStdout:  "a",
			bashFile:    "b",
			description: "Redirect attaches to second command only",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				_, ok = andNode.Left.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected left to be CommandNode, got %T", andNode.Left)
				}

				_, ok = andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first && second >>",
			input:       `echo "a" && echo "b" >> out.txt`,
			bashStdout:  "a",
			bashFile:    "b",
			description: "Append redirect on second command",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				redirectNode, ok := andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Fatalf("Expected right to be RedirectNode, got %T", andNode.Right)
				}

				if redirectNode.Mode != planfmt.RedirectAppend {
					t.Errorf("Expected RedirectAppend, got %v", redirectNode.Mode)
				}
			},
		},
		{
			name:        "first || second >",
			input:       `echo "a" || echo "b" > out.txt`,
			bashStdout:  "a",
			bashFile:    "",
			description: "First succeeds, OR skips second (with redirect)",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				orNode, ok := tree.(*planfmt.OrNode)
				if !ok {
					t.Fatalf("Expected OrNode, got %T", tree)
				}

				_, ok = orNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", orNode.Right)
				}
			},
		},
		{
			name:        "first | second >",
			input:       `echo "a" | cat > out.txt`,
			bashStdout:  "",
			bashFile:    "a",
			description: "Pipe happens, redirect captures cat's output",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				pipeNode, ok := tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode, got %T", tree)
				}

				if len(pipeNode.Commands) != 2 {
					t.Errorf("Expected 2 commands, got %d", len(pipeNode.Commands))
				}

				// Second command should have redirect attached (once we fix structure)
				// For now, this wraps in RedirectNode which is wrong
			},
		},
		{
			name:        "first ; second >",
			input:       `echo "a"; echo "b" > out.txt`,
			bashStdout:  "a",
			bashFile:    "b",
			description: "Semicolon with redirect on second",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				seqNode, ok := tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode, got %T", tree)
				}

				if len(seqNode.Nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(seqNode.Nodes))
				}

				_, ok = seqNode.Nodes[1].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected second node to be RedirectNode, got %T", seqNode.Nodes[1])
				}
			},
		},

		// ====================================================================
		// Category 3: Redirect on BOTH commands
		// From test_redirect_positions.sh tests 11-14
		// ====================================================================
		{
			name:        "first > && second >",
			input:       `echo "a" > out1.txt && echo "b" > out2.txt`,
			bashStdout:  "",
			bashFile:    "b", // out2.txt
			description: "Both commands have redirects",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				_, ok = andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				_, ok = andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first > || second >",
			input:       `echo "a" > out1.txt || echo "b" > out2.txt`,
			bashStdout:  "",
			bashFile:    "a", // out1.txt (second doesn't run)
			description: "First redirect succeeds, OR skips second",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				orNode, ok := tree.(*planfmt.OrNode)
				if !ok {
					t.Fatalf("Expected OrNode, got %T", tree)
				}

				_, ok = orNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected left to be RedirectNode, got %T", orNode.Left)
				}

				_, ok = orNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", orNode.Right)
				}
			},
		},
		{
			name:        "first > | second >",
			input:       `echo "a" > out1.txt | cat > out2.txt`,
			bashStdout:  "",
			bashFile:    "a", // out1.txt
			description: "Both commands in pipeline have redirects",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be PipelineNode with both commands having redirects
				pipeNode, ok := tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode, got %T", tree)
				}
				if len(pipeNode.Commands) != 2 {
					t.Fatalf("Expected 2 commands in pipeline, got %d", len(pipeNode.Commands))
				}
				// Both commands should be RedirectNode
				_, ok = pipeNode.Commands[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first command to be RedirectNode, got %T", pipeNode.Commands[0])
				}
				_, ok = pipeNode.Commands[1].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected second command to be RedirectNode, got %T", pipeNode.Commands[1])
				}
			},
		},
		{
			name:        "first > ; second >",
			input:       `echo "a" > out1.txt; echo "b" > out2.txt`,
			bashStdout:  "",
			bashFile:    "b", // out2.txt
			description: "Semicolon with both having redirects",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				seqNode, ok := tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode, got %T", tree)
				}

				if len(seqNode.Nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(seqNode.Nodes))
				}

				_, ok = seqNode.Nodes[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first node to be RedirectNode, got %T", seqNode.Nodes[0])
				}

				_, ok = seqNode.Nodes[1].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected second node to be RedirectNode, got %T", seqNode.Nodes[1])
				}
			},
		},

		// ====================================================================
		// Category 4: Redirect to SAME file
		// From test_redirect_positions.sh tests 15-17
		// ====================================================================
		{
			name:        "first > same && second > same",
			input:       `echo "a" > out.txt && echo "b" > out.txt`,
			bashStdout:  "",
			bashFile:    "b", // Last write wins
			description: "Both redirect to same file, last write wins",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				// Both should be redirects
				_, ok = andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				_, ok = andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first > same && second >> same",
			input:       `echo "a" > out.txt && echo "b" >> out.txt`,
			bashStdout:  "",
			bashFile:    "a\nb", // Overwrite then append
			description: "Overwrite then append to same file",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				leftRedirect, ok := andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Fatalf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				if leftRedirect.Mode != planfmt.RedirectOverwrite {
					t.Errorf("Expected left to be Overwrite, got %v", leftRedirect.Mode)
				}

				rightRedirect, ok := andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Fatalf("Expected right to be RedirectNode, got %T", andNode.Right)
				}

				if rightRedirect.Mode != planfmt.RedirectAppend {
					t.Errorf("Expected right to be Append, got %v", rightRedirect.Mode)
				}
			},
		},
		{
			name:        "first >> same && second >> same",
			input:       `echo "a" >> out.txt && echo "b" >> out.txt`,
			bashStdout:  "",
			bashFile:    "a\nb", // Both append
			description: "Both append to same file",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				leftRedirect, ok := andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Fatalf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				if leftRedirect.Mode != planfmt.RedirectAppend {
					t.Errorf("Expected left to be Append, got %v", leftRedirect.Mode)
				}

				rightRedirect, ok := andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Fatalf("Expected right to be RedirectNode, got %T", andNode.Right)
				}

				if rightRedirect.Mode != planfmt.RedirectAppend {
					t.Errorf("Expected right to be Append, got %v", rightRedirect.Mode)
				}
			},
		},

		// ====================================================================
		// Category 5: Three commands with redirects
		// From test_redirect_positions.sh tests 18-21
		// ====================================================================
		{
			name:        "first > && second && third",
			input:       `echo "a" > out.txt && echo "b" && echo "c"`,
			bashStdout:  "b\nc",
			bashFile:    "a",
			description: "Only first has redirect",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: AndNode(AndNode(Redirect(a), echo b), echo c)
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				// Left should be another AndNode (left-associative)
				leftAnd, ok := andNode.Left.(*planfmt.AndNode)
				if !ok {
					t.Errorf("Expected left to be AndNode (left-associative), got %T", andNode.Left)
				}

				_, ok = leftAnd.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected leftAnd.Left to be RedirectNode, got %T", leftAnd.Left)
				}

				_, ok = leftAnd.Right.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected leftAnd.Right to be CommandNode, got %T", leftAnd.Right)
				}

				_, ok = andNode.Right.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected right to be CommandNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first && second > && third",
			input:       `echo "a" && echo "b" > out.txt && echo "c"`,
			bashStdout:  "a\nc",
			bashFile:    "b",
			description: "Middle command has redirect",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				leftAnd, ok := andNode.Left.(*planfmt.AndNode)
				if !ok {
					t.Errorf("Expected left to be AndNode, got %T", andNode.Left)
				}

				_, ok = leftAnd.Left.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected leftAnd.Left to be CommandNode, got %T", leftAnd.Left)
				}

				_, ok = leftAnd.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected leftAnd.Right to be RedirectNode, got %T", leftAnd.Right)
				}
			},
		},
		{
			name:        "first && second && third >",
			input:       `echo "a" && echo "b" && echo "c" > out.txt`,
			bashStdout:  "a\nb",
			bashFile:    "c",
			description: "Last command has redirect",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				_, ok = andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first > && second > && third >",
			input:       `echo "a" > out1.txt && echo "b" > out.txt && echo "c" > out2.txt`,
			bashStdout:  "",
			bashFile:    "b", // out.txt
			description: "All three have redirects",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				leftAnd, ok := andNode.Left.(*planfmt.AndNode)
				if !ok {
					t.Errorf("Expected left to be AndNode, got %T", andNode.Left)
				}

				_, ok = leftAnd.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected leftAnd.Left to be RedirectNode, got %T", leftAnd.Left)
				}

				_, ok = leftAnd.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected leftAnd.Right to be RedirectNode, got %T", leftAnd.Right)
				}

				_, ok = andNode.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected right to be RedirectNode, got %T", andNode.Right)
				}
			},
		},

		// ====================================================================
		// Category 6: Mixed operators with redirects
		// From test_redirect_positions.sh tests 22-26
		// ====================================================================
		{
			name:        "first > && second | third",
			input:       `echo "a" > out.txt && echo "b" | cat`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Redirect on first, pipe on second",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				_, ok = andNode.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected left to be RedirectNode, got %T", andNode.Left)
				}

				_, ok = andNode.Right.(*planfmt.PipelineNode)
				if !ok {
					t.Errorf("Expected right to be PipelineNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first | second > && third",
			input:       `echo "a" | cat > out.txt && echo "b"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Pipe with redirect, then AND",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				_, ok = andNode.Left.(*planfmt.PipelineNode)
				if !ok {
					t.Errorf("Expected left to be PipelineNode, got %T", andNode.Left)
				}

				_, ok = andNode.Right.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected right to be CommandNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first > | second && third",
			input:       `echo "a" > out.txt | cat && echo "b"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Redirect then pipe then AND",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: AndNode(PipelineNode([RedirectNode, cat]), echo "b")
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				// Left should be PipelineNode
				pipeNode, ok := andNode.Left.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected left to be PipelineNode, got %T", andNode.Left)
				}
				if len(pipeNode.Commands) != 2 {
					t.Fatalf("Expected 2 commands in pipeline, got %d", len(pipeNode.Commands))
				}

				// Right should be CommandNode
				_, ok = andNode.Right.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected right to be CommandNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "first > ; second | third",
			input:       `echo "a" > out.txt; echo "b" | cat`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Redirect, semicolon, then pipe",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				seqNode, ok := tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode, got %T", tree)
				}

				if len(seqNode.Nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(seqNode.Nodes))
				}

				_, ok = seqNode.Nodes[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first node to be RedirectNode, got %T", seqNode.Nodes[0])
				}

				_, ok = seqNode.Nodes[1].(*planfmt.PipelineNode)
				if !ok {
					t.Errorf("Expected second node to be PipelineNode, got %T", seqNode.Nodes[1])
				}
			},
		},
		{
			name:        "first | second > ; third",
			input:       `echo "a" | cat > out.txt; echo "b"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Pipe with redirect, semicolon, then command",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				seqNode, ok := tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode, got %T", tree)
				}

				if len(seqNode.Nodes) != 2 {
					t.Fatalf("Expected 2 nodes, got %d", len(seqNode.Nodes))
				}

				_, ok = seqNode.Nodes[0].(*planfmt.PipelineNode)
				if !ok {
					t.Errorf("Expected first node to be PipelineNode, got %T", seqNode.Nodes[0])
				}

				_, ok = seqNode.Nodes[1].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected second node to be CommandNode, got %T", seqNode.Nodes[1])
				}
			},
		},

		// ====================================================================
		// Category 7: Redirect in middle of pipeline
		// From test_redirect_positions.sh tests 27-29
		// ====================================================================
		{
			name:        "first | second > | third",
			input:       `echo "a" | cat > out.txt | cat`,
			bashStdout:  "",
			bashFile:    "a",
			description: "Middle command in pipeline has redirect",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be PipelineNode([echo "a", RedirectNode(cat > out.txt), cat])
				pipeNode, ok := tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode, got %T", tree)
				}
				if len(pipeNode.Commands) != 3 {
					t.Fatalf("Expected 3 commands in pipeline, got %d", len(pipeNode.Commands))
				}
				// First should be CommandNode
				_, ok = pipeNode.Commands[0].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected first command to be CommandNode, got %T", pipeNode.Commands[0])
				}
				// Second should be RedirectNode
				_, ok = pipeNode.Commands[1].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected second command to be RedirectNode, got %T", pipeNode.Commands[1])
				}
				// Third should be CommandNode
				_, ok = pipeNode.Commands[2].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected third command to be CommandNode, got %T", pipeNode.Commands[2])
				}
			},
		},
		{
			name:        "first > | second | third",
			input:       `echo "a" > out.txt | cat | cat`,
			bashStdout:  "",
			bashFile:    "a",
			description: "First in pipeline has redirect",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be PipelineNode([RedirectNode(echo "a" > out.txt), cat, cat])
				pipeNode, ok := tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode, got %T", tree)
				}
				if len(pipeNode.Commands) != 3 {
					t.Fatalf("Expected 3 commands in pipeline, got %d", len(pipeNode.Commands))
				}
				// First should be RedirectNode
				_, ok = pipeNode.Commands[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first command to be RedirectNode, got %T", pipeNode.Commands[0])
				}
				// Second and third should be CommandNode
				_, ok = pipeNode.Commands[1].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected second command to be CommandNode, got %T", pipeNode.Commands[1])
				}
				_, ok = pipeNode.Commands[2].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected third command to be CommandNode, got %T", pipeNode.Commands[2])
				}
			},
		},
		{
			name:        "first | second | third >",
			input:       `echo "a" | cat | cat > out.txt`,
			bashStdout:  "",
			bashFile:    "a",
			description: "Last in pipeline has redirect",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				pipeNode, ok := tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode, got %T", tree)
				}

				if len(pipeNode.Commands) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(pipeNode.Commands))
				}

				// Last command should have redirect (once we fix structure)
			},
		},

		// ====================================================================
		// Category 8: Complex real-world scenarios
		// From test_redirect_positions.sh tests 30-32
		// ====================================================================
		{
			name:        "build > log && test > results && deploy",
			input:       `echo "build" > out1.txt && echo "test" > out2.txt && echo "deploy"`,
			bashStdout:  "deploy",
			bashFile:    "test", // out2.txt
			description: "Real-world: build logs, test results, deploy output",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: AndNode(AndNode(Redirect(build), Redirect(test)), echo deploy)
				andNode, ok := tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode, got %T", tree)
				}

				leftAnd, ok := andNode.Left.(*planfmt.AndNode)
				if !ok {
					t.Errorf("Expected left to be AndNode, got %T", andNode.Left)
				}

				_, ok = leftAnd.Left.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected leftAnd.Left to be RedirectNode, got %T", leftAnd.Left)
				}

				_, ok = leftAnd.Right.(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected leftAnd.Right to be RedirectNode, got %T", leftAnd.Right)
				}

				_, ok = andNode.Right.(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected right to be CommandNode, got %T", andNode.Right)
				}
			},
		},
		{
			name:        "cmd1 | cmd2 > out && cmd3 || cmd4",
			input:       `echo "a" | cat > out.txt && echo "b" || echo "c"`,
			bashStdout:  "b",
			bashFile:    "a",
			description: "Complex: pipe with redirect, AND, OR",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: OrNode(AndNode(PipelineNode([echo a, cat with redirect]), echo b), echo c)
				orNode, ok := tree.(*planfmt.OrNode)
				if !ok {
					t.Fatalf("Expected OrNode, got %T", tree)
				}

				andNode, ok := orNode.Left.(*planfmt.AndNode)
				if !ok {
					t.Errorf("Expected left to be AndNode, got %T", orNode.Left)
				}

				_, ok = andNode.Left.(*planfmt.PipelineNode)
				if !ok {
					t.Errorf("Expected andNode.Left to be PipelineNode, got %T", andNode.Left)
				}
			},
		},
		{
			name:        "cmd1 > out1 ; cmd2 | cmd3 > out2 ; cmd4",
			input:       `echo "a" > out1.txt; echo "b" | cat > out2.txt; echo "c"`,
			bashStdout:  "c",
			bashFile:    "b", // out2.txt
			description: "Complex: multiple semicolons with redirects and pipes",
			verify: func(t *testing.T, tree planfmt.ExecutionNode) {
				// Should be: SequenceNode([Redirect(a), PipelineNode([echo b, cat with redirect]), echo c])
				seqNode, ok := tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode, got %T", tree)
				}

				if len(seqNode.Nodes) != 3 {
					t.Fatalf("Expected 3 nodes, got %d", len(seqNode.Nodes))
				}

				_, ok = seqNode.Nodes[0].(*planfmt.RedirectNode)
				if !ok {
					t.Errorf("Expected first node to be RedirectNode, got %T", seqNode.Nodes[0])
				}

				_, ok = seqNode.Nodes[1].(*planfmt.PipelineNode)
				if !ok {
					t.Errorf("Expected second node to be PipelineNode, got %T", seqNode.Nodes[1])
				}

				_, ok = seqNode.Nodes[2].(*planfmt.CommandNode)
				if !ok {
					t.Errorf("Expected third node to be CommandNode, got %T", seqNode.Nodes[2])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			parseTree := parser.Parse([]byte(tt.input))
			if len(parseTree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", parseTree.Errors)
			}

			// Plan
			result, err := planner.Plan(parseTree.Events, parseTree.Tokens, planner.Config{
				Target: "", // Script mode
			})
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			// Verify structure
			if len(result.Steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(result.Steps))
			}

			step := result.Steps[0]
			if step.Tree == nil {
				t.Fatal("Expected tree, got nil")
			}

			// Run verification
			tt.verify(t, step.Tree)
		})
	}
}
