package btrfs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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

func (d *Driver) Snapshot(from, to string, readonly bool) error {
	fromPath := fmt.Sprintf("%s/%s", d.home, from)
	toPath := fmt.Sprintf("%s/%s", d.home, to)

	if _, err := os.Stat(fromPath); os.IsNotExist(err) {
		return fmt.Errorf("Volume does not exist: %s", fromPath)
	}
	if _, err := os.Stat(toPath); err == nil {
		return fmt.Errorf("Snapshot already exists: %s", toPath)
	}

	var cmd *exec.Cmd

	if readonly {
		cmd = raw("subvolume", "snapshot", "-r", fromPath, toPath)
	} else {
		cmd = raw("subvolume", "snapshot", fromPath, toPath)
	}
	return cmd.Run()
}

func (d *Driver) Subvolume(folder string) error {
	folderPath := fmt.Sprintf("%s/%s", d.home, folder)
	if _, err := os.Stat(folderPath); err == nil {
		return fmt.Errorf("Subvolume already exists: %s", folderPath)
	}

	return raw("subvolume", "create", folderPath).Run()
}

func (d *Driver) GetSubvolumeDetail(subvol, detail string) (string, error) {
	subvolPath := fmt.Sprintf("%s/%s", d.home, subvol)

	o, _ := raw("subvolume", "show", subvolPath).Output()

	output := string(o)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) > 1 {
			key, val := strings.Trim(fields[0], " \t"), strings.Trim(fields[1], " \t")
			if key == detail {
				return val, nil
			}
		}
	}
	return "", fmt.Errorf("Subvolume detail %s not found", detail)
}

func (d *Driver) GetSubvolumeParentUuid(folder string) (string, error) {
	return d.GetSubvolumeDetail(folder, "Parent uuid")

}

func (d *Driver) GetSubvolumeUuid(folder string) (string, error) {
	return d.GetSubvolumeDetail(folder, "uuid")
}

func (d *Driver) Remove(vol string) error {
	volPath := fmt.Sprintf("%s/%s", d.home, vol)

	if _, err := os.Stat(volPath); os.IsNotExist(err) {
		return fmt.Errorf("Volume does not exist: %s", volPath)
	}

	// find sub-subvolumes
	cmd := raw("subvolume", "list", "-o", volPath)

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

	return raw("subvolume", "delete", volPath).Run()
}

func (d *Driver) Show() string {
	details, _ := raw("subvolume", "show", d.home).Output()
	fmt.Printf("%s", details)

	return bytes.NewBuffer(details).String()
}

func raw(args ...string) *exec.Cmd {
	cmd := exec.Command("btrfs")
	cmd.Args = args
	return cmd
}
