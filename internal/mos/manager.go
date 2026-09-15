package mos

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// ManagerOptions configures a Manager.
type ManagerOptions struct {
	Identity      messages.Identity
	DialTimeout   time.Duration
	MachInfo      *messages.ListMachInfo
	InboxCapacity int
	Logf          func(string, ...any)
}

// Manager owns the bridge's MOS-side state: outbound peer clients and, in
// device mode, the inbound listener. It is the single entry point the MCP tools
// and web API use to talk MOS.
type Manager struct {
	opts  ManagerOptions
	inbox *Inbox

	mu     sync.RWMutex
	peers  map[string]*Peer
	device *DeviceServer
}

// PeerStatus is a snapshot of one peer's connection state.
type PeerStatus struct {
	PeerConfig
	LowerConnected bool `json:"lowerConnected"`
	UpperConnected bool `json:"upperConnected"`
	QueryConnected bool `json:"queryConnected"`
}

// Status is a snapshot of the whole MOS layer.
type Status struct {
	Identity     messages.Identity `json:"identity"`
	Peers        []PeerStatus      `json:"peers"`
	DeviceActive bool              `json:"deviceActive"`
	InboxCount   int               `json:"inboxCount"`
}

// NewManager creates a manager with no peers and the device server stopped.
func NewManager(opts ManagerOptions) *Manager {
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	return &Manager{
		opts:  opts,
		inbox: NewInbox(opts.InboxCapacity),
		peers: make(map[string]*Peer),
	}
}

// Inbox exposes the inbound-message buffer.
func (m *Manager) Inbox() *Inbox { return m.inbox }

// AddPeer registers (or replaces) an outbound peer by name.
func (m *Manager) AddPeer(cfg PeerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("peer name required")
	}
	if cfg.Host == "" {
		return fmt.Errorf("peer %q: host required", cfg.Name)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.peers[cfg.Name]; ok {
		_ = old.Close()
	}
	m.peers[cfg.Name] = NewPeer(cfg, m.opts.Identity, m.opts.DialTimeout)
	return nil
}

// RemovePeer closes and unregisters a peer.
func (m *Manager) RemovePeer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.peers[name]
	if !ok {
		return fmt.Errorf("unknown peer %q", name)
	}
	_ = p.Close()
	delete(m.peers, name)
	return nil
}

func (m *Manager) peer(name string) (*Peer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.peers[name]
	if !ok {
		return nil, fmt.Errorf("unknown peer %q", name)
	}
	return p, nil
}

// Send routes env to the named peer and returns the reply.
func (m *Manager) Send(ctx context.Context, peerName string, env *messages.Envelope) (*messages.Envelope, error) {
	p, err := m.peer(peerName)
	if err != nil {
		return nil, err
	}
	return p.Request(ctx, env)
}

// Identity returns the configured mosID/ncsID pair.
func (m *Manager) Identity() messages.Identity { return m.opts.Identity }

// StartDevice starts the inbound listener on the given ports (device role).
func (m *Manager) StartDevice(ports []int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.device != nil {
		return fmt.Errorf("device server already running")
	}
	dev := NewDeviceServer(m.opts.Identity, m.inbox, m.opts.MachInfo, m.opts.Logf)
	if err := dev.Listen(ports); err != nil {
		return err
	}
	m.device = dev
	return nil
}

// StopDevice stops the inbound listener if running.
func (m *Manager) StopDevice() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.device != nil {
		_ = m.device.Close()
		m.device = nil
	}
}

// Status returns a snapshot of all peers and the device server.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st := Status{
		Identity:     m.opts.Identity,
		DeviceActive: m.device != nil,
		InboxCount:   len(m.inbox.List(0)),
	}
	for _, p := range m.peers {
		lower, upper, query := p.Connected()
		st.Peers = append(st.Peers, PeerStatus{
			PeerConfig:     p.Config(),
			LowerConnected: lower,
			UpperConnected: upper,
			QueryConnected: query,
		})
	}
	sort.Slice(st.Peers, func(i, j int) bool { return st.Peers[i].Name < st.Peers[j].Name })
	return st
}

// Close shuts down every peer and the device server.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.peers {
		_ = p.Close()
	}
	m.peers = make(map[string]*Peer)
	if m.device != nil {
		_ = m.device.Close()
		m.device = nil
	}
}
