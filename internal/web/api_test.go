package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/config"
	"github.com/medcelerate/MOS-MCP/internal/mos"
	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

func newTestServer(t *testing.T) (*Server, *mos.Manager) {
	t.Helper()
	cfg := config.Default()
	mgr := mos.NewManager(mos.ManagerOptions{Identity: cfg.Identity()})
	t.Cleanup(mgr.Close)
	// cfgPath empty => in-memory only, no disk writes during tests.
	return New(mgr, &cfg, "", nil), mgr
}

func TestPeersCRUD(t *testing.T) {
	srv, _ := newTestServer(t)
	h := srv.Handler()

	// POST a peer.
	body := `{"name":"viz","host":"10.0.0.5"}`
	req := httptest.NewRequest(http.MethodPost, "/api/peers", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status = %d: %s", rec.Code, rec.Body.String())
	}

	// GET peers.
	req = httptest.NewRequest(http.MethodGet, "/api/peers", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var peers []mos.PeerConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &peers); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(peers) != 1 || peers[0].Name != "viz" {
		t.Fatalf("peers = %+v", peers)
	}

	// DELETE it.
	req = httptest.NewRequest(http.MethodDelete, "/api/peers?name=viz", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "viz") {
		t.Fatalf("peer not deleted: %s", rec.Body.String())
	}
}

func TestStatusEndpoint(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var st mos.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if st.Identity.MosID == "" {
		t.Fatal("identity missing from status")
	}
}

func TestTestPeerAgainstDevice(t *testing.T) {
	// Stand up a device server so the test endpoint has something to probe.
	devID := messages.Identity{MosID: "d.mos", NcsID: "d.ncs"}
	dev := mos.NewDeviceServer(devID, mos.NewInbox(4), &messages.ListMachInfo{Model: "Probe"}, nil)
	t.Cleanup(func() { _ = dev.Close() })
	ln := mustListen(t)
	dev.AddListener(ln)

	srv, _ := newTestServer(t)
	port := portOf(ln)
	body := `{"name":"probe","host":"127.0.0.1","lowerPort":` + itoa(port) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/peers/test", strings.NewReader(body))
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() { srv.Handler().ServeHTTP(rec, req); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("test-peer handler timed out")
	}

	var res testResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !res.Reachable {
		t.Fatalf("expected reachable, got error %q", res.Error)
	}
	if res.MachineInfo == nil || res.MachineInfo.Model != "Probe" {
		t.Fatalf("machineInfo = %+v", res.MachineInfo)
	}
}
