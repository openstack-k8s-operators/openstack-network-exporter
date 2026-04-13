// SPDX-License-Identifier: Apache-2.0

package netvf

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var trafficLabels = []string{"device", "vf", "pci_address", "numa_node"}

var infoLabels = []string{"device", "vf", "mac", "vlan", "link_state", "spoof_check", "trust", "pci_address", "numa_node"}

var infoMetric = lib.Metric{
	Name:        "net_vf_info",
	Description: "Virtual Function configuration information.",
	Labels:      infoLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var counterMetrics = map[string]*lib.Metric{
	"rx_bytes": {
		Name:        "net_vf_receive_bytes_total",
		Description: "Number of received bytes by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_bytes": {
		Name:        "net_vf_transmit_bytes_total",
		Description: "Number of transmitted bytes by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_packets": {
		Name:        "net_vf_receive_packets_total",
		Description: "Number of received packets by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_packets": {
		Name:        "net_vf_transmit_packets_total",
		Description: "Number of transmitted packets by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"broadcast": {
		Name:        "net_vf_broadcast_packets_total",
		Description: "Number of broadcast packets received by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"multicast": {
		Name:        "net_vf_multicast_packets_total",
		Description: "Number of multicast packets received by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_dropped": {
		Name:        "net_vf_receive_dropped_total",
		Description: "Number of dropped received packets by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_dropped": {
		Name:        "net_vf_transmit_dropped_total",
		Description: "Number of dropped transmitted packets by the VF.",
		Labels:      trafficLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
}
