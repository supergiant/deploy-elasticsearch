package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

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
		_, fn, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "[error] %s:%d %v\n", fn, line, err)

		os.Exit(1)
	}
}
