package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var cmdPs = &Command{
	Name:        "ps",
	Description: "List all conair containers",
	Summary:     "List all conair containers",
	Run:         runPs,
}

func runPs(args []string) (exit int) {

	path, err := exec.LookPath("machinectl")
	if err != nil {
		fmt.Fprintln(os.Stderr, "machinectl not found.")
	}

	args = append([]string{"list"}, args...)

	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "machinectl failed: machinctl %v: %s (%s)", strings.Join(args, " "), output, err)
	}
	fmt.Fprintln(os.Stdout, string(output[:]))

	return
}
