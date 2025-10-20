package planfmt_test

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestHashDeterminism verifies that same plan produces same hash
func TestHashDeterminism(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Header: planfmt.PlanHeader{
			SchemaID:  [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			CreatedAt: 1234567890,
			Compiler:  [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			PlanKind:  1,
		},
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
				},
			},
		},
	}

	// Write multiple times
	const numWrites = 5
	var hashes [numWrites][32]byte
	var buffers [numWrites]bytes.Buffer

	for i := 0; i < numWrites; i++ {
		hash, err := planfmt.Write(&buffers[i], plan)
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
		hashes[i] = hash
	}

	// Verify all hashes are identical
	firstHash := hashes[0]
	for i := 1; i < numWrites; i++ {
		if hashes[i] != firstHash {
			t.Errorf("Hash %d differs from first:\n  first: %x\n  hash %d: %x",
				i, firstHash, i, hashes[i])
		}
	}

	// Verify bytes are identical too
	firstBytes := buffers[0].Bytes()
	for i := 1; i < numWrites; i++ {
		if !bytes.Equal(firstBytes, buffers[i].Bytes()) {
			t.Errorf("Bytes %d differ from first", i)
		}
	}
}

// TestHashUniqueness verifies that different plans produce different hashes
func TestHashUniqueness(t *testing.T) {
	tests := []struct {
		name string
		plan *planfmt.Plan
	}{
		{
			name: "empty",
			plan: &planfmt.Plan{Target: ""},
		},
		{
			name: "different_target",
			plan: &planfmt.Plan{Target: "deploy"},
		},
		{
			name: "with_step",
			plan: &planfmt.Plan{
				Target: "deploy",
				Steps: []planfmt.Step{
					{
						ID:   1,
						Tree: &planfmt.CommandNode{Decorator: "@shell"},
					},
				},
			},
		},
		{
			name: "different_decorator",
			plan: &planfmt.Plan{
				Target: "deploy",
				Steps: []planfmt.Step{
					{
						ID:   1,
						Tree: &planfmt.CommandNode{Decorator: "@retry"},
					},
				},
			},
		},
	}

	// Compute all hashes
	hashes := make(map[[32]byte]string)
	for _, tt := range tests {
		var buf bytes.Buffer
		hash, err := planfmt.Write(&buf, tt.plan)
		if err != nil {
			t.Fatalf("%s: Write failed: %v", tt.name, err)
		}

		// Check for collision
		if existing, found := hashes[hash]; found {
			t.Errorf("Hash collision: %s and %s produced same hash %x",
				tt.name, existing, hash)
		}
		hashes[hash] = tt.name
	}
}

// TestHashRoundTrip verifies that read hash matches write hash
func TestHashRoundTrip(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "cmd", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
					},
				},
			},
		},
	}

	// Write
	var buf bytes.Buffer
	writeHash, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	_, readHash, err := planfmt.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Verify hashes match
	if writeHash != readHash {
		t.Errorf("Hash mismatch:\n  write: %x\n  read:  %x", writeHash, readHash)
	}
}

// TestHashNonZero verifies that hash is not all zeros
func TestHashNonZero(t *testing.T) {
	plan := &planfmt.Plan{Target: "test"}

	var buf bytes.Buffer
	hash, err := planfmt.Write(&buf, plan)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check hash is not zero
	zeroHash := [32]byte{}
	if hash == zeroHash {
		t.Error("Hash is all zeros - hashing not working")
	}
}
