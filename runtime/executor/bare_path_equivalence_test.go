package executor

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
	"github.com/opal-lang/opal/core/planfmt"
	_ "github.com/opal-lang/opal/runtime/decorators"
)

func TestBarePathEquivalence(t *testing.T) {
	t.Parallel()

	t.Run("output redirect bare path equals explicit file decorator", func(t *testing.T) {
		t.Parallel()

		workdir := t.TempDir()
		session := decorator.NewLocalSession().WithWorkdir(workdir)
		execCtx := decorator.ExecContext{Context: context.Background(), Session: session}

		relPath := filepath.Join("io", "out.txt")
		bareTarget := &planfmt.CommandNode{
			Decorator: "@shell",
			Args: []planfmt.Arg{{
				Key: "command",
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: relPath},
			}},
		}
		explicitTarget := &planfmt.CommandNode{
			Decorator: "@file",
			Args: []planfmt.Arg{{
				Key: "path",
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: relPath},
			}},
		}

		bareIO, bareIdentity, ok := resolvePlanIOSink(bareTarget)
		if !ok {
			t.Fatal("expected bare target to resolve")
		}
		explicitIO, explicitIdentity, ok := resolvePlanIOSink(explicitTarget)
		if !ok {
			t.Fatal("expected explicit target to resolve")
		}

		if diff := cmp.Diff(explicitIdentity, bareIdentity); diff != "" {
			t.Fatalf("sink identity mismatch (-want +got):\n%s", diff)
		}

		bareWriter, err := bareIO.OpenWrite(execCtx, false)
		if err != nil {
			t.Fatalf("open bare writer: %v", err)
		}
		if _, err := bareWriter.Write([]byte("bare\n")); err != nil {
			t.Fatalf("write bare output: %v", err)
		}
		if err := bareWriter.Close(); err != nil {
			t.Fatalf("close bare writer: %v", err)
		}

		explicitWriter, err := explicitIO.OpenWrite(execCtx, false)
		if err != nil {
			t.Fatalf("open explicit writer: %v", err)
		}
		if _, err := explicitWriter.Write([]byte("explicit\n")); err != nil {
			t.Fatalf("write explicit output: %v", err)
		}
		if err := explicitWriter.Close(); err != nil {
			t.Fatalf("close explicit writer: %v", err)
		}

		actual, err := session.Get(context.Background(), filepath.Join(workdir, relPath))
		if err != nil {
			t.Fatalf("read redirected output: %v", err)
		}
		if diff := cmp.Diff("explicit\n", string(actual)); diff != "" {
			t.Fatalf("redirected output mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("input source bare path equals explicit file decorator", func(t *testing.T) {
		t.Parallel()

		workdir := t.TempDir()
		session := decorator.NewLocalSession().WithWorkdir(workdir)
		execCtx := decorator.ExecContext{Context: context.Background(), Session: session}

		relPath := filepath.Join("io", "in.txt")
		expected := "from-input\n"
		if err := session.Put(context.Background(), []byte(expected), filepath.Join(workdir, relPath), 0o644); err != nil {
			t.Fatalf("seed input file: %v", err)
		}

		bareTarget := &planfmt.CommandNode{
			Decorator: "@shell",
			Args: []planfmt.Arg{{
				Key: "command",
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: relPath},
			}},
		}
		explicitTarget := &planfmt.CommandNode{
			Decorator: "@file",
			Args: []planfmt.Arg{{
				Key: "path",
				Val: planfmt.Value{Kind: planfmt.ValueString, Str: relPath},
			}},
		}

		bareIO, bareIdentity, ok := resolvePlanIOSink(bareTarget)
		if !ok {
			t.Fatal("expected bare target to resolve")
		}
		explicitIO, explicitIdentity, ok := resolvePlanIOSink(explicitTarget)
		if !ok {
			t.Fatal("expected explicit target to resolve")
		}

		if diff := cmp.Diff(explicitIdentity, bareIdentity); diff != "" {
			t.Fatalf("source identity mismatch (-want +got):\n%s", diff)
		}

		bareReader, err := bareIO.OpenRead(execCtx)
		if err != nil {
			t.Fatalf("open bare reader: %v", err)
		}
		bareData, err := io.ReadAll(bareReader)
		if err != nil {
			t.Fatalf("read bare source: %v", err)
		}
		if err := bareReader.Close(); err != nil {
			t.Fatalf("close bare reader: %v", err)
		}

		explicitReader, err := explicitIO.OpenRead(execCtx)
		if err != nil {
			t.Fatalf("open explicit reader: %v", err)
		}
		explicitData, err := io.ReadAll(explicitReader)
		if err != nil {
			t.Fatalf("read explicit source: %v", err)
		}
		if err := explicitReader.Close(); err != nil {
			t.Fatalf("close explicit reader: %v", err)
		}

		if diff := cmp.Diff(expected, string(bareData)); diff != "" {
			t.Fatalf("bare source data mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(expected, string(explicitData)); diff != "" {
			t.Fatalf("explicit source data mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(string(explicitData), string(bareData)); diff != "" {
			t.Fatalf("source equivalence mismatch (-want +got):\n%s", diff)
		}
	})
}
