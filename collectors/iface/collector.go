// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package iface

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
	return "interface"
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
	var bridges []ovs.Bridge
	var ports []ovs.Port
	var ifaces []ovs.Interface

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := ovsdb.List(ctx, &bridges)
	if err != nil {
		log.Errf("db.List(Bridge): %s", err)
		return
	}
	err = ovsdb.List(ctx, &ports)
	if err != nil {
		log.Errf("db.List(Port): %s", err)
		return
	}
	err = ovsdb.List(ctx, &ifaces)
	if err != nil {
		log.Errf("db.List(Interface): %s", err)
		return
	}

	portBridge := make(map[string]string)
	ifacePort := make(map[string]string)

	for _, br := range bridges {
		for _, p := range br.Ports {
			portBridge[p] = br.Name
		}
	}
	for _, p := range ports {
		for _, i := range p.Interfaces {
			ifacePort[i] = p.Name
			portBridge[ifacePort[i]] = portBridge[p.UUID]
		}
	}
	for _, i := range ifaces {
		port, ok := ifacePort[i.UUID]
		if !ok {
			continue
		}
		bridge, ok := portBridge[port]
		if !ok {
			continue
		}
		labels := []string{bridge, port, i.Name, i.Type}

		for _, m := range metrics {
			if config.MetricSets().Has(m.Set) {
				ch <- prometheus.MustNewConstMetric(m.Desc(),
					m.ValueType, m.GetValue(&i), labels...)
			}
		}
	}
}
