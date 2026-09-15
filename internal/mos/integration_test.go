package mos

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// TestClientDeviceRoundTrip stands up a device-role server and a client-role
// peer over a loopback socket, then verifies that a roCreate is acknowledged
// and recorded in the device inbox.
func TestClientDeviceRoundTrip(t *testing.T) {
	devID := messages.Identity{MosID: "device.mos", NcsID: "device.ncs"}
	inbox := NewInbox(10)
	machInfo := &messages.ListMachInfo{Model: "Test Device", MosRev: "2.8.5"}
	dev := NewDeviceServer(devID, inbox, machInfo, nil)
	defer func() { _ = dev.Close() }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	dev.AddListener(ln)

	ncsID := messages.Identity{MosID: "ncs.mos", NcsID: "ncs.ncs"}
	port := ln.Addr().(*net.TCPAddr).Port
	peer := NewPeer(PeerConfig{
		Name:      "dev",
		Host:      "127.0.0.1",
		LowerPort: port,
		UpperPort: port,
		QueryPort: port,
	}, ncsID, 3*time.Second)
	defer func() { _ = peer.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// reqMachInfo on the lower port.
	reply, err := peer.Request(ctx, ncsID.ReqMachInfo())
	if err != nil {
		t.Fatalf("reqMachInfo: %v", err)
	}
	if reply.ListMachInfo == nil || reply.ListMachInfo.Model != "Test Device" {
		t.Fatalf("unexpected machInfo reply: %s", reply.BodyName())
	}

	// roCreate on the upper port.
	create := ncsID.New()
	create.RoCreate = &messages.RoCreate{RoID: "RO1", RoSlug: "Test Show"}
	reply, err = peer.Request(ctx, create)
	if err != nil {
		t.Fatalf("roCreate: %v", err)
	}
	if reply.RoAck == nil || reply.RoAck.RoID != "RO1" {
		t.Fatalf("expected roAck for RO1, got %s", reply.BodyName())
	}

	// The device should have recorded both inbound requests.
	// Allow the server goroutine a moment to append the last entry.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(inbox.List(0)) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	entries := inbox.List(0)
	if len(entries) < 2 {
		t.Fatalf("expected >=2 inbox entries, got %d", len(entries))
	}
	if entries[0].Type != "roCreate" {
		t.Fatalf("newest inbox entry = %q, want roCreate", entries[0].Type)
	}
}
