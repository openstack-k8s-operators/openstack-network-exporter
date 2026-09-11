// SPDX-License-Identifier: Apache-2.0

package ovsdb

import (
	"context"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/ovsdb/ovnsb"
	"github.com/ovn-kubernetes/libovsdb/database/inmemory"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/server"
)

func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for %s", timeout, what)
}

// startSBServer runs an in-memory OVN_Southbound database on a unix socket.
func startSBServer(t *testing.T, path string) *server.OvsdbServer {
	t.Helper()
	clientModel, err := ovnsb.FullDatabaseModel()
	if err != nil {
		t.Fatal(err)
	}
	logger := logr.Discard()
	db := inmemory.NewDatabase(map[string]model.ClientDBModel{"OVN_Southbound": clientModel}, &logger)
	dbModel, errs := model.NewDatabaseModel(ovnsb.Schema(), clientModel)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	srv, err := server.NewOvsdbServer(db, &logger, dbModel)
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve("unix", path) }()
	waitFor(t, 2*time.Second, "the SB server", srv.Ready)
	return srv
}

// socketProxy stands for the SB database socket. Stopping it closes every
// client connection and removes the socket file, as a restart of the
// database does. The in-memory server cannot be used for that: its Close()
// only closes the listener and keeps accepted connections open.
type socketProxy struct {
	listener net.Listener
	mu       sync.Mutex
	conns    []net.Conn
}

func startProxy(t *testing.T, path, backend string) *socketProxy {
	t.Helper()
	l, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	p := &socketProxy{listener: l}
	go func() {
		for {
			front, err := l.Accept()
			if err != nil {
				return
			}
			back, err := net.Dial("unix", backend)
			if err != nil {
				front.Close()
				continue
			}
			p.mu.Lock()
			p.conns = append(p.conns, front, back)
			p.mu.Unlock()
			go func() { _, _ = io.Copy(back, front); back.Close() }()
			go func() { _, _ = io.Copy(front, back); front.Close() }()
		}
	}()
	return p
}

func (p *socketProxy) stop() {
	p.listener.Close() // also unlinks the socket file
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.conns {
		c.Close()
	}
	p.conns = nil
}

func listDatapaths() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var datapaths []ovnsb.DatapathBinding
	return SBList(ctx, &datapaths)
}

// The SB client is cached: a restart or leader change of the SB database
// must not leave it dead until the exporter restarts.
func TestSBReconnectAfterServerRestart(t *testing.T) {
	dir := t.TempDir()
	backend := filepath.Join(dir, "backend.sock")
	srv := startSBServer(t, backend)
	defer srv.Close()

	sock := filepath.Join(dir, "sb.sock")
	t.Setenv("OPENSTACK_NETWORK_EXPORTER_OVN_SB_CONNECTION", "unix:"+sock)
	if err := config.Parse(); err != nil {
		t.Fatal(err)
	}
	proxy := startProxy(t, sock, backend)
	if err := listDatapaths(); err != nil {
		t.Fatalf("first call: %v", err)
	}

	proxy.stop()
	waitFor(t, 5*time.Second, "calls to fail while the SB database is down", func() bool { return listDatapaths() != nil })

	proxy = startProxy(t, sock, backend)
	defer proxy.stop()
	waitFor(t, 5*time.Second, "calls to succeed after the SB database is back", func() bool { return listDatapaths() == nil })
}
