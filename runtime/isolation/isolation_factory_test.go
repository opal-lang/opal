package isolation

import "testing"

func TestNewIsolatorReturnsContext(t *testing.T) {
	isolator := NewIsolator()
	if isolator == nil {
		t.Fatal("expected non-nil isolator")
	}
}

func TestIsSupportedCanBeQueried(t *testing.T) {
	_ = IsSupported()
}
