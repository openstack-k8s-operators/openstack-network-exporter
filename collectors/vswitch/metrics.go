// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package vswitch

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovs"
	"github.com/prometheus/client_golang/prometheus"
)

type Metric struct {
	lib.Metric
	GetValue func(vs *ovs.OpenvSwitch) (value float64, labels []string)
}

var metrics = []Metric{
	{
		lib.Metric{
			Name:        "ovs_build_info",
			Description: "Version and library from which OVS binaries were built.",
			Labels:      []string{"ovs_version", "dpdk_version", "db_version"},
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(vs *ovs.OpenvSwitch) (float64, []string) {
			var ovs, dpdk, db string
			if vs.OVSVersion != nil {
				ovs = *vs.OVSVersion
			}
			if vs.DpdkVersion != nil {
				dpdk = *vs.DpdkVersion
			}
			if vs.DbVersion != nil {
				db = *vs.DbVersion
			}
			return 1, []string{ovs, dpdk, db}
		},
	},
	{
		lib.Metric{
			Name:        "ovs_dpdk_initialized",
			Description: "Has the DPDK subsystem been initialized.",
			ValueType:   prometheus.GaugeValue,
			Set:         config.METRICS_BASE,
		},
		func(vs *ovs.OpenvSwitch) (float64, []string) {
			if vs.DpdkInitialized {
				return 1, nil
			}
			return 0, nil
		},
	},
}
