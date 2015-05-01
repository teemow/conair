package main

import (
	"fmt"

	"os"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/nspawn"
)

var cmdPull = &Command{
	Name:        "pull",
	Description: "Pull an image (eg base)",
	Summary:     "Pull an image (eg base)",
	Run:         runPull,
}

func runPull(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	image := args[0]

	var newImage string
	if len(args) > 1 {
		newImage = args[1]
	} else {
		newImage = image
	}

	fs, err := btrfs.Init(home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't populate filesystem for conair.", err)
		return 1
	}

	err = fs.Subvolume(newImage)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't create subvolume for image %s.", newImage), err)
		return 1
	}

	err = nspawn.FetchImage(image, newImage, hub, home)
	if err != nil {
		_ = fs.Remove(newImage)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't create image %s.", newImage), err)
		return 1
	}

	return 0
}
