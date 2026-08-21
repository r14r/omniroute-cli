package main

import (
	"fmt"
	"os"

	"github.com/r14r/omniroute-cli/src/semver"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: semver-check <version>")
		os.Exit(2)
	}
	v, err := semver.Parse(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(v.String())
}
