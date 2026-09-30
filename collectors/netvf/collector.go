// SPDX-License-Identifier: Apache-2.0

package netvf

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jsimonetti/rtnetlink/v2"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	internalsysfs "github.com/openstack-k8s-operators/openstack-network-exporter/internal/sysfs"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovnsb"
	"github.com/prometheus/client_golang/prometheus"
)

const sysfsRoot = "/sys"

type Collector struct{}

func (Collector) Name() string {
	return "netvf"
}

func (Collector) Metrics() []lib.Metric {
	res := []lib.Metric{infoMetric}
	for _, m := range counterMetrics {
		res = append(res, *m)
	}
	return res
}

func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		log.Errf("netvf: failed to connect to rtnetlink: %s", err)
		return
	}
	defer conn.Close()

	links, err := conn.Link.ListWithVFInfo()
	if err != nil {
		log.Errf("netvf: failed to list interfaces: %s", err)
		return
	}

	portMap := buildVFPortMap()

	sets := config.MetricSets()
	buf := make(chan prometheus.Metric)
	go func() {
		collectFromLinks(links, sysfsRoot, portMap, buf)
		close(buf)
	}()
	for m := range buf {
		if sets.Has(metricSet(m)) {
			ch <- m
		}
	}
}

type vfPortInfo struct {
	portID string
	vmID   string
}

// buildVFPortMap queries OVN SB Port_Binding to map VF MAC addresses to
// Neutron port UUIDs and VM UUIDs. Returns an empty map if the SB database
// is unreachable.
func buildVFPortMap() map[string]vfPortInfo {
	portMap := make(map[string]vfPortInfo)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var ports []ovnsb.PortBinding
	if err := ovsdb.SBList(ctx, &ports); err != nil {
		log.Debugf("netvf: SB Port_Binding lookup failed: %s", err)
		return portMap
	}

	for _, p := range ports {
		vmID := p.ExternalIDs["neutron:device_id"]
		for _, macEntry := range p.MAC {
			mac := strings.SplitN(macEntry, " ", 2)[0]
			mac = strings.ToLower(mac)
			portMap[mac] = vfPortInfo{
				portID: p.LogicalPort,
				vmID:   vmID,
			}
		}
	}

	return portMap
}

// metricSet returns the MetricSet for a given prometheus.Metric by matching
// its Desc against the known metrics.
func metricSet(m prometheus.Metric) config.MetricSet {
	d := m.Desc()
	if d == infoMetric.Desc() {
		return infoMetric.Set
	}
	for _, cm := range counterMetrics {
		if d == cm.Desc() {
			return cm.Set
		}
	}
	return config.METRICS_BASE
}

// collectFromLinks is the testable core: processes rtnetlink link messages and
// emits metrics to ch. sysfsRoot is "/sys" in production, a temp dir in tests.
// It emits all metrics unconditionally; callers are responsible for filtering
// by MetricSet.
func collectFromLinks(links []rtnetlink.LinkMessage, sysfsRoot string, portMap map[string]vfPortInfo, ch chan<- prometheus.Metric) {
	for _, link := range links {
		if link.Attributes == nil {
			continue
		}
		if link.Attributes.NumVF == nil || *link.Attributes.NumVF == 0 {
			continue
		}

		device := link.Attributes.Name

		numaNode := "-1"
		var pciDev *internalsysfs.PciDevice
		if dev, err := internalsysfs.NetClassPCIDevice(sysfsRoot, device); err == nil {
			pciDev = dev
			if dev.NumaNode != nil {
				numaNode = fmt.Sprintf("%d", *dev.NumaNode)
			}
		}

		for _, vf := range link.Attributes.VFInfoList {
			vfID := fmt.Sprintf("%d", vf.ID)

			mac := ""
			if vf.MAC != nil {
				mac = vf.MAC.String()
			}
			vlan := fmt.Sprintf("%d", vf.Vlan)
			linkState := vfLinkStateString(vf.LinkState)
			spoofCheck := fmt.Sprintf("%t", vf.SpoofCheck)
			trust := fmt.Sprintf("%t", vf.Trust)

			pciAddress := ""
			if pciDev != nil {
				if addr, err := internalsysfs.PciDeviceVFAddress(sysfsRoot, pciDev, vf.ID); err == nil {
					pciAddress = addr
				}
			}

			info := portMap[strings.ToLower(mac)]

			ch <- prometheus.MustNewConstMetric(
				infoMetric.Desc(), prometheus.GaugeValue, 1,
				device, vfID, mac, vlan, linkState, spoofCheck, trust, pciAddress, numaNode, info.portID, info.vmID,
			)

			if vf.Stats == nil {
				continue
			}

			emitCounter(ch, "rx_bytes", float64(vf.Stats.RxBytes), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "tx_bytes", float64(vf.Stats.TxBytes), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "rx_packets", float64(vf.Stats.RxPackets), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "tx_packets", float64(vf.Stats.TxPackets), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "broadcast", float64(vf.Stats.Broadcast), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "multicast", float64(vf.Stats.Multicast), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "rx_dropped", float64(vf.Stats.RxDropped), device, vfID, pciAddress, numaNode)
			emitCounter(ch, "tx_dropped", float64(vf.Stats.TxDropped), device, vfID, pciAddress, numaNode)
		}
	}
}

func emitCounter(ch chan<- prometheus.Metric, key string, value float64, labels ...string) {
	m := counterMetrics[key]
	ch <- prometheus.MustNewConstMetric(m.Desc(), prometheus.CounterValue, value, labels...)
}

func vfLinkStateString(state rtnetlink.VFLinkState) string {
	switch state {
	case rtnetlink.VFLinkStateAuto:
		return "auto"
	case rtnetlink.VFLinkStateEnable:
		return "enable"
	case rtnetlink.VFLinkStateDisable:
		return "disable"
	default:
		return "unknown"
	}
}
