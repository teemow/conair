package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/nspawn"
)

var cmdInspect = &Command{
	Name:        "inspect",
	Description: "Inspect a containers network",
	Summary:     "Inspect a containers network",
	Run:         runInspect,
}

func runInspect(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))
	data, err := c.Inspect()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't inspect container %s.", container), err)
		return 1
	}
	fmt.Println(data)
	return 0
}
