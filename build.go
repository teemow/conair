package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"crypto/sha1"

	"code.google.com/p/go-uuid/uuid"
	"github.com/giantswarm/conair/btrfs"
	"github.com/giantswarm/conair/nspawn"
	"github.com/giantswarm/conair/parser"
)

var cmdBuild = &Command{
	Name:        "build",
	Description: "Build an image",
	Summary:     "Build an image",
	Run:         runBuild,
}

func readFile(filename string) (*parser.Conairfile, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, err
	}
	return parser.Parse(filename)
}

func runBuild(args []string) (exit int) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Image name missing.")
		return 1
	}

	f, err := readFile("./Conairfile")
	if err != nil {
		f, err = readFile("./Dockerfile")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Couldn't read Conairfile or Dockerfile.", err)
			return 1
		}
	}

	newImage := args[0]
	newImagePath := fmt.Sprintf("images/%s", newImage)

	image := f.From
	fromPath := fmt.Sprintf("images/%s", image)

	fs, _ := btrfs.Init(home)

	for _, cmd := range f.Commands {
		var err error
		fromPath, err = createContainer(fs, cmd, fromPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("Buildstep failed: %s %s.", cmd.Verb, cmd.Payload), err)
			if fromPath != "" {
				if err = fs.Remove(fromPath); err != nil {
					fmt.Fprintln(os.Stderr, "Couldn't remove temporary build container.", err)
				}
			}
			return 1
		}
	}
	if err = fs.Snapshot(fromPath, newImagePath); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't create filesystem for new image.", err)
		return 1
	}

	return 0
}

func createHash(id string, cmd parser.Command) (string, error) {
	h := sha1.New()

	io.WriteString(h, id)
	io.WriteString(h, cmd.Verb)
	io.WriteString(h, cmd.Payload)

	if cmd.Verb == "ADD" {
		p := strings.Split(cmd.Payload, " ")
		sourceFile := p[0]

		f, err := os.Open(sourceFile)
		if err != nil {
			return "", err
		}
		defer f.Close()
		reader := bufio.NewReader(f)

		_, err = io.Copy(h, reader)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func createContainer(fs *btrfs.Driver, cmd parser.Command, fromPath string) (string, error) {
	id, err := fs.GetSubvolumeId(fromPath)
	if err != nil {
		return "", err
	}

	container, err := createHash(id, cmd)
	if err != nil {
		return "", err
	}
	containerPath := fmt.Sprintf("layers/%s", container)

	if _, err := os.Stat(fmt.Sprintf("%s/%s", home, containerPath)); err == nil {
		// all fine - layer already exists
		return containerPath, nil
	}

	if err := fs.Snapshot(fromPath, containerPath); err != nil {
		return containerPath, fmt.Errorf("Couldn't create filesystem for build container. %v", err)
	}

	c := nspawn.Init(container, fmt.Sprintf("%s/%s", home, containerPath))

	if err := c.ReplaceMachineId(strings.Replace(uuid.New(), "-", "", -1)); err != nil {
		return containerPath, fmt.Errorf("Couldn't set machine-id for temporary build container. %v", err)
	}

	if cmd.Verb == "PKG" {
		if err := c.Build("RUN", "pacman -Sy --noconfirm"); err != nil {
			return containerPath, fmt.Errorf("Pacman update failed. %v", err)
		}
	}
	if err := c.Build(cmd.Verb, cmd.Payload); err != nil {
		return containerPath, fmt.Errorf("Buildstep failed: %s %s. %v", cmd.Verb, cmd.Payload, err)
	}

	// remove machine id at the end
	if err := c.ReplaceMachineId("REPLACE_ME"); err != nil {
		return containerPath, fmt.Errorf("Couldn't set machine-id placeholder for image. %v", err)
	}

	return containerPath, nil
}
