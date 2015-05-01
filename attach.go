package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/nspawn"
)

var cmdAttach = &Command{
	Name:        "attach",
	Description: "Attach to container",
	Summary:     "Attach to container",
	Run:         runAttach,
}

func runAttach(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	c := nspawn.Init(container, fmt.Sprintf("%s/.#%s", home, container))
	err := c.Attach()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't attach to container %s.", container), err)
		return 1
	}
	return 0
}
