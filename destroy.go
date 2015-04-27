package main

import (
	"fmt"
	"os"

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
	err := networkd.RemoveHostNetwork()
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
