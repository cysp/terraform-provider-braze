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

See the [latest released provider reference](https://registry.terraform.io/providers/cysp/braze/latest/docs)
for configuration and API permissions. The [reference in this checkout](docs/index.md)
also covers changes that may not have been released yet.
[Resource examples](examples/resources) include catalog values, Braze Liquid, and
import syntax. [Query examples](examples/list-resources) show how to discover
existing objects.

## Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md) for the repository map, test boundaries,
generation, and validation commands. Provider design contracts are in
[docs/design/](docs/design/).

## License

Licensed under the Mozilla Public License 2.0. See [LICENSE](LICENSE).
