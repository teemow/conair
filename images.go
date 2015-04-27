package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var cmdImages = &Command{
	Name:        "images",
	Description: "List all available conair images",
	Summary:     "List all available conair images",
	Run:         runImages,
}

func runImages(args []string) (exit int) {

	path, err := exec.LookPath("machinectl")
	if err != nil {
		fmt.Fprintln(os.Stderr, "machinectl not found.")
	}

	args = append([]string{"list-images"}, args...)

	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "machinectl failed: machinctl %v: %s (%s)", strings.Join(args, " "), output, err)
	}
	fmt.Fprintln(os.Stdout, string(output[:]))

	return
}
