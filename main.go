package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cysp/terraform-provider-braze/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Root generation requires Terraform on PATH.
//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs --provider-name=terraform-provider-braze

// set by goreleaser.
var version = "dev"

//nolint:gochecknoglobals // Set by GoReleaser.
var commit = "unknown"

func main() {
	var (
		debug       bool
		showVersion bool
	)

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.BoolVar(&showVersion, "version", false, "print the provider version and exit")
	flag.Parse()

	if showVersion {
		_, err := fmt.Fprintf(os.Stdout, "terraform-provider-braze %s (commit %s)\n", version, commit)
		if err != nil {
			log.Fatal(err.Error())
		}

		return
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/cysp/braze",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.Factory(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
