package executor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"

	"github.com/opal-lang/opal/core/decorator"
	sdkexec "github.com/opal-lang/opal/core/sdk/executor"
)

type sessionTransport struct {
	session decorator.Session
}

func newSessionTransport(session decorator.Session) *sessionTransport {
	return &sessionTransport{session: session}
}

func (t *sessionTransport) Exec(ctx context.Context, argv []string, opts sdkexec.ExecOpts) (int, error) {
	runSession := t.session
	if opts.Dir != "" {
		runSession = runSession.WithWorkdir(opts.Dir)
	}
	if len(opts.Env) > 0 {
		runSession = runSession.WithEnv(opts.Env)
	}

	result, err := runSession.Run(ctx, argv, decorator.RunOpts{
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})
	return result.ExitCode, err
}

func (t *sessionTransport) Put(ctx context.Context, src io.Reader, dst string, mode fs.FileMode) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	return t.session.Put(ctx, data, dst, mode)
}

func (t *sessionTransport) Get(ctx context.Context, src string, dst io.Writer) error {
	data, err := t.session.Get(ctx, src)
	if err != nil {
		return err
	}

	_, err = dst.Write(data)
	return err
}

func (t *sessionTransport) OpenFileWriter(ctx context.Context, path string, mode sdkexec.RedirectMode, perm fs.FileMode) (io.WriteCloser, error) {
	return &sessionFileWriter{
		session: t.session,
		ctx:     ctx,
		path:    path,
		mode:    mode,
		perm:    perm,
	}, nil
}

func (t *sessionTransport) Close() error {
	return nil
}

type sessionFileWriter struct {
	session decorator.Session
	ctx     context.Context
	path    string
	mode    sdkexec.RedirectMode
	perm    fs.FileMode
	buffer  bytes.Buffer
	closed  bool
}

func (w *sessionFileWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, errors.New("writer is closed")
	}
	return w.buffer.Write(p)
}

func (w *sessionFileWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	data := append([]byte(nil), w.buffer.Bytes()...)
	if w.mode == sdkexec.RedirectAppend {
		existing, err := w.session.Get(w.ctx, w.path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err == nil {
			data = append(existing, data...)
		}
	}

	return w.session.Put(w.ctx, data, w.path, w.perm)
}
