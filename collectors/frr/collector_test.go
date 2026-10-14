// SPDX-License-Identifier: Apache-2.0

package frr

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

func mustReadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read testdata/%s: %v", name, err)
	}
	return data
}

func collectMetrics(t *testing.T, f func(ch chan<- prometheus.Metric)) map[string][]io_prometheus_client.Metric {
	t.Helper()
	ch := make(chan prometheus.Metric, 100)
	f(ch)
	close(ch)

	result := make(map[string][]io_prometheus_client.Metric)
	for m := range ch {
		var dto io_prometheus_client.Metric
		if err := m.Write(&dto); err != nil {
			t.Fatal(err)
		}
		desc := m.Desc().String()
		result[desc] = append(result[desc], dto)
	}
	return result
}

func getLabelValue(m io_prometheus_client.Metric, name string) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == name {
			return lp.GetValue()
		}
	}
	return ""
}

func TestCollectBGPSummary(t *testing.T) {
	data := mustReadTestdata(t, "show_bgp_summary.json")
	dir := t.TempDir()
	sockPath := dir + "/bgpd.vty"
	ln := fakeVTYServer(t, sockPath, data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	projectMap := map[string]string{}

	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectBGPSummary(vty, projectMap, ch)
	})

	// 6 metrics per peer × 1 peer = 6
	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 6 {
		t.Errorf("expected 6 metrics, got %d", total)
	}

	// Verify state metric
	for desc, ms := range metrics {
		if !containsString(desc, "frr_bgp_peer_state") {
			continue
		}
		if len(ms) != 1 {
			t.Fatalf("expected 1 state metric, got %d", len(ms))
		}
		m := ms[0]
		if got := m.GetGauge().GetValue(); got != 5 {
			t.Errorf("peer state: got %v, want 5 (Established)", got)
		}
		if got := getLabelValue(m, "peer"); got != "172.16.0.1" {
			t.Errorf("peer label: got %q, want 172.16.0.1", got)
		}
		if got := getLabelValue(m, "vrf"); got != "default" {
			t.Errorf("vrf label: got %q, want default", got)
		}
		if got := getLabelValue(m, "afi"); got != "l2VpnEvpn" {
			t.Errorf("afi label: got %q, want l2VpnEvpn", got)
		}
		if got := getLabelValue(m, "peer_as"); got != "65000" {
			t.Errorf("peer_as label: got %q, want 65000", got)
		}
	}
}

func TestCollectBGPSummaryUptime(t *testing.T) {
	data := mustReadTestdata(t, "show_bgp_summary.json")
	dir := t.TempDir()
	ln := fakeVTYServer(t, dir+"/bgpd.vty", data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectBGPSummary(vty, map[string]string{}, ch)
	})

	for desc, ms := range metrics {
		if !containsString(desc, "frr_bgp_peer_uptime_seconds") {
			continue
		}
		if got := ms[0].GetGauge().GetValue(); got != 30.0 {
			t.Errorf("uptime: got %v, want 30.0", got)
		}
	}
}

func TestCollectEVPNRoutes(t *testing.T) {
	data := mustReadTestdata(t, "show_bgp_l2vpn_evpn.json")
	dir := t.TempDir()
	ln := fakeVTYServer(t, dir+"/bgpd.vty", data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectEVPNRoutes(vty, ch)
	})

	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	// 4 RDs, route types: Type-5 in 3 RDs (172.16.0.1:2 has 2, 172.16.0.2:2 has 2),
	// Type-3 in 2 RDs → expect distinct (rd, route_type) combos
	if total == 0 {
		t.Error("expected EVPN route metrics, got none")
	}

	// Verify we have route_type=5 and route_type=3
	hasType3, hasType5 := false, false
	for _, ms := range metrics {
		for _, m := range ms {
			rt := getLabelValue(m, "route_type")
			if rt == "3" {
				hasType3 = true
			}
			if rt == "5" {
				hasType5 = true
			}
		}
	}
	if !hasType3 {
		t.Error("missing route_type=3 (IMET)")
	}
	if !hasType5 {
		t.Error("missing route_type=5 (IP prefix)")
	}
}

func TestCollectVRFVNI(t *testing.T) {
	data := mustReadTestdata(t, "show_vrf_vni.json")
	dir := t.TempDir()
	ln := fakeVTYServer(t, dir+"/zebra.vty", data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	projectMap := map[string]string{"tenant": "test-project-id"}

	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectVRFVNI(vty, projectMap, ch)
	})

	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 1 {
		t.Fatalf("expected 1 VRF/VNI metric, got %d", total)
	}

	for _, ms := range metrics {
		m := ms[0]
		if got := getLabelValue(m, "vrf"); got != "tenant" {
			t.Errorf("vrf: got %q, want tenant", got)
		}
		if got := getLabelValue(m, "vni"); got != "1000" {
			t.Errorf("vni: got %q, want 1000", got)
		}
		if got := getLabelValue(m, "project_id"); got != "test-project-id" {
			t.Errorf("project_id: got %q, want test-project-id", got)
		}
		if got := getLabelValue(m, "state"); got != "Up" {
			t.Errorf("state: got %q, want Up", got)
		}
		if got := getLabelValue(m, "router_mac"); got != "0a:e7:54:a8:08:3f" {
			t.Errorf("router_mac: got %q, want 0a:e7:54:a8:08:3f", got)
		}
	}
}

func TestCollectEVPNRMAC(t *testing.T) {
	data := mustReadTestdata(t, "show_evpn_rmac.json")
	dir := t.TempDir()
	ln := fakeVTYServer(t, dir+"/zebra.vty", data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectEVPNRMAC(vty, ch)
	})

	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 1 {
		t.Fatalf("expected 1 RMAC metric, got %d", total)
	}

	for _, ms := range metrics {
		m := ms[0]
		if got := getLabelValue(m, "vni"); got != "1000" {
			t.Errorf("vni: got %q, want 1000", got)
		}
		if got := m.GetGauge().GetValue(); got != 1 {
			t.Errorf("numRmacs: got %v, want 1", got)
		}
	}
}

func TestCollectEVPNARPCache(t *testing.T) {
	data := mustReadTestdata(t, "show_evpn_arp_cache.json")
	dir := t.TempDir()
	ln := fakeVTYServer(t, dir+"/zebra.vty", data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectEVPNARPCache(vty, ch)
	})

	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 1 {
		t.Fatalf("expected 1 ARP cache metric, got %d", total)
	}

	for _, ms := range metrics {
		m := ms[0]
		if got := getLabelValue(m, "vni"); got != "100" {
			t.Errorf("vni: got %q, want 100", got)
		}
		if got := m.GetGauge().GetValue(); got != 1 {
			t.Errorf("numArpNd: got %v, want 1", got)
		}
	}
}

func TestCollectBGPSummaryDaemonDown(t *testing.T) {
	dir := t.TempDir()
	vty := &vtyClient{sockDir: dir}

	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectBGPSummary(vty, map[string]string{}, ch)
	})

	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 0 {
		t.Errorf("expected 0 metrics when daemon is down, got %d", total)
	}
}

func TestCollectRouteSummary(t *testing.T) {
	data := mustReadTestdata(t, "show_ip_route_vrf_all_summary.json")
	dir := t.TempDir()
	ln := fakeVTYServer(t, dir+"/zebra.vty", data)
	defer ln.Close()

	vty := &vtyClient{sockDir: dir}
	projectMap := map[string]string{"tenant": "test-project-id"}

	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectRouteSummary(vty, projectMap, ch)
	})

	// default VRF: kernel(1/1), connected(3/3), ibgp(2/2) = 3 protocols
	// (ebgp 0/0 skipped)
	// tenant VRF: connected(2/2), ibgp(1/1) = 2 protocols
	// Total: 5 protocols × 2 metrics each = 10
	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 10 {
		t.Errorf("expected 10 route metrics, got %d", total)
	}

	// Verify tenant VRF ibgp route
	for desc, ms := range metrics {
		if !containsString(desc, "frr_route_count_total") {
			continue
		}
		for _, m := range ms {
			if getLabelValue(m, "vrf") == "tenant" &&
				getLabelValue(m, "protocol") == "ibgp" {
				if got := m.GetGauge().GetValue(); got != 1 {
					t.Errorf("tenant ibgp rib: got %v, want 1", got)
				}
				if got := getLabelValue(m, "project_id"); got != "test-project-id" {
					t.Errorf("project_id: got %q, want test-project-id", got)
				}
			}
		}
	}
}

func TestCollectRouteSummaryDaemonDown(t *testing.T) {
	dir := t.TempDir()
	vty := &vtyClient{sockDir: dir}

	metrics := collectMetrics(t, func(ch chan<- prometheus.Metric) {
		collectRouteSummary(vty, map[string]string{}, ch)
	})

	total := 0
	for _, ms := range metrics {
		total += len(ms)
	}
	if total != 0 {
		t.Errorf("expected 0 metrics when daemon is down, got %d", total)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
