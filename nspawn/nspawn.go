package nspawn

import (
	"fmt"
	"io"
	"net/http"

	"os"
	"os/exec"
	"text/template"
)

const systemdPath string = "/etc/systemd/system"
const nspawnTemplate string = `[Unit]
Description=Container %i
Documentation=man:systemd-nspawn(1)
PartOf=machines.target
Before=machines.target

[Service]
ExecStartPre=/usr/bin/sed -i "s/REPLACE_ME/${MACHINE_ID}/" {{.Directory}}/%i/etc/machine-id
ExecStartPre=/usr/bin/chmod -w {{.Directory}}/%i/etc/machine-id
ExecStart=/usr/bin/systemd-nspawn --machine %i --uuid=${MACHINE_ID} --capability=all --quiet --network-veth --network-bridge={{.Bridge}} --keep-unit --boot --link-journal=try-guest --directory={{.Directory}}/%i $BIND
KillMode=mixed
Type=notify
RestartForceExitStatus=133
SuccessExitStatus=133
Delegate=yes

[Install]
WantedBy=machines.target
`

type unit struct {
	Bridge    string
	Directory string
}

func CreateUnit(bridge, containerPath string) error {
	u := unit{
		Bridge:    bridge,
		Directory: containerPath,
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

func CreateImage(name, path string) error {
	cmd := exec.Command("pacstrap", "-c", "-d", fmt.Sprintf("%s/%s", path, name),
		"bash", "bzip2", "coreutils", "diffutils", "file", "filesystem", "findutils",
		"gawk", "gcc-libs", "gettext", "glibc", "grep", "gzip", "iproute2", "iputils",
		"less", "libutil-linux", "licenses", "logrotate", "nano", "pacman", "procps-ng",
		"psmisc", "sed", "shadow", "sysfsutils", "systemd", "tar", "texinfo", "util-linux", "vi", "which")

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
	c.Build("RUN", "mkdir /etc/systemd/resolved.conf.d")
	c.Build("RUN", "echo '[Resolve]' > /etc/systemd/resolved.conf.d/dns.conf")
	c.Build("RUN", "echo 'DNS=8.8.8.8 8.8.4.4' >> /etc/systemd/resolved.conf.d/dns.conf")

	return nil
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
