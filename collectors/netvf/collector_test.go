// SPDX-License-Identifier: Apache-2.0

package netvf

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/jsimonetti/rtnetlink/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// buildFakeSysfs creates a minimal /sys tree for one PF with one VF:
//
//	/sys/class/net/enp3s0f0/device -> ../../../../bus/pci/devices/0000:03:00.0
//	/sys/bus/pci/devices/0000:03:00.0/numa_node  -> "1"
//	/sys/bus/pci/devices/0000:03:00.0/virtfn0    -> ../0000:03:01.0
func buildFakeSysfs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	pfBDF := "0000:03:00.0"
	vfBDF := "0000:03:01.0"

	pfDevPath := filepath.Join(root, "bus/pci/devices", pfBDF)
	if err := os.MkdirAll(pfDevPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pfDevPath, "numa_node"), []byte("1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	vfDevPath := filepath.Join(root, "bus/pci/devices", vfBDF)
	if err := os.MkdirAll(vfDevPath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink("../"+vfBDF, filepath.Join(pfDevPath, "virtfn0")); err != nil {
		t.Fatal(err)
	}

	netDevPath := filepath.Join(root, "class/net/enp3s0f0")
	if err := os.MkdirAll(netDevPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../../../bus/pci/devices/"+pfBDF, filepath.Join(netDevPath, "device")); err != nil {
		t.Fatal(err)
	}

	return root
}

func uint32Ptr(v uint32) *uint32 { return &v }

func TestCollectMetrics(t *testing.T) {
	sysfsRoot := buildFakeSysfs(t)

	mac, _ := net.ParseMAC("52:54:00:ab:cd:ef")
	numVF := uint32(1)

	links := []rtnetlink.LinkMessage{
		{
			Attributes: &rtnetlink.LinkAttributes{
				Name:  "enp3s0f0",
				NumVF: &numVF,
				VFInfoList: []rtnetlink.VFInfo{
					{
						ID:         0,
						MAC:        mac,
						Vlan:       100,
						LinkState:  rtnetlink.VFLinkStateAuto,
						SpoofCheck: false,
						Trust:      true,
						Stats: &rtnetlink.VFStats{
							RxPackets: 10,
							TxPackets: 20,
							RxBytes:   1000,
							TxBytes:   2000,
							Broadcast: 1,
							Multicast: 2,
							RxDropped: 3,
							TxDropped: 4,
						},
					},
				},
			},
		},
	}

	ch := make(chan prometheus.Metric, 20)
	collectFromLinks(links, sysfsRoot, ch)
	close(ch)

	got := map[string]bool{}
	for m := range ch {
		got[m.Desc().String()] = true
	}

	wantDescs := []string{
		infoMetric.Desc().String(),
		counterMetrics["rx_bytes"].Desc().String(),
		counterMetrics["tx_bytes"].Desc().String(),
		counterMetrics["rx_packets"].Desc().String(),
		counterMetrics["tx_packets"].Desc().String(),
		counterMetrics["broadcast"].Desc().String(),
		counterMetrics["multicast"].Desc().String(),
		counterMetrics["rx_dropped"].Desc().String(),
		counterMetrics["tx_dropped"].Desc().String(),
	}

	for _, desc := range wantDescs {
		if !got[desc] {
			t.Errorf("missing metric: %s", desc)
		}
	}
}

func TestSkipsLinksWithoutVFs(t *testing.T) {
	links := []rtnetlink.LinkMessage{
		{
			Attributes: &rtnetlink.LinkAttributes{
				Name:  "eth0",
				NumVF: uint32Ptr(0),
			},
		},
		{
			Attributes: &rtnetlink.LinkAttributes{
				Name: "eth1",
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collectFromLinks(links, "/nonexistent", ch)
	close(ch)

	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 metrics for links without VFs, got %d", count)
	}
}
