// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package bridge

import (
	"context"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovs"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "bridge"
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

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var bridges []ovs.Bridge
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := ovsdb.List(ctx, &bridges)
	if err != nil {
		log.Errf("db.List(Bridge): %s", err)
		return
	}

	for _, br := range bridges {
		labels := []string{br.Name, br.DatapathType}

		for _, m := range metrics {
			if config.MetricSets().Has(m.Set) {
				ch <- prometheus.MustNewConstMetric(m.Desc(),
					m.ValueType, m.GetValue(&br), labels...)
			}
		}
	}
}
