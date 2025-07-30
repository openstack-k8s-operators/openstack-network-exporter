// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2025 Yatin Karel

package ovnnorthd

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

// Coverage metrics for OVN northd
var coverageMetrics = map[string]lib.Metric{
	"pstream_open": {
		Name:        "ovn_northd_pstream_open_total",
		Description: "Specifies the number of time passive connections were opened for the remote peer to connect",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"stream_open": {
		Name:        "ovn_northd_stream_open_total",
		Description: "Specifies the number of attempts to connect to a remote peer",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"txn_success": {
		Name:        "ovn_northd_txn_success_total",
		Description: "Specifies the number of times the OVSDB transaction has successfully completed",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"txn_error": {
		Name:        "ovn_northd_txn_error_total",
		Description: "Specifies the number of times the OVSDB transaction has errored out",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_uncommitted": {
		Name:        "ovn_northd_txn_uncommitted_total",
		Description: "Specifies the number of times the OVSDB transaction were uncommitted",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_unchanged": {
		Name:        "ovn_northd_txn_unchanged_total",
		Description: "Specifies the number of times the OVSDB transaction resulted in no change to the database",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"txn_incomplete": {
		Name:        "ovn_northd_txn_incomplete_total",
		Description: "Specifies the number of times the OVSDB transaction did not complete and the client had to re-try",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_aborted": {
		Name:        "ovn_northd_txn_aborted_total",
		Description: "Specifies the number of times the OVSDB transaction has been aborted",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"txn_try_again": {
		Name:        "ovn_northd_txn_try_again_total",
		Description: "Specifies the number of times the OVSDB transaction failed and the client had to re-try",
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
}

// Status metric for OVN northd
var statusMetric = lib.Metric{
	Name:        "ovn_northd_status",
	Description: "Status of OVN northd (0=standby, 1=active, 2=paused)",
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}
