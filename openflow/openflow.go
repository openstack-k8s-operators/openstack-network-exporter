// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package openflow

import (
	"encoding/binary"
	"net"
	"path/filepath"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
)

// required constants taken from openvswitch
const (
	ofp10Version       uint8  = 0x01
	ofptHello          uint8  = 0
	ofpt10StatsRequest uint8  = 16
	ofpt10StatsReply   uint8  = 17
	nxVendorId         uint32 = 0x00002320 // Nicira
	ofppNone           uint16 = 0xffff     // no port
	ofpttAll           uint8  = 0xff       // all tables
	ofpstVendor        uint16 = 0xffff     // vendor stats
	nxstAggregate      uint32 = 1          // hardcoded in a comment
)

// struct ofp_header
type helloMsg struct {
	Version uint8
	Type    uint8
	Length  uint16
	Xid     uint32
}

// struct nicira10_stats_msg
type niciraStatsMsg struct {
	// struct ofp_header
	Version uint8
	Type    uint8
	Length  uint16
	Xid     uint32
	// struct ofp10_stats_msg
	Stat  uint16
	Flags uint16
	// struct ofp_vendor_header
	Vendor  uint32
	Subtype uint32
	Padding [4]byte
}

type nxAggregateStatsRequest struct {
	Header niciraStatsMsg
	// struct nx_flow_stats_request
	OutPort  uint16
	MatchLen uint16
	TableId  uint8
	Padding  [3]byte
}

type nxAggregateStatsReply struct {
	Header niciraStatsMsg
	// struct ofp_aggregate_stats_reply
	PacketCount uint64
	ByteCount   uint64
	FlowCount   uint32
	Padding     [4]byte
}

func connect(bridge string) (net.Conn, error) {
	sock := filepath.Join(config.OvsRundir(), bridge+".mgmt")

	conn, err := net.DialTimeout("unix", sock, 1*time.Second)
	if err != nil {
		return nil, err
	}
	err = conn.SetDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func sendRecv(conn net.Conn, request any, response any) error {
	err := binary.Write(conn, binary.BigEndian, request)
	if err != nil {
		return err
	}
	err = binary.Read(conn, binary.BigEndian, response)
	if err != nil {
		return err
	}
	return nil
}

type BridgeStats struct {
	Name    string
	Packets uint64
	Bytes   uint64
	Flows   uint32
}

func (s *BridgeStats) GetAggregateStats() error {
	conn, err := connect(s.Name)
	if err != nil {
		return err
	}
	defer conn.Close()

	helloReq := helloMsg{
		Version: ofp10Version,
		Type:    ofptHello,
		Length:  uint16(binary.Size(helloMsg{})),
		Xid:     1,
	}
	var helloResp helloMsg
	err = sendRecv(conn, &helloReq, &helloResp)
	if err != nil {
		return err
	}

	statsReq := nxAggregateStatsRequest{
		Header: niciraStatsMsg{
			Version: ofp10Version,
			Type:    ofpt10StatsRequest,
			Length:  uint16(binary.Size(nxAggregateStatsRequest{})),
			Xid:     1,
			Stat:    ofpstVendor,
			Vendor:  nxVendorId,
			Subtype: nxstAggregate,
		},
		OutPort: ofppNone,
		TableId: ofpttAll,
	}
	var statsResp nxAggregateStatsReply
	err = sendRecv(conn, &statsReq, &statsResp)
	if err != nil {
		return err
	}

	s.Packets = statsResp.PacketCount
	s.Bytes = statsResp.ByteCount
	s.Flows = statsResp.FlowCount

	return nil
}
