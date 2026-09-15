package mos

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// PeerConfig describes how to reach a remote MOS peer.
type PeerConfig struct {
	Name      string `json:"name" yaml:"name"`
	Host      string `json:"host" yaml:"host"`
	LowerPort int    `json:"lowerPort" yaml:"lowerPort"`
	UpperPort int    `json:"upperPort" yaml:"upperPort"`
	QueryPort int    `json:"queryPort" yaml:"queryPort"`
}

// withDefaults fills the standard MOS ports when unset.
func (c PeerConfig) withDefaults() PeerConfig {
	if c.LowerPort == 0 {
		c.LowerPort = 10540
	}
	if c.UpperPort == 0 {
		c.UpperPort = 10541
	}
	if c.QueryPort == 0 {
		c.QueryPort = 10542
	}
	return c
}

// Peer is an outbound (initiator) connection to a remote MOS peer. It lazily
// dials the lower, upper and query ports as different message classes require
// them, and keeps the sockets open between requests.
type Peer struct {
	cfg         PeerConfig
	id          messages.Identity
	dialTimeout time.Duration

	mu    sync.Mutex
	lower *Conn
	upper *Conn
	query *Conn
}

// NewPeer creates a peer client. It does not dial until the first request.
func NewPeer(cfg PeerConfig, id messages.Identity, dialTimeout time.Duration) *Peer {
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}
	return &Peer{cfg: cfg.withDefaults(), id: id, dialTimeout: dialTimeout}
}

// Config returns the peer's configuration.
func (p *Peer) Config() PeerConfig { return p.cfg }

func (p *Peer) dial(port int) (*Conn, error) {
	addr := net.JoinHostPort(p.cfg.Host, fmt.Sprintf("%d", port))
	nc, err := net.DialTimeout("tcp", addr, p.dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return NewConn(nc, p.id), nil
}

// connFor returns the connection appropriate for the given envelope, dialing
// it if necessary. The pointer-to-pointer lets us cache the dialed conn.
func (p *Peer) connFor(env *messages.Envelope) (*Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var target **Conn
	var port int
	switch {
	case env.MosReqObjList != nil || env.MosObjList != nil:
		target, port = &p.query, p.cfg.QueryPort
	case env.Profile() == 2 || env.Profile() == 4:
		target, port = &p.upper, p.cfg.UpperPort
	default:
		target, port = &p.lower, p.cfg.LowerPort
	}
	if *target == nil {
		conn, err := p.dial(port)
		if err != nil {
			return nil, err
		}
		*target = conn
	}
	return *target, nil
}

// Request sends env to the peer on the correct port and returns the reply.
func (p *Peer) Request(ctx context.Context, env *messages.Envelope) (*messages.Envelope, error) {
	conn, err := p.connFor(env)
	if err != nil {
		return nil, err
	}
	reply, err := conn.Request(ctx, env)
	if err != nil {
		// Drop the (likely broken) connection so the next call redials.
		p.reset(conn)
		return nil, err
	}
	return reply, nil
}

// reset clears any cached reference to conn after a failure.
func (p *Peer) reset(conn *Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_ = conn.Close()
	if p.lower == conn {
		p.lower = nil
	}
	if p.upper == conn {
		p.upper = nil
	}
	if p.query == conn {
		p.query = nil
	}
}

// Connected reports which ports currently hold an open socket.
func (p *Peer) Connected() (lower, upper, query bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lower != nil, p.upper != nil, p.query != nil
}

// Close closes all open sockets to the peer.
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range []*Conn{p.lower, p.upper, p.query} {
		if c != nil {
			_ = c.Close()
		}
	}
	p.lower, p.upper, p.query = nil, nil, nil
	return nil
}
