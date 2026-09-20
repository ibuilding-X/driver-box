package internal

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/orglibs/go-iecp5/asdu"
)

// testStation is a small wire-level peer, independent of the fork's server. It
// acknowledges I frames and echoes command ACT_CON so the real client transport,
// STARTDT, address mapping, and select/execute ordering are exercised together.
type testStation struct {
	conn            net.Conn
	observed        chan []byte
	sendSequence    uint16
	receiveSequence uint16
	reject          bool
	silence         bool
}

func (s *testStation) writeASDU(raw []byte) error {
	frame := make([]byte, 6+len(raw))
	frame[0] = 0x68
	frame[1] = byte(4 + len(raw))
	binary.LittleEndian.PutUint16(frame[2:4], s.sendSequence<<1)
	binary.LittleEndian.PutUint16(frame[4:6], s.receiveSequence<<1)
	copy(frame[6:], raw)
	s.sendSequence++
	_, err := s.conn.Write(frame)
	return err
}
func (s *testStation) run() {
	defer s.conn.Close()
	for {
		head := make([]byte, 2)
		if _, err := io.ReadFull(s.conn, head); err != nil {
			return
		}
		frame := make([]byte, int(head[1]))
		if _, err := io.ReadFull(s.conn, frame); err != nil {
			return
		}
		if len(frame) < 4 {
			return
		}
		if frame[0]&3 == 3 {
			switch frame[0] {
			case 7:
				_, _ = s.conn.Write([]byte{0x68, 4, 0x0b, 0, 0, 0})
			case 0x43:
				_, _ = s.conn.Write([]byte{0x68, 4, 0x83, 0, 0, 0})
			}
			continue
		}
		if frame[0]&1 != 0 {
			continue
		}
		s.receiveSequence = binary.LittleEndian.Uint16(frame[:2])>>1 + 1
		raw := append([]byte(nil), frame[4:]...)
		s.observed <- raw
		ack := []byte{0x68, 4, 1, 0, 0, 0}
		binary.LittleEndian.PutUint16(ack[4:], s.receiveSequence<<1)
		if _, err := s.conn.Write(ack); err != nil {
			return
		}
		switch asdu.TypeID(raw[0]) {
		case asdu.C_IC_NA_1:
			ca := binary.LittleEndian.Uint16(raw[4:6])
			addresses := []uint32{1001, 2001}
			if ca == 2 {
				addresses = []uint32{1}
			}
			for _, ioa := range addresses {
				value := []byte{1, 1, 20, 0, byte(ca), byte(ca >> 8), byte(ioa), byte(ioa >> 8), byte(ioa >> 16), 1}
				if err := s.writeASDU(value); err != nil {
					return
				}
			}
		case asdu.C_SC_NA_1:
			if s.silence {
				continue
			}
			response := append([]byte(nil), raw...)
			response[2] = 7
			if s.reject {
				response[2] |= 0x40
			}
			if err := s.writeASDU(response); err != nil {
				return
			}
		}
	}
}

func startTestConnector(t *testing.T, reject, silence bool) (*connector, <-chan []byte, <-chan plugin.DeviceData) {
	t.Helper()
	observed := make(chan []byte, 64)
	var peers sync.WaitGroup
	s, cfg := fixture()
	s.commandTimeout = 200 * time.Millisecond
	s.dialContext = func(ctx context.Context, _ *url.URL) (net.Conn, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		client, server := net.Pipe()
		peers.Add(1)
		go func() {
			defer peers.Done()
			(&testStation{conn: server, observed: observed, reject: reject, silence: silence}).run()
		}()
		return client, nil
	}
	reports := make(chan plugin.DeviceData, 64)
	c, err := newConnector(s, cfg, func(data []plugin.DeviceData) {
		for _, d := range data {
			reports <- d
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(); peers.Wait() })
	return c, observed, reports
}

func nextFrame(t *testing.T, frames <-chan []byte, kind byte) []byte {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case raw := <-frames:
			if raw[0] == kind {
				return raw
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for type %d", kind)
			return nil
		}
	}
}

func TestMasterMultipleDevicesAndControl(t *testing.T) {
	c, frames, reports := startTestConnector(t, false, false)
	first, second := nextFrame(t, frames, 100), nextFrame(t, frames, 100)
	if first[4] != 1 || second[4] != 2 {
		t.Fatalf("GI should be once per CA: %x %x", first, second)
	}
	got := make(map[string]bool)
	for len(got) < 3 {
		select {
		case report := <-reports:
			if len(report.Values) != 1 || report.Values[0].PointName != "switch" || report.Values[0].Value != 1 {
				t.Fatal(report)
			}
			got[report.ID] = true
		case <-time.After(3 * time.Second):
			t.Fatalf("missing device reports: %v", got)
		}
	}
	for _, id := range []string{"a", "b", "c"} {
		if !got[id] {
			t.Fatalf("missing %s", id)
		}
	}
	req, err := c.Encode("a", plugin.WriteMode, plugin.PointData{PointName: "switch", Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Send(req); err != nil {
		t.Fatal(err)
	}
	selected, executed := nextFrame(t, frames, 45), nextFrame(t, frames, 45)
	for _, raw := range [][]byte{selected, executed} {
		if raw[4] != 1 || raw[6] != 0xb9 || raw[7] != 0x0b {
			t.Fatalf("wrong device control IOA, expected 3001: %x", raw)
		}
	}
	if selected[9] != 0x81 || executed[9] != 1 {
		t.Fatalf("wrong select/execute: %x %x", selected, executed)
	}
	if err = c.Release(); err != nil || !c.client.IsActive() {
		t.Fatal("Release closed shared connection")
	}
	read, err := c.Encode("b", plugin.ReadMode, plugin.PointData{PointName: "switch"})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Send(read); err != nil {
		t.Fatal(err)
	}
	raw := nextFrame(t, frames, 102)
	if raw[6] != 0xd1 || raw[7] != 7 {
		t.Fatalf("wrong read IOA, expected 2001: %x", raw)
	}
	if _, err = c.Encode("missing", plugin.ReadMode, plugin.PointData{PointName: "switch"}); err == nil {
		t.Fatal("unknown device accepted")
	}
	// Force a transport reconnect; every CA must be interrogated again.
	c.client.Disconnect()
	nextFrame(t, frames, 100)
	nextFrame(t, frames, 100)
}

func TestRejectedSelectNeverExecutes(t *testing.T) {
	c, frames, _ := startTestConnector(t, true, false)
	nextFrame(t, frames, 100)
	nextFrame(t, frames, 100)
	req, err := c.Encode("a", plugin.WriteMode, plugin.PointData{PointName: "switch", Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Send(req); err == nil {
		t.Fatal("negative confirmation accepted")
	}
	raw := nextFrame(t, frames, 45)
	if raw[9]&0x80 == 0 {
		t.Fatal("execute before select")
	}
	select {
	case raw := <-frames:
		t.Fatalf("unexpected command after rejected select: %x", raw)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestCommandTimeoutAndClose(t *testing.T) {
	c, frames, _ := startTestConnector(t, false, true)
	nextFrame(t, frames, 100)
	nextFrame(t, frames, 100)
	req, _ := c.Encode("a", plugin.WriteMode, plugin.PointData{PointName: "switch", Value: 1})
	if err := c.Send(req); err == nil {
		t.Fatal("missing confirmation accepted")
	}
	if err := c.Send(req); err == nil {
		t.Fatal("ambiguous session reused")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Encode("a", plugin.ReadMode, plugin.PointData{PointName: "switch"}); err == nil {
		t.Fatal("closed connector accepted operation")
	}
}

func TestConfirmationMatchesValueAndSelection(t *testing.T) {
	s, cfg := fixture()
	c, err := newConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.cancel()
	req, _ := c.Encode("a", plugin.WriteMode, plugin.PointData{PointName: "switch", Value: 1})
	frame := req.(*request).operations[0].frame
	c.pending = &pendingCommand{frame: frame, result: make(chan error, 1)}
	wrong := frame.Clone()
	wrong.Coa.Cause = asdu.ActivationCon
	wrong.InfoObj[len(wrong.InfoObj)-1] &= 0x7f
	_ = c.confirm(wrong)
	select {
	case <-c.pending.result:
		t.Fatal("execute response confirmed selection")
	default:
	}
	right := frame.Clone()
	right.Coa.Cause = asdu.ActivationCon
	_ = c.confirm(right)
	select {
	case err := <-c.pending.result:
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("confirmation not delivered")
	}
}

func TestHandlerFiltersAndRoutesSpontaneous(t *testing.T) {
	s, cfg := fixture()
	c, err := newConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.cancel()
	for _, tt := range []struct {
		name               string
		ca                 uint16
		ioa                uint32
		kind, cot, quality byte
		expected           string
	}{
		{"device a", 1, 1001, 1, 3, 1, "a"}, {"device b", 1, 2001, 1, 3, 0, "b"}, {"other CA", 2, 1, 1, 3, 1, "c"},
		{"unknown address", 1, 999, 1, 3, 1, ""}, {"test ASDU", 1, 1001, 1, 0x83, 1, ""}, {"invalid", 1, 1001, 1, 3, 0x81, ""},
		{"not topical", 1, 1001, 1, 3, 0x41, ""}, {"wrong type family", 1, 1001, 3, 3, 1, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte{tt.kind, 1, tt.cot, 0, byte(tt.ca), byte(tt.ca >> 8), byte(tt.ioa), byte(tt.ioa >> 8), byte(tt.ioa >> 16), tt.quality}
			a := asdu.NewEmptyASDU(asdu.ParamsWide)
			if err := a.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			if err := c.ASDUHandler(nil, a); err != nil {
				t.Fatal(err)
			}
			select {
			case data := <-c.telemetry:
				if tt.expected == "" || len(data) != 1 || data[0].ID != tt.expected {
					t.Fatalf("unexpected data %+v", data)
				}
			default:
				if tt.expected != "" {
					t.Fatal("missing report")
				}
			}
		})
	}
}
