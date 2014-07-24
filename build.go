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

func readFile(filename string) (*parser.Conairfile, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}
	return parser.Parse(filename)
}

func runBuild(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	f, err := readFile("./Conairfile")
	if err != nil {
		f, err = readFile("./Dockerfile")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't read Conairfile or Dockerfile.", err)
			return 1
		}
	}

	newImage := args[0]
	newImagePath := fmt.Sprintf("images/%s", newImage)

	image := f.From
	imagePath := fmt.Sprintf("images/%s", image)

	container := fmt.Sprintf("tmp-%s", newImage)
	containerPath := fmt.Sprintf("container/%s", container)

	fs, _ := btrfs.Init(home)
	if err = fs.Snapshot(imagePath, containerPath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for build container.", err)
		return 1
	}

	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))

	if err := c.Build("RUN", "pacman -Sy --noconfirm"); err != nil {
		fmt.Fprintln(os.Stderr, "Pacman update failed.", err)

		if err = fs.Remove(containerPath); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
		}
		return 1
	}
	for _, cmd := range f.Commands {
		if err := c.Build(cmd.Verb, cmd.Payload); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Buildstep failed: %s %s.", cmd.Verb, cmd.Payload))
			if err = fs.Remove(containerPath); err != nil {
				fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
			}
			return 1
		}
	}
	// remove machine id at the end
	if err := c.ReplaceMachineId(); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove machine-id for new image.", err)
		return 1
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
