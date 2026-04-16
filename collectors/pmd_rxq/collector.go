// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package pmd_rxq

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/appctl"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "pmd-rxq"
}

func (Collector) Metrics() []lib.Metric {
	return []lib.Metric{isolatedMetric, overheadMetric, ctxtSwitchesMetric, nonVolCtxtSwitchesMetric, enabledMetric, usageMetric}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

var (
	// "pmd thread numa_id 0 core_id 39:"
	pmdThreadRe = regexp.MustCompile(`^pmd thread numa_id (\d+) core_id (\d+):$`)
	// "  isolated : true"
	isolatedRe = regexp.MustCompile(`^\s*isolated\s*:\s*(true|false)$`)
	// "  port: vhu-30            queue-id:  0 (enabled)   pmd usage: 43 %"
	rxqUsageRe = regexp.MustCompile(
		`^\s*port:\s*(\S+)\s+queue-id:\s*(\d+)\s+\((enabled|disabled)\)\s+pmd usage:\s*([\d\.]+)\s*%$`)
	// "  overhead: 11 %"
	overheadRe = regexp.MustCompile(`^\s*overhead\s*:\s*([\d\.]+)\s*%$`)
)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	stats := getVswitchdPmdStat()

	buf := appctl.OvsVSwitchd("dpif-netdev/pmd-rxq-show")
	if buf == "" {
		return
	}

	numa := ""
	cpu := ""

	scanner := bufio.NewScanner(strings.NewReader(buf))
	for scanner.Scan() {
		line := scanner.Text()

		if numa != "" && cpu != "" && config.MetricSets().Has(config.METRICS_PERF) {
			var val float64
			var err error

			if m := isolatedRe.FindStringSubmatch(line); m != nil {
				if m[1] == "true" {
					val = 1
				}
				ch <- prometheus.MustNewConstMetric(
					isolatedMetric.Desc(), isolatedMetric.ValueType,
					val, numa, cpu)
				continue
			} else if m := rxqUsageRe.FindStringSubmatch(line); m != nil {
				if m[3] == "enabled" {
					val = 1
				}
				ch <- prometheus.MustNewConstMetric(
					enabledMetric.Desc(), enabledMetric.ValueType,
					val, numa, cpu, m[1], m[2])

				val, err = strconv.ParseFloat(m[4], 64)
				if err != nil {
					log.Errf("pmd usage: %s", err)
					continue
				}
				ch <- prometheus.MustNewConstMetric(
					usageMetric.Desc(), usageMetric.ValueType,
					val, numa, cpu, m[1], m[2])
				continue
			} else if m := overheadRe.FindStringSubmatch(line); m != nil {
				val, err = strconv.ParseFloat(m[1], 64)
				if err != nil {
					log.Errf("overhead: %s", err)
					continue
				}
				ch <- prometheus.MustNewConstMetric(
					overheadMetric.Desc(), overheadMetric.ValueType,
					val, numa, cpu)
				continue
			}
		}

		match := pmdThreadRe.FindStringSubmatch(line)
		if match != nil {
			numa = match[1]
			cpu = match[2]
			c, _ := strconv.ParseUint(cpu, 10, 64)
			stat, ok := stats[c]
			if !ok {
				continue
			}
			if config.MetricSets().Has(ctxtSwitchesMetric.Set) {
				ch <- prometheus.MustNewConstMetric(
					ctxtSwitchesMetric.Desc(),
					ctxtSwitchesMetric.ValueType,
					float64(stat.ctxSwitches), numa, cpu)
			}
			if config.MetricSets().Has(nonVolCtxtSwitchesMetric.Set) {
				ch <- prometheus.MustNewConstMetric(
					nonVolCtxtSwitchesMetric.Desc(),
					nonVolCtxtSwitchesMetric.ValueType,
					float64(stat.nonVolCtxSwitches), numa, cpu)
			}
		}
	}
}

type pmdstat struct {
	name              string
	cpuAffinity       uint64
	numaAffinity      uint64
	ctxSwitches       uint64
	nonVolCtxSwitches uint64
}

var notPmdErr = errors.New("not a pmd thread")

func parseStatus(path string) (pmdstat, error) {
	var stat pmdstat
	f, err := os.Open(path)
	if err != nil {
		return stat, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		tokens := strings.Fields(scanner.Text())
		if len(tokens) != 2 {
			continue
		}
		name, value := tokens[0], tokens[1]

		switch name {
		case "Name:":
			if !strings.HasPrefix(value, "pmd-c") {
				return stat, notPmdErr
			}
			stat.name = value
		case "Cpus_allowed_list:":
			stat.cpuAffinity, _ = strconv.ParseUint(value, 10, 64)
		case "Mems_allowed_list:":
			stat.numaAffinity, _ = strconv.ParseUint(value, 10, 64)
		case "voluntary_ctxt_switches:":
			stat.ctxSwitches, _ = strconv.ParseUint(value, 10, 64)
		case "nonvoluntary_ctxt_switches:":
			stat.nonVolCtxSwitches, _ = strconv.ParseUint(value, 10, 64)
		}
	}
	if scanner.Err() != nil {
		return stat, scanner.Err()
	}

	return stat, nil
}

func getVswitchdPmdStat() map[uint64]pmdstat {
	pidfile := filepath.Join(config.OvsRundir(), "ovs-vswitchd.pid")
	f, err := os.Open(pidfile)
	if err != nil {
		log.Errf("open(%s): %s", pidfile, err)
		return nil
	}
	defer f.Close()
	buf, err := io.ReadAll(f)
	if err != nil {
		log.Errf("read(%s): %s", pidfile, err)
		return nil
	}
	tasks := filepath.Join(config.OvsProcdir(), strings.TrimSpace(string(buf)), "task")
	entries, err := os.ReadDir(tasks)
	if err != nil {
		log.Errf("readdir(%s): %s", tasks, err)
		return nil
	}

	stats := make(map[uint64]pmdstat)

	for _, e := range entries {
		if e.IsDir() {
			stat, err := parseStatus(filepath.Join(tasks, e.Name(), "status"))
			if err != nil {
				if !errors.Is(err, notPmdErr) {
					log.Errf("status(%s): %s", e.Name(), err)
				}
				continue
			}
			stats[stat.cpuAffinity] = stat
		}
	}

	return stats
}
