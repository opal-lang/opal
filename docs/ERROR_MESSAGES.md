---
title: "Error Message Guidelines"
audience: "Core Developers & AI Agents"
summary: "Writing clear, actionable error messages for humans and machines"
---

# Error Message Guidelines

**Goal: Error messages should be so clear that users can fix the problem without reading documentation.**

**Audience**: Core developers, AI agents, and contributors writing error messages.

## Core Principles

### 1. Precision in Wording

**Be specific about what's wrong and why:**
- ✅ "parameter 'times' expects integer between 1 and 100, got string"
- ❌ "invalid parameter value"

**Use lowercase without trailing punctuation:**
- ✅ "provided string was not `true` or `false`"
- ❌ "Invalid boolean value."

**Don't repeat what's obvious from the caret:**
- ✅ Message: "expects integer, got string" + caret points to `"not_a_number"`
- ❌ Message: "parameter 'times' with value \"not_a_number\" expects integer"

### 2. Show Exact Types and Values

**Include inferred types in error messages:**
- ✅ "invalid operands to binary expression ('int' and 'struct A')"
- ❌ "type mismatch"

**Show actual constraints, not generic descriptions:**
- ✅ "must be between 1 and 100"
- ❌ "must be a valid integer"

**Quote string values, show numeric values as-is:**
- ✅ `got string "not_a_number"`
- ✅ `got integer 200`
- ❌ `got not_a_number`

### 3. Provide Actionable Suggestions

**Show concrete, copy-pasteable fixes:**
- ✅ `= try: @retry(times=3) { echo "test" }`
- ❌ `= suggestion: use a valid integer value`

**Use realistic values from schema constraints:**
- For ranges: use midpoint or common value (e.g., 3 for 1-100, not random 42)
- For enums: show first valid value or most common
- For objects: show minimal valid structure
- For arrays: show 1-2 element example

**Don't use placeholder values:**
- ✅ `@retry(times=3)` (3 is valid and realistic)
- ❌ `@retry(times=VALUE)` (placeholder, not actionable)
- ❌ `@retry(times=42)` (random number, seems arbitrary)

### 4. Add Context When Helpful

**Brief explanation of what the parameter does:**
- ✅ `= help: 'times' controls the number of retry attempts`
- ❌ `= note: this parameter is required` (obvious)

**Only add context when it clarifies the error:**
- Add: When the purpose isn't obvious from the parameter name
- Skip: When the parameter name is self-explanatory
- Skip: When the error message already makes it clear

### 5. Avoid Redundancy

**Don't show information that's already visible:**
- ❌ `Parameter: times` (already in error message)
- ❌ `Got: "not_a_number"` (already highlighted with caret)
- ❌ `Code: SCHEMA_TYPE_MISMATCH` (technical jargon, not helpful by default)

**Exception: Show redundant info with `--verbose` flag for tooling/debugging**

## Error Format Structure

### Standard Format (Rust-style)

```
Error: <specific problem with types/constraints>
  --> <file>:<line>:<col>
   |
<line> | <source code>
   |    <caret and range highlighting>
   |
   = help: <brief context about why this matters>
   = try: <concrete, copy-pasteable fix>
```

### Examples

#### Type Mismatch
```
Error: parameter 'times' expects integer between 1 and 100, got string "not_a_number"
  --> test.opl:1:15
   |
 1 | @retry(times="not_a_number") { echo "test" }
   |               ^^^^^^^^^^^^^^ expected integer, found string
   |
   = try: @retry(times=3) { echo "test" }
```

#### Range Violation
```
Error: parameter 'times' value 200 exceeds maximum of 100
  --> test.opl:1:15
   |
 1 | @retry(times=200) { echo "test" }
   |               ^^^ must be between 1 and 100
   |
   = try: @retry(times=10) { echo "test" }
```

#### Invalid Enum Value
```
Error: parameter 'backoff' has invalid value "invalid"
  --> test.opl:1:17
   |
 1 | @retry(backoff="invalid") { echo "test" }
   |                 ^^^^^^^^^ must be one of: linear, exponential, constant
   |
   = try: @retry(backoff="exponential") { echo "test" }
```

#### Missing Required Parameter
```
Error: missing required parameter 'property'
  --> test.opl:1:1
   |
 1 | @env
   |     ^ expected environment variable name
   |
   = help: 'property' specifies which environment variable to read
   = try: @env.HOME
   = try: @env(property="HOME")
```

#### Object Field Type Mismatch
```
Error: object field 'timeout' expects duration, got integer 300
  --> test.opl:1:30
   |
 1 | @config(settings={timeout: 300})
   |                            ^^^ expected duration like "5m", found integer
   |
   = try: @config(settings={timeout: "5m"})
```

## Implementation Guidelines

### For Parser Errors

**Use `errorSchema()` for schema validation errors:**
```go
p.errorSchema(
    ErrorCodeSchemaRangeViolation,
    paramName,
    fmt.Sprintf("parameter '%s' value %v exceeds maximum of %v", 
        paramName, value, *schema.Maximum),
    p.generateConcreteSuggestion(paramName, paramSchema, decoratorName),
    fmt.Sprintf("integer between %v and %v", *schema.Minimum, *schema.Maximum),
    fmt.Sprintf("%v", value),
)
```

**Generate concrete suggestions:**
```go
func (p *parser) generateConcreteSuggestion(paramName string, schema types.ParamSchema, decoratorName string) string {
    // Use schema constraints to pick realistic value
    var exampleValue string
    
    switch schema.Type {
    case types.TypeInt:
        if schema.Minimum != nil && schema.Maximum != nil {
            // Use midpoint or common value
            mid := (*schema.Minimum + *schema.Maximum) / 2
            exampleValue = fmt.Sprintf("%d", int(mid))
        } else if len(schema.Examples) > 0 {
            exampleValue = schema.Examples[0]
        } else {
            exampleValue = "3" // Common small integer
        }
    case types.TypeEnum:
        if schema.EnumSchema != nil && len(schema.EnumSchema.Values) > 0 {
            // Use first non-deprecated value
            exampleValue = fmt.Sprintf("%q", schema.EnumSchema.Values[0])
        }
    // ... other types
    }
    
    return fmt.Sprintf("@%s(%s=%s) { ... }", decoratorName, paramName, exampleValue)
}
```

### For Runtime Errors

**Follow the same principles:**
- Be specific about what failed and why
- Show actual values and types
- Provide concrete fix suggestions
- Add context when helpful

**Example:**
```go
return fmt.Errorf(
    "failed to connect to %s:%d: connection refused\n" +
    "  help: ensure the service is running and accessible\n" +
    "  try: check with 'nc -zv %s %d'",
    host, port, host, port,
)
```

## Error Codes (For Tooling)

**Error codes are for programmatic handling, not human display:**
- Store in `ParseError.Code` field ✅
- Use for `--verbose` output ✅
- Use for `opal explain <CODE>` command ✅
- Use for IDE integration ✅
- **Don't show in default error output** ❌

**Error code naming:**
- Use SCREAMING_SNAKE_CASE
- Prefix with category: `SCHEMA_`, `PARSE_`, `RUNTIME_`
- Be specific: `SCHEMA_RANGE_VIOLATION` not `SCHEMA_ERROR`

**Available error codes:**
- `SCHEMA_TYPE_MISMATCH` - Parameter type doesn't match schema
- `SCHEMA_REQUIRED_MISSING` - Required parameter not provided
- `SCHEMA_ENUM_INVALID` - Value not in enum list
- `SCHEMA_ENUM_DEPRECATED` - Using deprecated enum value (warning)
- `SCHEMA_PATTERN_MISMATCH` - String doesn't match regex pattern
- `SCHEMA_ADDITIONAL_PROP` - Object has unexpected field
- `SCHEMA_RANGE_VIOLATION` - Number outside min/max range
- `SCHEMA_INT_REQUIRED` - Integer required but got float
- `SCHEMA_FORMAT_INVALID` - String doesn't match format (URI, CIDR, etc.)
- `SCHEMA_LENGTH_VIOLATION` - String/array length outside min/max
- `SCHEMA_ARRAY_ELEMENT_TYPE` - Array element has wrong type
- `SCHEMA_OBJECT_FIELD_TYPE` - Object field has wrong type

## Testing Error Messages

**Test complete error structure:**
```go
func TestErrorMessage(t *testing.T) {
    tree := Parse([]byte(`@retry(times=200) { echo "test" }`))
    
    if len(tree.Errors) == 0 {
        t.Fatal("expected error")
    }
    
    err := tree.Errors[0]
    
    // Test message is specific and includes constraints
    assert.Contains(t, err.Message, "exceeds maximum of 100")
    assert.Contains(t, err.Message, "200")
    
    // Test suggestion is concrete and actionable
    assert.Contains(t, err.Suggestion, "@retry(times=")
    assert.NotContains(t, err.Suggestion, "VALUE") // No placeholders
    assert.NotContains(t, err.Suggestion, "42") // No arbitrary values
    
    // Test error code is set (for tooling)
    assert.Equal(t, ErrorCodeSchemaRangeViolation, err.Code)
    
    // Test schema fields are populated
    assert.Equal(t, "times", err.Path)
    assert.Contains(t, err.ExpectedType, "between 1 and 100")
    assert.Equal(t, "200", err.GotValue)
}
```

**Golden tests for formatted output:**
```go
func TestErrorFormatting(t *testing.T) {
    tree := Parse([]byte(`@retry(times=200) { echo "test" }`))
    formatter := ErrorFormatter{Source: source}
    
    output := formatter.Format(tree.Errors[0])
    
    // Compare against golden file
    golden := readGoldenFile("testdata/error_range_violation.txt")
    if diff := cmp.Diff(golden, output); diff != "" {
        t.Errorf("Error format mismatch (-want +got):\n%s", diff)
    }
}
```

## Common Mistakes to Avoid

### ❌ Vague Messages
```
Error: invalid parameter
Error: validation failed
Error: type error
```

### ✅ Specific Messages
```
Error: parameter 'times' expects integer between 1 and 100, got string "not_a_number"
Error: parameter 'backoff' has invalid value "invalid", must be one of: linear, exponential, constant
Error: object field 'timeout' expects duration, got integer 300
```

### ❌ Generic Suggestions
```
= suggestion: use a valid value
= suggestion: fix the type
= suggestion: provide the correct parameter
```

### ✅ Concrete Suggestions
```
= try: @retry(times=3) { echo "test" }
= try: @retry(backoff="exponential") { echo "test" }
= try: @config(settings={timeout: "5m"})
```

### ❌ Arbitrary Examples
```
= try: @retry(times=42) { echo "test" }  // Why 42?
= try: @retry(times=VALUE) { echo "test" }  // Placeholder
= try: @retry(times=X) { echo "test" }  // Not actionable
```

### ✅ Realistic Examples
```
= try: @retry(times=3) { echo "test" }  // 3 is common for retries
= try: @timeout(duration="5m") { ... }  // 5m is common timeout
= try: @parallel(workers=4) { ... }  // 4 is common for CPU cores
```

## Error Code Reference

**All parser schema validation error codes with examples.**

Error codes are stored in `ParseError.Code` for programmatic handling (LSP, tooling, `--explain` flag). They are **not** shown in default error output to keep messages clean for humans.

### SCHEMA_TYPE_MISMATCH

**When**: Parameter type doesn't match schema expectation

**Example**:
```
Error: parameter 'times' expects integer between 1 and 100, got string
  --> test.opl:3:15
   |
 3 | @retry(times="not_a_number") {
   |               ^^^^^^^^^^^^^^
   |
   = try: @retry(times=50) { ... }
```

**Fields**:
- `Code`: `SCHEMA_TYPE_MISMATCH`
- `Path`: `times`
- `ExpectedType`: `"integer between 1 and 100"`
- `GotValue`: `"string"`

---

### SCHEMA_REQUIRED_MISSING

**When**: Required parameter not provided

**Example**:
```
Error: missing required parameter 'times'
  --> test.opl:3:1
   |
 3 | @retry {
   | ^^^^^^
   |
   = try: @retry(times=3) { ... }
```

**Fields**:
- `Code`: `SCHEMA_REQUIRED_MISSING`
- `Path`: `times`
- `ExpectedType`: `"integer between 1 and 100"`
- `GotValue`: `""` (empty)

---

### SCHEMA_ENUM_INVALID

**When**: Value not in enum list

**Example**:
```
Error: parameter 'backoff' has invalid value "invalid"
  --> test.opl:3:18
   |
 3 | @retry(backoff="invalid") {
   |                ^^^^^^^^^
   |
   = try: Use one of: "linear", "exponential", "constant"
```

**Fields**:
- `Code`: `SCHEMA_ENUM_INVALID`
- `Path`: `backoff`
- `ExpectedType`: `"one of [linear exponential constant]"`
- `GotValue`: `"\"invalid\""`

---

### SCHEMA_ENUM_DEPRECATED

**When**: Using deprecated enum value (warning, not error)

**Example**:
```
Warning: parameter 'strategy' value "old_name" is deprecated
  --> test.opl:3:20
   |
 3 | @config(strategy="old_name") {
   |                  ^^^^^^^^^^
   |
   = help: Use "new_name" instead
```

**Fields**:
- `Code`: `SCHEMA_ENUM_DEPRECATED`
- `Path`: `strategy`
- `ExpectedType`: `"one of [new_name other_value]"`
- `GotValue`: `"\"old_name\""`

---

### SCHEMA_RANGE_VIOLATION

**When**: Number outside min/max range

**Example**:
```
Error: invalid value for parameter 'times'
  --> test.opl:3:15
   |
 3 | @retry(times=200) {
   |               ^^^
   |
   = help: Value must be between 1 and 100
```

**Fields**:
- `Code`: `SCHEMA_RANGE_VIOLATION`
- `Path`: `times`
- `ExpectedType`: `"integer between 1 and 100"`
- `GotValue`: `"200"`

---

### SCHEMA_PATTERN_MISMATCH

**When**: String doesn't match regex pattern

**Example**:
```
Error: invalid value for parameter 'name'
  --> test.opl:3:14
   |
 3 | @config(name="123-invalid") {
   |              ^^^^^^^^^^^^^
   |
   = help: Must match pattern /^[a-z][a-z0-9-]*$/
```

**Fields**:
- `Code`: `SCHEMA_PATTERN_MISMATCH`
- `Path`: `name`
- `ExpectedType`: `"string matching /^[a-z][a-z0-9-]*$/"`
- `GotValue`: `"\"123-invalid\""`

---

### SCHEMA_FORMAT_INVALID

**When**: String doesn't match typed format (URI, CIDR, semver, etc.)

**Example**:
```
Error: invalid value for parameter 'endpoint'
  --> test.opl:3:19
   |
 3 | @api(endpoint="not-a-uri") {
   |               ^^^^^^^^^^^^
   |
   = help: Must be valid URI format
   = try: @api(endpoint="https://example.com")
```

**Fields**:
- `Code`: `SCHEMA_FORMAT_INVALID`
- `Path`: `endpoint`
- `ExpectedType`: `"uri format"`
- `GotValue`: `"\"not-a-uri\""`

**Supported formats**: `uri`, `hostname`, `ipv4`, `cidr`, `semver`, `duration`

---

### SCHEMA_INT_REQUIRED

**When**: Integer required but got float

**Example**:
```
Error: invalid value for parameter 'times'
  --> test.opl:3:15
   |
 3 | @retry(times=3.5) {
   |               ^^^
   |
   = help: Must be an integer (no decimal point)
   = try: @retry(times=3)
```

**Fields**:
- `Code`: `SCHEMA_INT_REQUIRED`
- `Path`: `times`
- `ExpectedType`: `"integer"`
- `GotValue`: `"3.5"`

---

### SCHEMA_LENGTH_VIOLATION

**When**: String or array length outside min/max

**Example**:
```
Error: invalid value for parameter 'name'
  --> test.opl:3:14
   |
 3 | @config(name="ab") {
   |              ^^^^
   |
   = help: Length must be between 3 and 50 characters
```

**Fields**:
- `Code`: `SCHEMA_LENGTH_VIOLATION`
- `Path`: `name`
- `ExpectedType`: `"string (length 3-50)"`
- `GotValue`: `"\"ab\""`

---

### SCHEMA_ADDITIONAL_PROP

**When**: Object has unexpected field (closed objects by default)

**Example**:
```
Error: invalid value for parameter 'config'
  --> test.opl:3:16
   |
 3 | @deploy(config={timeout: "5m", unknown: "value"}) {
   |                                ^^^^^^^^^^^^^^^^
   |
   = help: Object does not allow additional properties
   = note: Valid fields: timeout, retries, backoff
```

**Fields**:
- `Code`: `SCHEMA_ADDITIONAL_PROP`
- `Path`: `config.unknown`
- `ExpectedType`: `"object"`
- `GotValue`: `"{timeout: \"5m\", unknown: \"value\"}"`

---

### SCHEMA_ARRAY_ELEMENT_TYPE

**When**: Array element has wrong type

**Example**:
```
Error: invalid value for parameter 'ports'
  --> test.opl:3:15
   |
 3 | @expose(ports=[80, "443", 8080]) {
   |                    ^^^^^
   |
   = help: Array elements must be integers
```

**Fields**:
- `Code`: `SCHEMA_ARRAY_ELEMENT_TYPE`
- `Path`: `ports[1]`
- `ExpectedType`: `"integer"`
- `GotValue`: `"\"443\""`

---

### SCHEMA_OBJECT_FIELD_TYPE

**When**: Object field has wrong type

**Example**:
```
Error: object field 'timeout' expects duration, got integer
  --> test.opl:3:16
   |
 3 | @config(settings={timeout: 300}) {
   |                            ^^^
   |
   = try: @config(settings={timeout: "5m"})
```

**Fields**:
- `Code`: `SCHEMA_OBJECT_FIELD_TYPE`
- `Path`: `settings.timeout`
- `ExpectedType`: `"duration"`
- `GotValue`: `"300"`

---

### Error Code Usage

**For humans (default output)**:
- ❌ Don't show error codes in default output
- ✅ Show clear, actionable messages with examples

**For machines (LSP, tooling)**:
- ✅ Use `ParseError.Code` for programmatic handling
- ✅ Use `ParseError.Path` for navigation
- ✅ Use `ParseError.ExpectedType` and `ParseError.GotValue` for IDE hints

**For debugging (`--explain` flag, Phase 9)**:
- ✅ Show full error structure including code
- ✅ Show detailed explanation of error code
- ✅ Show links to documentation

---

## References

- [Rust API Guidelines - Error Messages](https://rust-lang.github.io/api-guidelines/interoperability.html#error-types-are-meaningful-and-well-behaved-c-good-err)
- [Clang - Expressive Diagnostics](https://clang.llvm.org/diagnostics.html)
- [Elm - Compiler Errors for Humans](https://elm-lang.org/news/compiler-errors-for-humans)

## Summary

**Good error messages are:**
1. **Specific** - Show exact types, values, and constraints
2. **Actionable** - Provide concrete, copy-pasteable fixes
3. **Contextual** - Explain why when it helps
4. **Clean** - No redundant information
5. **Realistic** - Use schema-derived examples, not arbitrary values

**Remember: Users should be able to fix the problem without reading documentation.**
