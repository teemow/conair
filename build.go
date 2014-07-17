package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/nspawn"
	"github.com/giantswarm/conair/parser"
)

var cmdBuild = &Command{
	Name:        "build",
	Description: "Build an image",
	Summary:     "Build an image",
	Run:         runBuild,
}

func runBuild(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	newImage := args[0]
	newImagePath := fmt.Sprintf("images/%s", newImage)

	d, err := parser.Dockerfile("./Dockerfile")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't read Dockerfile.", err)
		return 1
	}

	image := d.From
	imagePath := fmt.Sprintf("images/%s", image)

	container := fmt.Sprintf("tmp-%s", newImage)
	containerPath := fmt.Sprintf("container/%s", container)

	fs, _ := btrfs.Init(home)
	if err = fs.Snapshot(imagePath, containerPath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for build container.", err)
		return 1
	}

	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))

	for _, cmd := range d.Commands {
		if err := c.Build(cmd.Verb, cmd.Payload); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Buildstep failed: %s %s.", cmd.Verb, cmd.Payload))
			if err = fs.Remove(containerPath); err != nil {
				fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
			}
			return 1
		}
	}

	if err = fs.Snapshot(containerPath, newImagePath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for new image.", err)
		if err = fs.Remove(containerPath); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
		}
		return 1
	}

	if err = fs.Remove(containerPath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
		return 1
	}

	return 0
}
