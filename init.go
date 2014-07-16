package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/iptables"
	"github.com/giantswarm/conair/networkd"
	"github.com/giantswarm/conair/nspawn"
)

var cmdInit = &Command{
	Name:        "init",
	Description: "Initialize conair environment",
	Summary:     "Setup a bridge for the containers and add some iptables forwarding",
	Run:         runInit,
}

func runInit(args []string) (exit int) {
	err := networkd.CreateBridge(bridge, destination)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create bridge.", err)
		return 1
	}

	err = iptables.AddBridgeForwarding(bridge, destination)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't add forwarding to bridge.", err)
		return 1
	}

	err = nspawn.CreateUnit(bridge, getContainerPath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create unit file to start containers.", err)
		return 1
	}

	// create arch base with pacstrap
	// install packages inside containers
	// no ssh
	return 0
}

func remove(bridge, destination string) {
	err := iptables.DeleteBridgeForwarding(bridge, destination)
	if err != nil {
		fmt.Println(err)
	}

	err = networkd.DeleteBridge(bridge)
	if err != nil {
		fmt.Println(err)
	}
}

func create(bridge, destination string) {
}
