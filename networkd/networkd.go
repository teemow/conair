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
	Destination string
}

const (
	networkdPath         = "/etc/systemd/network"
	bridgeHostFile       = "80-container-bridge.netdev"
	networkHostFile      = "82-container-bridge.network"
	networkContainerFile = "80-container-host0.network"
)

const bridgeHostTemplate string = `[NetDev]
Name={{.Name}}
Kind=bridge
`

const networkHostTemplate string = `[Match]
Name={{.Name}}

[Network]
Address={{.Destination}}.1/24
LinkLocalAddressing=yes
DHCPServer=yes
DNS=8.8.8.8
IPMasquerade=yes
`

const networkContainerTemplate string = `[Match]
Virtualization=container
Name=host0

[Network]
DHCP=yes
LinkLocalAddressing=yes

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

func DefineHostNetwork(bridge, destination string) error {
	dev := networkDevice{
		Name:        bridge,
		Destination: strings.Replace(destination, ".0/24", "", 1),
	}

	bridgeHostPath := fmt.Sprintf("%s/%s", networkdPath, bridgeHostFile)
	err := storeNetworkDefinition(dev, bridgeHostTemplate, bridgeHostPath)
	if err != nil {
		return err
	}

	networkHostPath := fmt.Sprintf("%s/%s", networkdPath, networkHostFile)
	err = storeNetworkDefinition(dev, networkHostTemplate, networkHostPath)
	if err != nil {
		return err
	}

	return restart()
}

func DefineContainerNetwork(containerPath, destination string) error {
	dev := networkDevice{
		Destination: strings.Replace(destination, ".0/24", "", 1),
	}

	networkContainerPath := fmt.Sprintf("%s/%s/%s", containerPath, networkdPath, networkContainerFile)
	return storeNetworkDefinition(dev, networkContainerTemplate, networkContainerPath)
}

func RemoveHostNetwork() error {
	err := os.Remove(fmt.Sprintf("%s/%s", networkdPath, bridgeHostFile))
	if err != nil {
		return err
	}

	err = os.Remove(fmt.Sprintf("%s/%s", networkdPath, networkHostFile))
	if err != nil {
		return err
	}

	return restart()
}

func restart() error {
	return exec.Command("systemctl", "restart", "systemd-networkd").Run()
}
