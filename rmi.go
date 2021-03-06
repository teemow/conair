package main

import (
	"fmt"
	"os"
	"strings"

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

	imagePath := args[0]

	fs, _ := btrfs.Init(home)

	if !fs.Exists(imagePath) {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Image %s does not exists.", imagePath))
		return 1
	}

	for {
		var (
			uuid  string
			layer string
			err   error
		)
		uuid, err = fs.GetSubvolumeParentUuid(imagePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't get uuid.", err)
			return 1
		}

		layer, err = fs.GetLayerByUuid(uuid)
		noParent := false
		if err != nil {
			noParent = true
		}

		if err := fs.Remove(imagePath); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't remove filesystem.", err)
			return 1
		}

		fmt.Println(imagePath)
		if !strings.HasPrefix(layer, ".cnr-") || noParent {
			break
		} else {
			imagePath = layer
		}
	}
	return 0
}
