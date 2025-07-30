// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2025 Yatin Karel

package ovnnorthd

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

type Collector struct{}

func (Collector) Name() string {
	return "ovnnorthd"
}

func (Collector) Metrics() []lib.Metric {
	var res []lib.Metric
	for _, m := range coverageMetrics {
		res = append(res, m)
	}
	res = append(res, statusMetric)
	return res
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

func makeMetric(name, value string) prometheus.Metric {
	m, ok := coverageMetrics[name]
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

// "pstream_open                 0.0/sec     0.000/sec        0.0000/sec   total: 1"
var coverageRe = regexp.MustCompile(`^(\w+)\s+.*\s+total: (\d+)$`)

func collectCoverageMetrics(ch chan<- prometheus.Metric) {
	buf := appctl.OvnNorthd("coverage/show")
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

func collectStatusMetric(ch chan<- prometheus.Metric) {
	if !config.MetricSets().Has(statusMetric.Set) {
		return
	}

	buf := appctl.OvnNorthd("status")
	if buf == "" {
		return
	}

	var value float64
	status := strings.TrimSpace(buf)

	// Parse status from output like "Status: active"
	if strings.Contains(status, "Status:") {
		parts := strings.Split(status, ":")
		if len(parts) == 2 {
			statusValue := strings.TrimSpace(parts[1])
			switch statusValue {
			case "active":
				value = 1.0
			case "standby":
				value = 0.0
			case "paused":
				value = 2.0
			default:
				log.Warningf("Unknown northd status: %s", statusValue)
				return
			}
		} else {
			log.Warningf("Unexpected status format: %s", status)
			return
		}
	} else {
		log.Warningf("Status output does not contain 'Status:' prefix: %s", status)
		return
	}

	ch <- prometheus.MustNewConstMetric(statusMetric.Desc(), statusMetric.ValueType, value)
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	// Collect coverage metrics
	collectCoverageMetrics(ch)

	// Collect status metric
	collectStatusMetric(ch)
}
