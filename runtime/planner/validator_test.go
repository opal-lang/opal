package planner

import (
	"testing"

	"github.com/builtwithtofu/sigil/core/planfmt"
	"github.com/google/go-cmp/cmp"
)

func TestDetectCycle_LinearChainPasses(t *testing.T) {
	transports := []planfmt.Transport{
		{ID: "A", ParentID: ""},
		{ID: "B", ParentID: "A"},
		{ID: "C", ParentID: "B"},
	}

	err := detectCycle(transports)
	if diff := cmp.Diff(nil, err); diff != "" {
		t.Errorf("detectCycle() mismatch (-want +got):\n%s", diff)
	}
}

func TestDetectCycle_CycleFails(t *testing.T) {
	transports := []planfmt.Transport{
		{ID: "A", ParentID: "B"},
		{ID: "B", ParentID: "A"},
	}

	err := detectCycle(transports)
	if err == nil {
		t.Fatal("detectCycle() expected error, got nil")
	}

	expected := "transport cycle detected: A -> B -> A"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("detectCycle() error mismatch (-want +got):\n%s", diff)
	}
}

func TestDetectCycle_SelfCycleFails(t *testing.T) {
	transports := []planfmt.Transport{
		{ID: "A", ParentID: "A"},
	}

	err := detectCycle(transports)
	if err == nil {
		t.Fatal("detectCycle() expected error, got nil")
	}

	expected := "transport cycle detected: A -> A"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("detectCycle() error mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateTransportTable_DuplicateIDFails(t *testing.T) {
	transports := []planfmt.Transport{
		{ID: "transport:alpha"},
		{ID: "transport:alpha"},
	}

	err := validateTransportTable(transports)
	if err == nil {
		t.Fatal("validateTransportTable() expected error, got nil")
	}

	expected := "duplicate transport ID \"transport:alpha\""
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("validateTransportTable() error mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateTransportTable_ReservedIDFails(t *testing.T) {
	transports := []planfmt.Transport{
		{ID: "local"},
	}

	err := validateTransportTable(transports)
	if err == nil {
		t.Fatal("validateTransportTable() expected error, got nil")
	}

	expected := "transport ID \"local\" is reserved"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("validateTransportTable() error mismatch (-want +got):\n%s", diff)
	}
}

func TestValidateTransportTable_EmptyIDFails(t *testing.T) {
	transports := []planfmt.Transport{{ID: ""}}

	err := validateTransportTable(transports)
	if err == nil {
		t.Fatal("validateTransportTable() expected error, got nil")
	}

	expected := "transport ID must not be empty"
	if diff := cmp.Diff(expected, err.Error()); diff != "" {
		t.Errorf("validateTransportTable() error mismatch (-want +got):\n%s", diff)
	}
}
