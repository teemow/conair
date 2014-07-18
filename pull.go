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
	var newImage string
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	image := args[0]

	if len(args) > 1 {
		newImage = args[1]
	} else {
		newImage = image
	}
	newImagePath := fmt.Sprintf("images/%s", newImage)

	fs, err := btrfs.Init(home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't populate filesystem for conair.", err)
		return 1
	}

	err = fs.Subvolume(newImagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't create subvolume for image %s.", newImage), err)
		return 1
	}

	err = nspawn.FetchImage(image, newImage, hub, getImagesPath())
	if err != nil {
		_ = fs.Remove(newImagePath)
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't create image %s.", newImage), err)
		return 1
	}

	return 0
}
