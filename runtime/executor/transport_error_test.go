package executor

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/opal-lang/opal/core/decorator"
)

func TestTransportError(t *testing.T) {
	cause := errors.New("dial tcp 127.0.0.1:22: connection refused")
	err := decorator.TransportError{
		Code:      decorator.TransportErrorCodeConnect,
		Message:   "ssh dial failed",
		Retryable: true,
		Cause:     cause,
	}

	if diff := cmp.Diff("TRANSPORT_CONNECT_FAILED", err.Code); diff != "" {
		t.Fatalf("Code mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(true, err.Retryable); diff != "" {
		t.Fatalf("Retryable mismatch (-want +got):\n%s", diff)
	}

	wantError := "transport [TRANSPORT_CONNECT_FAILED]: ssh dial failed: dial tcp 127.0.0.1:22: connection refused"
	if diff := cmp.Diff(wantError, err.Error()); diff != "" {
		t.Fatalf("Error() mismatch (-want +got):\n%s", diff)
	}

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is should match wrapped cause")
	}
}
