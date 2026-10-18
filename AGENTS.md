# AGENTS.md

> **Attention AI Assistant**: This document contains strict rules for generating code in this repository. You must follow these conventions when writing, modifying, or reviewing code. 
> *Note: For project-specific domain context, feature descriptions, and tool purpose, refer to `README.md`.*

## 1. General Context
* **Language**: Go 1.27+ (Utilize latest features like range over func, improved generics, etc., where applicable).
* **Architecture**: Hexagonal (Ports and Adapters). Strict separation of concerns.
* **CLI Framework**: `github.com/spf13/cobra` (subcommands, help, completions).
* **Mocking**: `go.uber.org/mock` (gomock) or `github.com/stretchr/testify/mock`.

## 2. Strict File Placement Rules
When adding new code, you MUST place it in the correct directory based on its role:
* **`main.go`**: ONLY dependency wiring and app startup. NO domain logic.
* **`cmd/<subcommand>/`**: ONLY CLI parsing, flag definitions, and formatting stdout/stderr.
* **`port/<portname>/`**: ONLY `interface` definitions and `struct` definitions for messages/DTOs. NO logic.
* **`adapter/<adaptername>/`**: ONLY implementations of `port/` interfaces that interact with the OS, network, filesystem, or heavy libraries.
* **`coremodule/<modulename>/`**: ONLY pure business logic. **NEVER** import `os`, `net`, `io`, `database`, or heavy third-party libs here.

## 3. Dependency & Import Rules
* **NEVER** import an `adapter/` package from a `coremodule/` or `cmd/` package.
* **NEVER** import a `coremodule/` package from an `adapter/` package.
* All cross-module dependencies MUST be routed through interfaces defined in `port/`.
* Use **Constructor Injection** for all dependencies. Do not use global variables or package-level state for dependencies.

## 4. Code Generation Rules
* **Interfaces**: Define interfaces in `port/`. Keep them small and focused on the consumer's needs.
* **Validation**: If a function receives external data, create a `validate.go` file in that package. Write pure validation functions (e.g., `func ValidateConfig(c Config) error`).
* **Mapping**: If translating between two different struct types (e.g., CLI flags to Core DTO, or Core DTO to API request), create a `convert.go` file. Write pure mapping functions.
* **Error Handling**: Use standard Go error wrapping (`fmt.Errorf("context: %w", err)`). Define custom error types in the `port/` package if the core needs to react to specific domain errors.

## 5. Testing Rules
* **Core Modules (`coremodule/`)**: 
  * MUST use **Table-Driven Tests** (data-driven). 
  * Define test cases as a slice of structs containing inputs, expected outputs, and setup/teardown mocks.
* **Adapters & CLI (`adapter/`, `cmd/`)**:
  * Use the mocking library to mock the `port/` interfaces.
  * Verify that the adapter correctly handles I/O errors and maps them to domain errors.
  * Verify that CLI commands correctly parse flags and call the core module with the right parameters.
* **No Real I/O in Unit Tests**: Never touch the real file system or network in unit tests. 

## 6. Go Idioms to Enforce
* Use `context.Context` as the first parameter for all methods crossing boundaries (ports/adapters).
* Keep structs small. Prefer passing interfaces to structs.
* Use `errors.Is` and `errors.As` for error checking.
* Format code with `gofmt` and ensure compliance with standard `golangci-lint` strict settings.