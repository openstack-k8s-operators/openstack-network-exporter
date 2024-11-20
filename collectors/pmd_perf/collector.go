// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package pmd_perf

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

func makeMetric(numa, cpu, name, value string) prometheus.Metric {
	m, ok := metrics[name]
	if !ok {
		return nil
	}
	if !config.MetricSets().Has(m.Set) {
		return nil
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Errf("%s: %s: %s", name, value, err)
		return nil
	}

	return prometheus.MustNewConstMetric(m.Desc(), m.ValueType, val, numa, cpu)
}

type Collector struct{}

func (Collector) Name() string {
	return "pmd-perf"
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

var (
	// "pmd thread numa_id 0 core_id 39:"
	pmdThreadRe = regexp.MustCompile(`(?m)^pmd thread numa_id (\d+) core_id (\d+):$`)
	// "  - Used TSC cycles:  997990776491190  ( 99.8 % of total cycles)"
	pmdPerfStatRe = regexp.MustCompile(`(?m)^\s*([^:]+):\s+(\d+)\s*(.*)$`)
)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	buf := appctl.Call("dpif-netdev/pmd-perf-show")
	if buf == "" {
		return
	}

	numa := ""
	cpu := ""

	scanner := bufio.NewScanner(strings.NewReader(buf))
	for scanner.Scan() {
		line := scanner.Text()

		if numa != "" && cpu != "" {
			match := pmdPerfStatRe.FindStringSubmatch(line)
			if match != nil {
				metric := makeMetric(numa, cpu, match[1], match[2])
				if metric != nil {
					ch <- metric
				}
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
