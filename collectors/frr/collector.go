// SPDX-License-Identifier: Apache-2.0

package frr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string { return "frr" }

func (Collector) Metrics() []lib.Metric { return allMetrics }

func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	sockDir := config.FrrRundir()
	vty := &vtyClient{sockDir: sockDir}

	projectMap := buildVRFProjectMap()

	collectBGPSummary(vty, projectMap, ch)
	collectEVPNRoutes(vty, ch)
	collectVRFVNI(vty, projectMap, ch)
	collectEVPNRMAC(vty, ch)
	collectEVPNARPCache(vty, ch)
	collectRouteSummary(vty, projectMap, ch)
}

// buildVRFProjectMap returns a VRF name → project_id map.
// Stub for now — will query OVN NB once NB client is available.
func buildVRFProjectMap() map[string]string {
	return make(map[string]string)
}

// BGP summary parsing

type bgpSummaryAFI struct {
	RouterID  string             `json:"routerId"`
	AS        uint32             `json:"as"`
	VrfName   string             `json:"vrfName"`
	PeerCount int                `json:"peerCount"`
	Peers     map[string]bgpPeer `json:"peers"`
}

type bgpPeer struct {
	RemoteAs       uint32  `json:"remoteAs"`
	LocalAs        uint32  `json:"localAs"`
	State          string  `json:"state"`
	PfxRcd         int     `json:"pfxRcd"`
	PfxSnt         int     `json:"pfxSnt"`
	MsgRcvd        uint64  `json:"msgRcvd"`
	MsgSent        uint64  `json:"msgSent"`
	PeerUptimeMsec float64 `json:"peerUptimeMsec"`
}

func collectBGPSummary(vty *vtyClient, projectMap map[string]string, ch chan<- prometheus.Metric) {
	data, err := vty.Exec("bgpd", "show bgp summary json")
	if err != nil {
		log.Debugf("frr: bgp summary: %s", err)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Errf("frr: parse bgp summary: %s", err)
		return
	}

	for afiKey, afiData := range raw {
		var afi bgpSummaryAFI
		if err := json.Unmarshal(afiData, &afi); err != nil {
			continue
		}
		if afi.Peers == nil {
			continue
		}

		vrfName := afi.VrfName
		if vrfName == "" {
			vrfName = "default"
		}
		projectID := projectMap[vrfName]

		for peerAddr, peer := range afi.Peers {
			labels := []string{
				vrfName,
				projectID,
				peerAddr,
				fmt.Sprintf("%d", peer.RemoteAs),
				fmt.Sprintf("%d", peer.LocalAs),
				afiKey,
			}

			ch <- prometheus.MustNewConstMetric(
				bgpPeerState.Desc(), prometheus.GaugeValue,
				bgpStateToFloat(peer.State), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				bgpPeerUptimeSeconds.Desc(), prometheus.GaugeValue,
				peer.PeerUptimeMsec/1000.0, labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				bgpPeerPrefixesReceived.Desc(), prometheus.GaugeValue,
				float64(peer.PfxRcd), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				bgpPeerPrefixesAdvertised.Desc(), prometheus.GaugeValue,
				float64(peer.PfxSnt), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				bgpPeerMessagesSent.Desc(), prometheus.CounterValue,
				float64(peer.MsgSent), labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				bgpPeerMessagesReceived.Desc(), prometheus.CounterValue,
				float64(peer.MsgRcvd), labels...,
			)
		}
	}
}

func bgpStateToFloat(state string) float64 {
	switch strings.ToLower(state) {
	case "idle":
		return 0
	case "connect":
		return 1
	case "active":
		return 2
	case "opensent":
		return 3
	case "openconfirm":
		return 4
	case "established":
		return 5
	default:
		return -1
	}
}

// EVPN route parsing

type evpnRouteEntry struct {
	Paths []struct {
		RouteType int `json:"routeType"`
	} `json:"paths"`
}

func collectEVPNRoutes(vty *vtyClient, ch chan<- prometheus.Metric) {
	data, err := vty.Exec("bgpd", "show bgp l2vpn evpn json")
	if err != nil {
		log.Debugf("frr: evpn routes: %s", err)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Errf("frr: parse evpn routes: %s", err)
		return
	}

	// Count routes per RD and route type
	type rdRouteType struct {
		rd        string
		routeType string
	}
	counts := make(map[rdRouteType]int)

	for key, val := range raw {
		// Skip metadata keys
		if !strings.Contains(key, ":") {
			continue
		}
		rd := key

		// Parse the RD block — keys are EVPN prefixes
		var rdBlock map[string]json.RawMessage
		if err := json.Unmarshal(val, &rdBlock); err != nil {
			continue
		}

		for prefixKey, prefixData := range rdBlock {
			if prefixKey == "rd" {
				continue
			}
			var entry evpnRouteEntry
			if err := json.Unmarshal(prefixData, &entry); err != nil {
				continue
			}
			for _, path := range entry.Paths {
				k := rdRouteType{rd: rd, routeType: fmt.Sprintf("%d", path.RouteType)}
				counts[k]++
			}
		}
	}

	for k, count := range counts {
		ch <- prometheus.MustNewConstMetric(
			evpnRouteCount.Desc(), prometheus.GaugeValue,
			float64(count), k.rd, k.routeType,
		)
	}
}

// VRF/VNI parsing

type vrfVNIResponse struct {
	VRFs []vrfVNIEntry `json:"vrfs"`
}

type vrfVNIEntry struct {
	VRF       string `json:"vrf"`
	VNI       int    `json:"vni"`
	VxlanIntf string `json:"vxlanIntf"`
	SVIIntf   string `json:"sviIntf"`
	State     string `json:"state"`
	RouterMAC string `json:"routerMac"`
}

func collectVRFVNI(vty *vtyClient, projectMap map[string]string, ch chan<- prometheus.Metric) {
	data, err := vty.Exec("zebra", "show vrf vni json")
	if err != nil {
		log.Debugf("frr: vrf vni: %s", err)
		return
	}

	var resp vrfVNIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Errf("frr: parse vrf vni: %s", err)
		return
	}

	for _, v := range resp.VRFs {
		ch <- prometheus.MustNewConstMetric(
			vrfVNIInfo.Desc(), prometheus.GaugeValue, 1,
			v.VRF,
			fmt.Sprintf("%d", v.VNI),
			projectMap[v.VRF],
			v.VxlanIntf,
			v.SVIIntf,
			v.State,
			v.RouterMAC,
		)
	}
}

// EVPN RMAC parsing

func collectEVPNRMAC(vty *vtyClient, ch chan<- prometheus.Metric) {
	data, err := vty.Exec("zebra", "show evpn rmac vni all json")
	if err != nil {
		log.Debugf("frr: evpn rmac: %s", err)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Errf("frr: parse evpn rmac: %s", err)
		return
	}

	for vni, vniData := range raw {
		var vniBlock map[string]json.RawMessage
		if err := json.Unmarshal(vniData, &vniBlock); err != nil {
			continue
		}
		// numRmacs is directly available
		var numRmacs float64
		if v, ok := vniBlock["numRmacs"]; ok {
			if err := json.Unmarshal(v, &numRmacs); err != nil {
				continue
			}
		}

		ch <- prometheus.MustNewConstMetric(
			evpnRMACCount.Desc(), prometheus.GaugeValue,
			numRmacs, vni,
		)
	}
}

// EVPN ARP cache parsing

func collectEVPNARPCache(vty *vtyClient, ch chan<- prometheus.Metric) {
	data, err := vty.Exec("zebra", "show evpn arp-cache vni all json")
	if err != nil {
		log.Debugf("frr: evpn arp-cache: %s", err)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Errf("frr: parse evpn arp-cache: %s", err)
		return
	}

	for vni, vniData := range raw {
		var vniBlock map[string]json.RawMessage
		if err := json.Unmarshal(vniData, &vniBlock); err != nil {
			continue
		}
		var numArpNd float64
		if v, ok := vniBlock["numArpNd"]; ok {
			if err := json.Unmarshal(v, &numArpNd); err != nil {
				continue
			}
		}

		ch <- prometheus.MustNewConstMetric(
			evpnARPCacheCount.Desc(), prometheus.GaugeValue,
			numArpNd, vni,
		)
	}
}

// Route summary parsing

type routeSummaryVRF struct {
	Routes []routeSummaryEntry `json:"routes"`
}

type routeSummaryEntry struct {
	Type string  `json:"type"`
	RIB  float64 `json:"rib"`
	FIB  float64 `json:"fib"`
}

func collectRouteSummary(vty *vtyClient, projectMap map[string]string, ch chan<- prometheus.Metric) {
	data, err := vty.Exec("zebra", "show ip route vrf all summary json")
	if err != nil {
		log.Debugf("frr: route summary: %s", err)
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Errf("frr: parse route summary: %s", err)
		return
	}

	for vrfName, vrfData := range raw {
		var vrf routeSummaryVRF
		if err := json.Unmarshal(vrfData, &vrf); err != nil {
			continue
		}

		projectID := projectMap[vrfName]

		for _, route := range vrf.Routes {
			if route.RIB == 0 && route.FIB == 0 {
				continue
			}
			labels := []string{vrfName, projectID, route.Type}

			ch <- prometheus.MustNewConstMetric(
				routeCountTotal.Desc(), prometheus.GaugeValue,
				route.RIB, labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				routeFIBCountTotal.Desc(), prometheus.GaugeValue,
				route.FIB, labels...,
			)
		}
	}
}
