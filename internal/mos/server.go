package mos

import (
	"fmt"
	"net"
	"sync"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// DeviceServer accepts inbound MOS connections from an NCS while the bridge is
// acting as a MOS device. Incoming messages are recorded to the inbox and
// answered with the appropriate acknowledgement.
type DeviceServer struct {
	id       messages.Identity
	inbox    *Inbox
	machInfo *messages.ListMachInfo
	logf     func(string, ...any)

	mu        sync.Mutex
	listeners []net.Listener
	closed    bool
}

// NewDeviceServer creates a device-role server. machInfo is returned in
// response to reqMachInfo; logf receives operational log lines (may be nil).
func NewDeviceServer(id messages.Identity, inbox *Inbox, machInfo *messages.ListMachInfo, logf func(string, ...any)) *DeviceServer {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &DeviceServer{id: id, inbox: inbox, machInfo: machInfo, logf: logf}
}

// Listen opens a TCP listener on each of the given ports and serves accepted
// connections until Close is called.
func (s *DeviceServer) Listen(ports []int) error {
	seen := map[int]bool{}
	for _, port := range ports {
		if port == 0 || seen[port] {
			continue
		}
		seen[port] = true
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("listen %s: %w", addr, err)
		}
		s.mu.Lock()
		s.listeners = append(s.listeners, ln)
		s.mu.Unlock()
		s.logf("mos device listening on %s", addr)
		go s.acceptLoop(ln)
	}
	return nil
}

// AddListener serves an already-opened listener (used by tests to bind an
// ephemeral port without a bind race).
func (s *DeviceServer) AddListener(ln net.Listener) {
	s.mu.Lock()
	s.listeners = append(s.listeners, ln)
	s.mu.Unlock()
	go s.acceptLoop(ln)
}

func (s *DeviceServer) acceptLoop(ln net.Listener) {
	for {
		nc, err := ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if !closed {
				s.logf("accept error on %s: %v", ln.Addr(), err)
			}
			return
		}
		peer := nc.RemoteAddr().String()
		s.logf("mos device accepted connection from %s", peer)
		conn := NewConn(nc, s.id)
		go func() {
			err := conn.Serve(func(env *messages.Envelope) *messages.Envelope {
				s.inbox.Add(peer, env)
				return s.reply(env)
			})
			s.logf("mos device connection from %s closed: %v", peer, err)
		}()
	}
}

// reply builds the standard acknowledgement for an inbound message.
func (s *DeviceServer) reply(env *messages.Envelope) *messages.Envelope {
	switch {
	case env.Heartbeat != nil:
		return s.id.Heartbeat()
	case env.ReqMachInfo != nil:
		out := s.id.New()
		out.ListMachInfo = s.machInfo
		return out
	case env.MosReqObj != nil:
		return s.id.MosAck(env.MosReqObj.ObjID, messages.StatusAck, "")
	case env.MosObj != nil:
		return s.id.MosAck(env.MosObj.ObjID, messages.StatusAck, "")
	case env.MosReqAll != nil:
		return s.id.MosAck("", messages.StatusAck, "")
	case env.RoCreate != nil:
		return s.id.RoAck(env.RoCreate.RoID, "OK")
	case env.RoReplace != nil:
		return s.id.RoAck(env.RoReplace.RoID, "OK")
	case env.RoDelete != nil:
		return s.id.RoAck(env.RoDelete.RoID, "OK")
	case env.RoElementAction != nil:
		return s.id.RoAck(env.RoElementAction.RoID, "OK")
	case env.RoReadyToAir != nil:
		return s.id.RoAck(env.RoReadyToAir.RoID, "OK")
	case env.RoStorySend != nil:
		return s.id.RoAck(env.RoStorySend.RoID, "OK")
	default:
		return nil
	}
}

// Close stops all listeners.
func (s *DeviceServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, ln := range s.listeners {
		_ = ln.Close()
	}
	s.listeners = nil
	return nil
}
