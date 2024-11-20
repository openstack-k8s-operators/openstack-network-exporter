// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package datapath

import (
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var commonLabels = []string{"type", "name"}

var flowsMetric = lib.Metric{
	Name:        "ovs_datapath_flows_total",
	Description: "The number of datapath flows.",
	ValueType:   prometheus.GaugeValue,
	Labels:      commonLabels,
	Set:         config.METRICS_PERF,
}

var hitsMetric = lib.Metric{
	Name:        "ovs_datapath_lookup_hits_total",
	Description: "The total number of lookups in the datapath flow cache.",
	ValueType:   prometheus.GaugeValue,
	Labels:      commonLabels,
	Set:         config.METRICS_PERF,
}

var missedMetric = lib.Metric{
	Name:        "ovs_datapath_lookup_missed_total",
	Description: "Number of missed lookups in the datapath flow cache.",
	ValueType:   prometheus.GaugeValue,
	Labels:      commonLabels,
	Set:         config.METRICS_PERF,
}

var lostMetric = lib.Metric{
	Name:        "ovs_datapath_lookup_lost_total",
	Description: "Number of lost lookups in the datapath flow cache.",
	ValueType:   prometheus.GaugeValue,
	Labels:      commonLabels,
	Set:         config.METRICS_PERF,
}
