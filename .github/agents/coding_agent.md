# Terraform Provider for Braze - Coding Agent Configuration

## Project Overview

This is a **Terraform provider for the Braze messaging platform**, written in Go. It enables infrastructure-as-code management of Braze resources (e.g., content blocks) using Terraform.

### Key Technologies
- **Language**: Go 1.25.4
- **Framework**: Terraform Plugin Framework v1.16.1
- **Code Generation**: ogen v1.17.0 (for Braze API client)
- **Testing**: Standard Go testing + Terraform acceptance tests
- **Linting**: golangci-lint with comprehensive configuration

## Project Structure

```
.
├── internal/
│   ├── braze-client-go/          # Auto-generated Braze API client
│   │   ├── openapi/               # OpenAPI specifications (YAML files)
│   │   ├── ogen.yml               # ogen configuration for code generation
│   │   ├── oas_*_gen.go           # Generated client code (DO NOT EDIT)
│   │   └── testing/               # Local test server for integration tests
│   │       ├── server.go          # Test server implementation
│   │       ├── handler*.go        # Request handlers
│   │       └── server_test.go     # Server tests
│   └── provider/                  # Terraform provider implementation
│       ├── braze_provider.go      # Provider configuration
│       ├── braze_content_block_resource.go  # Content block resource
│       ├── *_model.go             # Data models
│       ├── *_schema.go            # Terraform schemas
│       └── *_test.go              # Unit and acceptance tests
├── main.go                        # Provider entry point
├── tools/                         # Build tools
│   └── tools.go                   # Tool dependencies
├── examples/                      # Example Terraform configurations
├── docs/                          # Generated documentation
├── .golangci.yml                  # Linting configuration
└── .github/workflows/             # CI/CD workflows
```

## Code Generation

### Braze API Client (ogen)

The Braze API client is **auto-generated** from OpenAPI specifications using ogen:

- **Source**: `internal/braze-client-go/openapi/openapi.yml`
- **Generated files**: `internal/braze-client-go/oas_*_gen.go`
- **Generation command**: 
  ```bash
  go generate ./internal/braze-client-go/
  ```
  Or via the go:generate directive in `internal/braze-client-go/ogen.go`:
  ```go
  //go:generate go run github.com/ogen-go/ogen/cmd/ogen -target . -package brazeclient -clean openapi/openapi.yml
  ```

**Important**: Never manually edit generated files (files with `_gen.go` suffix). Always modify the OpenAPI spec and regenerate.

### Documentation Generation

Documentation is auto-generated from provider code:

```bash
go generate ./...
```

This runs `terraform-plugin-docs` to generate documentation in the `docs/` directory.

## Local Test Server

The project includes a **local test server** for integration testing without requiring a real Braze API instance:

- **Location**: `internal/braze-client-go/testing/`
- **Purpose**: Mock Braze API for acceptance tests
- **Implementation**: Implements the generated ogen server interface
- **Usage**: Tests create a local HTTP server with the test handlers

Example usage pattern:
```go
server, err := testing.NewBrazeServer()
// Configure test data via server.Handler()
// Use in tests with httptest
```

## Building and Testing

### Build Commands

```bash
# Build the provider
go build -v .

# Download dependencies
go mod download

# Install tools
go mod download
```

### Testing Commands

```bash
# Run all unit tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.txt -covermode=atomic -coverpkg=./... ./...

# Run acceptance tests (requires TF_ACC=1)
TF_ACC=1 go test -v ./internal/provider/

# Run specific test
go test -v ./internal/provider/ -run TestAccBrazeContentBlockResource
```

### Linting

```bash
# Run golangci-lint
golangci-lint run

# With auto-fix
golangci-lint run --fix
```

**Linter configuration**: `.golangci.yml`
- All linters enabled by default
- Exceptions: depguard, exhaustruct, funlen, lll, wsl
- Generated code (`_gen.go`) is automatically excluded
- Max cyclomatic complexity: 20

## Development Conventions

### Code Style

1. **Formatting**: Code uses `gofumpt`, `gofmt`, and `goimports`
2. **Comments**: Match existing style; not required for simple code
3. **Naming**: 
   - Terraform resources: `braze_<resource_name>`
   - Go types: CamelCase
   - Variables: camelCase (single letters allowed: i, k, r, v, w)

### Testing Patterns

1. **Unit tests**: Standard Go tests in `*_test.go` files
2. **Acceptance tests**: Use `terraform-plugin-testing` framework
   - Must set `TF_ACC=1` environment variable
   - Test against local test server (not real API)
   - Example: `internal/provider/braze_content_block_resource_test.go`

3. **Test naming**: `Test<Type><Name>` (e.g., `TestAccBrazeContentBlockResource`)

### Provider Resources

When adding new resources:
1. Define OpenAPI spec in `internal/braze-client-go/openapi/`
2. Regenerate client: `go generate ./internal/braze-client-go/`
3. Implement resource in `internal/provider/`:
   - `braze_<resource>_resource.go` - Resource implementation
   - `braze_<resource>_schema.go` - Terraform schema
   - `braze_<resource>_model*.go` - Data models
   - `braze_<resource>_resource_test.go` - Tests
4. Add test handlers in `internal/braze-client-go/testing/`
5. Update provider registration in `braze_provider.go`
6. Generate documentation: `go generate ./...`

## Dependencies

### Core Dependencies
- `github.com/hashicorp/terraform-plugin-framework` - Terraform plugin SDK
- `github.com/hashicorp/terraform-plugin-testing` - Testing framework
- `github.com/ogen-go/ogen` - OpenAPI code generation
- `github.com/go-faster/jx` - JSON library (used by ogen)
- `github.com/google/uuid` - UUID generation
- `github.com/hashicorp/go-retryablehttp` - HTTP client with retries

### Development Tools
- `github.com/hashicorp/terraform-plugin-docs` - Doc generation
- `github.com/ogen-go/ogen/cmd/ogen` - Code generation CLI

## CI/CD Workflows

Located in `.github/workflows/`:

1. **test.yml**: Build, unit tests, and acceptance tests
2. **golangci.yml**: Linting with golangci-lint
3. **gogenerate.yml**: Verify generated code is up-to-date
4. **release.yml**: GoReleaser for versioned releases

## Common Tasks

### Add a new Braze resource

1. Define OpenAPI spec in `internal/braze-client-go/openapi/schemas/<resource>/`
2. Update `internal/braze-client-go/openapi/openapi.yml` to reference new schemas
3. Regenerate client: `go generate ./internal/braze-client-go/`
4. Create resource implementation in `internal/provider/`
5. Add test handlers in `internal/braze-client-go/testing/`
6. Write tests
7. Register resource in provider
8. Generate docs: `go generate ./...`

### Update Braze API client

1. Modify OpenAPI specs in `internal/braze-client-go/openapi/`
2. Run: `go generate ./internal/braze-client-go/`
3. Update affected provider code
4. Update tests
5. Verify: `go test ./...`

### Fix linting issues

```bash
golangci-lint run --fix
```

Most issues are auto-fixable. For complex issues, check `.golangci.yml` for configured rules.

## Important Notes

1. **DO NOT** manually edit files ending in `_gen.go` - they are auto-generated
2. **Always** run `go generate` after modifying OpenAPI specs
3. **Use the local test server** for acceptance tests, not production Braze API
4. **Follow existing patterns** when adding new resources
5. **Run tests and linters** before committing code
6. Generated code exclusions are handled automatically by golangci-lint

## Useful Commands Reference

```bash
# Full development workflow
go mod download                    # Install dependencies
go generate ./...                  # Generate all code
go build -v .                      # Build
golangci-lint run                  # Lint
go test -v ./...                   # Test
TF_ACC=1 go test -v ./internal/provider/  # Acceptance tests

# Code generation
go generate ./internal/braze-client-go/   # Regenerate API client
go generate ./...                          # Regenerate all (includes docs)

# Testing specific components
go test -v ./internal/provider/            # Provider tests
go test -v ./internal/braze-client-go/testing/  # Test server tests
```

## Getting Help

- **Terraform Plugin Framework**: https://developer.hashicorp.com/terraform/plugin/framework
- **ogen Documentation**: https://ogen.dev/
- **Project README**: `/README.md`
