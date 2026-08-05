// SPDX-License-Identifier: Apache-2.0

package openflow

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/skydive-project/goloxi"
	"github.com/skydive-project/goloxi/of10"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	readData  *bytes.Buffer
	writeData *bytes.Buffer
	readErr   error
	writeErr  error
	deadline  time.Time
	closed    bool
}

func newMockConn() *mockConn {
	return &mockConn{
		readData:  &bytes.Buffer{},
		writeData: &bytes.Buffer{},
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	if !m.deadline.IsZero() && time.Now().After(m.deadline) {
		return 0, &timeoutError{}
	}
	if m.readData.Len() == 0 {
		return 0, &timeoutError{}
	}
	return m.readData.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writeData.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return &net.UnixAddr{Name: "local"} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.UnixAddr{Name: "remote"} }
func (m *mockConn) SetDeadline(t time.Time) error      { m.deadline = t; return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { m.deadline = t; return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type timeoutError struct{}

func (e *timeoutError) Error() string { return "timeout" }
func (e *timeoutError) Timeout() bool { return true }

func createOpenFlowHeader(version, msgType uint8, length uint16, xid uint32) []byte {
	header := make([]byte, 8)
	header[0] = version
	header[1] = msgType
	binary.BigEndian.PutUint16(header[2:4], length)
	binary.BigEndian.PutUint32(header[4:8], xid)
	return header
}

func TestDrainPendingMessages_Timeout(t *testing.T) {
	conn := newMockConn()
	start := time.Now()
	drainPendingMessages(conn, 50*time.Millisecond)
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("drainPendingMessages took too long: %v", elapsed)
	}
}

func TestDrainPendingMessages_EchoRequest(t *testing.T) {
	conn := newMockConn()

	echoHeader := createOpenFlowHeader(0x01, 2, 8, 0)
	conn.readData.Write(echoHeader)

	initialLen := conn.readData.Len()
	drainPendingMessages(conn, 50*time.Millisecond)

	if conn.readData.Len() != 0 {
		t.Errorf("expected all data to be drained, got %d bytes remaining (started with %d)",
			conn.readData.Len(), initialLen)
	}
}

func TestDrainPendingMessages_MultipleMessages(t *testing.T) {
	conn := newMockConn()

	for i := 0; i < 3; i++ {
		echoHeader := createOpenFlowHeader(0x01, 2, 8, uint32(i))
		conn.readData.Write(echoHeader)
	}

	initialLen := conn.readData.Len()
	drainPendingMessages(conn, 50*time.Millisecond)

	if conn.readData.Len() != 0 {
		t.Errorf("expected all data to be drained, got %d bytes remaining (started with %d)",
			conn.readData.Len(), initialLen)
	}
}

func TestDrainPendingMessages_MessageWithBody(t *testing.T) {
	conn := newMockConn()

	header := createOpenFlowHeader(0x01, 2, 24, 0)
	body := make([]byte, 16)
	conn.readData.Write(header)
	conn.readData.Write(body)

	initialLen := conn.readData.Len()
	drainPendingMessages(conn, 50*time.Millisecond)

	if conn.readData.Len() != 0 {
		t.Errorf("expected all data to be drained, got %d bytes remaining (started with %d)",
			conn.readData.Len(), initialLen)
	}
}

func TestDrainPendingMessages_PartialHeader(t *testing.T) {
	conn := newMockConn()

	conn.readData.Write([]byte{0x01, 0x02, 0x00})

	drainPendingMessages(conn, 50*time.Millisecond)
}

func TestDrainPendingMessages_ReadError(t *testing.T) {
	conn := newMockConn()
	conn.readErr = io.ErrUnexpectedEOF

	drainPendingMessages(conn, 50*time.Millisecond)
}

func createNiciraFlowStatsReply(flags of10.StatsReplyFlags) []byte {
	reply := of10.NewNiciraFlowStatsReply()
	reply.SetFlags(flags)

	encoder := goloxi.NewEncoder()
	if err := reply.Serialize(encoder); err != nil {
		panic(err)
	}
	return encoder.Bytes()
}

func TestGetFlowStats_SingleReply(t *testing.T) {
	replyData := createNiciraFlowStatsReply(0)

	buf := bytes.NewBuffer(replyData)
	reader := bufio.NewReader(buf)

	stats, err := readFlowStatsReplies(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stats) != 0 {
		t.Errorf("expected 0 flow stats from empty reply, got %d", len(stats))
	}

	remaining := reader.Buffered()
	if remaining != 0 {
		t.Errorf("expected all data to be consumed, but %d bytes remain buffered", remaining)
	}
}

func TestGetFlowStats_MultiPartReply(t *testing.T) {
	part1Data := createNiciraFlowStatsReply(of10.OFPSFReplyMore)
	part2Data := createNiciraFlowStatsReply(of10.OFPSFReplyMore)
	part3Data := createNiciraFlowStatsReply(0)

	buf := bytes.NewBuffer(nil)
	buf.Write(part1Data)
	buf.Write(part2Data)
	buf.Write(part3Data)

	initialLen := buf.Len()
	reader := bufio.NewReader(buf)

	_, err := readFlowStatsReplies(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remaining := reader.Buffered()
	if remaining != 0 {
		t.Errorf("expected all %d bytes to be consumed, but %d bytes remain buffered", initialLen, remaining)
	}
}

func TestGetFlowStats_MultiPartReply_StopsAtFinal(t *testing.T) {
	part1Data := createNiciraFlowStatsReply(of10.OFPSFReplyMore)
	part2Data := createNiciraFlowStatsReply(0)
	extraData := createOpenFlowHeader(0x01, 2, 8, 99)

	buf := bytes.NewBuffer(nil)
	buf.Write(part1Data)
	buf.Write(part2Data)
	buf.Write(extraData)

	reader := bufio.NewReader(buf)

	_, err := readFlowStatsReplies(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	remaining := reader.Buffered()
	if remaining != len(extraData) {
		t.Errorf("expected %d bytes to remain (ECHO request), got %d", len(extraData), remaining)
	}
}
