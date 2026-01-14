// SPDX-License-Identifier: Apache-2.0

package sriov

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var extendedLabels = []string{
	"interface",
	"type",
	"parent_pf",
	"vf_num",
	"driver",
	"data_source",
	"numa_node",
}

var metrics = map[string]lib.Metric{
	"rx_bytes": {
		Name:        "sriov_rx_bytes_total",
		Description: "Total number of received bytes on SR-IOV interface",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_bytes": {
		Name:        "sriov_tx_bytes_total",
		Description: "Total number of transmitted bytes on SR-IOV interface",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_unicast": {
		Name:        "sriov_rx_unicast_packets_total",
		Description: "Total number of received unicast packets",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_unicast": {
		Name:        "sriov_tx_unicast_packets_total",
		Description: "Total number of transmitted unicast packets",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_multicast": {
		Name:        "sriov_rx_multicast_packets_total",
		Description: "Total number of received multicast packets",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_multicast": {
		Name:        "sriov_tx_multicast_packets_total",
		Description: "Total number of transmitted multicast packets",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_broadcast": {
		Name:        "sriov_rx_broadcast_packets_total",
		Description: "Total number of received broadcast packets",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_broadcast": {
		Name:        "sriov_tx_broadcast_packets_total",
		Description: "Total number of transmitted broadcast packets",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_dropped": {
		Name:        "sriov_rx_dropped_total",
		Description: "Total number of received packets dropped",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_errors": {
		Name:        "sriov_tx_errors_total",
		Description: "Total number of transmit errors",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_alloc_fail": {
		Name:        "sriov_rx_alloc_fail_total",
		Description: "Total number of RX buffer allocation failures",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_pg_alloc_fail": {
		Name:        "sriov_rx_pg_alloc_fail_total",
		Description: "Total number of RX page allocation failures",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_linearize": {
		Name:        "sriov_tx_linearize_total",
		Description: "Number of times TX linearization was needed",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_busy": {
		Name:        "sriov_tx_busy_total",
		Description: "Number of times TX queue was busy",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_restart": {
		Name:        "sriov_tx_restart_total",
		Description: "Number of TX queue restarts",
		Labels:      extendedLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
}
