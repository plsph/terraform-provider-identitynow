package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set via -ldflags during build
var version string = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/plsph/identitynow",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
