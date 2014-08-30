package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/giantswarm/conair/btrfs"
)

var cmdSnapshot = &Command{
	Name:        "snapshot",
	Description: "Manage snapshots",
	Summary:     "Manage snapshots",
	Run:         runSnapshot,
}

func runSnapshot(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Snapshot command missing.")
		return 1
	}

	switch args[0] {
	case "create":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Snapshot name missing.")
			return 1
		}

		snapshot := args[1]
		snapshotPath := fmt.Sprintf("snapshots/%s", snapshot)

		fs, _ := btrfs.Init(home)

		if fs.Exists(snapshotPath) {
			fmt.Fprintln(os.Stderr, "Snapshot already exists.")
			return 1
		}

		if err := fs.Subvolume(snapshotPath); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't create snapshot.", err)
			return 1
		}
	case "rm":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Snapshot name missing.")
			return 1
		}

		snapshot := args[1]
		snapshotPath := fmt.Sprintf("snapshots/%s", snapshot)

		fs, _ := btrfs.Init(home)

		if !fs.Exists(snapshotPath) {
			fmt.Fprintln(os.Stderr, "Snapshot doesn't exist.")
			return 1
		}

		if err := fs.Remove(snapshotPath); err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't remove snapshot.", err)
			return 1
		}
	case "ls":
		snapshots, _ := ioutil.ReadDir(fmt.Sprintf("%s/snapshots", home))
		if len(snapshots) < 1 {
			fmt.Println("No snapshots found.")
			return
		}

		for _, s := range snapshots {
			fmt.Println(s.Name())
		}

	default:
		fmt.Fprintln(os.Stderr, "Snapshot command unknown.")
		return 1
	}

	return 0
}
