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

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
)

type appctlDaemon string

const (
	ovsVswitchd   appctlDaemon = "ovs-vswitchd"
	ovnController appctlDaemon = "ovn-controller"
)

func call(daemon appctlDaemon, method string, args ...string) string {
	var rundir string
	var err error

	switch daemon {
	case ovsVswitchd:
		rundir = config.OvsRundir()
	case ovnController:
		rundir = config.OvnRundir()
	default:
		panic(fmt.Errorf("unknown daemon value: %v", daemon))
	}

	pidfile := filepath.Join(rundir, fmt.Sprintf("%s.pid", daemon))

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
	sockpath := filepath.Join(rundir, fmt.Sprintf("%s.%d.ctl", daemon, pid))
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

func OvsVSwitchd(method string, args ...string) string {
	return call(ovsVswitchd, method, args...)
}

func OvnController(method string, args ...string) string {
	return call(ovnController, method, args...)
}
