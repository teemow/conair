package nspawn

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

const systemdPath string = "/etc/systemd/system"
const nspawnTemplate string = `[Unit]
Description=Container %i
Documentation=man:systemd-nspawn(1)

[Service]
ExecStart=/usr/bin/systemd-nspawn --machine %i --quiet --private-network --network-veth --network-bridge={{.Bridge}} --keep-unit --boot --link-journal=guest --directory={{.Directory}}/%i
KillMode=mixed
Type=notify

[Install]
WantedBy=multi-user.target
`

type unit struct {
	Bridge    string
	Directory string
}

func CreateUnit(bridge, containerPath string) error {
	u := unit{
		bridge,
		containerPath,
	}

	f, err := os.Create(fmt.Sprintf("%s/conair@.service", systemdPath))
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	var tmpl *template.Template
	tmpl, err = template.New("conair-unit").Parse(nspawnTemplate)
	if err != nil {
		return err
	}
	err = tmpl.Execute(f, u)
	if err != nil {
		return err
	}
	return nil
}

func RemoveUnit() error {
	return os.Remove(fmt.Sprintf("%s/conair@.service", systemdPath))
}

func Init(name string) container {
	return container{
		name,
		fmt.Sprintf("conair@%s.service", name),
	}
}

type container struct {
	Name string
	Unit string
}

func (c *container) Enable() error {
	return exec.Command("systemctl", "enable", c.Unit).Run()
}

func (c *container) Start() error {
	return exec.Command("systemctl", "start", c.Unit).Run()
}

func (c *container) Disable() error {
	return exec.Command("systemctl", "disable", c.Unit).Run()
}

func (c *container) Stop() error {
	return exec.Command("systemctl", "stop", c.Unit).Run()
}

func (c *container) Status() (string, error) {
	o, err := exec.Command("systemctl", "status", c.Unit).Output()

	if err != nil {
		return "", err
	}

	return bytes.NewBuffer(o).String(), nil
}

func (c *container) Attach() error {
	o, _ := exec.Command("machinectl", "-p", "Leader", "show", c.Name).Output()

	leader := strings.TrimSpace(strings.Split(bytes.NewBuffer(o).String(), "=")[1])

	// prepare the shell
	cmd := exec.Command("/usr/bin/nsenter", "-m", "-u", "-i", "-n", "-p", "-t", leader, "/bin/bash")

	cmd.Env = []string{"TERM=vt102", "SHELL=/bin/bash", "USER=root", "LANG=C", "HOME=/root", "PWD=/root", "PATH=/usr/local/sbin:/usr/local/bin:/usr/bin:/usr/bin/core_perl"}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.Run()

	return nil
}
