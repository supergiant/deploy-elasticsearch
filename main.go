package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/supergiant/deploy-elasticsearch/pkg"
	supergiant "github.com/supergiant/supergiant/pkg/client"
)

func main() {
	var (
		sgURL       string
		sgToken     string
		sgCertFile  string
		componentID int64
	)

	flag.StringVar(&sgURL, "sg-url", "", "URL of the relevant Supergiant server")
	flag.StringVar(&sgToken, "sg-token", "", "The API token of the requesting Supergiant User")
	flag.StringVar(&sgCertFile, "sg-cert", "", "SSL certificate file of the Supergiant server")
	flag.Int64Var(&componentID, "component-id", 0, "ID of the Supergiant Component")
	flag.Parse()

	if sgURL == "" {
		panic("--sg-url required")
	}
	if sgToken == "" {
		panic("--sg-token required")
	}
	if sgCertFile == "" {
		panic("--sg-cert required")
	}
	if componentID == 0 {
		panic("--component-id required")
	}

	sg := supergiant.New(sgURL, "token", sgToken, sgCertFile)

	if err := pkg.Deploy(sg, &componentID); err != nil {
		_, fn, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "[error] %s:%d %v\n", fn, line, err)
		os.Exit(1)
	}
}
