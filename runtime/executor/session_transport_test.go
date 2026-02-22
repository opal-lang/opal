package executor

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestSessionTransportOpenFileReader(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	session := decorator.NewLocalSession().WithWorkdir(workdir)
	transport := newSessionTransport(session)

	path := filepath.Join(workdir, "input.txt")
	if err := session.Put(context.Background(), []byte("input-data\n"), path, 0o644); err != nil {
		t.Fatalf("seed input file: %v", err)
	}

	reader, err := transport.OpenFileReader(context.Background(), path)
	if err != nil {
		t.Fatalf("open file reader: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := reader.Close(); closeErr != nil {
			t.Fatalf("close reader: %v", closeErr)
		}
	})

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read file content: %v", err)
	}
	if diff := cmp.Diff("input-data\n", string(data)); diff != "" {
		t.Fatalf("reader content mismatch (-want +got):\n%s", diff)
	}
}

func TestSessionTransportOpenFileReaderMissingPath(t *testing.T) {
	t.Parallel()

	transport := newSessionTransport(decorator.NewLocalSession())
	if _, err := transport.OpenFileReader(context.Background(), filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("expected open file reader to fail for missing path")
	}
}
