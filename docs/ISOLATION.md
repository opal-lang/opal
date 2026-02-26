---
title: "Isolation Decorators"
audience: "End Users and Script Authors"
summary: "Process isolation decorators for secure, sandboxed execution"
---

# Isolation Decorators

Isolation decorators wrap execution in sandboxed contexts using operating system isolation primitives.

These decorators use namespace-based isolation on Linux. Other platforms have limited support or return errors at plan time.

## Available Decorators

### `@isolated.network.loopback()`

Executes commands with loopback-only network access. External network calls fail.

```opal
@isolated.network.loopback() {
    curl http://localhost:8080/health  # works
    curl https://external.api.com      # fails
}
```

**Platform support:** Linux only. Returns plan-time error on macOS and Windows.

---

### `@isolated.filesystem.readonly()`

Executes commands with read-only filesystem access. Write operations fail.

```opal
@isolated.filesystem.readonly() {
    cat /etc/config.yaml    # works
    echo "test" > /tmp/out  # fails
}
```

**Platform support:** Linux only. Returns plan-time error on macOS and Windows.

---

### `@isolated.filesystem.ephemeral()`

Executes commands with an ephemeral filesystem. Changes are discarded after the block completes.

```opal
@isolated.filesystem.ephemeral() {
    echo "temp data" > /tmp/tempfile
    cat /tmp/tempfile  # works, shows "temp data"
}
# /tmp/tempfile no longer exists outside the block
```

**Platform support:** Linux only. Returns plan-time error on macOS and Windows.

---

### `@isolated.memory.lock()`

Locks process memory to prevent swapping. Useful for handling sensitive data that should not be written to disk.

```opal
@isolated.memory.lock() {
    # Sensitive operations with secrets in memory
    decrypt-secrets --key @secret.api_key
}
```

**Platform support:** Linux and macOS only. Returns plan-time error on Windows.

---

### `@isolated.privileges.drop()`

Drops supplementary privileges from the process. Reduces attack surface by removing unnecessary capabilities.

```opal
@isolated.privileges.drop() {
    # Run with minimal privileges
    process-user-data --input @var.data_file
}
```

**Platform support:** Linux and macOS only. Returns plan-time error on Windows.

---

## Platform Support Matrix

| Decorator | Linux | macOS | Windows |
|-----------|-------|-------|---------|
| `network.loopback` | Full support | Error | Error |
| `filesystem.readonly` | Full support | Error | Error |
| `filesystem.ephemeral` | Full support | Error | Error |
| `memory.lock` | Full support | Full support | Error |
| `privileges.drop` | Full support | Full support | Error |

**Legend:**
- **Full support:** Isolation feature works as documented
- **Error:** Plan-time error when decorator is used
- **No-op:** Decorator succeeds but performs no isolation

---

## Stacking Decorators

Isolation decorators can be nested to combine protections:

```opal
@isolated.network.loopback() {
    @isolated.memory.lock() {
        @isolated.privileges.drop() {
            # Network isolated, memory locked, privileges dropped
            process-sensitive-data
        }
    }
}
```

The innermost block runs with all stacked isolation properties applied.

---

## Privilege Requirements

Some isolation decorators require elevated privileges:

- **Linux:** `network.loopback` requires `CAP_NET_ADMIN` capability or root
- **Linux:** `filesystem.ephemeral` may require `CAP_SYS_ADMIN` for certain mount operations
- **Windows:** Network isolation via WFP (Windows Filtering Platform) requires Administrator privileges

Run scripts with appropriate privileges or use capability management tools on Linux.
