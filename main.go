package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/supergiant/deploy-elasticsearch/pkg"
)

func main() {
	var (
		appName  string
		compName string
	)
	flag.StringVar(&appName, "app-name", "", "Name of the Supergiant App")
	flag.StringVar(&compName, "component-name", "", "Name of the Supergiant Component")
	flag.Parse()

	if err := pkg.Deploy(&appName, &compName); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		debug.PrintStack()
		os.Exit(1)
	}
}
