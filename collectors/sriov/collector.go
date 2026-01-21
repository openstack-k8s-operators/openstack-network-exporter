// SPDX-License-Identifier: Apache-2.0

package sriov

import (
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/safchain/ethtool"
	"github.com/vishvananda/netlink"
)

type Collector struct{}

func (Collector) Name() string {
	return "sriov"
}

func (Collector) Metrics() []lib.Metric {
	var res []lib.Metric
	for _, m := range metrics {
		res = append(res, m)
	}
	for _, m := range vfNetlinkMetrics {
		res = append(res, m)
	}
	return res
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

type InterfaceInfo struct {
	Name     string
	IsPF     bool
	IsVF     bool
	ParentPF string
	VFNum    int
	NumVFs   int
	Driver   string
	PCIAddr  string
	NumaNode string
}

func discoverSriovInterfaces() ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	netPath := "/sys/class/net"
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		name := entry.Name()

		if name == "lo" {
			continue
		}

		info := InterfaceInfo{Name: name, VFNum: -1}

		devicePath := filepath.Join(netPath, name, "device")

		if pciAddr, err := filepath.EvalSymlinks(devicePath); err == nil {
			info.PCIAddr = filepath.Base(pciAddr)
		}

		info.Driver = getDriver(devicePath)
		info.NumaNode = getNumaNode(devicePath)

		numVFsPath := filepath.Join(devicePath, "sriov_numvfs")
		if numVFs, err := readIntFromFile(numVFsPath); err == nil {
			info.IsPF = true
			info.NumVFs = numVFs
			interfaces = append(interfaces, info)
			continue
		}

		physfnPath := filepath.Join(devicePath, "physfn")
		if _, err := os.Lstat(physfnPath); err == nil {
			info.IsVF = true

			if vfNum, pfPCI := getVFNumber(devicePath); vfNum >= 0 {
				info.VFNum = vfNum
				info.ParentPF = pfPCI
			}

			interfaces = append(interfaces, info)
		}
	}

	return interfaces, nil
}

func getDriver(devicePath string) string {
	driverPath := filepath.Join(devicePath, "driver")
	if target, err := os.Readlink(driverPath); err == nil {
		return filepath.Base(target)
	}
	return "none"
}

func getNumaNode(devicePath string) string {
	numaPath := filepath.Join(devicePath, "numa_node")
	data, err := os.ReadFile(numaPath)
	if err != nil {
		return "-1"
	}
	numaNode := strings.TrimSpace(string(data))
	if numaNode == "" {
		return "-1"
	}
	return numaNode
}

func readIntFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func getVFNumber(devicePath string) (int, string) {
	physfnPath := filepath.Join(devicePath, "physfn")
	pfDevice, err := os.Readlink(physfnPath)
	if err != nil {
		return -1, ""
	}

	pfDevicePath := filepath.Join(devicePath, pfDevice)
	entries, err := os.ReadDir(pfDevicePath)
	if err != nil {
		return -1, ""
	}

	myDevice, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return -1, ""
	}

	virtfnRe := regexp.MustCompile(`^virtfn(\d+)$`)
	for _, entry := range entries {
		match := virtfnRe.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}

		virtfnPath := filepath.Join(pfDevicePath, entry.Name())
		target, err := os.Readlink(virtfnPath)
		if err != nil {
			continue
		}

		targetPath := filepath.Join(pfDevicePath, target)
		targetAbs, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			continue
		}

		if targetAbs == myDevice {
			vfNum, _ := strconv.Atoi(match[1])
			return vfNum, filepath.Base(pfDevicePath)
		}
	}

	return -1, ""
}

func getEthtoolStats(iface string) (map[string]uint64, error) {
	eth, err := ethtool.NewEthtool()
	if err != nil {
		return nil, err
	}
	defer eth.Close()

	return eth.Stats(iface)
}

func buildLabels(info InterfaceInfo, dataSource string) []string {
	vfNum := ""
	if info.VFNum >= 0 {
		vfNum = strconv.Itoa(info.VFNum)
	}

	ifType := "unknown"
	if info.IsPF {
		ifType = "pf"
	} else if info.IsVF {
		ifType = "vf"
	}

	return []string{
		info.Name,
		ifType,
		info.ParentPF,
		vfNum,
		info.Driver,
		dataSource,
		info.NumaNode,
	}
}

// VFStats holds statistics for a VF obtained via netlink
type VFStats struct {
	VFNum     int
	MAC       net.HardwareAddr
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
	Multicast uint64
	Broadcast uint64
	RxDropped uint64
	TxDropped uint64
}

// getVFStatsFromNetlink retrieves per-VF statistics using netlink (ip -s link show)
func getVFStatsFromNetlink(ifaceName string) ([]VFStats, error) {
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return nil, err
	}

	vfInfos := link.Attrs().Vfs
	if len(vfInfos) == 0 {
		return nil, nil
	}

	var stats []VFStats
	for _, vf := range vfInfos {
		s := VFStats{
			VFNum:     vf.ID,
			MAC:       vf.Mac,
			RxBytes:   vf.RxBytes,
			TxBytes:   vf.TxBytes,
			RxPackets: vf.RxPackets,
			TxPackets: vf.TxPackets,
			Multicast: vf.Multicast,
			Broadcast: vf.Broadcast,
			RxDropped: vf.RxDropped,
			TxDropped: vf.TxDropped,
		}
		stats = append(stats, s)
	}

	return stats, nil
}

var queueStatRe = regexp.MustCompile(`^(tx|rx)_queue_(\d+)_(packets|bytes)$`)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	interfaces, err := discoverSriovInterfaces()
	if err != nil {
		log.Errf("failed to discover SR-IOV interfaces: %s", err)
		return
	}

	log.Debugf("discovered %d SR-IOV interfaces", len(interfaces))

	seenVFStats := make(map[string]bool)

	for _, iface := range interfaces {
		stats, err := getEthtoolStats(iface.Name)
		if err != nil {
			log.Debugf("ethtool stats %s: %s", iface.Name, err)
			continue
		}

		log.Debugf("collected %d stats for %s (PF=%v, VF=%v, driver=%s)",
			len(stats), iface.Name, iface.IsPF, iface.IsVF, iface.Driver)

		if iface.IsVF {
			labels := buildLabels(iface, "direct")
			collectInterfaceStats(ch, labels, stats)
			key := iface.ParentPF + ":" + strconv.Itoa(iface.VFNum)
			seenVFStats[key] = true
		}

		if iface.IsPF {
			labels := buildLabels(iface, "direct")
			collectInterfaceStats(ch, labels, stats)
			// Use netlink to get per-VF stats (works on ice, mlx5_core, etc.)
			collectVFStatsFromNetlink(ch, iface, seenVFStats)
		}
	}
}

func collectInterfaceStats(ch chan<- prometheus.Metric, labels []string, stats map[string]uint64) {
	for statName, value := range stats {
		if match := queueStatRe.FindStringSubmatch(statName); match != nil {
			direction := match[1]
			queueNum := match[2]
			statType := match[3]

			metricName := "sriov_" + direction + "_queue_" + statType + "_total"
			queueLabels := append(append([]string{}, labels...), queueNum)

			desc := prometheus.NewDesc(
				metricName,
				statType+" "+direction+" on queue",
				append(extendedLabels, "queue"),
				nil,
			)

			if config.MetricSets().Has(config.METRICS_PERF) {
				ch <- prometheus.MustNewConstMetric(
					desc, prometheus.CounterValue, float64(value), queueLabels...)
			}
			continue
		}

		if m, ok := metrics[statName]; ok {
			if config.MetricSets().Has(m.Set) {
				ch <- prometheus.MustNewConstMetric(
					m.Desc(), m.ValueType, float64(value), labels...)
			}
		}
	}
}

func emitVFMetric(ch chan<- prometheus.Metric, metricKey string, value uint64, labels []string) {
	m := vfNetlinkMetrics[metricKey]
	ch <- prometheus.MustNewConstMetric(m.Desc(), m.ValueType, float64(value), labels...)
}

func collectVFStatsFromNetlink(ch chan<- prometheus.Metric, pfInfo InterfaceInfo, seenVFStats map[string]bool) {
	vfStats, err := getVFStatsFromNetlink(pfInfo.Name)
	if err != nil {
		log.Debugf("netlink VF stats for %s: %s", pfInfo.Name, err)
		return
	}

	if len(vfStats) == 0 {
		log.Debugf("no VF stats from netlink for %s", pfInfo.Name)
		return
	}

	log.Debugf("collected %d VF stats via netlink for PF %s", len(vfStats), pfInfo.Name)

	for _, vf := range vfStats {
		key := pfInfo.PCIAddr + ":" + strconv.Itoa(vf.VFNum)

		// Skip if we already collected direct stats for this VF
		if seenVFStats[key] {
			log.Debugf("skipping VF %d on %s - already have direct stats", vf.VFNum, pfInfo.Name)
			continue
		}

		// Determine driver - if not in seenVFStats, it's likely vfio-pci
		driver := "vfio-pci"

		vfLabels := []string{
			"",                        // interface (empty for vfio-pci)
			"vf",                      // type
			pfInfo.PCIAddr,            // parent_pf
			strconv.Itoa(vf.VFNum),    // vf_num
			driver,                    // driver
			"netlink",                 // data_source
			pfInfo.NumaNode,           // numa_node
		}

		// Emit counter metrics
		if config.MetricSets().Has(config.METRICS_COUNTERS) {
			emitVFMetric(ch, "rx_bytes", vf.RxBytes, vfLabels)
			emitVFMetric(ch, "tx_bytes", vf.TxBytes, vfLabels)
			emitVFMetric(ch, "rx_packets", vf.RxPackets, vfLabels)
			emitVFMetric(ch, "tx_packets", vf.TxPackets, vfLabels)
			emitVFMetric(ch, "rx_multicast", vf.Multicast, vfLabels)
			emitVFMetric(ch, "rx_broadcast", vf.Broadcast, vfLabels)
		}

		// Emit error metrics
		if config.MetricSets().Has(config.METRICS_ERRORS) {
			emitVFMetric(ch, "rx_dropped", vf.RxDropped, vfLabels)
			emitVFMetric(ch, "tx_dropped", vf.TxDropped, vfLabels)
		}
	}
}
