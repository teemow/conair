package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/nspawn"
)

var cmdStop = &Command{
	Name:        "stop",
	Description: "Stop a container",
	Summary:     "Stop a container",
	Run:         runStop,
}

func runStop(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	c := nspawn.Init(container, fmt.Sprintf("%s/.#%s", home, container))
	err := c.Stop()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't stop container.", err)
		return 1
	}

	return 0
}
