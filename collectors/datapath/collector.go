// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package datapath

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/appctl"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "datapath"
}

func (Collector) Metrics() []lib.Metric {
	return []lib.Metric{flowsMetric, hitsMetric, missedMetric, lostMetric}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

var (
	// "netdev@ovs-netdev:"
	datapathRe = regexp.MustCompile(`^([\w-]+)@([\w-]+):$`)
	// "  lookups: hit:57723911358512 missed:132 lost:20"
	lookupsRe = regexp.MustCompile(`^  lookups:\s*hit:\s*(\d+)\s+missed:\s*(\d+)\s+lost:(\d+)$`)
	// "  flows: 76"
	flowsRe = regexp.MustCompile(`^  flows:\s*(\d+)$`)
)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	if !config.MetricSets().Has(config.METRICS_PERF) {
		return
	}

	buf := appctl.OvsVSwitchd("dpctl/show")
	if buf == "" {
		return
	}

	dptype := ""
	dpname := ""

	scanner := bufio.NewScanner(strings.NewReader(buf))
	for scanner.Scan() {
		line := scanner.Text()

		if dptype != "" && dpname != "" {
			if m := lookupsRe.FindStringSubmatch(line); m != nil {
				val, _ := strconv.ParseFloat(m[1], 64)
				ch <- prometheus.MustNewConstMetric(
					hitsMetric.Desc(), hitsMetric.ValueType,
					val, dptype, dpname)
				val, _ = strconv.ParseFloat(m[2], 64)
				ch <- prometheus.MustNewConstMetric(
					missedMetric.Desc(), missedMetric.ValueType,
					val, dptype, dpname)
				val, _ = strconv.ParseFloat(m[3], 64)
				ch <- prometheus.MustNewConstMetric(
					lostMetric.Desc(), lostMetric.ValueType,
					val, dptype, dpname)
				continue
			} else if m := flowsRe.FindStringSubmatch(line); m != nil {
				val, _ := strconv.ParseFloat(m[1], 64)
				ch <- prometheus.MustNewConstMetric(
					flowsMetric.Desc(), flowsMetric.ValueType,
					val, dptype, dpname)
				continue
			}
		}

		match := datapathRe.FindStringSubmatch(line)
		if match != nil {
			dptype = match[1]
			dpname = match[2]
		}
	}
}
