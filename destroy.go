package main

import (
	"fmt"
	"os"

	"github.com/giantswarm/conair/iptables"
	"github.com/giantswarm/conair/networkd"
	"github.com/giantswarm/conair/nspawn"
)

var cmdDestroy = &Command{
	Name:        "destroy",
	Description: "Destroy conair environment",
	Summary:     "Remove bridge, iptables and unit file",
	Run:         runDestroy,
}

func runDestroy(args []string) (exit int) {
	err := iptables.DeleteBridgeForwarding(bridge, destination)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove iptables rules.", err)
		return 1
	}

	err = networkd.DeleteBridge(bridge)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove bridge.", err)
		return 1
	}

	err = nspawn.RemoveUnit()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't remove unit file to start containers.", err)
		return 1
	}
	return 0
}
