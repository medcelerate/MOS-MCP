package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/config"
	"github.com/medcelerate/MOS-MCP/internal/mos"
	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GET /api/status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.mgr.Status())
}

// GET /api/config, PUT /api/config
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, http.StatusOK, s.cfg)
	case http.MethodPut:
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if err := incoming.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		*s.cfg = incoming
		if err := s.applyPeers(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := s.persist(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s.cfg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /api/peers, POST /api/peers, DELETE /api/peers?name=...
func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		defer s.mu.Unlock()
		writeJSON(w, http.StatusOK, s.cfg.Peers)

	case http.MethodPost:
		var peer mos.PeerConfig
		if err := json.NewDecoder(r.Body).Decode(&peer); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if peer.Name == "" || peer.Host == "" {
			writeError(w, http.StatusBadRequest, "name and host are required")
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.upsertPeer(peer)
		if err := s.mgr.AddPeer(peer); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.persist(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s.cfg.Peers)

	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "name query parameter required")
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.deletePeer(name)
		_ = s.mgr.RemovePeer(name)
		if err := s.persist(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s.cfg.Peers)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// testResult is the response from POST /api/peers/test.
type testResult struct {
	Peer          string                 `json:"peer"`
	Reachable     bool                   `json:"reachable"`
	HeartbeatType string                 `json:"heartbeatType,omitempty"`
	MachineInfo   *messages.ListMachInfo `json:"machineInfo,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

// POST /api/peers/test — body is a PeerConfig. The peer is registered (or
// updated) in the manager, then probed with a heartbeat and reqMachInfo. This
// is the manual capability check that stands in for MOS's absent discovery.
func (s *Server) handleTestPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var peer mos.PeerConfig
	if err := json.NewDecoder(r.Body).Decode(&peer); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if peer.Host == "" {
		writeError(w, http.StatusBadRequest, "host is required")
		return
	}
	if peer.Name == "" {
		peer.Name = "__test__"
	}
	if err := s.mgr.AddPeer(peer); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	res := testResult{Peer: peer.Name}
	id := s.mgr.Identity()

	hb, err := s.mgr.Send(ctx, peer.Name, id.Heartbeat())
	if err != nil {
		res.Error = "heartbeat: " + err.Error()
		writeJSON(w, http.StatusOK, res)
		return
	}
	res.Reachable = true
	res.HeartbeatType = hb.BodyName()

	mi, err := s.mgr.Send(ctx, peer.Name, id.ReqMachInfo())
	if err != nil {
		res.Error = "reqMachInfo: " + err.Error()
		writeJSON(w, http.StatusOK, res)
		return
	}
	res.MachineInfo = mi.ListMachInfo
	writeJSON(w, http.StatusOK, res)
}

// GET /api/inbox?limit=N
func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = jsonUnmarshalInt(v, &limit)
	}
	writeJSON(w, http.StatusOK, s.mgr.Inbox().List(limit))
}

// --- config mutation helpers (caller holds s.mu) ---

func (s *Server) upsertPeer(peer mos.PeerConfig) {
	for i := range s.cfg.Peers {
		if s.cfg.Peers[i].Name == peer.Name {
			s.cfg.Peers[i] = peer
			return
		}
	}
	s.cfg.Peers = append(s.cfg.Peers, peer)
}

func (s *Server) deletePeer(name string) {
	out := s.cfg.Peers[:0]
	for _, p := range s.cfg.Peers {
		if p.Name != name {
			out = append(out, p)
		}
	}
	s.cfg.Peers = out
}

// applyPeers re-syncs the manager's peers to match the current config.
func (s *Server) applyPeers() error {
	for _, st := range s.mgr.Status().Peers {
		_ = s.mgr.RemovePeer(st.Name)
	}
	for _, p := range s.cfg.Peers {
		if err := s.mgr.AddPeer(p); err != nil {
			return err
		}
	}
	return nil
}

// persist writes the config to disk if a path is configured.
func (s *Server) persist() error {
	if s.cfgPath == "" {
		return nil
	}
	return s.cfg.Save(s.cfgPath)
}

func jsonUnmarshalInt(s string, out *int) (bool, error) {
	return true, json.Unmarshal([]byte(s), out)
}
