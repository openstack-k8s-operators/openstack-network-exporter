// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package pmd_rxq

import (
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var isolatedMetric = lib.Metric{
	Name:        "ovs_pmd_cpu_isolated",
	Description: "1 or 0 whether the CPU is excluded from automatic Rxq balancing.",
	ValueType:   prometheus.GaugeValue,
	Labels:      []string{"numa", "cpu"},
	Set:         config.METRICS_PERF,
}

var overheadMetric = lib.Metric{
	Name:        "ovs_pmd_cpu_overhead",
	Description: "Percentage of CPU cycles not related to one specific Rxq.",
	ValueType:   prometheus.GaugeValue,
	Labels:      []string{"numa", "cpu"},
	Set:         config.METRICS_PERF,
}

var ctxtSwitchesMetric = lib.Metric{
	Name:        "ovs_pmd_context_switches",
	Description: "Number of voluntary context switches per PMD thread.",
	ValueType:   prometheus.CounterValue,
	Labels:      []string{"numa", "cpu"},
	Set:         config.METRICS_PERF,
}

var nonVolCtxtSwitchesMetric = lib.Metric{
	Name:        "ovs_pmd_nonvol_context_switches",
	Description: "Number of non-voluntary context switches per PMD thread.",
	ValueType:   prometheus.CounterValue,
	Labels:      []string{"numa", "cpu"},
	Set:         config.METRICS_ERRORS,
}

var usageMetric = lib.Metric{
	Name:        "ovs_pmd_rxq_usage",
	Description: "Percentage of CPU cycles used to process packets from one Rxq.",
	ValueType:   prometheus.GaugeValue,
	Labels:      []string{"numa", "cpu", "interface", "rxq"},
	Set:         config.METRICS_PERF,
}

var enabledMetric = lib.Metric{
	Name:        "ovs_pmd_rxq_enabled",
	Description: "1 or 0 whether a vhost-user Rxq is enabled by a guest.",
	ValueType:   prometheus.GaugeValue,
	Labels:      []string{"numa", "cpu", "interface", "rxq"},
	Set:         config.METRICS_PERF,
}
