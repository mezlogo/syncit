# DESIGN.md

## Design Principles & Rationale

### 1. Hexagonal Architecture (Ports & Adapters)
* **Rationale**: CLI tools often degrade into unmaintainable "script-like" code where I/O is tightly coupled with business logic. Enforcing Ports and Adapters ensures the core logic is 100% testable in isolation, without needing to mock the file system, network, or CLI framework.
* **Trade-off**: Requires more boilerplate and upfront structural design than a flat `main.go` script. We accept this for long-term maintainability, testability, and adaptability.

### 2. Strict Module Isolation & Dependency Inversion
* **Rationale**: Modules communicate *only* via interfaces defined in the `port/` directory. This prevents circular dependencies and ensures that swapping an implementation (e.g., changing a local file reader to an S3 reader) requires zero changes to the core logic or the CLI layer.
* **Rule**: A package in `coremodule/` or `adapter/` may import `port/`, but **never** another `adapter/` or `coremodule/` directly.

### 3. Explicit Dependency Injection
* **Rationale**: Go lacks a standard runtime DI framework. We use explicit **Constructor Injection**. 
* **Pattern**: Dependencies are passed via `New...` constructor functions.
* **Trade-off**: Manual wiring in `main.go` can become verbose. We accept this to avoid the "magic", reflection, and hidden control flow of third-party DI containers, keeping the codebase idiomatic and easy to trace.

### 4. Boundary Logic Extraction (`validate.go` & `convert.go`)
* **Rationale**: To keep main logic files focused on a single responsibility, boundary-crossing logic is extracted into dedicated files within the same package.
* **Pattern**: 
  * `validate.go`: Contains pure functions to validate incoming DTOs/messages before they enter the core or adapter.
  * `convert.go`: Contains pure functions to map external domain models to internal port messages, and vice versa.

### 5. Testing Philosophy
* **Core Modules (`coremodule/`)**: Tested using **Data-Driven (Table-Driven) Tests**. Since this layer contains the actual business logic, we use comprehensive table-driven tests to cover edge cases, happy paths, and error states without external dependencies.
* **Adapters & CLI (`adapter/`, `cmd/`)**: Tested using a **Mocking Library**. We mock the `port/` interfaces to verify that the adapter correctly translates I/O and that the CLI correctly parses flags and formats output.
