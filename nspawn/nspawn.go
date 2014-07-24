package nspawn

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
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

func (c *Container) run(payload string) (*exec.Cmd, error) {
	if err := createBuildstep(fmt.Sprintf("%s/%s", c.Path, c.Buildstep), payload); err != nil {
		return nil, err
	}

	return exec.Command(
		"/usr/bin/systemd-nspawn",
		"--quiet",
		fmt.Sprintf("--directory=%s", c.Path),
		fmt.Sprintf("/%s", c.Buildstep),
	), nil
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

func (c *Container) Build(verb, payload string) error {
	var (
		cmd *exec.Cmd
		err error
	)

	switch verb {
	case "RUN":
		cmd, err = c.run(payload)
	case "ADD":
		cmd, err = c.add(payload)
	case "PKG":
		cmd, err = c.pkg(payload)
	case "ENABLE":
		cmd, err = c.enable(payload)
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

func FetchImage(image, newImage, url, path string) error {
	tarFile := fmt.Sprintf("%s/%s.tar.bz2", path, image)
	out, err := os.Create(tarFile)
	defer out.Close()
	if err != nil {
		return err
	}

	fmt.Printf("Fetching %s image.\n", image)
	var resp *http.Response
	resp, err = http.Get(fmt.Sprintf("%s/%s.tar.bz2", url, image))
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	var n int64
	n, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("Fetched %s image. Downloaded %d bytes.\n", image, n)

	fmt.Printf("Extracting %s...\n", tarFile)
	cmd := exec.Command("tar", "xjf", tarFile)
	cmd.Dir = fmt.Sprintf("%s/%s", path, newImage)
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

	if err := os.Remove(tarFile); err != nil {
		return err
	}
	return nil
}

func CreateImage(name, path string) error {
	cmd := exec.Command("pacstrap", "-c", "-d", fmt.Sprintf("%s/%s", path, name),
		"bash", "bzip2", "coreutils", "diffutils", "file", "filesystem", "findutils",
		"gawk", "gcc-libs", "gettext", "glibc", "grep", "gzip", "iproute2", "iputils",
		"less", "libutil-linux", "licenses", "logrotate", "nano", "pacman", "procps-ng",
		"psmisc", "sed", "shadow", "sysfsutils", "tar", "texinfo", "util-linux", "vi", "which")

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

	c := Init(name, fmt.Sprintf("%s/%s", path, name))

	c.Build("ENABLE", "systemd-networkd systemd-resolved")
	c.Build("RUN", "rm -f /etc/resolv.conf")
	c.Build("RUN", "ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf")

	return nil
}
