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
const buildstepTemplate string = `#!/bin/sh
mkdir -p /run/systemd/resolve
echo 'nameserver 8.8.8.8' > /run/systemd/resolve/resolv.conf

{{.Payload}}

rc=$?

rm -f /run/systemd/resolve/resolv.conf

exit $rc
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

func Init(name, path string) Container {
	return Container{
		name,
		fmt.Sprintf("conair@%s.service", name),
		path,
		".conairbuildstep",
	}
}

type Container struct {
	Name      string
	Unit      string
	Path      string
	Buildstep string
}

func (c *Container) Enable() error {
	return exec.Command("systemctl", "enable", c.Unit).Run()
}

func (c *Container) Start() error {
	return exec.Command("systemctl", "start", c.Unit).Run()
}

func (c *Container) Disable() error {
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
	o, _ := exec.Command("machinectl", "-p", "Leader", "show", c.Name).Output()

	leader := strings.TrimSpace(strings.Split(bytes.NewBuffer(o).String(), "=")[1])

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

func (c *Container) Build(verb, payload string) error {
	var cmd *exec.Cmd

	switch verb {
	case "RUN":
		if err := createBuildstep(fmt.Sprintf("%s/%s", c.Path, c.Buildstep), payload); err != nil {
			return err
		}

		cmd = exec.Command(
			"/usr/bin/systemd-nspawn",
			"--quiet",
			fmt.Sprintf("--directory=%s", c.Path),
			fmt.Sprintf("/%s", c.Buildstep),
		)
	case "ADD":
		paths := strings.Split(payload, " ")
		if len(paths) < 2 {
			return fmt.Errorf("Failed to add: %s", payload)
		}
		cmd = exec.Command("cp", paths[0], fmt.Sprintf("%s%s", c.Path, paths[1]))
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

	if verb == "RUN" {
		if err := os.Remove(fmt.Sprintf("%s/%s", c.Path, c.Buildstep)); err != nil {
			return err
		}
	}
	return nil
}

type buildstep struct {
	Payload string
}

func createBuildstep(file, payload string) error {
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
