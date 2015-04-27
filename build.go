package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/layer"
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

	newImage := args[0]
	newImagePath := fmt.Sprintf("machines/%s", newImage)

	fs, _ := btrfs.Init(home)

	// remove existing layer
	if fs.Exists(newImagePath) {
		if err := fs.Remove(newImagePath); err != nil {
			fmt.Printf("%v", err)
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't remove existing image. %v", err))
			return 1
		}
	}

	// read build file
	f, err := readFile("./Conairfile")
	if err != nil {
		f, err = readFile("./Dockerfile")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't read Conairfile or Dockerfile.", err)
			return 1
		}
	}

	image := f.From
	parentPath := fmt.Sprintf("machines/%s", image)

	for i, snap := range f.Snapshots {
		paths := strings.Split(snap, ":")
		if len(paths) < 2 {
			fmt.Fprintln(os.Stderr, "SNAPSHOT definition is unreadable. Please use <snapshot name>:<container path> notation.")
			return 1
		}

		// check if snapshot exists - otherwise create a new subvolume
		snapshotPath := fmt.Sprintf("conair/snapshots/%s", paths[0])
		if !fs.Exists(snapshotPath) {
			if err := fs.Subvolume(snapshotPath); err != nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't create snapshot '%s'.", snapshotPath))
				return 1
			}
		}

		f.Snapshots[i] = fmt.Sprintf("%s/conair/snapshots/%s", home, snap)
	}

	for _, cmd := range f.Commands {
		l, err := layer.Create(fs, cmd.Verb, cmd.Payload, parentPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't create layer: %v.", err))
			return 1
		}
		fmt.Println(l.Hash, cmd.Verb, cmd.Payload)

		if l.Exists == true {
			parentPath = l.Path
			continue
		}

		c := nspawn.Init(l.Hash, fmt.Sprintf("%s/%s", home, l.Path))
		c.SetBinds(append(f.Binds, f.Snapshots...))

		if err := c.Build(cmd.Verb, cmd.Payload); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Buildstep failed: %v.", err))
			if err = l.Remove(); err != nil {
				fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
			}
			return 1
		}

		parentPath = l.Path
	}
	if err = fs.Snapshot(parentPath, newImagePath, true); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for new image.", err)
		return 1
	}

	return 0
}
