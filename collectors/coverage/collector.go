// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package coverage

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/appctl"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

func makeMetric(name, value string) prometheus.Metric {
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

	return prometheus.MustNewConstMetric(m.Desc(), m.ValueType, val)
}

type Collector struct{}

func (Collector) Name() string {
	return "coverage"
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

// "netdev_sent       967178.4/sec 966510.667/sec   880482.1181/sec   total: 21235468562413"
var coverageRe = regexp.MustCompile(`^(\w+)\s+.*\s+total: (\d+)$`)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	buf := appctl.OvsVSwitchd("coverage/show")
	if buf == "" {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(buf))
	for scanner.Scan() {
		line := scanner.Text()

		match := coverageRe.FindStringSubmatch(line)
		if match != nil {
			metric := makeMetric(match[1], match[2])
			if metric != nil {
				ch <- metric
			}
		}
	}
}
