// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package collectors

import "github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"

// All supported collectors. Please keep alpha sorted.
var collectors = []lib.Collector{}

func Collectors() []lib.Collector {
	return collectors
}
