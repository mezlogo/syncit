# TESTING.md

> **Attention AI Assistant & Developers**: This document defines the strict testing rules, guidelines, and decision-making frameworks for this repository. You MUST adhere to these conventions when writing, modifying, or reviewing tests. 
> *Note: This guide is project-agnostic and applies to any Go CLI tool built using this Hexagonal Architecture.*

## 1. Core Testing Philosophy

Because we enforce **Hexagonal Architecture (Ports and Adapters)**, our testing strategy is strictly divided by architectural layer. The goal is to achieve **fast, isolated, and high-coverage** feedback loops without the flakiness of real I/O in business logic tests.

*   **Core (`coremodule/`)**: Tested with **Table-Driven Unit Tests**. Fast, deterministic, 100% isolated.
*   **CLI (`cmd/`)**: Tested with **Argument Mapping & Formatting Tests**. Verifies the "translation" layer between human input and domain DTOs.
*   **Adapters (`adapter/`)**: Tested with **Sandboxed Integration Tests**. The *only* place real I/O is permitted.

---

## 2. Layer 1: Core Modules (`coremodule/`)
**Focus**: Pure business logic, state transitions, edge cases, and boundary validation (`validate.go` / `convert.go`).

### Rules
1.  **NO Real I/O**: NEVER import `os`, `net`, `io`, or touch the filesystem/network.
2.  **Mocking**: MUST mock all `port/` interfaces using `go.uber.org/mock` (gomock) or `github.com/stretchr/testify/mock`.
3.  **Table-Driven Tests**: MUST use data-driven tests to cover happy paths, error states, and edge cases efficiently.

### Guidelines & Skeleton
Define test cases as a slice of structs. Group assertions logically.

```go
// coremodule/<module>/<module>_test.go
package <module>

import (
    "context"
    "errors"
    "testing"
    // port imports...
)

func Test<Service>_<Method>(t *testing.T) {
    ctx := context.Background()
    errBoom := errors.New("boom")

    tests := []struct {
        name        string
        req         port<module>.Request
        setupMock   func(m *Mock<Port>)
        wantResp    port<module>.Response
        wantErr     bool
        errContains string
    }{
        {
            name: "Success: happy path",
            // ... setup ...
        },
        {
            name:        "Validation: invalid input",
            // ... setup ...
            wantErr:     true,
            errContains: "validation failed",
        },
        {
            name: "Error: port returns failure",
            // ... setup ...
            wantErr:     true,
            errContains: "boom",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            mockPort := NewMock<Port>(ctrl)
            if tt.setupMock != nil {
                tt.setupMock(mockPort)
            }

            svc := New(mockPort)
            resp, err := svc.<Method>(ctx, tt.req)

            // Assertions using errors.Is / errors.As and standard checks
            if (err != nil) != tt.wantErr {
                t.Fatalf("unexpected error status: got %v, wantErr %v", err, tt.wantErr)
            }
            if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
                t.Errorf("error mismatch: got %q, want contains %q", err.Error(), tt.errContains)
            }
            // Assert resp...
        })
    }
}
```

---

## 3. Layer 2: CLI Commands (`cmd/`)
**Focus**: Cobra flag parsing, argument mapping to `port/` Request DTOs, and standard output (stdout/stderr) formatting.

### Rules
1.  **Mock the Core**: NEVER execute real core logic. Mock the `port/` interfaces that the command invokes.
2.  **Cobra Testing Utilities**: Use `cmd.SetArgs()`, `cmd.SetOut()`, and `cmd.SetErr()` to capture I/O.
3.  **No Business Logic Testing**: If you find yourself testing *how* data is processed, you are in the wrong layer. Move that test to `coremodule/`.

### Guidelines & Skeleton
Verify that CLI flags correctly populate the Domain DTO, and that the Domain Response is correctly formatted to the user.

```go
// cmd/<subcommand>/<subcommand>_test.go
package <subcommand>

import (
    "bytes"
    "testing"
    // port imports...
)

func Test<Subcommand>Command_FlagMapping(t *testing.T) {
    tests := []struct {
        name          string
        args          []string
        expectedReq   port<subcommand>.Request
        mockResp      port<subcommand>.Response
        expectedOut   string
    }{
        {
            name: "Maps all flags correctly",
            args: []string{"--flag1", "val1", "-f", "val2"},
            expectedReq: port<subcommand>.Request{ /* ... */ },
            mockResp:    port<subcommand>.Response{ /* ... */ },
            expectedOut: "expected formatted output\n",
        },
        {
            name: "Defaults when no flags provided",
            args: []string{},
            expectedReq: port<subcommand>.Request{ /* defaults */ },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockCore := NewMock<Port>(gomock.NewController(t))
            
            // Expect the core to be called with the mapped request
            if tt.expectedReq != (port<subcommand>.Request{}) {
                mockCore.EXPECT().
                    <Method>(gomock.Any(), tt.expectedReq).
                    Return(tt.mockResp, nil)
            }

            cmd := New(mockCore)
            buf := new(bytes.Buffer)
            cmd.SetOut(buf)
            cmd.SetErr(buf)
            cmd.SetArgs(tt.args)

            err := cmd.Execute()
            if err != nil {
                t.Fatalf("Unexpected execution error: %v", err)
            }

            // Verify Output Formatting
            if buf.String() != tt.expectedOut {
                t.Errorf("Output mismatch:\ngot:  %q\nwant: %q", buf.String(), tt.expectedOut)
            }
        })
    }
}
```

---

## 4. Layer 3: Adapters (`adapter/`)
**Focus**: Real I/O integration, OS/Network behavior, error translation, and context cancellation.

### Rules
1.  **Real I/O ONLY**: This is the **only** directory allowed to touch the real filesystem, network, or OS calls. **NO MOCKS** for the external system being adapted.
2.  **Sandboxing**: MUST use `t.TempDir()` for filesystem tests to ensure isolation and automatic cleanup. Use `httptest.NewServer()` for network adapters.
3.  **Error & Context Testing**: MUST verify that OS errors (e.g., `os.ErrNotExist`) are correctly mapped to domain errors, and that `context.Context` cancellation correctly aborts operations.

### Guidelines & Skeleton
Test the boundaries. How does the adapter handle a broken symlink? How does it handle a timeout?

```go
// adapter/<adapter>/<adapter>_test.go
package <adapter>

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func Test<Adapter>_<Method>(t *testing.T) {
    // 1. Setup sandboxed real environment
    dir := t.TempDir()
    targetPath := filepath.Join(dir, "target.txt")
    os.WriteFile(targetPath, []byte("data"), 0644)

    adapter := New()
    ctx := context.Background()

    // 2. Test Happy Path
    result, err := adapter.<Method>(ctx, targetPath)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // Assert result...

    // 3. Test OS Error Mapping
    _, err = adapter.<Method>(ctx, filepath.Join(dir, "nonexistent"))
    if !errors.Is(err, port<adapter>.ErrNotFound) { // Or whatever domain error is expected
        t.Errorf("expected domain not found error, got: %v", err)
    }
}

func Test<Adapter>_ContextCancellation(t *testing.T) {
    adapter := New()
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    _, err := adapter.<Method>(ctx, t.TempDir())
    if !errors.Is(err, context.Canceled) {
        t.Errorf("expected context.Canceled, got: %v", err)
    }
}
```

---

## 5. Decision Matrix for Agents & Developers

When writing or reviewing a test, use this matrix to make decisions and enforce boundaries.

| Question / Scenario | Rule / Action |
| :--- | :--- |
| **Am I in `coremodule/` and importing `os` / `net` / `io`?** | **FAIL**. Core must be pure. Remove I/O, use `port/` interfaces, and mock them. |
| **Am I in `coremodule/` and using real filesystem/network?** | **FAIL**. Use Table-Driven tests with Mocks. |
| **Am I in `cmd/` and testing business logic / data processing?** | **FAIL**. Move logic to `coremodule/`. CLI should only test flag mapping and output formatting. |
| **Am I in `adapter/` and using Mocks for the external system?** | **FAIL**. Adapters MUST test real integrations using `t.TempDir()` or `httptest`. |
| **How do I test `validate.go` or `convert.go`?** | Test them as pure functions within the `coremodule/` table-driven tests, or write specific unit tests for them in the same package. |
| **How do I assert errors?** | NEVER use string matching for errors unless checking for a specific substring in a wrapped error. Use `errors.Is()` and `errors.As()`. |
| **How do I handle `context.Context` in tests?** | Always pass `context.Background()` or `t.Context()` (Go 1.24+) for happy paths. Explicitly test `context.WithCancel()` in `adapter/` to ensure graceful shutdowns. |

---

## 6. Go Idioms & Tooling Standards

1.  **Mocking Libraries**: 
    *   Prefer `go.uber.org/mock` (gomock) for strict interface mocking.
    *   Use `github.com/stretchr/testify/mock` if flexible, dynamic return values are needed.
    *   *Never* write manual mock structs unless absolutely necessary for simple function-based interfaces.
2.  **Test Naming**: Use descriptive names in table-driven tests: `name: "Success: handles empty directory"`.
3.  **Subtests**: Always use `t.Run(tt.name, func(t *testing.T) { ... })` to isolate test cases and allow parallel execution where safe.
4.  **Cleanup**: Rely on `t.TempDir()` and `t.Cleanup()` instead of manual `os.RemoveAll` to prevent test pollution.
5.  **Linting**: All tests must pass `golangci-lint` with strict settings enabled (e.g., `errcheck`, `govet`, `staticcheck`).
