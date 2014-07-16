package networkd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

type networkDevice struct {
	Name        string
	Kind        string
	Destination string
}

const networkdPath string = "/etc/systemd/network"

const netDevTemplate string = `[NetDev]
Name={{.Name}}
Kind={{.Kind}}
`

const networkTemplate string = `[Match]
Name={{.Name}}

[Network]
Address={{.Destination}}.1/24
DHCPServer=yes
DNS=8.8.8.8

[Route]
Gateway={{.Destination}}.1
Destination={{.Destination}}.0/24
`

func storeNetworkDefinition(dev networkDevice, text, filename string) error {
	path := []string{networkdPath, "/", filename}
	f, err := os.Create(strings.Join(path, ""))
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	var tmpl *template.Template
	tmpl, err = template.New("device").Parse(text)
	if err != nil {
		return err
	}
	err = tmpl.Execute(f, dev)
	if err != nil {
		return err
	}
	return nil
}

func getDeviceFilename(bridge string) string {
	return strings.Join([]string{"80-", bridge, ".netdev"}, "")
}

func getNetworkFilename(bridge string) string {
	return strings.Join([]string{"82-", bridge, ".network"}, "")
}

func CreateBridge(bridge, destination string) error {
	dev := networkDevice{bridge, "bridge", strings.Replace(destination, ".0/24", "", 1)}

	err := storeNetworkDefinition(dev, netDevTemplate, getDeviceFilename(bridge))
	if err != nil {
		return err
	}

	err = storeNetworkDefinition(dev, networkTemplate, getNetworkFilename(bridge))
	if err != nil {
		return err
	}

	return restart()
}

func DeleteBridge(bridge string) error {
	err := os.Remove(fmt.Sprintf("%s/%s", networkdPath, getDeviceFilename(bridge)))
	if err != nil {
		return err
	}

	err = os.Remove(fmt.Sprintf("%s/%s", networkdPath, getNetworkFilename(bridge)))
	if err != nil {
		return err
	}

	err = exec.Command("ip", "link", "set", bridge, "down").Run()
	if err != nil {
		return err
	}

	err = exec.Command("brctl", "delbr", bridge).Run()
	if err != nil {
		return err
	}

	return restart()
}

func restart() error {
	return exec.Command("systemctl", "restart", "systemd-networkd").Run()
}
