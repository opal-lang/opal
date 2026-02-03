package planner

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/opal-lang/opal/core/decorator"
)

func deriveTransportID(planKey []byte, decoratorName string, args map[string]any, parentID string) (string, error) {
	descriptor, err := buildTransportDescriptor(decoratorName, args, parentID)
	if err != nil {
		return "", err
	}

	if len(planKey) == 0 {
		sum := sha256.Sum256(descriptor)
		return "transport:" + base64.RawURLEncoding.EncodeToString(sum[:16]), nil
	}

	h := hmac.New(sha256.New, planKey)
	if _, err := h.Write(descriptor); err != nil {
		return "", fmt.Errorf("hash transport descriptor: %w", err)
	}
	mac := h.Sum(nil)
	return "transport:" + base64.RawURLEncoding.EncodeToString(mac[:16]), nil
}

func localTransportID(planKey []byte) string {
	localID, err := deriveTransportID(planKey, "local", nil, "")
	if err != nil {
		return "local"
	}
	return localID
}

func buildTransportDescriptor(decoratorName string, args map[string]any, parentID string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(decoratorName)
	buf.WriteByte(0)
	buf.WriteString(parentID)
	buf.WriteByte(0)

	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		buf.WriteString(key)
		buf.WriteByte(0)
		valueBytes, err := transportValueBytes(args[key])
		if err != nil {
			return nil, fmt.Errorf("transport arg %q: %w", key, err)
		}
		buf.Write(valueBytes)
		buf.WriteByte(0)
	}

	return buf.Bytes(), nil
}

func transportValueBytes(value any) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return []byte("null:"), nil
	case string:
		return []byte("string:" + v), nil
	case []byte:
		return append([]byte("bytes:"), v...), nil
	case bool:
		return []byte("bool:" + strconv.FormatBool(v)), nil
	case int:
		return []byte("int:" + strconv.FormatInt(int64(v), 10)), nil
	case int64:
		return []byte("int64:" + strconv.FormatInt(v, 10)), nil
	case float64:
		return []byte("float64:" + strconv.FormatFloat(v, 'g', -1, 64)), nil
	default:
		canonical, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal value: %w", err)
		}
		return append([]byte("json:"), canonical...), nil
	}
}

func evaluateArgs(args []ArgIR, getValue ValueLookup) (map[string]any, error) {
	params := make(map[string]any, len(args))
	for _, arg := range args {
		if arg.Value == nil {
			return nil, fmt.Errorf("transport arg %q has no value", arg.Name)
		}
		val, err := EvaluateExpr(arg.Value, getValue)
		if err != nil {
			return nil, fmt.Errorf("evaluate transport arg %q: %w", arg.Name, err)
		}
		params[arg.Name] = val
	}
	return params, nil
}

func lookupTransportDecorator(name string) (decorator.Transport, decorator.Descriptor, bool) {
	trimmed := strings.TrimPrefix(name, "@")
	if trimmed == "" {
		return nil, decorator.Descriptor{}, false
	}
	entry, ok := decorator.Global().Lookup(trimmed)
	if !ok {
		return nil, decorator.Descriptor{}, false
	}
	transport, ok := entry.Impl.(decorator.Transport)
	if !ok {
		return nil, decorator.Descriptor{}, false
	}
	return transport, entry.Impl.Descriptor(), true
}

func isTransportDecoratorName(name string) bool {
	_, _, ok := lookupTransportDecorator(name)
	return ok
}
