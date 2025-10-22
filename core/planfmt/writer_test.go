package planfmt_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestWriteEmptyPlan verifies we can write a minimal plan with correct magic and version
func TestWriteEmptyPlan(t *testing.T) {
	// Given: empty plan
	plan := &planfmt.Plan{}

	// When: write to buffer
	var buf bytes.Buffer
	hash, err := planfmt.Write(&buf, plan)
	// Then: no error, valid hash, valid magic number
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("Expected 32-byte hash, got %d", len(hash))
	}

	// Verify hash is non-zero (actual content hashed)
	allZero := true
	for _, b := range hash {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("Expected non-zero hash, got all zeros")
	}

	// Verify magic number "OPAL"
	data := buf.Bytes()
	if len(data) < 4 {
		t.Fatalf("Output too short: %d bytes", len(data))
	}

	magic := string(data[0:4])
	if magic != "OPAL" {
		t.Errorf("Expected magic 'OPAL', got %q", magic)
	}

	// Verify version is present (bytes 4-5, little-endian uint16)
	if len(data) < 6 {
		t.Fatalf("Output missing version: %d bytes", len(data))
	}
}

// TestWriteFlags verifies flags field is written correctly
func TestWriteFlags(t *testing.T) {
	// Given: empty plan (no compression, no signature)
	plan := &planfmt.Plan{}

	// When: write to buffer
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Then: flags field should be 0x0000 (no flags set)
	data := buf.Bytes()
	if len(data) < 8 {
		t.Fatalf("Output too short for flags: %d bytes", len(data))
	}

	// Flags are at offset 6-7 (after magic + version)
	flags := binary.LittleEndian.Uint16(data[6:8])
	if flags != 0 {
		t.Errorf("Expected flags 0x0000, got 0x%04x", flags)
	}
}

// TestWriteHeaderLengths verifies header and body length fields
func TestWriteHeaderLengths(t *testing.T) {
	// Given: empty plan
	plan := &planfmt.Plan{
		Target: "deploy",
	}

	// When: write to buffer
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Then: verify we have header length and body length fields
	data := buf.Bytes()

	// Format: MAGIC(4) | VERSION(2) | FLAGS(2) | HEADER_LEN(4) | BODY_LEN(8)
	// Minimum size: 4 + 2 + 2 + 4 + 8 = 20 bytes
	if len(data) < 20 {
		t.Fatalf("Output too short for header lengths: %d bytes", len(data))
	}

	// HEADER_LEN at offset 8-11 (uint32, little-endian)
	headerLen := binary.LittleEndian.Uint32(data[8:12])
	if headerLen == 0 {
		t.Error("Expected non-zero header length")
	}

	// BODY_LEN at offset 12-19 (uint64, little-endian)
	bodyLen := binary.LittleEndian.Uint64(data[12:20])
	// Body length can be 0 for empty plan, just verify field exists
	_ = bodyLen
}

// TestWriteActualHeader verifies the actual header bytes are written correctly
func TestWriteActualHeader(t *testing.T) {
	// Given: plan with target
	plan := &planfmt.Plan{
		Target: "deploy",
		Header: planfmt.PlanHeader{
			SchemaID:  [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			CreatedAt: 1234567890,
			Compiler:  [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			PlanKind:  1, // contract
		},
	}

	// When: write to buffer
	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data := buf.Bytes()

	// Then: verify total size matches preamble + header
	// Preamble: 20 bytes
	// Header: 44 bytes (PlanHeader) + 2 (target length) + 6 (target string "deploy")
	expectedHeaderLen := uint32(44 + 2 + 6)
	expectedTotalLen := 20 + expectedHeaderLen

	if uint32(len(data)) < expectedTotalLen {
		t.Fatalf("Output too short: got %d bytes, expected at least %d", len(data), expectedTotalLen)
	}

	// Verify HEADER_LEN field matches actual header size
	headerLen := binary.LittleEndian.Uint32(data[8:12])
	if headerLen != expectedHeaderLen {
		t.Errorf("HEADER_LEN mismatch: got %d, expected %d", headerLen, expectedHeaderLen)
	}

	// Verify header starts at offset 20 (after preamble)
	headerStart := 20

	// Verify SchemaID (16 bytes at offset 20)
	schemaID := data[headerStart : headerStart+16]
	for i, b := range plan.Header.SchemaID {
		if schemaID[i] != b {
			t.Errorf("SchemaID[%d] mismatch: got %d, expected %d", i, schemaID[i], b)
		}
	}

	// Verify CreatedAt (8 bytes at offset 36, little-endian uint64)
	createdAt := binary.LittleEndian.Uint64(data[headerStart+16 : headerStart+24])
	if createdAt != plan.Header.CreatedAt {
		t.Errorf("CreatedAt mismatch: got %d, expected %d", createdAt, plan.Header.CreatedAt)
	}

	// Verify Compiler (16 bytes at offset 44)
	compiler := data[headerStart+24 : headerStart+40]
	for i, b := range plan.Header.Compiler {
		if compiler[i] != b {
			t.Errorf("Compiler[%d] mismatch: got %d, expected %d", i, compiler[i], b)
		}
	}

	// Verify PlanKind (1 byte at offset 60)
	planKind := data[headerStart+40]
	if planKind != plan.Header.PlanKind {
		t.Errorf("PlanKind mismatch: got %d, expected %d", planKind, plan.Header.PlanKind)
	}

	// Verify Target length (2 bytes at offset 64, little-endian uint16)
	targetLenOffset := headerStart + 44
	targetLen := binary.LittleEndian.Uint16(data[targetLenOffset : targetLenOffset+2])
	if targetLen != uint16(len(plan.Target)) {
		t.Errorf("Target length mismatch: got %d, expected %d", targetLen, len(plan.Target))
	}

	// Verify Target string (6 bytes at offset 66)
	targetOffset := targetLenOffset + 2
	target := string(data[targetOffset : targetOffset+int(targetLen)])
	if target != plan.Target {
		t.Errorf("Target mismatch: got %q, expected %q", target, plan.Target)
	}
}

// TestContractRoundtrip verifies WriteContract and ReadContract work together
func TestContractRoundtrip(t *testing.T) {
	// Given: a plan with steps
	plan := &planfmt.Plan{
		Target: "hello",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: `echo "Hello"`}},
					},
				},
			},
		},
	}

	// When: compute hash first
	var hashBuf bytes.Buffer
	hash, err := planfmt.Write(&hashBuf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Write contract to separate buffer
	var contractBuf bytes.Buffer
	err = planfmt.WriteContract(&contractBuf, plan.Target, hash, plan)
	if err != nil {
		t.Fatalf("WriteContract failed: %v", err)
	}

	// Then: read contract back
	target, readHash, readPlan, err := planfmt.ReadContract(&contractBuf)
	if err != nil {
		t.Fatalf("ReadContract failed: %v", err)
	}

	// Verify target
	if target != plan.Target {
		t.Errorf("Target mismatch: got %q, want %q", target, plan.Target)
	}

	// Verify hash
	if readHash != hash {
		t.Errorf("Hash mismatch: got %x, want %x", readHash, hash)
	}

	// Verify plan structure
	if len(readPlan.Steps) != len(plan.Steps) {
		t.Errorf("Step count mismatch: got %d, want %d", len(readPlan.Steps), len(plan.Steps))
	}

	if readPlan.Steps[0].ID != plan.Steps[0].ID {
		t.Errorf("Step ID mismatch: got %d, want %d", readPlan.Steps[0].ID, plan.Steps[0].ID)
	}
}

// TestWriteTargetTooLong tests that target strings exceeding uint16 max are rejected
func TestWriteTargetTooLong(t *testing.T) {
	// Create a target string longer than uint16 max (65535 bytes)
	longTarget := string(make([]byte, 65536))

	plan := &planfmt.Plan{
		Target: longTarget,
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for target exceeding uint16 max, got nil")
	}

	if err.Error() != "target length 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestWriteTooManySteps tests that step counts exceeding uint16 max are rejected
func TestWriteTooManySteps(t *testing.T) {
	// Create more steps than uint16 max (65535)
	steps := make([]planfmt.Step, 65536)
	for i := range steps {
		steps[i] = planfmt.Step{
			ID:   uint64(i + 1),
			Tree: &planfmt.CommandNode{Decorator: "test"},
		}
	}

	plan := &planfmt.Plan{
		Target: "test",
		Steps:  steps,
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for step count exceeding uint16 max, got nil")
	}

	if err.Error() != "step count 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestWriteDecoratorNameTooLong tests that decorator names exceeding uint16 max are rejected
func TestWriteDecoratorNameTooLong(t *testing.T) {
	longDecorator := string(make([]byte, 65536))

	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: &planfmt.CommandNode{Decorator: longDecorator},
			},
		},
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for decorator name exceeding uint16 max, got nil")
	}

	if err.Error() != "decorator name length 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestWriteTooManyArgs tests that argument counts exceeding uint16 max are rejected
func TestWriteTooManyArgs(t *testing.T) {
	// Create more args than uint16 max (65535)
	args := make([]planfmt.Arg, 65536)
	for i := range args {
		args[i] = planfmt.Arg{
			Key: "arg",
			Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value"},
		}
	}

	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "test",
					Args:      args,
				},
			},
		},
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for argument count exceeding uint16 max, got nil")
	}

	if err.Error() != "argument count 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestWriteTooManyBlockSteps tests that block step counts exceeding uint16 max are rejected
func TestWriteTooManyBlockSteps(t *testing.T) {
	// Create more block steps than uint16 max (65535)
	blockSteps := make([]planfmt.Step, 65536)
	for i := range blockSteps {
		blockSteps[i] = planfmt.Step{
			ID:   uint64(i + 1),
			Tree: &planfmt.CommandNode{Decorator: "test"},
		}
	}

	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "test",
					Block:     blockSteps,
				},
			},
		},
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for block step count exceeding uint16 max, got nil")
	}

	if err.Error() != "block step count 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestWriteTooManyPipelineCommands tests that pipeline command counts exceeding uint16 max are rejected
func TestWriteTooManyPipelineCommands(t *testing.T) {
	// Create more pipeline commands than uint16 max (65535)
	commands := make([]planfmt.CommandNode, 65536)
	for i := range commands {
		commands[i] = planfmt.CommandNode{Decorator: "test"}
	}

	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: &planfmt.PipelineNode{Commands: commands},
			},
		},
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for pipeline command count exceeding uint16 max, got nil")
	}

	if err.Error() != "pipeline command count 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestWriteTooManySequenceNodes tests that sequence node counts exceeding uint16 max are rejected
func TestWriteTooManySequenceNodes(t *testing.T) {
	// Create more sequence nodes than uint16 max (65535)
	nodes := make([]planfmt.ExecutionNode, 65536)
	for i := range nodes {
		nodes[i] = &planfmt.CommandNode{Decorator: "test"}
	}

	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID:   1,
				Tree: &planfmt.SequenceNode{Nodes: nodes},
			},
		},
	}

	var buf bytes.Buffer
	_, err := planfmt.Write(&buf, plan)

	if err == nil {
		t.Fatal("Expected error for sequence node count exceeding uint16 max, got nil")
	}

	if err.Error() != "sequence node count 65536 exceeds maximum 65535" {
		t.Errorf("Wrong error message: %v", err)
	}
}
