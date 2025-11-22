# Terraform Provider for Braze

A Terraform provider for managing Braze configuration.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.12
- [Go](https://golang.org/doc/install) >= 1.25

## Building the Provider

```shell
go build -v .
```

## Using the Provider

```terraform
terraform {
  required_providers {
    braze = {
      source = "cysp/braze"
    }
  }
}

provider "braze" {
  api_key = "your-api-key-here"
}
```

## Developing the Provider

### Testing

```shell
go test -v ./...
```

### Acceptance Tests

```shell
TF_ACC=1 go test -v ./internal/provider/
```

### Generating Documentation

```shell
go generate ./...
```

## License

Mozilla Public License Version 2.0
