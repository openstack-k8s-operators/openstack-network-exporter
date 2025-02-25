// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package memory

import (
	"regexp"
	"strconv"

	"github.com/openstack-k8s-operators/openstack-network-exporter/appctl"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "memory"
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

var memoryCountRe = regexp.MustCompile(`(\w+):(\d+)`)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	buf := appctl.OvsVSwitchd("memory/show")
	if buf == "" {
		return
	}

	for _, match := range memoryCountRe.FindAllStringSubmatch(buf, -1) {
		m, ok := metrics[match[1]]
		if !ok {
			continue
		}
		if !config.MetricSets().Has(m.Set) {
			continue
		}
		val, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			log.Errf("%s: %s: %s", match[1], match[2], err)
			continue
		}
		ch <- prometheus.MustNewConstMetric(m.Desc(), m.ValueType, val)
	}
}
