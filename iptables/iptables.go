package iptables

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Action string

const (
	Insert Action = "-I"
	Append Action = "-A"
	Check  Action = "-C"
	Delete Action = "-D"
)

var (
	ErrIptablesNotFound = errors.New("Iptables not found")
	supportsXlock       = false
)

func init() {
	supportsXlock = exec.Command("iptables", "--wait", "-L", "-n").Run() == nil
}

func AddBridgeForwarding(bridge, destination string) error {
	return setupBridgeForwarding(Insert, bridge, destination)
}

func DeleteBridgeForwarding(bridge, destination string) error {
	return setupBridgeForwarding(Delete, bridge, destination)
}

func setupBridgeForwarding(action Action, bridge, destination string) error {
	actionArgs := []string{fmt.Sprint(action)}
	check := (action == Delete)

	// Enable NAT
	natArgs := []string{"POSTROUTING", "-t", "nat", "-s", destination, "!", "-o", bridge, "-j", "MASQUERADE"}
	if exists(natArgs...) == check {
		if output, err := raw(append(actionArgs, natArgs...)...); err != nil {
			return fmt.Errorf("Unable to change network bridge NAT: %s", err)
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables postrouting: %s", output)
		}
	}

	forwardArgs := []string{"FORWARD", "-t", "filter"}

	// icc
	iccArgs := append(forwardArgs, "-i", bridge, "-o", bridge, "-j", "ACCEPT")
	if exists(iccArgs...) == check {
		if output, err := raw(append(actionArgs, iccArgs...)...); err != nil {
			return fmt.Errorf("Unable to change intercontainer communication: %s", err)
		} else if len(output) != 0 {
			return fmt.Errorf("Error changing intercontainer communication: %s", output)
		}
	}

	// Accept all non-intercontainer outgoing packets
	outgoingArgs := append(forwardArgs, "-i", bridge, "!", "-o", bridge, "-j", "ACCEPT")
	if exists(outgoingArgs...) == check {
		if output, err := raw(append(actionArgs, outgoingArgs...)...); err != nil {
			return fmt.Errorf("Unable to change outgoing packets: %s", err)
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables change outgoing: %s", output)
		}
	}

	// Accept incoming packets for existing connections
	existingArgs := append(forwardArgs, "-o", bridge, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT")
	if exists(existingArgs...) == check {
		if output, err := raw(append(actionArgs, existingArgs...)...); err != nil {
			return fmt.Errorf("Unable to change incoming packets: %s", err)
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables change incoming: %s", output)
		}
	}
	return nil
}

// Check if an existing rule exists
func exists(args ...string) bool {
	if _, err := raw(append([]string{fmt.Sprint(Check)}, args...)...); err != nil {
		return false
	}
	return true
}

func raw(args ...string) ([]byte, error) {
	path, err := exec.LookPath("iptables")
	if err != nil {
		return nil, ErrIptablesNotFound
	}

	if supportsXlock {
		args = append([]string{"--wait"}, args...)
	}

	if os.Getenv("DEBUG") != "" {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("[debug] %s, %v\n", path, args))
	}

	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("iptables failed: iptables %v: %s (%s)", strings.Join(args, " "), output, err)
	}

	// ignore iptables' message about xtables lock
	if strings.Contains(string(output), "waiting for it to exit") {
		output = []byte("")
	}

	return output, err
}
