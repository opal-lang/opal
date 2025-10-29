# core/decorator - Clean Decorator Architecture

This package contains the clean, new decorator architecture with auto-inference and the `database/sql` registration pattern.

## Key Features

### 1. Auto-Inference of Roles
Decorators just implement interfaces. The registry automatically figures out what they can do:

```go
// Decorator implements Value interface → RoleProvider inferred
type VarDecorator struct{}

func (d *VarDecorator) Descriptor() Descriptor {
    return Descriptor{Path: "var"}
}

func (d *VarDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
    return ctx.Vars[*call.Primary], nil
}

// Registration (database/sql pattern)
func init() {
    decorator.Register("var", &VarDecorator{})
}
```

### 2. Multi-Role Decorators
A single decorator can implement multiple interfaces:

```go
// Implements both Value and Endpoint
type S3ObjectDecorator struct{}

func (d *S3ObjectDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
    // Return metadata
    return map[string]any{"size": 1024, "etag": "abc123"}, nil
}

func (d *S3ObjectDecorator) Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error) {
    // Return writer for upload
    return s3.NewWriter(bucket, key), nil
}

// Registry infers: Roles = [RoleProvider, RoleEndpoint]
```

### 3. URI-Based Path Lookup
Paths are like URIs for the Opal namespace:

```go
decorator.Register("env", &EnvDecorator{})
decorator.Register("aws.secret", &AWSSecretDecorator{})
decorator.Register("aws.s3.object", &AWSS3ObjectDecorator{})

// Lookup
entry, ok := decorator.Global().Lookup("aws.s3.object")
```

### 4. Telemetry Ownership Model
- **Opal runtime**: Tracks decorator entry/exit, total timing, call hierarchy
- **Decorator implementations**: Can create child spans for internal tracking

## Interfaces

- **Value**: Produces values (@var, @env, @aws.secret)
- **Exec**: Wraps execution (@retry, @timeout)
- **Transport**: Creates transport boundaries (@ssh.connect, @docker.exec)
- **Endpoint**: Provides I/O capabilities (@file.read, @s3.put)
- **Session**: Execution context (env, cwd, transport)
- **Span**: Telemetry span for observability

## Migration Plan

This package is **parallel** to `core/types`. The old `core/types` will be deleted in Phase 3 after:
1. Migrating existing decorators to new interfaces
2. Updating runtime/executor to use new registry
3. Updating parser to use new types

## Testing

```bash
cd core/decorator && go test -v
```

All tests demonstrate:
- Auto-inference of roles
- Multi-role decorator support
- URI-based lookup
- database/sql registration pattern
