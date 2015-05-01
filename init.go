package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/networkd"
	"github.com/giantswarm/conair/nspawn"
)

var cmdInit = &Command{
	Name:        "init",
	Description: "Initialize conair environment",
	Summary:     "Setup a network and systemd units for the containers",
	Run:         runInit,
}

func runInit(args []string) (exit int) {
	fmt.Println("Create network")
	err := networkd.DefineHostNetwork(bridge, destination)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create bridge.", err)
		return 1
	}

	fmt.Println("Create systemd unit for conair containers.")
	err = nspawn.CreateUnit(bridge, home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create unit file to start containers.", err)
		return 1
	}

	_, err = btrfs.Init(home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't populate filesystem structure for conair.", err)
		return 1
	}

	return 0
}
