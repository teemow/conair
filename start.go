package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/nspawn"
)

var cmdStart = &Command{
	Name:        "start",
	Description: "Start a container",
	Summary:     "Start a container",
	Run:         runStart,
}

func runStart(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))
	err := c.Start()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't start container.", err)
		return 1
	}

	return 0
}
