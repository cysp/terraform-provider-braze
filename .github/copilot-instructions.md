# Copilot Instructions for terraform-provider-braze

## Project Overview

This is a Terraform provider for managing Braze configuration. It's built using the Terraform Plugin Framework and provides resources and data sources for interacting with the Braze API.

### Key Technologies
- **Language**: Go 1.25+
- **Framework**: Terraform Plugin Framework
- **API Client**: Auto-generated using ogen from OpenAPI specs
- **Testing**: Standard Go testing + Terraform acceptance tests

## Repository Structure

```
.
├── internal/
│   ├── braze-client-go/     # Auto-generated Braze API client (from OpenAPI)
│   └── provider/            # Terraform provider implementation
│       ├── *_resource.go    # Resource implementations
│       ├── *_resource_test.go # Resource tests
│       └── braze_provider.go # Provider configuration
├── examples/                 # Example Terraform configurations
├── docs/                     # Auto-generated documentation
├── templates/                # Documentation templates
└── main.go                   # Provider entry point
```

## Development Workflow

### Building the Provider

```bash
go build -v .
```

The build process:
1. Downloads dependencies from `go.mod`
2. Compiles the provider binary
3. Binary can be used locally for testing

### Running Tests

**Unit Tests**:
```bash
go test -v ./...
```

**Acceptance Tests** (requires BRAZE_API_KEY):
```bash
TF_ACC=1 go test -v ./internal/provider/
```

Acceptance tests:
- Test against real Braze API
- Require `BRAZE_API_KEY` environment variable
- Create/modify/delete actual resources
- Run in CI with matrix of Terraform versions (1.13, 1.14)

### Linting

```bash
golangci-lint run
```

Configuration in `.golangci.yml`:
- Uses `version: "2"` format
- Enables most linters by default
- Key disabled linters: `depguard`, `exhaustruct`, `funlen`, `lll`, `wsl`
- Custom settings for `cyclop` (max-complexity: 20) and `varnamelen`
- Excludes: generated code, examples, third_party

### Generating Code & Documentation

```bash
go generate ./...
```

This command:
1. Formats Terraform examples (`terraform fmt -recursive ./examples/`)
2. Generates provider documentation using `tfplugindocs`
3. All generated files must be committed (CI checks for uncommitted changes)

## Coding Standards

### General Guidelines

1. **Minimal Changes**: Make the smallest possible changes to achieve the goal
2. **No Breaking Changes**: Don't modify working code unless absolutely necessary
3. **Follow Existing Patterns**: Match the style and structure of existing code
4. **Test Coverage**: Maintain or improve test coverage
5. **Generated Code**: Never manually edit files with `_gen.go` suffix

### Go Code Style

- Follow standard Go conventions
- Use `gofmt`, `gofumpt`, and `goimports` for formatting
- Variable naming: avoid meaningless names except `i`, `k`, `r`, `v`, `w`
- Avoid dot imports except in test files
- Keep cyclomatic complexity under 20

### Terraform Provider Patterns

**Resource Structure**:
- Each resource has: `*_resource.go`, `*_resource_schema.go`, `*_resource_test.go`
- Use Terraform Plugin Framework types (not SDK v2)
- Implement CRUD operations: Create, Read, Update, Delete
- Handle state management properly

**Testing Requirements**:
- Unit tests for logic and transformations
- Acceptance tests for resources (with `TF_ACC=1`)
- Use `testdata/` subdirectory for test Terraform configs
- Test both success and error cases

**API Client Usage**:
- The Braze API client is auto-generated from OpenAPI specs
- Located in `internal/braze-client-go/`
- To update: modify `ogen.yml` and regenerate
- Use retryable HTTP client for resilience

## Common Tasks

### Adding a New Resource

1. Create resource implementation: `internal/provider/braze_<resource>_resource.go`
2. Create schema definition: `internal/provider/braze_<resource>_resource_schema.go`
3. Create tests: `internal/provider/braze_<resource>_resource_test.go`
4. Register in provider: add to `Resources()` method in `braze_provider.go`
5. Add example: `examples/resources/braze_<resource>/resource.tf`
6. Run `go generate ./...` to update documentation
7. Run tests and linting

### Adding a New Data Source

1. Create data source implementation: `internal/provider/braze_<datasource>_data_source.go`
2. Create schema definition: `internal/provider/braze_<datasource>_data_source_schema.go`
3. Create tests: `internal/provider/braze_<datasource>_data_source_test.go`
4. Register in provider: add to `DataSources()` method in `braze_provider.go`
5. Add example: `examples/data-sources/braze_<datasource>/data-source.tf`
6. Run `go generate ./...` to update documentation
7. Run tests and linting

### Modifying the API Client

1. Update OpenAPI spec in `internal/braze-client-go/openapi/`
2. Modify `internal/braze-client-go/ogen.yml` if needed
3. Run `go generate ./internal/braze-client-go/`
4. Update provider code to use new client features
5. Test changes thoroughly

### Debugging

- Use `-debug` flag when running provider locally
- Check Terraform logs with `TF_LOG=DEBUG`
- Use `terraform-plugin-log` for structured logging
- Acceptance tests show full Terraform output

## Dependencies

### Adding New Dependencies

1. Use `go get <package>` to add dependency
2. Run `go mod tidy` to clean up
3. Dependencies are managed in `go.mod`
4. CI validates indirect dependencies are up-to-date

### Key Dependencies

- `github.com/hashicorp/terraform-plugin-framework` - Provider framework
- `github.com/hashicorp/terraform-plugin-testing` - Acceptance testing
- `github.com/ogen-go/ogen` - OpenAPI client generator
- `github.com/hashicorp/go-retryablehttp` - Resilient HTTP client

## CI/CD

### GitHub Actions Workflows

- **test.yml**: Builds, runs unit tests, and acceptance tests
- **golangci.yml**: Runs linting checks
- **gogenerate.yml**: Ensures generated code is up-to-date
- **release.yml**: Publishes releases using goreleaser
- **update-indirect-dependencies.yml**: Keeps dependencies current

### Pre-commit Checklist

Before committing changes:

1. ✅ Run `go generate ./...` if examples or schema changed
2. ✅ Run `go test -v ./...` to ensure tests pass
3. ✅ Run `golangci-lint run` to check code quality
4. ✅ Run `go mod tidy` if dependencies changed
5. ✅ Verify no uncommitted generated files
6. ✅ Add acceptance tests for new resources/data sources

## Testing Philosophy

- **Unit Tests**: Fast, isolated, mock external dependencies
- **Acceptance Tests**: Slow, integration, use real API (with test credentials)
- **Test Data**: Keep test Terraform configs in `testdata/` subdirectories
- **Coverage**: Aim for high coverage on business logic
- **CI**: All tests must pass before merge

## Common Patterns

### Error Handling

```go
if err != nil {
    resp.Diagnostics.AddError(
        "Error Creating Resource",
        fmt.Sprintf("Could not create resource: %s", err),
    )
    return
}
```

### State Management

```go
// Read from plan
var data ResourceModel
resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

// Write to state
resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
```

### API Client Usage

```go
client := data.Client.(*brazeclient.Client)
response, err := client.CreateContentBlock(ctx, request)
```

## Documentation

- Auto-generated from Go code and examples
- Templates in `templates/`
- Examples in `examples/`
- Published to Terraform Registry
- Do not manually edit `docs/` directory

## Security

- API keys handled securely through provider configuration
- Never commit credentials or API keys
- Use environment variables for sensitive data in tests
- Secrets managed through GitHub Actions secrets

## Release Process

1. Tag version: `git tag v1.x.x`
2. Push tag: `git push origin v1.x.x`
3. GitHub Actions runs goreleaser
4. Binary published to GitHub Releases
5. Provider published to Terraform Registry

## Getting Help

- Check existing resources for patterns
- Review Terraform Plugin Framework docs
- Look at acceptance tests for usage examples
- Generated API client has detailed type information
