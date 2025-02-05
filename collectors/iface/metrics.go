package iface

import (
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/ovsdb/ovs"
	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	lib.Metric
	GetValue func(iface *ovs.Interface) float64
}

var commonLabels = []string{"bridge", "port", "interface", "type"}

var metrics = []Metric{
	{
		lib.Metric{
			Name:        "ovs_interface_admin_state",
			Description: "The administrative state of the interface. Possible values are: up(1), down(0) or unknown(-1).",
			Labels:      commonLabels,
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(iface *ovs.Interface) float64 {
			if iface.AdminState != nil {
				switch *iface.AdminState {
				case "up":
					return 1
				case "down":
					return 0
				}
			}
			return -1
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_link_state",
			Description: "The link state of the interface. Possible values are: up(1), down(0) or unknown(-1).",
			Labels:      commonLabels,
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(iface *ovs.Interface) float64 {
			if iface.LinkState != nil {
				switch *iface.LinkState {
				case "up":
					return 1
				case "down":
					return 0
				}
			}
			return -1
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_mtu_bytes",
			Description: "Maximum transmission unit size in bytes.",
			Labels:      commonLabels,
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(iface *ovs.Interface) float64 {
			if iface.MTU != nil {
				return float64(*iface.MTU)
			}
			return 0
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_link_speed_bps",
			Description: "Link speed in bits per second.",
			Labels:      commonLabels,
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(iface *ovs.Interface) float64 {
			if iface.LinkSpeed != nil {
				return float64(*iface.LinkSpeed)
			}
			return 0
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_link_resets",
			Description: "The number of times the link_state has changed.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_COUNTERS,
		},
		func(iface *ovs.Interface) float64 {
			if iface.LinkResets != nil {
				return float64(*iface.LinkResets)
			}
			return 0
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_rx_packets",
			Description: "Number of received packets.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_COUNTERS,
		},
		func(iface *ovs.Interface) float64 {
			return float64(iface.Statistics["rx_packets"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_rx_bytes",
			Description: "Number of received bytes.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_COUNTERS,
		},
		func(iface *ovs.Interface) float64 {
			return float64(iface.Statistics["rx_bytes"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_rx_errors",
			Description: "Number of invalid packets received.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_ERRORS,
		},
		func(iface *ovs.Interface) float64 {
			return float64(iface.Statistics["rx_errors"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_rx_dropped",
			Description: "Number of packets dropped by hardware because the Rx ring was full.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_ERRORS,
		},
		func(iface *ovs.Interface) float64 {
			if x := iface.Statistics["rx_missed_errors"]; x > 0 {
				return float64(x)
			}
			return float64(iface.Statistics["rx_dropped"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_tx_packets",
			Description: "Number of transmitted packets.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_COUNTERS,
		},
		func(iface *ovs.Interface) float64 {
			return float64(iface.Statistics["tx_packets"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_tx_bytes",
			Description: "Number of transmitted bytes.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_COUNTERS,
		},
		func(iface *ovs.Interface) float64 {
			return float64(iface.Statistics["tx_bytes"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_tx_errors",
			Description: "Number of errors while transmitting packets.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_ERRORS,
		},
		func(iface *ovs.Interface) float64 {
			if x := iface.Statistics["ovs_tx_failure_drops"]; x > 0 {
				return float64(x)
			}
			return float64(iface.Statistics["tx_errors"])
		},
	},
	{
		lib.Metric{
			Name:        "ovs_interface_tx_retries",
			Description: "Number of times a packet transmission was retried because the destination queue was full.",
			Labels:      commonLabels,
			ValueType:   prometheus.CounterValue,
			Set:         config.METRICS_ERRORS,
		},
		func(iface *ovs.Interface) float64 {
			return float64(iface.Statistics["ovs_tx_retries"])
		},
	},
}
