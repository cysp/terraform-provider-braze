# Development

## Prerequisites

Tool requirements come from [go.mod](go.mod), the
[lint workflow](.github/workflows/golangci.yml), and the Terraform matrix in the
[test workflow](.github/workflows/test.yml). Install Terraform 1.14 or later on
`PATH` for documentation generation and the full acceptance suite.

## Repository map

- `internal/provider/braze_provider.go` configures the provider and registers
  resources and list resources. There are no data sources.
- `internal/provider/*_resource_schema.go` defines resource and identity schemas;
  `*_resource.go` implements Terraform lifecycle operations.
- `internal/provider/*_adapter.go` translates between provider models and the
  generated Braze client. `*_model*.go` handles Terraform values and API payloads.
- `internal/provider/*_test.go` tests those boundaries and Terraform workflows.
  `braze_provider_testing_test.go` wires the mocked acceptance harness.
- `internal/braze-client-go/openapi/` and `ogen.yml` define the generated client.
  `internal/braze-client-go/testing/` implements the local Braze server used by
  tests; it is an API fixture, not independent evidence of live Braze behavior.
- `docs/design/` records SDK Authentication resource contracts and their API
  evidence. Read the relevant singular or collection design before changing key
  ownership, primary selection, operation ordering, or failure recovery.

## Tests

With `TF_ACC` unset, run unit, provider protocol, and local HTTP integration tests:

```sh
go test ./...
```

Acceptance tests invoke Terraform against local HTTP fixtures and require no
Braze credentials. Run a focused scenario while iterating, then broaden when
shared planning, state, or lifecycle behavior changes:

```sh
TF_ACC=1 TF_ACC_MOCKED=1 go test ./internal/provider -run '^TestAccBrazeCatalogAndCatalogItem$' -count=1
TF_ACC=1 TF_ACC_MOCKED=1 go test ./internal/provider -run '^TestAcc' -count=1 -timeout 15m
```

`TF_ACC` enables acceptance tests. `TF_ACC_MOCKED` matches CI's invocation; this
repository's tests do not offer a live Braze mode when it is unset. Mocked test
results establish provider behavior against the fixtures. Check primary Braze
documentation or a separately authorized live experiment for service behavior.

Set `TF_ACC_TERRAFORM_PATH` to an existing Terraform executable for a reproducible
run. The acceptance framework can otherwise find or install Terraform. The test
that exercises query-generated imports invokes the CLI directly and requires
`TF_ACC_TERRAFORM_PATH` or `terraform` on `PATH`. Query tests skip on Terraform
1.12 and 1.13; inspect skips before reporting that query behavior was tested.
The CI matrix also checks older resource compatibility and is not a requirement
to replay every version locally.

### Choosing the test boundary

Use the smallest boundary that proves the contract:

| Behavior | Test boundary |
| --- | --- |
| Model conversion, validators, and SDK key operation ordering | Unit tests with independently stated expected values or invariants |
| HTTP methods, payloads, error decoding, and pagination | Adapter tests with explicit request and response fixtures |
| Plan, apply, refresh, identity import, and retained state | Focused Terraform acceptance steps |
| Query-generated configuration importing without drift | Direct CLI integration test |
| Local server behavior used by multiple tests | Mock server tests in `internal/braze-client-go/testing/` |

Keep independent cases in tables and sequential state transitions in explicit
acceptance steps. Avoid repeating the same invalid input matrix at every layer;
use acceptance tests where Terraform itself changes the outcome. Preserve API
assertions when Terraform state alone cannot prove whether a remote object was
retained or deleted. Use typed plan/state checks when null, unknown, or collection
semantics matter. Add coverage for a changed contract, rather than assertions
that merely repeat implementation details.

## Documentation and code generation

Practitioner documentation is generated with `terraform-plugin-docs`. Edit the
source for the information being changed:

| Documentation concern | Authoritative input |
| --- | --- |
| Resource and attribute contracts | Schema descriptions under `internal/provider/` |
| Configuration, query, and import syntax | `examples/` |
| Provider setup and permissions | `templates/index.md.tmpl` |
| Resource workflows and shared discovery instructions | `templates/` |
| Algorithms, provider invariants, and API evidence | Handwritten `docs/design/` |

Keep short contracts in schemas and operational workflows in templates. Resource
examples are reference snippets; the caller supplies provider configuration and
any referenced inputs or public-key files. Configuration and import examples
must address the same resource. SDK rotation directories are alternative
snapshots, not modules to combine. Do not run alternative import examples
together.

Run all generators from the repository root:

```sh
go generate ./...
```

The root generator formats `examples/` with Terraform and regenerates Registry
pages. The client generator uses `ogen` to regenerate `oas_*_gen.go`. For focused
iteration, use `go generate .` or `go generate ./internal/braze-client-go`.
For direct documentation generation, pass
`--provider-name=terraform-provider-braze` explicitly to `tfplugindocs generate`
so the provider name does not depend on the checkout directory.

The generated documentation is `docs/index.md`, `docs/resources/`, and
`docs/list-resources/`. Keep handwritten `docs/design/` when regenerating;
never clear the entire `docs/` tree.

Validate Registry structure and review the complete generation diff:

```sh
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name=terraform-provider-braze
git diff --check
git diff
git status --short --untracked-files=all
```

The validator checks document structure, not prose accuracy or freshness. Review
rendered links, behavior, API terminology, and example addresses. Validate resource
snippets in an isolated configuration with the built provider; validate discovery
examples with `terraform validate -query`. See HashiCorp's
[query workflow](https://developer.hashicorp.com/terraform/language/import/bulk).

To check generation reproducibility, rerun `go generate ./...` after committing
both inputs and outputs, then require `git status --short --untracked-files=all`
to be empty. `git diff --exit-code` alone misses newly generated files.

## Build and lint

```sh
go build .
golangci-lint run
golangci-lint fmt --diff
```

Use the [official golangci-lint binary](https://golangci-lint.run/docs/welcome/install/local/#binaries)
at the version selected by the lint workflow. `go build .` creates
`terraform-provider-braze` in the repository root. For manual Terraform use, the
provider address is `cysp/braze`; configure
[development overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

For a prose-only change, review links and run `git diff --check`. For Go changes,
run affected tests, build, lint, and formatting checks. Schema, examples, templates,
and OpenAPI changes also require regeneration. Inspect remote checks for the
actual revision before reporting CI success; record skipped or blocked checks
separately from passing checks.

Release automation is defined by the
[release workflow](.github/workflows/release.yml) and [.goreleaser.yml](.goreleaser.yml).
