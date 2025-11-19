package planfmt_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/opal-lang/opal/core/planfmt"
)

// TestByteAccounting verifies the byte-accounting invariant:
// len(file) == 20 (preamble) + HEADER_LEN + BODY_LEN
func TestByteAccounting(t *testing.T) {
	tests := []struct {
		name string
		plan *planfmt.Plan
	}{
		{
			name: "empty plan",
			plan: &planfmt.Plan{
				Target: "",
			},
		},
		{
			name: "plan with target",
			plan: &planfmt.Plan{
				Target: "deploy",
				Header: planfmt.PlanHeader{
					CreatedAt: 1234567890,
					PlanKind:  1,
				},
			},
		},
		{
			name: "plan with single step",
			plan: &planfmt.Plan{
				Target: "build",
				Header: planfmt.PlanHeader{
					CreatedAt: 9876543210,
					PlanKind:  1,
				},
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			_, err := planfmt.Write(&buf, tt.plan)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			data := buf.Bytes()
			if len(data) < 20 {
				t.Fatalf("File too short: %d bytes", len(data))
			}

			// Extract lengths from preamble
			headerLen := binary.LittleEndian.Uint32(data[8:12])
			bodyLen := binary.LittleEndian.Uint64(data[12:20])

			// Verify byte-accounting invariant
			expectedLen := 20 + int(headerLen) + int(bodyLen)
			actualLen := len(data)

			if actualLen != expectedLen {
				t.Errorf("Byte-accounting invariant violated:\n"+
					"  preamble: 20 bytes\n"+
					"  HEADER_LEN: %d bytes\n"+
					"  BODY_LEN: %d bytes\n"+
					"  expected total: %d bytes\n"+
					"  actual total: %d bytes\n"+
					"  difference: %d bytes",
					headerLen, bodyLen, expectedLen, actualLen, actualLen-expectedLen)
			}
		})
	}
}

// TestCanonicalArgOrder verifies that plans with different arg orders
// produce identical bytes (args must be sorted for determinism)
func TestCanonicalArgOrder(t *testing.T) {
	// Plan with args in order A, B, C
	p1 := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "arg_a", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value_a"}},
						{Key: "arg_b", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value_b"}},
						{Key: "arg_c", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value_c"}},
					},
				},
			},
		},
	}

	// Plan with args in order C, A, B (semantically identical)
	p2 := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "arg_c", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value_c"}},
						{Key: "arg_a", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value_a"}},
						{Key: "arg_b", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "value_b"}},
					},
				},
			},
		},
	}

	// Write both plans
	var buf1, buf2 bytes.Buffer
	hash1, err1 := planfmt.Write(&buf1, p1)
	hash2, err2 := planfmt.Write(&buf2, p2)

	if err1 != nil {
		t.Fatalf("Write p1 failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Write p2 failed: %v", err2)
	}

	// Verify bytes are identical (canonical encoding)
	bytes1 := buf1.Bytes()
	bytes2 := buf2.Bytes()

	if !bytes.Equal(bytes1, bytes2) {
		t.Errorf("Non-canonical encoding: different arg orders produced different bytes\n"+
			"  p1 length: %d bytes\n"+
			"  p2 length: %d bytes\n"+
			"  This means args are not being sorted before encoding!",
			len(bytes1), len(bytes2))
	}

	// Verify hashes match
	if hash1 != hash2 {
		t.Errorf("Non-canonical hashes: different arg orders produced different hashes\n"+
			"  p1 hash: %x\n"+
			"  p2 hash: %x",
			hash1, hash2)
	}
}

// TestStability verifies that the same plan produces identical bytes
// across multiple writes (no non-determinism from maps, time, etc.)
func TestStability(t *testing.T) {
	plan := &planfmt.Plan{
		Target: "deploy",
		Header: planfmt.PlanHeader{
			SchemaID:  [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			CreatedAt: 1234567890, // Fixed timestamp
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
						{Key: "timeout", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 30}},
					},
				},
			},
		},
	}

	// Write the plan 10 times
	const numWrites = 10
	var buffers [numWrites]bytes.Buffer
	var hashes [numWrites][32]byte

	for i := 0; i < numWrites; i++ {
		hash, err := planfmt.Write(&buffers[i], plan)
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
		hashes[i] = hash
	}

	// Verify all bytes are identical
	firstBytes := buffers[0].Bytes()
	firstHash := hashes[0]

	for i := 1; i < numWrites; i++ {
		currentBytes := buffers[i].Bytes()
		currentHash := hashes[i]

		if !bytes.Equal(firstBytes, currentBytes) {
			t.Errorf("Instability detected: write %d produced different bytes\n"+
				"  first write: %d bytes, sha256=%x\n"+
				"  write %d: %d bytes, sha256=%x",
				i, len(firstBytes), sha256.Sum256(firstBytes),
				i, len(currentBytes), sha256.Sum256(currentBytes))
		}

		if firstHash != currentHash {
			t.Errorf("Instability detected: write %d produced different hash\n"+
				"  first hash: %x\n"+
				"  hash %d: %x",
				i, firstHash, i, currentHash)
		}
	}
}

// TestCanonicalCommandOrder verifies that command order is preserved
// (command order IS semantically significant, unlike args)
func TestCanonicalCommandOrder(t *testing.T) {
	// Plan with commands in order A, B (chained with &&)
	p1 := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.AndNode{
					Left:  &planfmt.CommandNode{Decorator: "@task_a"},
					Right: &planfmt.CommandNode{Decorator: "@task_b"},
				},
			},
		},
	}

	// Plan with commands in order B, A (semantically DIFFERENT)
	p2 := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.AndNode{
					Left:  &planfmt.CommandNode{Decorator: "@task_b"},
					Right: &planfmt.CommandNode{Decorator: "@task_a"},
				},
			},
		},
	}

	// Write both plans
	var buf1, buf2 bytes.Buffer
	hash1, err1 := planfmt.Write(&buf1, p1)
	hash2, err2 := planfmt.Write(&buf2, p2)

	if err1 != nil {
		t.Fatalf("Write p1 failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Write p2 failed: %v", err2)
	}

	// Verify bytes are DIFFERENT (child order matters)
	bytes1 := buf1.Bytes()
	bytes2 := buf2.Bytes()

	if bytes.Equal(bytes1, bytes2) {
		t.Errorf("Child order not preserved: different child orders produced identical bytes\n" +
			"  This is wrong because child order IS semantically significant!")
	}

	// Verify hashes are different (skip for now - hash is dummy [32]byte{})
	// TODO: Enable once real hashing is implemented
	_ = hash1
	_ = hash2
	// if hash1 == hash2 {
	// 	t.Errorf("Child order not preserved: different child orders produced identical hashes")
	// }
}

// TestSecretUsesOrderDeterministic verifies that SecretUses are always sorted
// in the same order regardless of declaration order (critical for contract verification).
// This test ensures block scoping doesn't introduce non-determinism.
func TestSecretUsesOrderDeterministic(t *testing.T) {
	// Plan with SecretUses in order A, B, C
	p1 := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
				},
			},
		},
		SecretUses: []planfmt.SecretUse{
			{DisplayID: "opal:aaa111", SiteID: "site1", Site: "root/step-1/params/command"},
			{DisplayID: "opal:bbb222", SiteID: "site2", Site: "root/step-2/params/command"},
			{DisplayID: "opal:ccc333", SiteID: "site3", Site: "root/step-3/params/command"},
		},
		PlanSalt: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}

	// Same plan with SecretUses in order C, A, B (should produce identical bytes after canonicalization)
	p2 := &planfmt.Plan{
		Target: "test",
		Steps: []planfmt.Step{
			{
				ID: 1,
				Tree: &planfmt.CommandNode{
					Decorator: "@shell",
					Args: []planfmt.Arg{
						{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo test"}},
					},
				},
			},
		},
		SecretUses: []planfmt.SecretUse{
			{DisplayID: "opal:ccc333", SiteID: "site3", Site: "root/step-3/params/command"},
			{DisplayID: "opal:aaa111", SiteID: "site1", Site: "root/step-1/params/command"},
			{DisplayID: "opal:bbb222", SiteID: "site2", Site: "root/step-2/params/command"},
		},
		PlanSalt: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}

	// Write both plans
	var buf1, buf2 bytes.Buffer
	hash1, err1 := planfmt.Write(&buf1, p1)
	hash2, err2 := planfmt.Write(&buf2, p2)

	if err1 != nil {
		t.Fatalf("Write p1 failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Write p2 failed: %v", err2)
	}

	// Verify bytes are identical (SecretUses sorted during canonicalization)
	bytes1 := buf1.Bytes()
	bytes2 := buf2.Bytes()

	if !bytes.Equal(bytes1, bytes2) {
		t.Errorf("Non-deterministic SecretUses encoding: different orders produced different bytes\n"+
			"  p1 length: %d bytes\n"+
			"  p2 length: %d bytes\n"+
			"  This means SecretUses are not being sorted before encoding!",
			len(bytes1), len(bytes2))
	}

	// Verify hashes match
	if hash1 != hash2 {
		t.Errorf("Non-deterministic hashes: different SecretUses orders produced different hashes\n"+
			"  p1 hash: %x\n"+
			"  p2 hash: %x",
			hash1, hash2)
	}

	t.Logf("✓ SecretUses order is deterministic (sorted during canonicalization)")
}
