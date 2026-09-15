package mos

import (
	"sync"
	"time"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
)

// InboxEntry is a single inbound message recorded while acting as a MOS device.
type InboxEntry struct {
	Received time.Time          `json:"received"`
	Peer     string             `json:"peer"`
	Type     string             `json:"type"`
	Envelope *messages.Envelope `json:"-"`
	XML      string             `json:"xml"`
}

// Inbox is a fixed-capacity ring buffer of recently received messages. It lets
// an AI client inspect what a connected NCS has pushed to the bridge.
type Inbox struct {
	mu      sync.Mutex
	entries []InboxEntry
	cap     int
}

// NewInbox creates an inbox retaining up to capacity entries.
func NewInbox(capacity int) *Inbox {
	if capacity <= 0 {
		capacity = 100
	}
	return &Inbox{cap: capacity}
}

// Add records an inbound envelope from the named peer.
func (b *Inbox) Add(peer string, env *messages.Envelope) {
	xmlBytes, _ := env.Marshal()
	entry := InboxEntry{
		Received: time.Now(),
		Peer:     peer,
		Type:     env.BodyName(),
		Envelope: env,
		XML:      string(xmlBytes),
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, entry)
	if len(b.entries) > b.cap {
		b.entries = b.entries[len(b.entries)-b.cap:]
	}
}

// List returns up to limit most-recent entries, newest first. limit <= 0
// returns all retained entries.
func (b *Inbox) List(limit int) []InboxEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(b.entries)
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]InboxEntry, 0, n)
	for i := len(b.entries) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, b.entries[i])
	}
	return out
}
