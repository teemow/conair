package main

import (
	"fmt"
)

var cmdVersion = &Command{
	Name:        "version",
	Description: "Print the version and exit",
	Summary:     "Print the version and exit",
	Run:         runVersion,
}

func runVersion(args []string) (exit int) {
	fmt.Println("conair version", projectVersion)
	return
}
