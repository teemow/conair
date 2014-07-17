package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/nspawn"
)

var cmdRm = &Command{
	Name:        "rm",
	Description: "Remove a container",
	Summary:     "Remove a container",
	Run:         runRm,
}

func runRm(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	containerPath := fmt.Sprintf("container/%s", container)

	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))
	if err := c.Stop(); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't stop container.", err)
	}

	if err := c.Disable(); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't disable container.", err)
	}

	fs, _ := btrfs.Init(home)
	if err := fs.Remove(containerPath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove filesystem for container.", err)
		return 1
	}

	return 0
}
