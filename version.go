package main

import (
	"fmt"

	"github.com/giantswarm/conair/version"
)

var cmdVersion = &Command{
	Name:        "version",
	Description: "Print the version and exit",
	Summary:     "Print the version and exit",
	Run:         runVersion,
}

func runVersion(args []string) (exit int) {
	fmt.Println("conair version", version.Version)
	return
}
