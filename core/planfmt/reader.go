package planfmt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/crypto/blake2b"
)

// Read reads a plan from r and returns the plan and its hash.
func Read(r io.Reader) (*Plan, [32]byte, error) {
	rd := &Reader{r: r}
	return rd.ReadPlan()
}

// Reader handles reading plans from binary format.
type Reader struct {
	r io.Reader
}

// ReadPlan reads the plan from the underlying reader and returns the computed hash.
func (rd *Reader) ReadPlan() (*Plan, [32]byte, error) {
	// Create hasher to compute hash while reading
	hasher, err := blake2b.New256(nil)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("create hasher: %w", err)
	}

	// Read preamble (20 bytes)
	var preamble [20]byte
	if _, err := io.ReadFull(rd.r, preamble[:]); err != nil {
		return nil, [32]byte{}, fmt.Errorf("read preamble: %w", err)
	}
	hasher.Write(preamble[:])

	// Verify magic
	magic := string(preamble[0:4])
	if magic != Magic {
		return nil, [32]byte{}, fmt.Errorf("invalid magic: got %q, expected %q", magic, Magic)
	}

	// Read version
	version := binary.LittleEndian.Uint16(preamble[4:6])
	if version != Version {
		return nil, [32]byte{}, fmt.Errorf("unsupported version: got 0x%04x, expected 0x%04x", version, Version)
	}

	// Read flags
	flags := Flags(binary.LittleEndian.Uint16(preamble[6:8]))

	// Reject unknown flags for this version
	knownFlags := FlagCompressed | FlagSigned
	if flags&^knownFlags != 0 {
		return nil, [32]byte{}, fmt.Errorf("unsupported flags: 0x%04x (unknown bits: 0x%04x)", flags, flags&^knownFlags)
	}

	// TODO: Implement compression and signature verification
	if flags&FlagCompressed != 0 {
		return nil, [32]byte{}, fmt.Errorf("compressed plans not yet supported")
	}
	if flags&FlagSigned != 0 {
		return nil, [32]byte{}, fmt.Errorf("signed plans not yet supported")
	}

	// Read header length
	headerLen := binary.LittleEndian.Uint32(preamble[8:12])

	// Read body length
	bodyLen := binary.LittleEndian.Uint64(preamble[12:20])

	// Validate lengths to prevent OOM attacks
	// Header: metadata only, should be < 1KB typically
	// Body: even 10,000 steps should fit in ~10MB
	const maxHeaderLen = 64 * 1024      // 64KB max header (very generous)
	const maxBodyLen = 32 * 1024 * 1024 // 32MB max body (lowered for fuzz safety)
	const maxDepth = 1000               // Max recursion depth to prevent stack overflow

	if headerLen > maxHeaderLen {
		return nil, [32]byte{}, fmt.Errorf("header length %d exceeds maximum %d", headerLen, maxHeaderLen)
	}
	if bodyLen > maxBodyLen {
		return nil, [32]byte{}, fmt.Errorf("body length %d exceeds maximum %d", bodyLen, maxBodyLen)
	}

	// Read header
	headerBuf := make([]byte, headerLen)
	if _, err := io.ReadFull(rd.r, headerBuf); err != nil {
		return nil, [32]byte{}, fmt.Errorf("read header: %w", err)
	}
	hasher.Write(headerBuf)

	plan, err := rd.readHeader(bytes.NewReader(headerBuf))
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("parse header: %w", err)
	}

	// Read body
	bodyBuf := make([]byte, bodyLen)
	if _, err := io.ReadFull(rd.r, bodyBuf); err != nil {
		return nil, [32]byte{}, fmt.Errorf("read body: %w", err)
	}
	hasher.Write(bodyBuf)

	if err := rd.readBody(bytes.NewReader(bodyBuf), plan, maxDepth); err != nil {
		return nil, [32]byte{}, fmt.Errorf("parse body: %w", err)
	}

	// Extract hash
	var digest [32]byte
	copy(digest[:], hasher.Sum(nil))
	return plan, digest, nil
}

// readHeader reads the plan header
func (rd *Reader) readHeader(r io.Reader) (*Plan, error) {
	plan := &Plan{}

	// Read SchemaID (16 bytes)
	if _, err := io.ReadFull(r, plan.Header.SchemaID[:]); err != nil {
		return nil, fmt.Errorf("read schema ID: %w", err)
	}

	// Read CreatedAt (8 bytes, uint64, little-endian)
	if err := binary.Read(r, binary.LittleEndian, &plan.Header.CreatedAt); err != nil {
		return nil, fmt.Errorf("read created at: %w", err)
	}

	// Read Compiler (16 bytes)
	if _, err := io.ReadFull(r, plan.Header.Compiler[:]); err != nil {
		return nil, fmt.Errorf("read compiler: %w", err)
	}

	// Read PlanKind (1 byte)
	var planKind byte
	if err := binary.Read(r, binary.LittleEndian, &planKind); err != nil {
		return nil, fmt.Errorf("read plan kind: %w", err)
	}
	plan.Header.PlanKind = planKind

	// Skip reserved (3 bytes)
	var reserved [3]byte
	if _, err := io.ReadFull(r, reserved[:]); err != nil {
		return nil, fmt.Errorf("read reserved: %w", err)
	}

	// Read Target length (2 bytes, uint16, little-endian)
	var targetLen uint16
	if err := binary.Read(r, binary.LittleEndian, &targetLen); err != nil {
		return nil, fmt.Errorf("read target length: %w", err)
	}

	// Read Target string
	targetBuf := make([]byte, targetLen)
	if _, err := io.ReadFull(r, targetBuf); err != nil {
		return nil, fmt.Errorf("read target: %w", err)
	}
	plan.Target = string(targetBuf)

	return plan, nil
}

// readBody reads the plan body (steps)
func (rd *Reader) readBody(r io.Reader, plan *Plan, maxDepth int) error {
	// Check if body is empty
	var peek [1]byte
	n, err := r.Read(peek[:])
	if err == io.EOF || n == 0 {
		// Empty body, no root step
		return nil
	}

	// Body has content, read root step
	// Create a new reader with the peeked byte prepended
	bodyReader := io.MultiReader(bytes.NewReader(peek[:n]), r)
	root, err := rd.readStep(bodyReader, 0, maxDepth)
	if err != nil {
		return err
	}
	plan.Root = root

	return nil
}

// readStep reads a single step and its children recursively
func (rd *Reader) readStep(r io.Reader, depth int, maxDepth int) (*Step, error) {
	// Check depth limit to prevent stack overflow
	if depth >= maxDepth {
		return nil, fmt.Errorf("max recursion depth %d exceeded", maxDepth)
	}

	step := &Step{}

	// Read step ID (8 bytes, uint64, little-endian)
	if err := binary.Read(r, binary.LittleEndian, &step.ID); err != nil {
		return nil, fmt.Errorf("read step ID: %w", err)
	}

	// Read kind (1 byte)
	var kind byte
	if err := binary.Read(r, binary.LittleEndian, &kind); err != nil {
		return nil, fmt.Errorf("read step kind: %w", err)
	}
	step.Kind = StepKind(kind)

	// Read op length (2 bytes, uint16, little-endian)
	var opLen uint16
	if err := binary.Read(r, binary.LittleEndian, &opLen); err != nil {
		return nil, fmt.Errorf("read op length: %w", err)
	}

	// Read op string
	opBuf := make([]byte, opLen)
	if _, err := io.ReadFull(r, opBuf); err != nil {
		return nil, fmt.Errorf("read op: %w", err)
	}
	step.Op = string(opBuf)

	// Read args count (2 bytes, uint16, little-endian)
	var argsCount uint16
	if err := binary.Read(r, binary.LittleEndian, &argsCount); err != nil {
		return nil, fmt.Errorf("read args count: %w", err)
	}

	// Read each arg
	if argsCount > 0 {
		step.Args = make([]Arg, argsCount)
		for i := 0; i < int(argsCount); i++ {
			arg, err := rd.readArg(r)
			if err != nil {
				return nil, fmt.Errorf("read arg %d: %w", i, err)
			}
			step.Args[i] = *arg
		}
	}

	// Read children count (2 bytes, uint16, little-endian)
	var childrenCount uint16
	if err := binary.Read(r, binary.LittleEndian, &childrenCount); err != nil {
		return nil, fmt.Errorf("read children count: %w", err)
	}

	// Read each child recursively
	if childrenCount > 0 {
		step.Children = make([]*Step, childrenCount)
		for i := 0; i < int(childrenCount); i++ {
			child, err := rd.readStep(r, depth+1, maxDepth)
			if err != nil {
				return nil, fmt.Errorf("read child %d: %w", i, err)
			}
			step.Children[i] = child
		}
	}

	return step, nil
}

// readArg reads a single argument
func (rd *Reader) readArg(r io.Reader) (*Arg, error) {
	arg := &Arg{}

	// Read key length (2 bytes, uint16, little-endian)
	var keyLen uint16
	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
		return nil, fmt.Errorf("read key length: %w", err)
	}

	// Read key string
	keyBuf := make([]byte, keyLen)
	if _, err := io.ReadFull(r, keyBuf); err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	arg.Key = string(keyBuf)

	// Read value kind (1 byte)
	var kind byte
	if err := binary.Read(r, binary.LittleEndian, &kind); err != nil {
		return nil, fmt.Errorf("read value kind: %w", err)
	}
	arg.Val.Kind = ValueKind(kind)

	// Read value based on kind
	switch arg.Val.Kind {
	case ValueString:
		// String: 2-byte length + string
		var strLen uint16
		if err := binary.Read(r, binary.LittleEndian, &strLen); err != nil {
			return nil, fmt.Errorf("read string length: %w", err)
		}
		strBuf := make([]byte, strLen)
		if _, err := io.ReadFull(r, strBuf); err != nil {
			return nil, fmt.Errorf("read string: %w", err)
		}
		arg.Val.Str = string(strBuf)

	case ValueInt:
		// Int: 8 bytes, int64, little-endian
		if err := binary.Read(r, binary.LittleEndian, &arg.Val.Int); err != nil {
			return nil, fmt.Errorf("read int: %w", err)
		}

	case ValueBool:
		// Bool: 1 byte (0 or 1)
		var b byte
		if err := binary.Read(r, binary.LittleEndian, &b); err != nil {
			return nil, fmt.Errorf("read bool: %w", err)
		}
		arg.Val.Bool = b != 0

	case ValuePlaceholder:
		// Placeholder: 4 bytes, uint32 (index into placeholder table)
		if err := binary.Read(r, binary.LittleEndian, &arg.Val.Ref); err != nil {
			return nil, fmt.Errorf("read placeholder ref: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown value kind: %d", kind)
	}

	return arg, nil
}
