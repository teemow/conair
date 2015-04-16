package nspawn

import (
	"bytes"
	"fmt"

	"os"
	"os/exec"
	"strings"
	"text/template"

	"code.google.com/p/go-uuid/uuid"
)

const (
	nspawnConfigTemplate string = `[Service]
Environment="MACHINE_ID={{.MachineId}}"
Environment="BIND={{.Bind}}"
`
	nspawnMachineIdTemplate string = `{{.MachineId}}
`
	buildstepTemplate string = `#!/bin/sh

{{.Payload}}

rc=$?

exit $rc
`
)

type Container struct {
	Name       string
	Unit       string
	Path       string
	Buildstep  string
	ConfigPath string
	Binds      []string
	Snapshots  []string
}

type config struct {
	MachineId string
	Bind      string
}

func Init(name, path string) Container {
	c := Container{
		Name:      name,
		Unit:      fmt.Sprintf("conair@%s.service", name),
		Path:      path,
		Buildstep: ".conairbuildstep",
		Binds:     make([]string, 0),
		Snapshots: make([]string, 0),
	}
	c.ConfigPath = fmt.Sprintf("%s/%s.d", systemdPath, c.Unit)

	return c
}

func (c *Container) SetSnapshots(snapshots []string) {
	c.Snapshots = snapshots
}

func (c *Container) SetBinds(binds []string) {
	c.Binds = binds
}

func (c *Container) createConfig() error {
	conf := config{
		MachineId: strings.Replace(uuid.New(), "-", "", -1),
		Bind:      "",
	}

	if len(c.Binds) > 0 {
		tmp := make([]string, 0)
		for _, bind := range c.Binds {
			tmp = append(tmp, fmt.Sprintf("--bind=%s", bind))
		}
		conf.Bind = strings.Join(tmp, " ")
	}

	if err := os.Mkdir(c.ConfigPath, 0755); err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf("%s/10-container.conf", c.ConfigPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	var tmpl *template.Template
	tmpl, err = template.New("container-config").Parse(nspawnConfigTemplate)
	if err != nil {
		return err
	}
	err = tmpl.Execute(f, conf)
	if err != nil {
		return err
	}
	return nil
}

func (c *Container) removeConfig() error {
	if err := os.Remove(fmt.Sprintf("%s/10-container.conf", c.ConfigPath)); err != nil {
		return err
	}
	if err := os.Remove(c.ConfigPath); err != nil {
		return err
	}
	return nil
}

func (c *Container) Enable() error {
	if err := c.createConfig(); err != nil {
		return err
	}

	return exec.Command("systemctl", "enable", c.Unit).Run()
}

func (c *Container) Start() error {
	return exec.Command("systemctl", "start", c.Unit).Run()
}

func (c *Container) Disable() error {
	if err := c.removeConfig(); err != nil {
		return err
	}

	return exec.Command("systemctl", "disable", c.Unit).Run()
}

func (c *Container) Stop() error {
	return exec.Command("systemctl", "stop", c.Unit).Run()
}

func (c *Container) Status() (string, error) {
	o, err := exec.Command("systemctl", "status", c.Unit).Output()

	if err != nil {
		return "", err
	}

	return bytes.NewBuffer(o).String(), nil
}

func (c *Container) Attach() error {
	leader, err := c.getLeader()
	if err != nil {
		return err
	}

	// prepare the shell
	cmd := exec.Command("/usr/bin/nsenter", "-m", "-u", "-i", "-n", "-p", "-t", leader, "/bin/bash")

	cmd.Env = []string{
		"TERM=vt102",
		"SHELL=/bin/bash",
		"USER=root",
		"LANG=C",
		"HOME=/root",
		"PWD=/root",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/bin:/usr/bin/core_perl",
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.Run()

	return nil
}

func (c *Container) getLeader() (string, error) {
	o, err := exec.Command("machinectl", "-p", "Leader", "show", c.Name).Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.Split(bytes.NewBuffer(o).String(), "=")[1]), nil
}

func (c *Container) Execute(payload string) (string, error) {
	leader, err := c.getLeader()
	if err != nil {
		return "", err
	}

	// prepare the shell
	cmd := exec.Command("/usr/bin/nsenter", "-m", "-u", "-i", "-n", "-p", "-t", leader, "/bin/bash")

	cmd.Env = []string{
		"TERM=vt102",
		"SHELL=/bin/bash",
		"USER=root",
		"LANG=C",
		"HOME=/root",
		"PWD=/root",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/bin:/usr/bin/core_perl",
	}
	cmd.Stdin = strings.NewReader(payload)

	bs, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func (c *Container) Inspect() (string, error) {
	return c.Execute("ip a show dev host0")
}

func (c *Container) Ip() (string, error) {
	return c.Execute("ip route get 128.193.4.20 | awk '{print $7}'")
}

func (c *Container) Build(verb, payload string) error {
	var (
		cmd *exec.Cmd
		err error
	)

	if verb == "PKG" {
		if err := c.Build("RUN", "pacman -Sy --noconfirm"); err != nil {
			return err
		}
	}

	switch verb {
	case "RUN":
		cmd, err = c.run(payload)
	case "RUN_NOCACHE":
		cmd, err = c.run(payload)
	case "ADD":
		cmd, err = c.add(payload)
	case "PKG":
		cmd, err = c.pkg(payload)
	case "ENABLE":
		cmd, err = c.enable(payload)
	default:
		return nil
	}

	if err != nil {
		return err
	}

	cmd.Env = []string{
		"TERM=vt102",
		"SHELL=/bin/bash",
		"USER=root",
		"LANG=C",
		"HOME=/root",
		"PWD=/root",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/bin:/usr/bin/core_perl",
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return err
	}

	if verb != "ADD" {
		if err := c.cleanupBuildstep(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) run(payload string) (*exec.Cmd, error) {
	if err := c.prepareBuildstep(payload); err != nil {
		return nil, err
	}

	params := make([]string, 0)
	params = append(params, "--quiet", fmt.Sprintf("--directory=%s", c.Path))
	for _, bind := range c.Binds {
		params = append(params, fmt.Sprintf("--bind=%s", bind))
	}
	params = append(params, fmt.Sprintf("/%s", c.Buildstep))

	return exec.Command("/usr/bin/systemd-nspawn", params...), nil
}

func (c *Container) add(payload string) (*exec.Cmd, error) {
	paths := strings.Split(payload, " ")
	if len(paths) < 2 {
		return nil, fmt.Errorf("Failed to add: %s", payload)
	}
	return exec.Command("cp", paths[0], fmt.Sprintf("%s%s", c.Path, paths[1])), nil
}

func (c *Container) enable(payload string) (*exec.Cmd, error) {
	return c.run(fmt.Sprintf("systemctl enable %s", payload))
}

func (c *Container) pkg(payload string) (*exec.Cmd, error) {
	return c.run(fmt.Sprintf("pacman -S --noconfirm %s", payload))
}

func (c *Container) prepareBuildstep(payload string) error {
	if err := c.replaceMachineId(strings.Replace(uuid.New(), "-", "", -1)); err != nil {
		return fmt.Errorf("Couldn't set machine-id for temporary build container. %v", err)
	}
	if err := c.createBuildstep(payload); err != nil {
		return fmt.Errorf("Couldn't buildstep for temporary build container. %v", err)
	}

	return nil
}

func (c *Container) cleanupBuildstep() error {
	if err := c.replaceMachineId("REPLACE_ME"); err != nil {
		return fmt.Errorf("Couldn't set machine-id placeholder for image. %v", err)
	}

	if err := c.removeBuildstep(); err != nil {
		return fmt.Errorf("Couldn't remove buildstep for temporary build container. %v", err)
	}

	return nil
}

func (c *Container) replaceMachineId(machineId string) error {
	conf := config{
		MachineId: machineId,
	}

	f, err := os.Create(fmt.Sprintf("%s/etc/machine-id", c.Path))
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	var tmpl *template.Template
	tmpl, err = template.New("container-machine-id").Parse(nspawnMachineIdTemplate)
	if err != nil {
		return err
	}
	err = tmpl.Execute(f, conf)
	if err != nil {
		return err
	}
	return nil
}

type buildstep struct {
	Payload string
}

func (c *Container) createBuildstep(payload string) error {
	file := fmt.Sprintf("%s/%s", c.Path, c.Buildstep)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	if err := f.Chmod(0755); err != nil {
		return err
	}

	var tmpl *template.Template
	tmpl, err = template.New("conair-buildstep").Parse(buildstepTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(f, buildstep{
		payload,
	})
}

func (c *Container) removeBuildstep() error {
	return os.Remove(fmt.Sprintf("%s/%s", c.Path, c.Buildstep))
}
