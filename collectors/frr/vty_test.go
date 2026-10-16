// SPDX-License-Identifier: Apache-2.0

package frr

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeVTYServer simulates a FRR VTY Unix socket (vtysh protocol).
// Accepts one connection, reads a command, responds with JSON + end marker.
func fakeVTYServer(t *testing.T, sockPath string, response []byte) net.Listener {
	t.Helper()

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Read command (null-terminated)
		buf := make([]byte, 256)
		conn.Read(buf)

		// Send response + end marker (\0\0\0<status>)
		conn.Write(response)
		conn.Write([]byte{0, 0, 0, 0}) // \0\0\0 + CMD_SUCCESS(0)
	}()

	return ln
}

func TestVTYExec(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "bgpd.vty")
	want := []byte(`{"as":65000,"peers":{}}`)

	ln := fakeVTYServer(t, sockPath, want)
	defer ln.Close()

	c := &vtyClient{sockDir: dir}
	got, err := c.Exec("bgpd", "show bgp summary json")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVTYExecDaemonNotRunning(t *testing.T) {
	dir := t.TempDir()

	c := &vtyClient{sockDir: dir}
	_, err := c.Exec("bgpd", "show bgp summary json")
	if err == nil {
		t.Fatal("expected error for missing socket")
	}
}

func TestVTYExecTimeout(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "bgpd.vty")

	// Server that accepts but never responds
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Hang — never write anything
		time.Sleep(10 * time.Second)
	}()

	c := &vtyClient{sockDir: dir}
	_, err = c.Exec("bgpd", "show bgp summary json")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestVTYExecLargeResponse(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "zebra.vty")

	// Build a response larger than the 4096 read buffer
	large := make([]byte, 10000)
	for i := range large {
		large[i] = 'x'
	}
	large[0] = '{'
	large[len(large)-1] = '}'

	ln := fakeVTYServer(t, sockPath, large)
	defer ln.Close()

	c := &vtyClient{sockDir: dir}
	got, err := c.Exec("zebra", "show vrf vni json")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	if len(got) != len(large) {
		t.Errorf("got %d bytes, want %d", len(got), len(large))
	}
}

func TestVTYExecWithTestdata(t *testing.T) {
	testdata := map[string]string{
		"show bgp summary json":          "testdata/show_bgp_summary.json",
		"show bgp l2vpn evpn json":       "testdata/show_bgp_l2vpn_evpn.json",
		"show vrf vni json":              "testdata/show_vrf_vni.json",
		"show evpn rmac vni all json":    "testdata/show_evpn_rmac.json",
		"show evpn arp-cache vni all json":    "testdata/show_evpn_arp_cache.json",
		"show ip route vrf all summary json": "testdata/show_ip_route_vrf_all_summary.json",
	}

	for cmd, file := range testdata {
		t.Run(cmd, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read testdata: %v", err)
			}

			dir := t.TempDir()
			daemon := "bgpd"
			if cmd == "show vrf vni json" || cmd == "show evpn rmac vni all json" || cmd == "show evpn arp-cache vni all json" || cmd == "show ip route vrf all summary json" {
				daemon = "zebra"
			}
			sockPath := filepath.Join(dir, daemon+".vty")

			ln := fakeVTYServer(t, sockPath, data)
			defer ln.Close()

			c := &vtyClient{sockDir: dir}
			got, err := c.Exec(daemon, cmd)
			if err != nil {
				t.Fatalf("Exec failed: %v", err)
			}

			if string(got) != string(data) {
				t.Errorf("response mismatch: got %d bytes, want %d bytes", len(got), len(data))
			}
		})
	}
}
