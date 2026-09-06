# Terraform Provider for Braze

Manage Braze catalogs, catalog items, content blocks, email templates, and SDK authentication keys.
Requires Terraform 1.12 or later; discovery with `terraform query` requires 1.14
or later.

## Setup

```terraform
terraform {
  required_version = ">= 1.12"
  required_providers {
    braze = { source = "cysp/braze" }
  }
}

provider "braze" {
  base_url = "https://rest.iad-01.braze.com"
}
```

Choose the REST endpoint for your [Braze instance](https://www.braze.com/docs/api/basics/)
and set `BRAZE_API_KEY` in your environment.

See the [provider reference](docs/index.md) for configuration and API permissions.
[Resource examples](examples/resources) include catalog values, Braze Liquid, and
import syntax. [Query examples](examples/list-resources) show how to discover
existing objects.

## Developing

Go requirements and tool dependencies are declared in [go.mod](go.mod). Use
Terraform 1.14 or later for the full acceptance suite and the golangci-lint version
pinned in the [lint workflow](.github/workflows/golangci.yml).

```shell
go build ./...
go test -timeout 10m ./...
TF_ACC=1 TF_ACC_MOCKED=1 go test -timeout 15m ./internal/provider/
golangci-lint fmt --diff
golangci-lint run --timeout 5m
```

Acceptance tests use local HTTP fixtures and require no Braze credentials.
Set `TF_ACC_TERRAFORM_PATH` to test with a specific Terraform executable.

Edit schema descriptions, `templates/`, and `examples/` to change the Registry
documentation. Edit `internal/braze-client-go/openapi/` to change the generated
client. Regenerate and review both modified and new files:

```shell
go generate ./...
git diff
git status --short
```

## License

Mozilla Public License Version 2.0
