// SPDX-License-Identifier: Apache-2.0
// Vendored from https://github.com/prometheus/procfs (branch: sriov)
// sysfs/net_class.go and sysfs/pci_device.go
// TODO: remove this package and import github.com/prometheus/procfs/sysfs
// once the procfs sriov branch is merged upstream.

package sysfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	netclassPath   = "class/net"
	pciDevicesPath = "bus/pci/devices"
)

// PciDeviceLocation represents a PCI BDF address (Segment:Bus:Device.Function).
type PciDeviceLocation struct {
	Segment  int
	Bus      int
	Device   int
	Function int
}

func (l PciDeviceLocation) String() string {
	return fmt.Sprintf("%04x:%02x:%02x.%x", l.Segment, l.Bus, l.Device, l.Function)
}

// PciDevice holds PCI device information relevant for SR-IOV.
type PciDevice struct {
	Location PciDeviceLocation
	NumaNode *int32
}

// NetClassPCIDevice returns the PciDevice backing the given network interface
// by resolving /sys/class/net/<iface>/device symlink.
// sysfsRoot is the path to the sysfs mount (normally "/sys").
func NetClassPCIDevice(sysfsRoot, iface string) (*PciDevice, error) {
	deviceSymlink := filepath.Join(sysfsRoot, netclassPath, iface, "device")
	resolved, err := os.Readlink(deviceSymlink)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve device symlink for %q: %w", iface, err)
	}
	return parsePciDevice(sysfsRoot, filepath.Base(resolved))
}

// PciDeviceVFAddress returns the PCI BDF address of Virtual Function vfIndex
// by resolving /sys/bus/pci/devices/<bdf>/virtfn<vfIndex> symlink.
// sysfsRoot is the path to the sysfs mount (normally "/sys").
func PciDeviceVFAddress(sysfsRoot string, device *PciDevice, vfIndex uint32) (string, error) {
	bdf := device.Location.String()
	virtfnPath := filepath.Join(sysfsRoot, pciDevicesPath, bdf, fmt.Sprintf("virtfn%d", vfIndex))
	resolved, err := os.Readlink(virtfnPath)
	if err != nil {
		return "", fmt.Errorf("failed to read virtfn%d symlink for %q: %w", vfIndex, bdf, err)
	}
	return filepath.Base(resolved), nil
}

func parsePciDevice(sysfsRoot, bdf string) (*PciDevice, error) {
	loc, err := parsePciDeviceLocation(bdf)
	if err != nil {
		return nil, err
	}
	dev := &PciDevice{Location: *loc}

	numaPath := filepath.Join(sysfsRoot, pciDevicesPath, bdf, "numa_node")
	if data, err := os.ReadFile(numaPath); err == nil {
		if val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 32); err == nil {
			n := int32(val)
			dev.NumaNode = &n
		}
	}

	return dev, nil
}

func parsePciDeviceLocation(bdf string) (*PciDeviceLocation, error) {
	// Format: "0000:a2:00.0"
	parts := strings.Split(bdf, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid PCI BDF %q: expected segment:bus:device.function", bdf)
	}
	segment, err := strconv.ParseInt(parts[0], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid PCI segment in %q: %w", bdf, err)
	}
	bus, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid PCI bus in %q: %w", bdf, err)
	}
	devFunc := strings.Split(parts[2], ".")
	if len(devFunc) != 2 {
		return nil, fmt.Errorf("invalid PCI device.function in %q", bdf)
	}
	device, err := strconv.ParseInt(devFunc[0], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid PCI device in %q: %w", bdf, err)
	}
	function, err := strconv.ParseInt(devFunc[1], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid PCI function in %q: %w", bdf, err)
	}
	return &PciDeviceLocation{
		Segment:  int(segment),
		Bus:      int(bus),
		Device:   int(device),
		Function: int(function),
	}, nil
}
