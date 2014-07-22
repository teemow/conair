package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/nspawn"
)

var cmdIp = &Command{
	Name:        "ip",
	Description: "Get the ip of a container",
	Summary:     "Get the ip of a container",
	Run:         runIp,
}

func runIp(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Container name missing.")
		return 1
	}

	container := args[0]
	c := nspawn.Init(container, fmt.Sprintf("%s/%s", getContainerPath(), container))
	data, err := c.Ip()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Couldn't the ip of container %s.", container), err)
		return 1
	}
	fmt.Println(data)
	return 0
}
