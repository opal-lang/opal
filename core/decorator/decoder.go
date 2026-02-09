package decorator

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/opal-lang/opal/core/types"
)

type DecodeWarning struct {
	Parameter string
	Message   string
}

type Decoder struct {
	schema     types.DecoratorSchema
	ordered    []types.ParamSchema
	positional []types.ParamSchema
	params     map[string]types.ParamSchema
	validator  *types.Validator
}

type positionalArg struct {
	index int
	value any
}

func CompileDecoder(schema types.DecoratorSchema) *Decoder {
	params := schema.Parameters
	if params == nil {
		params = map[string]types.ParamSchema{}
	}

	ordered := schema.GetOrderedParameters()
	if len(ordered) == 0 && len(params) > 0 {
		names := make([]string, 0, len(params))
		for name := range params {
			names = append(names, name)
		}
		sort.Strings(names)
		ordered = make([]types.ParamSchema, 0, len(names))
		for _, name := range names {
			ordered = append(ordered, params[name])
		}
	}

	positional := positionalBindingOrder(ordered)

	return &Decoder{
		schema:     schema,
		ordered:    ordered,
		positional: positional,
		params:     params,
		validator:  types.NewValidator(types.DefaultValidationConfig()),
	}
}

func NormalizeArgs(schema types.DecoratorSchema, primary *string, raw map[string]any) (map[string]any, []DecodeWarning, error) {
	return CompileDecoder(schema).NormalizeArgs(primary, raw)
}

func ValidateArgs(schema types.DecoratorSchema, canonical map[string]any) ([]DecodeWarning, error) {
	return CompileDecoder(schema).ValidateArgs(canonical)
}

func DecodeInto[T any](schema types.DecoratorSchema, primary *string, raw map[string]any) (T, []DecodeWarning, error) {
	var zero T
	decoder := CompileDecoder(schema)

	canonical, normalizeWarnings, err := decoder.NormalizeArgs(primary, raw)
	if err != nil {
		return zero, nil, err
	}

	validateWarnings, err := decoder.ValidateArgs(canonical)
	if err != nil {
		return zero, append(normalizeWarnings, validateWarnings...), err
	}

	decoded, err := decodeCanonical[T](canonical)
	if err != nil {
		return zero, append(normalizeWarnings, validateWarnings...), err
	}

	return decoded, append(normalizeWarnings, validateWarnings...), nil
}

func (d *Decoder) NormalizeArgs(primary *string, raw map[string]any) (map[string]any, []DecodeWarning, error) {
	canonical := make(map[string]any)
	warnings := make([]DecodeWarning, 0)

	if primary != nil {
		if d.schema.PrimaryParameter == "" {
			return nil, nil, fmt.Errorf("decorator %q does not accept a primary parameter", d.schema.Path)
		}
		canonical[d.schema.PrimaryParameter] = *primary
	}

	if raw == nil {
		raw = map[string]any{}
	}

	positionals := make([]positionalArg, 0)
	namedCount := 0

	for key, value := range raw {
		if index, ok := parsePositionalArgKey(key); ok {
			positionals = append(positionals, positionalArg{index: index, value: value})
			continue
		}

		targetName := key
		if replacement, ok := d.schema.DeprecatedParameters[key]; ok {
			targetName = replacement
			warnings = append(warnings, DecodeWarning{
				Parameter: key,
				Message:   fmt.Sprintf("parameter %q is deprecated, use %q", key, replacement),
			})
		}

		if _, ok := d.params[targetName]; !ok {
			return nil, nil, fmt.Errorf("unknown parameter %q", key)
		}

		if _, exists := canonical[targetName]; exists {
			return nil, nil, fmt.Errorf("duplicate parameter %q", targetName)
		}

		canonical[targetName] = value
		namedCount++
	}

	sort.Slice(positionals, func(i, j int) bool {
		return positionals[i].index < positionals[j].index
	})

	if err := validatePositionalIndexes(positionals, namedCount); err != nil {
		return nil, nil, err
	}

	for _, positional := range positionals {
		found := ""
		for _, param := range d.positional {
			if _, exists := canonical[param.Name]; exists {
				continue
			}
			found = param.Name
			break
		}

		if found == "" {
			return nil, nil, fmt.Errorf("too many positional arguments")
		}

		canonical[found] = positional.value
	}

	return canonical, warnings, nil
}

func validatePositionalIndexes(positionals []positionalArg, namedCount int) error {
	if len(positionals) == 0 {
		return nil
	}

	present := make(map[int]bool, len(positionals))
	maxIndex := 0
	for _, positional := range positionals {
		present[positional.index] = true
		if positional.index > maxIndex {
			maxIndex = positional.index
		}
	}

	missingCount := 0
	firstMissing := 0
	for i := 1; i <= maxIndex; i++ {
		if present[i] {
			continue
		}
		missingCount++
		if firstMissing == 0 {
			firstMissing = i
		}
	}

	if missingCount > namedCount {
		return fmt.Errorf("invalid positional argument index: missing arg%d", firstMissing)
	}

	return nil
}

func (d *Decoder) ValidateArgs(canonical map[string]any) ([]DecodeWarning, error) {
	if canonical == nil {
		canonical = map[string]any{}
	}

	for key := range canonical {
		if _, ok := d.params[key]; !ok {
			return nil, fmt.Errorf("unknown parameter %q", key)
		}
	}

	for name, param := range d.params {
		if _, exists := canonical[name]; !exists && param.Default != nil {
			canonical[name] = param.Default
		}
	}

	for _, name := range d.requiredParameterNames() {
		param := d.params[name]
		if !param.Required {
			continue
		}
		if _, exists := canonical[name]; !exists {
			return nil, fmt.Errorf("missing required parameter %q", name)
		}
	}

	warnings := make([]DecodeWarning, 0)
	for _, name := range sortedArgKeys(canonical) {
		value := canonical[name]
		param := d.params[name]

		if param.EnumSchema != nil {
			if strValue, ok := value.(string); ok {
				if replacement, deprecated := param.EnumSchema.DeprecatedValues[strValue]; deprecated {
					canonical[name] = replacement
					value = replacement
					warnings = append(warnings, DecodeWarning{
						Parameter: name,
						Message:   fmt.Sprintf("value %q for parameter %q is deprecated, use %q", strValue, name, replacement),
					})
				}
			}
		}

		if err := validateStrictType(name, param, value); err != nil {
			return nil, err
		}

		if err := d.validator.ValidateParams(&param, value); err != nil {
			return nil, fmt.Errorf("invalid parameter %q: %w", name, err)
		}
	}

	return warnings, nil
}

func decodeCanonical[T any](canonical map[string]any) (T, error) {
	var out T

	target := reflect.ValueOf(&out).Elem()
	switch target.Kind() {
	case reflect.Struct:
		if err := decodeIntoStruct(target, canonical); err != nil {
			return out, err
		}
		return out, nil
	case reflect.Map:
		if target.Type().Key().Kind() != reflect.String {
			return out, fmt.Errorf("map decode requires string keys")
		}
		if target.IsNil() {
			target.Set(reflect.MakeMap(target.Type()))
		}
		for _, key := range sortedArgKeys(canonical) {
			value := canonical[key]
			v := reflect.ValueOf(value)
			if !v.IsValid() {
				target.SetMapIndex(reflect.ValueOf(key), reflect.Zero(target.Type().Elem()))
				continue
			}
			if !v.Type().AssignableTo(target.Type().Elem()) {
				return out, fmt.Errorf("cannot assign parameter %q (%T) to map value type %s", key, value, target.Type().Elem())
			}
			target.SetMapIndex(reflect.ValueOf(key), v)
		}
		return out, nil
	default:
		return out, fmt.Errorf("DecodeInto target must be struct or map, got %s", target.Kind())
	}
}

func decodeIntoStruct(target reflect.Value, canonical map[string]any) error {
	targetType := target.Type()
	fieldByName := make(map[string]int)

	for i := 0; i < target.NumField(); i++ {
		field := targetType.Field(i)
		if !field.IsExported() {
			continue
		}

		aliases := make([]string, 0, 3)
		if tag := field.Tag.Get("decorator"); tag != "" {
			aliases = append(aliases, tag)
		}
		if tag := field.Tag.Get("json"); tag != "" {
			name := strings.Split(tag, ",")[0]
			if name != "" && name != "-" {
				aliases = append(aliases, name)
			}
		}
		aliases = append(aliases, field.Name)

		for _, alias := range aliases {
			fieldByName[normalizeKey(alias)] = i
		}
	}

	for _, key := range sortedArgKeys(canonical) {
		index, ok := fieldByName[normalizeKey(key)]
		if !ok {
			return fmt.Errorf("no struct field mapped for parameter %q", key)
		}

		field := target.Field(index)
		value := canonical[key]
		refValue := reflect.ValueOf(value)
		if !refValue.IsValid() {
			field.SetZero()
			continue
		}

		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			strValue, ok := value.(string)
			if !ok {
				return fmt.Errorf("cannot assign parameter %q (%T) to field %q (%s)", key, value, targetType.Field(index).Name, field.Type())
			}
			d, err := types.ParseDuration(strValue)
			if err != nil {
				return fmt.Errorf("cannot parse duration for parameter %q: %w", key, err)
			}
			field.Set(reflect.ValueOf(time.Duration(d.Nanoseconds())))
			continue
		}

		if !refValue.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("cannot assign parameter %q (%T) to field %q (%s)", key, value, targetType.Field(index).Name, field.Type())
		}

		field.Set(refValue)
	}

	return nil
}

func validateStrictType(name string, schema types.ParamSchema, value any) error {
	strictType := schema.Type
	switch strictType {
	case types.TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %q expects string", name)
		}
	case types.TypeInt:
		if !isStrictInt(value) {
			return fmt.Errorf("parameter %q expects integer", name)
		}
	case types.TypeFloat:
		if !isStrictFloat(value) {
			return fmt.Errorf("parameter %q expects float", name)
		}
	case types.TypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter %q expects boolean", name)
		}
	case types.TypeDuration:
		switch duration := value.(type) {
		case string:
			if _, err := types.ParseDuration(duration); err != nil {
				return fmt.Errorf("parameter %q expects duration", name)
			}
		case types.Duration:
		case time.Duration:
		default:
			return fmt.Errorf("parameter %q expects duration", name)
		}
	case types.TypeObject:
		if !isStrictObject(value) {
			return fmt.Errorf("parameter %q expects object", name)
		}
	case types.TypeArray:
		if !isStrictArray(value) {
			return fmt.Errorf("parameter %q expects array", name)
		}
	case types.TypeEnum, types.TypeScrubMode:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %q expects string", name)
		}
	case types.TypeAuthHandle, types.TypeSecretHandle:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %q expects string", name)
		}
	default:
		return fmt.Errorf("parameter %q has unsupported type %q", name, strictType)
	}

	return nil
}

func isStrictInt(value any) bool {
	t := reflect.TypeOf(value)
	if t == nil {
		return false
	}
	kind := t.Kind()
	return kind >= reflect.Int && kind <= reflect.Int64
}

func isStrictFloat(value any) bool {
	t := reflect.TypeOf(value)
	if t == nil {
		return false
	}
	kind := t.Kind()
	return kind == reflect.Float32 || kind == reflect.Float64
}

func isStrictObject(value any) bool {
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.Kind() != reflect.Map {
		return false
	}
	return v.Type().Key().Kind() == reflect.String
}

func isStrictArray(value any) bool {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return false
	}
	return v.Kind() == reflect.Array || v.Kind() == reflect.Slice
}

func (d *Decoder) requiredParameterNames() []string {
	names := make([]string, 0, len(d.params))
	for name, param := range d.params {
		if param.Required {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func parsePositionalArgKey(key string) (int, bool) {
	if !strings.HasPrefix(key, "arg") {
		return 0, false
	}
	if len(key) <= 3 {
		return 0, false
	}

	index, err := strconv.Atoi(key[3:])
	if err != nil || index <= 0 {
		return 0, false
	}

	return index, true
}

func sortedArgKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '_' || r == '-' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func positionalBindingOrder(params []types.ParamSchema) []types.ParamSchema {
	if len(params) == 0 {
		return nil
	}

	ordered := make([]types.ParamSchema, 0, len(params))
	for _, param := range params {
		if param.Required {
			ordered = append(ordered, param)
		}
	}
	for _, param := range params {
		if !param.Required {
			ordered = append(ordered, param)
		}
	}

	return ordered
}
