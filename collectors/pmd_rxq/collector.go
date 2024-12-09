// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package pmd_rxq

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/appctl"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "pmd-rxq"
}

func (Collector) Metrics() []lib.Metric {
	return []lib.Metric{isolatedMetric, overheadMetric, enabledMetric, usageMetric}
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
	if !config.MetricSets().Has(config.METRICS_PERF) {
		return
	}

	buf := appctl.OvsVSwitchd("dpif-netdev/pmd-rxq-show")
	if buf == "" {
		return
	}

	numa := ""
	cpu := ""

	scanner := bufio.NewScanner(strings.NewReader(buf))
	for scanner.Scan() {
		line := scanner.Text()

		if numa != "" && cpu != "" {
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
		}
	}
}
