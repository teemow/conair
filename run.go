package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/nspawn"
)

var (
	flagBind     string
	flagSnapshot string
	cmdRun       = &Command{
		Name:    "run",
		Summary: "Run a container",
		Usage:   "[-bind=S] <image> [<container>]",
		Run:     runRun,
		Description: `Run a new container

Example:
conair run base test

You can either bind mount a directory into the container or take a snapshot of a volume that will be deleted with the container.

conair run -bind=/var/data:/data base test
conair run -snapshot=mysnapshot:/data base test

`,
	}
)

func init() {
	cmdRun.Flags.StringVar(&flagBind, "bind", "", "Bind mount a directory into the container")
	cmdRun.Flags.StringVar(&flagSnapshot, "snapshot", "", "Add a snapshot into the container")
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
	if err := fs.Snapshot(imagePath, containerPath, false); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for container.", err)
		return 1
	}

	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))
	if flagBind != "" {
		c.SetBinds([]string{flagBind})
	}
	if flagSnapshot != "" {
		c.SetSnapshots([]string{flagSnapshot})
	}

	for _, snap := range c.Snapshots {
		paths := strings.Split(snap, ":")

		if len(paths) < 2 {
			fmt.Fprintln(os.Stderr, "Couldn't create snapshot for container.")
			return 1
		}

		from := fmt.Sprintf("snapshots/%s", paths[0])
		to := fmt.Sprintf("%s/%s", containerPath, paths[1])

		if fs.Exists(to) {
			if err := os.Remove(fmt.Sprintf("%s/%s", home, to)); err != nil {
				fmt.Fprintln(os.Stderr, "Couldn't remove existing directory for snapshot.")
				return 1
			}
		}

		if err := fs.Snapshot(from, to, false); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't create snapshot for container.", err)
			return 1
		}
	}

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
