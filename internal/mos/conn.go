package mos

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// Role describes which side of a MOS conversation a connection drives.
type Role int

const (
	// Initiator dials a peer and drives request/response exchanges (NCS-style
	// when talking to a MOS device, or vice versa). Every inbound message is
	// treated as the reply to the most recent request.
	Initiator Role = iota
	// Responder accepts a connection and replies to inbound requests via a
	// handler.
	Responder
)

// Handler produces an optional reply for an inbound message on a responder
// connection. Returning nil sends nothing.
type Handler func(*messages.Envelope) *messages.Envelope

// Conn wraps a single MOS TCP connection and its message framing.
//
// MOS forbids pipelining: a peer must not send another message on a socket
// until the previous one has been acknowledged. Conn honours this for
// initiator connections by serialising Request calls, treating the next inbound
// document as the response.
type Conn struct {
	net    net.Conn
	reader *StreamReader
	id     messages.Identity

	writeMu sync.Mutex
	seq     atomic.Uint64

	readCh chan *messages.Envelope
	errCh  chan error

	reqMu     sync.Mutex // serialises initiator request/response
	closeOnce sync.Once
}

// NewConn wraps an established net.Conn and starts its read loop.
func NewConn(nc net.Conn, id messages.Identity) *Conn {
	c := &Conn{
		net:    nc,
		reader: NewStreamReader(nc),
		id:     id,
		readCh: make(chan *messages.Envelope),
		errCh:  make(chan error, 1),
	}
	go c.readLoop()
	return c
}

func (c *Conn) readLoop() {
	for {
		env, err := c.reader.Next()
		if err != nil {
			select {
			case c.errCh <- err:
			default:
			}
			close(c.readCh)
			return
		}
		c.readCh <- env
	}
}

func (c *Conn) nextMessageID() string {
	return strconv.FormatUint(c.seq.Add(1), 10)
}

// Send stamps a messageID (if absent) onto env and writes it to the socket.
func (c *Conn) Send(env *messages.Envelope) error {
	if env.MosID == "" {
		env.MosID = c.id.MosID
	}
	if env.NcsID == "" {
		env.NcsID = c.id.NcsID
	}
	if env.MessageID == "" {
		env.MessageID = c.nextMessageID()
	}
	data, err := env.Marshal()
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := c.net.Write(data); err != nil {
		return fmt.Errorf("write mos message: %w", err)
	}
	return nil
}

// Request sends env and waits for the next inbound document as its reply. It is
// safe for concurrent use; requests are serialised to satisfy the MOS
// no-pipelining rule.
func (c *Conn) Request(ctx context.Context, env *messages.Envelope) (*messages.Envelope, error) {
	c.reqMu.Lock()
	defer c.reqMu.Unlock()

	if err := c.Send(env); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-c.errCh:
		return nil, fmt.Errorf("connection closed awaiting reply: %w", err)
	case reply, ok := <-c.readCh:
		if !ok {
			return nil, errors.New("connection closed awaiting reply")
		}
		return reply, nil
	}
}

// Serve reads inbound messages and dispatches them to handler until the
// connection closes. Any reply handler returns is written back.
func (c *Conn) Serve(handler Handler) error {
	for {
		select {
		case err := <-c.errCh:
			return err
		case env, ok := <-c.readCh:
			if !ok {
				return <-c.errCh
			}
			if reply := handler(env); reply != nil {
				if err := c.Send(reply); err != nil {
					return err
				}
			}
		}
	}
}

// Close closes the underlying socket.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() { err = c.net.Close() })
	return err
}
