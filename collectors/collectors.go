// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package collectors

import (
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/bridge"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/coverage"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/datapath"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/iface"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/memory"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/pmd_perf"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/pmd_rxq"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/vswitch"
)

// All supported collectors. Please keep alpha sorted.
var collectors = []lib.Collector{
	new(bridge.Collector),
	new(coverage.Collector),
	new(datapath.Collector),
	new(iface.Collector),
	new(memory.Collector),
	new(pmd_perf.Collector),
	new(pmd_rxq.Collector),
	new(vswitch.Collector),
}

func Collectors() []lib.Collector {
	return collectors
}
