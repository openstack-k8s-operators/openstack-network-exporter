// SPDX-License-Identifier: Apache-2.0

package frr

import (
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
)

const vtyTimeout = 2 * time.Second

type vtyClient struct {
	sockDir string
}

// Exec connects to a FRR daemon's VTY socket, sends the given command, and
// returns the JSON response. Each call opens a new connection (no persistent
// state).
//
// FRR .vty Unix sockets use the vtysh protocol: no banner on connect, no
// "enable" step. Client sends command+\0, daemon responds with
// output+\0\0\0<status>.
func (c *vtyClient) Exec(daemon, command string) ([]byte, error) {
	sockPath := filepath.Join(c.sockDir, daemon+".vty")

	conn, err := net.DialTimeout("unix", sockPath, vtyTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", sockPath, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(vtyTimeout)); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	// Send the command (null-terminated)
	if _, err := conn.Write(append([]byte(command), 0)); err != nil {
		return nil, fmt.Errorf("send command: %w", err)
	}

	// Read response until the first null byte (start of end marker)
	resp, err := vtyRead(conn)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	log.Debugf("frr: %s %q → %d bytes", daemon, command, len(resp))

	return resp, nil
}

// vtyRead reads from conn until a null byte is encountered, returning
// everything before it.
func vtyRead(conn net.Conn) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			return buf, err
		}
		for i := 0; i < n; i++ {
			if tmp[i] == 0 {
				return append(buf, tmp[:i]...), nil
			}
		}
		buf = append(buf, tmp[:n]...)
	}
}
