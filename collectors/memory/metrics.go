// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package memory

import (
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

// handlers:29 idl-cells-Open_vSwitch:7351 ports:114 revalidators:11 rules:190 udpif keys:76
var metrics = map[string]lib.Metric{
	"handlers": {
		Name:        "ovs_memory_handlers_total",
		Description: "Total number of handler threads.",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_PERF,
	},
	"ports": {
		Name:        "ovs_memory_ports_total",
		Description: "Total number of ports.",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_PERF,
	},
	"revalidators": {
		Name:        "ovs_memory_revalidators_total",
		Description: "Total number of revalidator threads.",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_PERF,
	},
	"rules": {
		Name:        "ovs_memory_rules_total",
		Description: "Total number of rules.",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_PERF,
	},
	"keys": {
		Name:        "ovs_memory_keys_total",
		Description: "Total number of udpif keys.",
		ValueType:   prometheus.GaugeValue,
		Set:         config.METRICS_PERF,
	},
}
