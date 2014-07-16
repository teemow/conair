package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/btrfs"
)

var cmdRmi = &Command{
	Name:        "rmi",
	Description: "Remove an image",
	Summary:     "Remove an image",
	Run:         runRmi,
}

func runRmi(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	image := args[0]
	imagePath := fmt.Sprintf("images/%s", image)

	fs, _ := btrfs.Init(home)
	if err := fs.Remove(imagePath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove filesystem for image.", err)
		return 1
	}

	return 0
}
