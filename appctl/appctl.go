// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package appctl

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/dataplane-node-exporter/config"
	"github.com/openstack-k8s-operators/dataplane-node-exporter/log"
)

func Call(method string, args ...string) string {
	var err error

	pidfile := filepath.Join(config.OvsRundir(), "ovs-vswitchd.pid")

	var f *os.File
	if f, err = os.Open(pidfile); err != nil {
		log.Errf("os.Open: %s", err)
		return ""
	}
	var buf []byte
	if buf, err = io.ReadAll(f); err != nil {
		log.Errf("io.ReadAll: %s", err)
		return ""
	}

	var pid int
	if pid, err = strconv.Atoi(strings.TrimSpace(string(buf))); err != nil {
		log.Errf("strconv.Atoi: %s", err)
		return ""
	}
	sockpath := filepath.Join(config.OvsRundir(), fmt.Sprintf("ovs-vswitchd.%d.ctl", pid))
	conn, err := net.Dial("unix", sockpath)
	if err != nil {
		log.Errf("net.Dial: %s", err)
		return ""
	}

	client := rpc.NewClientWithCodec(NewClientCodec(conn))
	defer func() {
		err := client.Close()
		if err != nil {
			log.Warningf("close: %s", err)
		}
	}()

	if args == nil {
		args = make([]string, 0)
	}

	var reply string

	log.Debugf("calling: %s %s", method, args)
	if err = client.Call(method, args, &reply); err != nil {
		log.Errf("call(%s): %s", method, err)
		return ""
	}

	return reply
}
