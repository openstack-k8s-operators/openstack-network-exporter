// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package pmd_perf

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

var commonLabels = []string{"numa", "cpu"}

var metrics = map[string]lib.Metric{
	//  Iterations:         184304299142  (2.35 us/it)
	"Iterations": {
		Name:        "ovs_pmd_total_iterations",
		Description: "Total number of iterations",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  - Used TSC cycles:  997990776491190  ( 99.8 % of total cycles)
	//  - idle iterations:  157612371830  (  7.3 % of used cycles)
	"- idle iterations": {
		Name:        "ovs_pmd_idle_iterations",
		Description: "Number of iterations where zero packets where received",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  - busy iterations:   26691927312  ( 92.7 % of used cycles)
	"- busy iterations": {
		Name:        "ovs_pmd_busy_iterations",
		Description: "Number of iterations spent processing packets",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  - sleep iterations:            0  (  0.0 % of iterations)
	"- sleep iterations": {
		Name:        "ovs_pmd_sleep_iterations",
		Description: "Number of iterations spent sleeping",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  Sleep time (us):               0  (  0 us/iteration avg.)
	"Sleep time (us)": {
		Name:        "ovs_pmd_sleep_microseconds",
		Description: "Total time spent sleeping in microseconds",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  Rx packets:         1553492231479  (3572 Kpps, 595 cycles/pkt)
	"Rx packets": {
		Name:        "ovs_pmd_rx_packets",
		Description: "Number of packets received",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  Datapath passes:    1553492231479  (1.00 passes/pkt)
	"Datapath passes": {
		Name:        "ovs_pmd_datapath_passes",
		Description: "Number of datapath passes",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},
	//  - PHWOL hits:                  0  (  0.0 %)
	"- PHWOL hits": {
		Name:        "ovs_pmd_phwol_hits",
		Description: "phwol hits",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},
	//  - MFEX Opt hits:               0  (  0.0 %)
	"- MFEX Opt hits": {
		Name:        "ovs_pmd_mfex_opt_hits",
		Description: "mfex opt hits",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},

	//  - Simple Match hits:           0  (  0.0 %)
	"- Simple Match hits": {
		Name:        "ovs_pmd_simple_match_hits",
		Description: "simple match hits",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},
	//  - EMC hits:         1548027394828  ( 99.6 %)
	"- EMC hits": {
		Name:        "ovs_pmd_emc_hits",
		Description: "emc hits",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},
	//  - SMC hits:                    0  (  0.0 %)
	"- SMC hits": {
		Name:        "ovs_pmd_smc_hits",
		Description: "smc hits",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},
	//  - Megaflow hits:      5464836649  (  0.4 %, 1.00 subtbl lookups/hit)
	"- Megaflow hits": {
		Name:        "ovs_pmd_megaflow_hits",
		Description: "megaflow hits",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_DEBUG,
	},
	//  - Upcalls:                     2  (  0.0 %, 0.0 us/upcall)
	"- Upcalls": {
		Name:        "ovs_pmd_total_upcalls",
		Description: "Number of upcalls",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  - Lost upcalls:                0  (  0.0 %)
	"- Lost upcalls": {
		Name:        "ovs_pmd_lost_upcalls",
		Description: "Number of lost upcalls",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_ERRORS,
	},
	//  Tx packets:         1553492699287  (3572 Kpps)
	"Tx packets": {
		Name:        "ovs_pmd_tx_packets",
		Description: "Number of Tx packets",
		ValueType:   prometheus.CounterValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
	//  Tx batches:          50017523886  (31.06 pkts/batch)
	"Tx batches": {
		Name:        "ovs_pmd_tx_batches",
		Description: "Number of Tx batches",
		ValueType:   prometheus.GaugeValue,
		Labels:      commonLabels,
		Set:         config.METRICS_PERF,
	},
}
