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

const networkClientTemplate string = `[Match]
Virtualization=container
Name=host0

[Network]
DHCP=v4
IPv4LL=no

[Route]
Gateway={{.Destination}}.1
`

func storeNetworkDefinition(dev networkDevice, text, path string) error {
	f, err := os.Create(path)
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

	netDevPath := fmt.Sprintf("%s/%s", networkdPath, getDeviceFilename(bridge))
	err := storeNetworkDefinition(dev, netDevTemplate, netDevPath)
	if err != nil {
		return err
	}

	networkPath := fmt.Sprintf("%s/%s", networkdPath, getNetworkFilename(bridge))
	err = storeNetworkDefinition(dev, networkTemplate, networkPath)
	if err != nil {
		return err
	}

	return restart()
}

func CreateClientNetwork(containerPath, destination string) error {
	dev := networkDevice{Destination: strings.Replace(destination, ".0/24", "", 1)}

	networkClientPath := fmt.Sprintf("%s/%s/80-container-host0.network", containerPath, networkdPath)
	return storeNetworkDefinition(dev, networkClientTemplate, networkClientPath)
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
