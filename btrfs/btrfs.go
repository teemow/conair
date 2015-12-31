package btrfs

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

type FsMagic int64

const (
	FsMagicBtrfs      = FsMagic(0x9123683E)
	FsMagicBtrfs32Bit = FsMagic(-1859950530)
)

var (
	ErrPrerequisites = errors.New("prerequisites for driver not satisfied (wrong filesystem?)")
)

func Init(home string) (*Driver, error) {
	rootdir := path.Dir(home + "/")

	var buf syscall.Statfs_t
	if err := syscall.Statfs(rootdir, &buf); err != nil {
		return nil, err
	}

	if !(FsMagic(buf.Type) == FsMagicBtrfs || FsMagic(buf.Type) == FsMagicBtrfs32Bit) {
		return nil, ErrPrerequisites
	}

	if err := os.MkdirAll(home, 0700); err != nil {
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

	if !d.Exists(from) {
		return fmt.Errorf("Volume does not exist: %s", fromPath)
	}
	if d.Exists(to) {
		return fmt.Errorf("Snapshot already exists: %s", toPath)
	}

	var cmd *exec.Cmd

	if readonly {
		cmd = raw("subvolume", "snapshot", "-r", fromPath, toPath)
	} else {
		cmd = raw("subvolume", "snapshot", fromPath, toPath)
	}
	if err := cmd.Run(); err != nil {
		return err
	}

	// create recursive snapshots.
	subvolumes, err := d.ListSubSubvolumes(from)
	if err != nil {
		return err
	}

	for _, subvol := range subvolumes {
		subvolFrom := fmt.Sprintf("%s/%s", from, subvol)
		subvolTo := fmt.Sprintf("%s/%s", to, subvol)

		// delete empty directory
		if d.Exists(subvolTo) {
			if err := os.Remove(fmt.Sprintf("%s/%s", d.home, subvolTo)); err != nil {
				return err
			}
		}

		err = d.Snapshot(subvolFrom, subvolTo, readonly)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) Subvolume(vol string) error {
	volPath := fmt.Sprintf("%s/%s", d.home, vol)
	if _, err := os.Stat(volPath); err == nil {
		return fmt.Errorf("Subvolume already exists: %s", volPath)
	}

	return raw("subvolume", "create", volPath).Run()
}

func (d *Driver) Exists(vol string) bool {
	volPath := fmt.Sprintf("%s/%s", d.home, vol)
	_, err := os.Stat(volPath)
	if err == nil {
		return true
	} else {
		// check os.IsNotExist(err) ?
		return false
	}
}

func (d *Driver) GetSubvolumeDetail(vol, detail string) (string, error) {
	volPath := fmt.Sprintf("%s/%s", d.home, vol)

	o, _ := raw("subvolume", "show", volPath).Output()

	output := string(o)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) > 1 {
			key, val := strings.Trim(fields[0], " \t"), strings.Trim(fields[1], " \t")
			if strings.EqualFold(key, detail) {
				return val, nil
			}
		}
	}
	return "", fmt.Errorf("Subvolume detail %s not found", detail)
}

func (d *Driver) GetSubvolumeParentUuid(vol string) (string, error) {
	return d.GetSubvolumeDetail(vol, "Parent uuid")

}

func (d *Driver) GetSubvolumeUuid(vol string) (string, error) {
	return d.GetSubvolumeDetail(vol, "uuid")
}

func (d *Driver) ListSubvolumes() ([]string, error) {
	var volumes []string

	// find sub-subvolumes
	cmd := raw("subvolume", "list", "-o", d.home, "-u")

	output, err := cmd.StdoutPipe()
	if err != nil {
		return volumes, fmt.Errorf("Can't access subvolume list of %s: %v", d.home, err)
	}
	defer output.Close()
	err = cmd.Start()
	if err != nil {
		return volumes, err
	}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "__active") {
			line = strings.Replace(line, "__active/", "", 1)
		}
		volumes = append(volumes, line)
	}
	return volumes, nil
}

func (d *Driver) GetLayerByUuid(uuid string) (string, error) {
	layers, err := d.ListSubvolumes()
	if err != nil {
		return "", err
	}

	replaceHome := strings.Replace(d.home, "/", "", 1) + "/"
	for _, layer := range layers {
		if strings.Contains(layer, fmt.Sprintf(" uuid %s ", uuid)) {
			layerDetails := strings.Split(layer, " ")
			if len(layerDetails) > 10 {
				return strings.Replace(layerDetails[10], replaceHome, "", 1), nil
			}
		}
	}
	return "", fmt.Errorf("No layer found")
}

func (d *Driver) ListSubSubvolumes(vol string) ([]string, error) {
	var volumes []string

	volPath := fmt.Sprintf("%s/%s", d.home, vol)

	if !d.Exists(vol) {
		return volumes, fmt.Errorf("Volume does not exist: %s", volPath)
	}

	// find sub-subvolumes
	cmd := raw("subvolume", "list", "-o", volPath)

	output, err := cmd.StdoutPipe()
	if err != nil {
		return volumes, fmt.Errorf("Can't access subvolume list of %s: %v", volPath, err)
	}
	defer output.Close()
	err = cmd.Start()
	if err != nil {
		return volumes, err
	}

	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), " ")
		if len(line) > 8 {
			subvol := strings.Join(line[8:], " ")
			// remove beginning of volume path - relative to conair home
			if strings.Contains(subvol, "__active") {
				subvol = strings.Replace(subvol, "__active/", "", 1)
			}

			if strings.HasPrefix(subvol, volPath) {
				volumes = append(volumes, strings.Replace(subvol, fmt.Sprintf("%s/", strings.Replace(volPath, "/", "", 1)), "", 1))
			}

			if strings.HasPrefix(subvol, vol) {
				volumes = append(volumes, strings.Replace(subvol, fmt.Sprintf("%s/", vol), "", 1))
			}
		}
	}
	err = scanner.Err()
	if err != nil {
		return volumes, fmt.Errorf("Can't read subvolume list of %s: %v", volPath, err)
	}
	return volumes, nil
}

func (d *Driver) Remove(vol string) error {
	volPath := fmt.Sprintf("%s/%s", d.home, vol)

	if !d.Exists(vol) {
		return fmt.Errorf("Volume does not exist: %s", volPath)
	}

	subvolumes, err := d.ListSubSubvolumes(vol)
	if err != nil {
		return err
	}

	for _, subvol := range subvolumes {
		if err := d.Remove(fmt.Sprintf("%s/%s", vol, subvol)); err != nil {
			return err
		}
	}

	return raw("subvolume", "delete", volPath).Run()
}

func raw(args ...string) *exec.Cmd {
	return exec.Command("btrfs", args...)
}
