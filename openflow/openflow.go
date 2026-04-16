// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2024 Robin Jarry

package openflow

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/skydive-project/goloxi"
	"github.com/skydive-project/goloxi/of10"
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
	ofTblLogToPhys     uint8  = 65
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

func handShake(conn net.Conn) error {

	helloReq := helloMsg{
		Version: ofp10Version,
		Type:    ofptHello,
		Length:  uint16(binary.Size(helloMsg{})),
		Xid:     1,
	}
	var helloResp helloMsg
	return sendRecv(conn, &helloReq, &helloResp)
}

func (s *BridgeStats) GetAggregateStats() error {
	conn, err := connect(s.Name)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = handShake(conn)
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

type RouterPortsStats struct {
	DPTunnelKey   uint64
	PortTunnelKey uint32
	PacketCount   uint64
	ByteCount     uint64
}

func GetRouterPortsStats() ([]RouterPortsStats, error) {
	var isDataPathJump bool
	var routerStats []RouterPortsStats
	var dpTunnK uint64
	var pTunnK uint32

	stats, err := getFlowStats(config.IntBrdNam(), ofTblLogToPhys)
	if err != nil {
		return nil, err
	}

	for _, entry := range stats.GetStats() {
		isDataPathJump = false

		for _, anAction := range entry.GetActions() {
			if anAction.GetActionName() == "nx_clone" {
				isDataPathJump = true
				break
			}
		}

		if isDataPathJump {
			for _, aMatch := range entry.GetMatch().NxmEntries {
				mName := aMatch.GetOXMName()
				if mName == "reg15" {
					pTunnK = aMatch.GetOXMValue().(uint32)
				} else if mName == "metadata" {
					dpTunnK = aMatch.GetOXMValue().(uint64)
				}
			}

			routerStats = append(
				routerStats,
				RouterPortsStats{
					DPTunnelKey:   dpTunnK,
					PortTunnelKey: pTunnK,
					PacketCount:   entry.GetPacketCount(),
					ByteCount:     entry.GetByteCount(),
				})
		}
	}
	return routerStats, nil
}

func getFlowStats(bridge string, table uint8) (*of10.NiciraFlowStatsReply, error) {

	conn, err := connect(bridge)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return nil, err
	}

	err = handShake(conn)
	if err != nil {
		return nil, err
	}

	request := of10.NewNiciraFlowStatsRequest()
	request.SetXid(1)
	request.SetTableId(table)
	request.SetOutPort(of10.Port(ofppNone))
	request.SetMatchLen(0)
	encoder := goloxi.NewEncoder()
	if err = request.Serialize(encoder); err != nil {
		return nil, err
	}
	_, err = conn.Write(encoder.Bytes())
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	data, err := reader.Peek(8)
	if err != nil {
		return nil, err
	}
	header := &goloxi.Header{}
	if err := header.Decode(goloxi.NewDecoder(data)); err != nil {
		return nil, err
	}
	data = make([]byte, header.Length)
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return nil, err
	}
	flows, err := of10.DecodeMessage(data)
	if err != nil {
		return nil, err
	}
	switch t := flows.(type) {
	case *of10.NiciraFlowStatsReply:
		return t, nil
	default:
		return nil, fmt.Errorf("unexpected openflow response of type %T from bridge", t)
	}
}
