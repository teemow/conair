package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/nspawn"
)

var cmdRun = &Command{
	Name:        "run",
	Description: "Run a container",
	Summary:     "Run a container",
	Run:         runRun,
}

func runRun(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	image := args[0]
	imagePath := fmt.Sprintf("images/%s", image)

	var container string
	if len(args) < 2 {
		// add some hashing here
		container = image
	} else {
		container = args[1]
	}
	containerPath := fmt.Sprintf("container/%s", container)

	fs, _ := btrfs.Init(home)
	if err := fs.Snapshot(imagePath, containerPath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for container.", err)
		return 1
	}

	c := nspawn.Init(container)

	if err := c.Enable(); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't enable container.", err)
		return 1
	}

	if err := c.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't start container.", err)
		return 1
	}

	return 0
}
