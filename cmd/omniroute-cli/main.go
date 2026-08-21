package main

import (
	"fmt"
	"os"

	"github.com/r14r/omniroute-cli/src/cli"
	"github.com/r14r/omniroute-cli/src/semver"
)

var version = "dev"

func main() {
	v, err := semver.ParseBuild(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid build version %q: %v\n", version, err)
		os.Exit(2)
	}
	os.Exit(cli.Run(os.Args[1:], v, os.Stdout, os.Stderr))
}
