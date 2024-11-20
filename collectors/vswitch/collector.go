// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package vswitch

import (
	"context"
	"time"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/ovsdb"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/ovsdb/ovs"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "vswitch"
}

func (Collector) Metrics() []lib.Metric {
	var res []lib.Metric
	for _, m := range metrics {
		res = append(res, m.Metric)
	}
	return res
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	if !config.MetricSets().Has(config.METRICS_BASE) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var vswitch ovs.OpenvSwitch
	err := ovsdb.Get(ctx, &vswitch)
	if err != nil {
		log.Errf("OvsdbGet(vswitch): %s", err)
		return
	}

	for _, m := range metrics {
		value, labels := m.GetValue(&vswitch)
		ch <- prometheus.MustNewConstMetric(m.Desc(), m.ValueType, value, labels...)
	}
}
