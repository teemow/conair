package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/btrfs"
)

var cmdCommit = &Command{
	Name:        "commit",
	Description: "Commit a container",
	Summary:     "Commit a container",
	Run:         runCommit,
}

func runCommit(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	containerPath := fmt.Sprintf("container/%s", container)

	var image string
	if len(args) < 2 {
		image = container
	} else {
		image = args[1]
	}
	imagePath := fmt.Sprintf("machines/%s", image)

	fs, _ := btrfs.Init(home)
	if err := fs.Snapshot(containerPath, imagePath, true); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create snapshot of container.", err)
		return 1
	}

	return 0
}
