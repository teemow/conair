package main

import (
	"fmt"
	"io/ioutil"
)

var cmdPs = &Command{
	Name:        "ps",
	Description: "List all conair containers",
	Summary:     "List all conair containers",
	Run:         runPs,
}

func runPs(args []string) (exit int) {

	files, _ := ioutil.ReadDir(getContainerPath())
	if len(files) < 1 {
		fmt.Println("No containers found.")
		return
	}

	fmt.Println("Running containers:")
	for _, f := range files {
		fmt.Println(f.Name())
	}

	fmt.Println("\nYou should also take a look at machinectl to manage your containers!")
	return
}
