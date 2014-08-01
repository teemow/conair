package btrfs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

type FsMagic uint64

const (
	FsMagicBtrfs = FsMagic(0x9123683E)
)

var (
	ErrPrerequisites = errors.New("prerequisites for driver not satisfied (wrong filesystem?)")
)

func Init(home string) (*Driver, error) {
	rootdir := path.Dir(home)

	var buf syscall.Statfs_t
	if err := syscall.Statfs(rootdir, &buf); err != nil {
		return nil, err
	}

	if FsMagic(buf.Type) != FsMagicBtrfs {
		return nil, ErrPrerequisites
	}

	if err := os.MkdirAll(home, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fmt.Sprintf("%s/%s", home, "/images"), 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fmt.Sprintf("%s/%s", home, "/container"), 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(fmt.Sprintf("%s/%s", home, "/layers"), 0700); err != nil {
		return nil, err
	}

	return &Driver{
		home: home,
	}, nil
}

type Driver struct {
	home string
}

func (d *Driver) Snapshot(from, to string) error {
	fromPath := fmt.Sprintf("%s/%s", d.home, from)
	toPath := fmt.Sprintf("%s/%s", d.home, to)

	if _, err := os.Stat(fromPath); os.IsNotExist(err) {
		return fmt.Errorf("Volume does not exist: %s", fromPath)
	}
	if _, err := os.Stat(toPath); err == nil {
		return fmt.Errorf("Snapshot already exists: %s", toPath)
	}

	return exec.Command("btrfs", "subvolume", "snapshot", fromPath, toPath).Run()
}

func (d *Driver) Subvolume(folder string) error {
	folderPath := fmt.Sprintf("%s/%s", d.home, folder)
	if _, err := os.Stat(folderPath); err == nil {
		return fmt.Errorf("Subvolume already exists: %s", folderPath)
	}

	return exec.Command("btrfs", "subvolume", "create", folderPath).Run()
}

func (d *Driver) GetSubvolumeId(folder string) (string, error) {
	folderPath := fmt.Sprintf("%s/%s", d.home, folder)

	cmd := exec.Command("btrfs", "subvolume", "list", "-o", d.home)
	stdout, _ := cmd.StdoutPipe()
	reader := bufio.NewReader(stdout)
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var id string

	for {
		l, isPrefix, err := reader.ReadLine()
		if isPrefix {
			return "", fmt.Errorf("Couldn't get subvolume id: %s", folder)
		}
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}

		line := string(l)
		if strings.Contains(line, folderPath) {
			id = strings.Fields(line)[1]
			break
		}

	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return id, nil

}

func (d *Driver) Remove(vol string) error {
	volPath := fmt.Sprintf("%s/%s", d.home, vol)

	if _, err := os.Stat(volPath); os.IsNotExist(err) {
		return fmt.Errorf("Volume does not exist: %s", volPath)
	}

	// find sub-subvolumes
	cmd := exec.Command("btrfs", "subvolume", "list", "-o", volPath)

	output, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Can't access subvolume list of %s: %v", volPath, err)
	}
	defer output.Close()
	err = cmd.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), " ")
		if len(line) > 8 {
			subvol := strings.Join(line[8:], " ")
			// remove beginning of volume path - relative to conair home
			subvol = strings.Replace(subvol, fmt.Sprintf("__active%s", d.home), "", 1)
			if err := d.Remove(subvol); err != nil {
				return err
			}
		}
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("Can't read subvolume list of %s: %v", volPath, err)
	}

	return exec.Command("btrfs", "subvolume", "delete", volPath).Run()
}

func (d *Driver) Show() string {
	details, _ := exec.Command("btrfs", "subvolume", "show", d.home).Output()
	fmt.Printf("%s", details)

	return bytes.NewBuffer(details).String()
}
