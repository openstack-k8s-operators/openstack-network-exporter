// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package bridge

import (
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/openflow"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/ovsdb/ovs"
	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	lib.Metric
	GetValue func(br *ovs.Bridge) float64
}

var labels = []string{"bridge", "datapath_type"}

var metrics = []Metric{
	{
		lib.Metric{
			Name:        "ovs_bridge_port_count",
			Description: "The number of ports in a bridge.",
			Labels:      labels,
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(br *ovs.Bridge) float64 {
			return float64(len(br.Ports))
		},
	},
	{
		lib.Metric{
			Name:        "ovs_bridge_flow_count",
			Description: "The number of openflow rules configured on a bridge.",
			Labels:      labels,
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(br *ovs.Bridge) float64 {
			bs := openflow.BridgeStats{Name: br.Name}
			err := bs.GetAggregateStats()
			if err != nil {
				log.Errf("bs.GetAggregateStats: %s", err)
				return 0
			}
			return float64(bs.Flows)
		},
	},
}
