// SPDX-License-Identifier: Apache-2.0

package frr

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

// BGP peer metrics

var bgpPeerLabels = []string{"vrf", "project_id", "peer", "peer_as", "local_as", "afi"}

var bgpPeerState = lib.Metric{
	Name:        "frr_bgp_peer_state",
	Description: "BGP peer session state (0=idle, 1=connect, 2=active, 3=opensent, 4=openconfirm, 5=established).",
	Labels:      bgpPeerLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var bgpPeerUptimeSeconds = lib.Metric{
	Name:        "frr_bgp_peer_uptime_seconds",
	Description: "BGP peer session uptime in seconds.",
	Labels:      bgpPeerLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var bgpPeerPrefixesReceived = lib.Metric{
	Name:        "frr_bgp_peer_prefixes_received",
	Description: "Number of prefixes received from the BGP peer.",
	Labels:      bgpPeerLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var bgpPeerPrefixesAdvertised = lib.Metric{
	Name:        "frr_bgp_peer_prefixes_advertised",
	Description: "Number of prefixes advertised to the BGP peer.",
	Labels:      bgpPeerLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var bgpPeerMessagesSent = lib.Metric{
	Name:        "frr_bgp_peer_messages_sent_total",
	Description: "Total number of messages sent to the BGP peer.",
	Labels:      bgpPeerLabels,
	ValueType:   prometheus.CounterValue,
	Set:         config.METRICS_COUNTERS,
}

var bgpPeerMessagesReceived = lib.Metric{
	Name:        "frr_bgp_peer_messages_received_total",
	Description: "Total number of messages received from the BGP peer.",
	Labels:      bgpPeerLabels,
	ValueType:   prometheus.CounterValue,
	Set:         config.METRICS_COUNTERS,
}

// EVPN route metrics

var evpnRouteCount = lib.Metric{
	Name:        "frr_evpn_route_count",
	Description: "Number of EVPN routes per route distinguisher and route type.",
	Labels:      []string{"rd", "route_type"},
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

// VRF/VNI metrics

var vrfVNIInfo = lib.Metric{
	Name:        "frr_vrf_vni_info",
	Description: "VRF to VNI mapping information.",
	Labels:      []string{"vrf", "vni", "project_id", "vxlan_intf", "svi_intf", "state", "router_mac"},
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

// EVPN RMAC metrics

var evpnRMACCount = lib.Metric{
	Name:        "frr_evpn_rmac_count",
	Description: "Number of remote router MACs per VNI.",
	Labels:      []string{"vni"},
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

// EVPN ARP cache metrics

var evpnARPCacheCount = lib.Metric{
	Name:        "frr_evpn_arp_cache_count",
	Description: "Number of ARP/ND cache entries per VNI.",
	Labels:      []string{"vni"},
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

// Route count metrics

var routeLabels = []string{"vrf", "project_id", "protocol"}

var routeCountTotal = lib.Metric{
	Name:        "frr_route_count_total",
	Description: "Number of routes in the RIB per VRF and protocol.",
	Labels:      routeLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

var routeFIBCountTotal = lib.Metric{
	Name:        "frr_route_fib_count_total",
	Description: "Number of routes installed in the FIB (kernel) per VRF and protocol.",
	Labels:      routeLabels,
	ValueType:   prometheus.GaugeValue,
	Set:         config.METRICS_BASE,
}

// allMetrics returns all FRR metrics for the collector interface.
var allMetrics = []lib.Metric{
	bgpPeerState,
	bgpPeerUptimeSeconds,
	bgpPeerPrefixesReceived,
	bgpPeerPrefixesAdvertised,
	bgpPeerMessagesSent,
	bgpPeerMessagesReceived,
	evpnRouteCount,
	vrfVNIInfo,
	evpnRMACCount,
	evpnARPCacheCount,
	routeCountTotal,
	routeFIBCountTotal,
}
