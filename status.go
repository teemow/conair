package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/nspawn"
)

var cmdStatus = &Command{
	Name:        "status",
	Description: "Status of container",
	Summary:     "Status of container",
	Run:         runStatus,
}

func runStatus(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	c := nspawn.Init(container)
	status, err := c.Status()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf(status)

	return 0
}
