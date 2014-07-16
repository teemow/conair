package main

import (
	"fmt"
	"io/ioutil"
)

var cmdImages = &Command{
	Name:        "images",
	Description: "List all available conair images",
	Summary:     "List all available conair images",
	Run:         runImages,
}

func runImages(args []string) (exit int) {

	files, _ := ioutil.ReadDir(getImagesPath())
	if len(files) < 1 {
		fmt.Println("No images found.")
		return
	}

	fmt.Println("Available images:")
	for _, f := range files {
		fmt.Println(f.Name())
	}
	return
}
