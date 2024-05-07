package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/taliesins/terraform-provider-cleaneks/internal/provider"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address:         "registry.terraform.io/taliesins/terraform-provider-cleaneks",
		Debug:           debug,
		ProtocolVersion: 6,
	})

	if err != nil {
		fmt.Printf("failed to initialize provider: %v\n", err)
		os.Exit(1)
	}
}
