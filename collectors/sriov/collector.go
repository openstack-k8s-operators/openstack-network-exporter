// SPDX-License-Identifier: Apache-2.0

package sriov

import (
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

var vfStatPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^vf_(\w+)\[(\d+)\]$`), // ixgbe: vf_rx_packets[0]
	regexp.MustCompile(`^vf-(\d+)-(\w+)$`),    // i40e: vf-0-rx_packets
	regexp.MustCompile(`^vf_(\d+)_(\w+)$`),    // ice: vf_0_rx_packets
}

func parseVFStat(statName string) (vfNum int, statType string, ok bool) {
	if match := vfStatPatterns[0].FindStringSubmatch(statName); match != nil {
		statType = match[1]
		vfNum, _ = strconv.Atoi(match[2])
		return vfNum, statType, true
	}

	if match := vfStatPatterns[1].FindStringSubmatch(statName); match != nil {
		vfNum, _ = strconv.Atoi(match[1])
		statType = match[2]
		return vfNum, statType, true
	}

	if match := vfStatPatterns[2].FindStringSubmatch(statName); match != nil {
		vfNum, _ = strconv.Atoi(match[1])
		statType = match[2]
		return vfNum, statType, true
	}

	return 0, "", false
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
			collectPerVFStatsFromPF(ch, iface, stats, seenVFStats)
		}
	}
}

func collectInterfaceStats(ch chan<- prometheus.Metric, labels []string, stats map[string]uint64) {
	for statName, value := range stats {
		if _, _, ok := parseVFStat(statName); ok {
			continue
		}

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

func collectPerVFStatsFromPF(ch chan<- prometheus.Metric, pfInfo InterfaceInfo, stats map[string]uint64, seenVFStats map[string]bool) {
	for statName, value := range stats {
		vfNum, statType, ok := parseVFStat(statName)
		if !ok {
			continue
		}

		key := pfInfo.PCIAddr + ":" + strconv.Itoa(vfNum)
		if seenVFStats[key] {
			continue
		}

		metricName := "sriov_vf_" + statType + "_total"

		vfLabels := []string{
			"",
			"vf",
			pfInfo.PCIAddr,
			strconv.Itoa(vfNum),
			"vfio-pci",
			"pf_aggregate",
		}

		desc := prometheus.NewDesc(
			metricName,
			"VF "+statType+" collected from PF",
			extendedLabels,
			nil,
		)

		var metricSet config.MetricSet
		if strings.Contains(statType, "error") || strings.Contains(statType, "drop") {
			metricSet = config.METRICS_ERRORS
		} else {
			metricSet = config.METRICS_COUNTERS
		}

		if config.MetricSets().Has(metricSet) {
			ch <- prometheus.MustNewConstMetric(
				desc, prometheus.CounterValue, float64(value), vfLabels...)
		}
	}
}
