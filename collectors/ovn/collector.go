// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Miguel Lavalle

package ovn

import (
	"bufio"
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/appctl"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/ovsdb"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/ovsdb/ovs"
	"github.com/prometheus/client_golang/prometheus"
)

func collectopenvSwitch(externaIds map[string]string, ch chan<- prometheus.Metric) {
	for name, metric := range openvSwitch {
		value, ok := externaIds[name]
		if !ok {
			continue
		}
		if !config.MetricSets().Has(metric.Set) {
			continue
		}

		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Errf("%s: %s: %s", name, value, err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(metric.Desc(), metric.ValueType, val)
	}
}

func collectopenvSwitchBoolean(externaIds map[string]string, ch chan<- prometheus.Metric) {
	for name, metric := range openvSwitchBoolean {
		value, ok := externaIds[name]
		if !ok {
			continue
		}
		if !config.MetricSets().Has(metric.Set) {
			continue
		}

		val := 1.0
		if strings.ToLower(value) != "true" {
			val = 0.0
		}

		ch <- prometheus.MustNewConstMetric(metric.Desc(), metric.ValueType, val)
	}
}

func parse_mappings(bridgeMappings string) map[string]string {
	// This function is based on the one that Neutron uses to parse bridge
	// mappings:
	// https://github.com/openstack/neutron-lib/blob/ef72d4cd6e2e74a0452dcea916613357c3627a22/neutron_lib/utils/helpers.py
	mappings := make(map[string]string)
	mappingsSlice := strings.Split(bridgeMappings, ",")
	for _, m := range mappingsSlice {
		mSplit := strings.Split(m, ":")
		mappings[strings.TrimSpace(mSplit[0])] = strings.TrimSpace(mSplit[1])
	}
	return mappings
}

func collectopenvSwitchLabels(externaIds map[string]string, ch chan<- prometheus.Metric) {
	value := 1.0
	for name, metric := range openvSwitchLabels {
		label, ok := externaIds[name]
		if !ok {
			continue
		}
		if !config.MetricSets().Has(metric.Set) {
			continue
		}

		ch <- prometheus.MustNewConstMetric(metric.Desc(), metric.ValueType, value, []string{label}...)
	}

	extIds, ok := externaIds["ovn-bridge-mappings"]
	if !ok {
		return
	}
	if !config.MetricSets().Has(bridgeMappings.Set) {
		return
	}
	for network, bridge := range parse_mappings(extIds) {
		labels := []string{network, bridge}
		ch <- prometheus.MustNewConstMetric(bridgeMappings.Desc(), bridgeMappings.ValueType, value, labels...)
	}
}

func makeMetric(name, value string) prometheus.Metric {
	m, ok := ovnController[name]
	if !ok {
		return nil
	}
	if !config.MetricSets().Has(m.Set) {
		return nil
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Errf("%s: %s: %s", name, value, err)
		return nil
	}

	return prometheus.MustNewConstMetric(m.Desc(), m.ValueType, val)
}

const (
	packetInDrop           = "packet_in_drop"
	dropBufferedPacketsMap = "pinctrl_drop_buffered_packets_map"
	dropControllerEvent    = "pinctrl_drop_controller_event"
)

func isPacketInDropComponent(name string) bool {
	if name == dropBufferedPacketsMap || name == dropControllerEvent {
		return true
	}
	return false
}

// "vconn_sent                 0.0/sec     0.083/sec        0.0767/sec   total: 131870"
var coverageRe = regexp.MustCompile(`^(\w+)\s+.*\s+total: (\d+)$`)

func collectCoverageMetrics(ch chan<- prometheus.Metric) {

	packetInDropComponets := map[string]string{
		dropBufferedPacketsMap: "",
		dropControllerEvent:    "",
	}

	buf := appctl.OvnController("coverage/show")
	if buf == "" {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(buf))
	for scanner.Scan() {
		line := scanner.Text()

		match := coverageRe.FindStringSubmatch(line)
		if match != nil {
			if isPacketInDropComponent(match[1]) {
				packetInDropComponets[match[1]] = match[2]
			} else {
				metric := makeMetric(match[1], match[2])
				if metric != nil {
					ch <- metric
				}
			}
		}
	}

	total := 0
	for m, v := range packetInDropComponets {
		if v != "" {
			i, e := strconv.Atoi(v)
			if e != nil {
				log.Errf("%s: %s: %s", m, v, e)
				continue
			}
			total += i
		}
	}
	if total > 0 {
		metric := makeMetric(packetInDrop, strconv.Itoa(total))
		if metric != nil {
			ch <- metric
		}
	}
}

type Collector struct{}

func (Collector) Name() string {
	return "ovn"
}

func (Collector) Metrics() []lib.Metric {
	var res []lib.Metric
	for _, g := range metrics {
		for _, m := range *g {
			res = append(res, m)
		}
	}
	return res
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	// collect items from the ExternalIDs field in the OpenvSwitch table
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	var vswitch ovs.OpenvSwitch
	err := ovsdb.Get(ctx, &vswitch)
	if err != nil {
		log.Errf("OvsdbGet(vswitch): %s", err)
		return
	}
	collectopenvSwitch(vswitch.ExternalIDs, ch)
	collectopenvSwitchBoolean(vswitch.ExternalIDs, ch)
	collectopenvSwitchLabels(vswitch.ExternalIDs, ch)

	// collect the ovn-controller coverage metrics
	collectCoverageMetrics(ch)
}
