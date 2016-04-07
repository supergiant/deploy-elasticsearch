package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/supergiant/deploy-elasticsearch/pkg"
)

func main() {
	var (
		appName  string
		compName string
	)
	flag.StringVar(&appName, "App name", "", "Name of the Supergiant App")
	flag.StringVar(&compName, "Component name", "", "Name of the Supergiant Component")

	if err := pkg.Deploy(&appName, &compName); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
