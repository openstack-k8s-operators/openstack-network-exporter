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
	"regexp"
	"strconv"
	"strings"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
)

type appctlDaemon string

const (
	ovsVswitchd   appctlDaemon = "ovs-vswitchd"
	ovnController appctlDaemon = "ovn-controller"
	ovnNorthd     appctlDaemon = "ovn-northd"
)

func getPidFromFile(pidfile string) (int, error) {
	f, err := os.Open(pidfile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(buf)))
	if err != nil {
		return 0, err
	}

	return pid, nil
}

func getPidFromCtlFiles(rundir string, daemon appctlDaemon) (int, error) {
	// Look for .ctl files matching the daemon pattern
	pattern := fmt.Sprintf("%s.*.ctl", daemon)
	matches, err := filepath.Glob(filepath.Join(rundir, pattern))
	if err != nil {
		return 0, err
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no control socket files found for %s", daemon)
	}

	// Extract PID from the first matching file
	// Expected format: daemon.pid.ctl
	re := regexp.MustCompile(fmt.Sprintf(`%s\.(\d+)\.ctl$`, regexp.QuoteMeta(string(daemon))))
	for _, match := range matches {
		basename := filepath.Base(match)
		submatch := re.FindStringSubmatch(basename)
		if len(submatch) == 2 {
			pid, err := strconv.Atoi(submatch[1])
			if err == nil {
				return pid, nil
			}
		}
	}

	return 0, fmt.Errorf("could not extract PID from control socket files for %s", daemon)
}

func call(daemon appctlDaemon, method string, args ...string) string {
	var rundir string
	var err error

	switch daemon {
	case ovsVswitchd:
		rundir = config.OvsRundir()
	case ovnController:
		rundir = config.OvnRundir()
	case ovnNorthd:
		rundir = config.OvnRundir()
	default:
		panic(fmt.Errorf("unknown daemon value: %v", daemon))
	}

	pidfile := filepath.Join(rundir, fmt.Sprintf("%s.pid", daemon))

	// First try to get PID from .pid file
	pid, err := getPidFromFile(pidfile)
	if err != nil {
		log.Debugf("Failed to read PID file %s: %s, trying to find PID from .ctl files", pidfile, err)
		// If that fails, try to extract PID from .ctl files
		pid, err = getPidFromCtlFiles(rundir, daemon)
		if err != nil {
			log.Errf("Failed to get PID for %s: %s", daemon, err)
			return ""
		}
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

func OvnNorthd(method string, args ...string) string {
	return call(ovnNorthd, method, args...)
}
