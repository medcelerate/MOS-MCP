package messages

import "time"

// Identity carries the mosID/ncsID pair stamped onto every outbound envelope.
type Identity struct {
	MosID string `json:"mosID"`
	NcsID string `json:"ncsID"`
}

// New returns an envelope pre-populated with the identity. Callers set exactly
// one body field before sending. MessageID is assigned by the connection layer.
func (id Identity) New() *Envelope {
	return &Envelope{MosID: id.MosID, NcsID: id.NcsID}
}

// NowEBU formats the current time in the ISO-8601/EBU form MOS uses for time
// fields, e.g. "2009-04-11T14:22:07,125-05:00".
func NowEBU() string {
	return time.Now().Format("2006-01-02T15:04:05Z07:00")
}

// Heartbeat builds a heartbeat envelope.
func (id Identity) Heartbeat() *Envelope {
	e := id.New()
	e.Heartbeat = &Heartbeat{Time: NowEBU()}
	return e
}

// ReqMachInfo builds a reqMachInfo envelope.
func (id Identity) ReqMachInfo() *Envelope {
	e := id.New()
	e.ReqMachInfo = &ReqMachInfo{}
	return e
}

// MosAck builds a mosAck envelope.
func (id Identity) MosAck(objID, status, desc string) *Envelope {
	e := id.New()
	e.MosAck = &MosAck{ObjID: objID, Status: status, StatusDescription: desc}
	return e
}

// RoAck builds a roAck envelope.
func (id Identity) RoAck(roID, status string) *Envelope {
	e := id.New()
	e.RoAck = &RoAck{RoID: roID, RoStatus: status}
	return e
}
