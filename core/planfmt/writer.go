package planfmt

import (
	"bytes"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/blake2b"
)

const (
	// Magic is the file magic number "OPAL" (4 bytes)
	Magic = "OPAL"

	// Version is the format version (uint16, little-endian)
	// Version scheme: major.minor encoded as single uint16
	// 0x0001 = version 1.0
	// Breaking changes increment major, additions increment minor
	Version uint16 = 0x0001
)

// Flags is a bitmask for optional features
type Flags uint16

const (
	// FlagCompressed indicates STEPS and VALUES sections are zstd-compressed
	FlagCompressed Flags = 1 << 0

	// FlagSigned indicates a detached Ed25519 signature is present
	FlagSigned Flags = 1 << 1

	// Bits 2-15 reserved for future use
)

// Write writes a plan to w and returns the 32-byte file hash (BLAKE3).
// The plan is canonicalized before writing to ensure deterministic output.
func Write(w io.Writer, p *Plan) ([32]byte, error) {
	wr := &Writer{w: w}
	return wr.WritePlan(p)
}

// Writer handles writing plans to binary format.
type Writer struct {
	w io.Writer
}

// WritePlan writes the plan to the underlying writer.
// Format: MAGIC(4) | VERSION(2) | FLAGS(2) | HEADER_LEN(4) | BODY_LEN(8) | HEADER | BODY
// Returns the BLAKE2b-256 hash of the entire file.
func (wr *Writer) WritePlan(p *Plan) ([32]byte, error) {
	// Canonicalize plan for deterministic encoding (sorts args, preserves child order)
	p.Canonicalize()

	// Use buffer-then-write pattern: build header and body first, then write preamble with correct lengths
	var headerBuf, bodyBuf bytes.Buffer

	// Build header in buffer
	if err := wr.writeHeader(&headerBuf, p); err != nil {
		return [32]byte{}, err
	}

	// Build body in buffer (TODO: implement sections)
	if err := wr.writeBody(&bodyBuf, p); err != nil {
		return [32]byte{}, err
	}

	// Create hasher and multi-writer to compute hash while writing
	hasher, err := blake2b.New256(nil)
	if err != nil {
		return [32]byte{}, err
	}
	mw := io.MultiWriter(hasher, wr.w)

	// Write preamble with actual lengths (to both hasher and output)
	var preambleBuf bytes.Buffer
	if err := wr.writePreambleToBuffer(&preambleBuf, uint32(headerBuf.Len()), uint64(bodyBuf.Len())); err != nil {
		return [32]byte{}, err
	}
	if _, err := mw.Write(preambleBuf.Bytes()); err != nil {
		return [32]byte{}, err
	}

	// Write header (to both hasher and output)
	if _, err := mw.Write(headerBuf.Bytes()); err != nil {
		return [32]byte{}, err
	}

	// Write body (to both hasher and output)
	if _, err := mw.Write(bodyBuf.Bytes()); err != nil {
		return [32]byte{}, err
	}

	// Extract hash
	var digest [32]byte
	copy(digest[:], hasher.Sum(nil))
	return digest, nil
}

// writePreambleToBuffer writes the fixed-size preamble (20 bytes) to a buffer
func (wr *Writer) writePreambleToBuffer(buf *bytes.Buffer, headerLen uint32, bodyLen uint64) error {
	// Magic number (4 bytes)
	if _, err := buf.Write([]byte(Magic)); err != nil {
		return err
	}

	// Version (2 bytes, little-endian)
	if err := binary.Write(buf, binary.LittleEndian, Version); err != nil {
		return err
	}

	// Flags (2 bytes, little-endian)
	flags := Flags(0) // No compression, no signature
	if err := binary.Write(buf, binary.LittleEndian, uint16(flags)); err != nil {
		return err
	}

	// Header length (4 bytes, uint32, little-endian)
	if err := binary.Write(buf, binary.LittleEndian, headerLen); err != nil {
		return err
	}

	// Body length (8 bytes, uint64, little-endian)
	if err := binary.Write(buf, binary.LittleEndian, bodyLen); err != nil {
		return err
	}

	return nil
}

// writeHeader writes the plan header to the buffer
func (wr *Writer) writeHeader(buf *bytes.Buffer, p *Plan) error {
	// Write PlanHeader struct (44 bytes fixed)
	// SchemaID (16 bytes)
	if _, err := buf.Write(p.Header.SchemaID[:]); err != nil {
		return err
	}

	// CreatedAt (8 bytes, uint64, little-endian)
	if err := binary.Write(buf, binary.LittleEndian, p.Header.CreatedAt); err != nil {
		return err
	}

	// Compiler (16 bytes)
	if _, err := buf.Write(p.Header.Compiler[:]); err != nil {
		return err
	}

	// PlanKind (1 byte)
	if err := buf.WriteByte(p.Header.PlanKind); err != nil {
		return err
	}

	// Reserved (3 bytes)
	if _, err := buf.Write([]byte{0, 0, 0}); err != nil {
		return err
	}

	// Target (variable length: 2-byte length prefix + string bytes)
	targetLen := uint16(len(p.Target))
	if err := binary.Write(buf, binary.LittleEndian, targetLen); err != nil {
		return err
	}
	if _, err := buf.WriteString(p.Target); err != nil {
		return err
	}

	return nil
}

// writeBody writes the plan body (TOC + sections) to the buffer
func (wr *Writer) writeBody(buf *bytes.Buffer, p *Plan) error {
	// For now, write a minimal STEPS section (no TOC yet)
	// This is just enough to make child order tests pass
	if p.Root != nil {
		return wr.writeStep(buf, p.Root)
	}
	return nil
}

// writeStep writes a single step and its children recursively
func (wr *Writer) writeStep(buf *bytes.Buffer, s *Step) error {
	// Write step ID (8 bytes, uint64, little-endian)
	if err := binary.Write(buf, binary.LittleEndian, s.ID); err != nil {
		return err
	}

	// Write kind (1 byte)
	if err := buf.WriteByte(uint8(s.Kind)); err != nil {
		return err
	}

	// Write op (2-byte length + string)
	opLen := uint16(len(s.Op))
	if err := binary.Write(buf, binary.LittleEndian, opLen); err != nil {
		return err
	}
	if _, err := buf.WriteString(s.Op); err != nil {
		return err
	}

	// Write args count (2 bytes, uint16)
	argsCount := uint16(len(s.Args))
	if err := binary.Write(buf, binary.LittleEndian, argsCount); err != nil {
		return err
	}

	// Write each arg
	for _, arg := range s.Args {
		if err := wr.writeArg(buf, &arg); err != nil {
			return err
		}
	}

	// Write children count (2 bytes, uint16)
	childrenCount := uint16(len(s.Children))
	if err := binary.Write(buf, binary.LittleEndian, childrenCount); err != nil {
		return err
	}

	// Write each child recursively
	for _, child := range s.Children {
		if err := wr.writeStep(buf, child); err != nil {
			return err
		}
	}

	return nil
}

// writeArg writes a single argument
func (wr *Writer) writeArg(buf *bytes.Buffer, arg *Arg) error {
	// Write key (2-byte length + string)
	keyLen := uint16(len(arg.Key))
	if err := binary.Write(buf, binary.LittleEndian, keyLen); err != nil {
		return err
	}
	if _, err := buf.WriteString(arg.Key); err != nil {
		return err
	}

	// Write value kind (1 byte)
	if err := buf.WriteByte(uint8(arg.Val.Kind)); err != nil {
		return err
	}

	// Write value based on kind
	switch arg.Val.Kind {
	case ValueString:
		// String: 2-byte length + string
		strLen := uint16(len(arg.Val.Str))
		if err := binary.Write(buf, binary.LittleEndian, strLen); err != nil {
			return err
		}
		if _, err := buf.WriteString(arg.Val.Str); err != nil {
			return err
		}
	case ValueInt:
		// Int: 8 bytes, int64, little-endian
		if err := binary.Write(buf, binary.LittleEndian, arg.Val.Int); err != nil {
			return err
		}
	case ValueBool:
		// Bool: 1 byte (0 or 1)
		var b byte
		if arg.Val.Bool {
			b = 1
		}
		if err := buf.WriteByte(b); err != nil {
			return err
		}
	case ValuePlaceholder:
		// Placeholder: 4 bytes, uint32 (index into placeholder table)
		if err := binary.Write(buf, binary.LittleEndian, arg.Val.Ref); err != nil {
			return err
		}
	}

	return nil
}
