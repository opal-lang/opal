package builtins

import (
	"context"
	"io"
	"io/fs"
	"testing"

	"github.com/builtwithtofu/sigil/core/plugin"
	"github.com/google/go-cmp/cmp"
)

func TestFileRedirectTargetOverwriteWriteAndRead(t *testing.T) {
	capability := FileRedirectCapability{}
	session := &memoryParentTransport{snapshot: plugin.SessionSnapshot{Workdir: "/tmp"}, files: map[string][]byte{}}
	ctx := fakeExecContext{ctx: context.Background(), session: session}
	args := fakeArgs{strings: map[string]string{"path": "out.txt"}}

	writer, err := capability.OpenForWrite(ctx, args, false)
	if err != nil {
		t.Fatalf("OpenForWrite() error = %v", err)
	}
	if _, err := writer.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := capability.OpenForRead(ctx, args)
	if err != nil {
		t.Fatalf("OpenForRead() error = %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if diff := cmp.Diff("hello\n", string(data)); diff != "" {
		t.Fatalf("file content mismatch (-want +got):\n%s", diff)
	}
}

func TestFileRedirectTargetAppendPersistsPartialOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	capability := FileRedirectCapability{}
	session := &memoryParentTransport{
		snapshot: plugin.SessionSnapshot{Workdir: "/tmp"},
		files:    map[string][]byte{"/tmp/out.txt": []byte("start\n")},
	}
	execCtx := fakeExecContext{ctx: ctx, session: session}
	args := fakeArgs{strings: map[string]string{"path": "out.txt"}}

	writer, err := capability.OpenForWrite(execCtx, args, true)
	if err != nil {
		t.Fatalf("OpenForWrite() error = %v", err)
	}
	if _, err := writer.Write([]byte("one\n")); err != nil {
		t.Fatalf("Write() first error = %v", err)
	}
	cancel()
	if _, err := writer.Write([]byte("two\n")); err == nil {
		t.Fatal("Write() second error = nil, want cancellation error")
	}
	_ = writer.Close()

	reader, err := capability.OpenForRead(fakeExecContext{ctx: context.Background(), session: session}, args)
	if err != nil {
		t.Fatalf("OpenForRead() error = %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if diff := cmp.Diff("start\none\n", string(data)); diff != "" {
		t.Fatalf("file content mismatch (-want +got):\n%s", diff)
	}
}

func TestFileRedirectTargetAppendFlushesAfterCancelWithContextAwareSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	capability := FileRedirectCapability{}
	session := &memoryParentTransport{
		snapshot:              plugin.SessionSnapshot{Workdir: "/tmp"},
		files:                 map[string][]byte{"/tmp/out.txt": []byte("start\n")},
		failOnCanceledContext: true,
	}
	execCtx := fakeExecContext{ctx: ctx, session: session}
	args := fakeArgs{strings: map[string]string{"path": "out.txt"}}

	writer, err := capability.OpenForWrite(execCtx, args, true)
	if err != nil {
		t.Fatalf("OpenForWrite() error = %v", err)
	}
	if _, err := writer.Write([]byte("one\n")); err != nil {
		t.Fatalf("Write() first error = %v", err)
	}
	cancel()
	if _, err := writer.Write([]byte("two\n")); err == nil {
		t.Fatal("Write() second error = nil, want cancellation error")
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := capability.OpenForRead(fakeExecContext{ctx: context.Background(), session: session}, args)
	if err != nil {
		t.Fatalf("OpenForRead() error = %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if diff := cmp.Diff("start\none\n", string(data)); diff != "" {
		t.Fatalf("file content mismatch (-want +got):\n%s", diff)
	}
}

func TestFileRedirectTargetAppendFlushesOnceOnClose(t *testing.T) {
	capability := FileRedirectCapability{}
	session := &memoryParentTransport{
		snapshot: plugin.SessionSnapshot{Workdir: "/tmp"},
		files:    map[string][]byte{"/tmp/out.txt": []byte("start\n")},
	}
	execCtx := fakeExecContext{ctx: context.Background(), session: session}
	args := fakeArgs{strings: map[string]string{"path": "out.txt"}}

	writer, err := capability.OpenForWrite(execCtx, args, true)
	if err != nil {
		t.Fatalf("OpenForWrite() error = %v", err)
	}
	if _, err := writer.Write([]byte("one\n")); err != nil {
		t.Fatalf("Write() first error = %v", err)
	}
	if _, err := writer.Write([]byte("two\n")); err != nil {
		t.Fatalf("Write() second error = %v", err)
	}
	if diff := cmp.Diff(0, session.putCalls); diff != "" {
		t.Fatalf("Put() call count before close mismatch (-want +got):\n%s", diff)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if diff := cmp.Diff(1, session.putCalls); diff != "" {
		t.Fatalf("Put() call count after close mismatch (-want +got):\n%s", diff)
	}

	reader, err := capability.OpenForRead(execCtx, args)
	if err != nil {
		t.Fatalf("OpenForRead() error = %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if diff := cmp.Diff("start\none\ntwo\n", string(data)); diff != "" {
		t.Fatalf("file content mismatch (-want +got):\n%s", diff)
	}
}

type memoryParentTransport struct {
	snapshot              plugin.SessionSnapshot
	files                 map[string][]byte
	putCalls              int
	failOnCanceledContext bool
}

func (m *memoryParentTransport) Run(ctx context.Context, argv []string, opts plugin.RunOpts) (plugin.Result, error) {
	return plugin.Result{ExitCode: plugin.ExitSuccess}, nil
}

func (m *memoryParentTransport) Put(ctx context.Context, data []byte, path string, mode fs.FileMode) error {
	if m.failOnCanceledContext {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	m.putCalls++
	copyData := make([]byte, len(data))
	copy(copyData, data)
	m.files[path] = copyData
	return nil
}

func (m *memoryParentTransport) Get(ctx context.Context, path string) ([]byte, error) {
	if m.failOnCanceledContext {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	data := m.files[path]
	copyData := make([]byte, len(data))
	copy(copyData, data)
	return copyData, nil
}

func (m *memoryParentTransport) Snapshot() plugin.SessionSnapshot { return m.snapshot }
func (m *memoryParentTransport) Close() error                     { return nil }
