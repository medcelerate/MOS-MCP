// Package messages defines Go representations of MOS (Media Object Server)
// protocol messages as specified in MOS Protocol v2.8.5, along with helpers to
// marshal and unmarshal them to/from the on-the-wire XML form.
//
// Every MOS message shares a common envelope rooted at <mos> carrying a mosID,
// ncsID and messageID, followed by exactly one body element identifying the
// message type. The Envelope type models this with one optional (pointer) field
// per supported body; the non-nil field identifies the message.
package messages

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// Envelope is the root <mos> element common to every MOS message.
//
// Exactly one of the body pointer fields is expected to be set on a valid
// message. Unknown body elements received from a peer are ignored by the XML
// decoder, which provides the forward compatibility the spec calls for.
type Envelope struct {
	XMLName   xml.Name `xml:"mos"`
	MosID     string   `xml:"mosID"`
	NcsID     string   `xml:"ncsID"`
	MessageID string   `xml:"messageID,omitempty"`

	// Profile 0 — Basic connection.
	Heartbeat    *Heartbeat    `xml:"heartbeat,omitempty"`
	ReqMachInfo  *ReqMachInfo  `xml:"reqMachInfo,omitempty"`
	ListMachInfo *ListMachInfo `xml:"listMachInfo,omitempty"`

	// Profile 1 — Basic object workflow.
	MosAck     *MosAck     `xml:"mosAck,omitempty"`
	MosReqObj  *MosReqObj  `xml:"mosReqObj,omitempty"`
	MosObj     *MosObj     `xml:"mosObj,omitempty"`
	MosReqAll  *MosReqAll  `xml:"mosReqAll,omitempty"`
	MosListAll *MosListAll `xml:"mosListAll,omitempty"`

	// Profile 2 — Running order / content list.
	RoCreate        *RoCreate        `xml:"roCreate,omitempty"`
	RoReplace       *RoReplace       `xml:"roReplace,omitempty"`
	RoDelete        *RoDelete        `xml:"roDelete,omitempty"`
	RoAck           *RoAck           `xml:"roAck,omitempty"`
	RoElementAction *RoElementAction `xml:"roElementAction,omitempty"`
	RoReadyToAir    *RoReadyToAir    `xml:"roReadyToAir,omitempty"`

	// Profile 3 — Advanced object workflow (search).
	MosReqObjList *MosReqObjList `xml:"mosReqObjList,omitempty"`
	MosObjList    *MosObjList    `xml:"mosObjList,omitempty"`

	// Profile 4 — Advanced running-order workflow.
	RoStorySend *RoStorySend `xml:"roStorySend,omitempty"`
	RoReqAll    *RoReqAll    `xml:"roReqAll,omitempty"`
	RoListAll   *RoListAll   `xml:"roListAll,omitempty"`
}

// BodyName returns the XML element name of the message body, or "" if no known
// body is set. It is used for routing and logging.
func (e *Envelope) BodyName() string {
	switch {
	case e.Heartbeat != nil:
		return "heartbeat"
	case e.ReqMachInfo != nil:
		return "reqMachInfo"
	case e.ListMachInfo != nil:
		return "listMachInfo"
	case e.MosAck != nil:
		return "mosAck"
	case e.MosReqObj != nil:
		return "mosReqObj"
	case e.MosObj != nil:
		return "mosObj"
	case e.MosReqAll != nil:
		return "mosReqAll"
	case e.MosListAll != nil:
		return "mosListAll"
	case e.RoCreate != nil:
		return "roCreate"
	case e.RoReplace != nil:
		return "roReplace"
	case e.RoDelete != nil:
		return "roDelete"
	case e.RoAck != nil:
		return "roAck"
	case e.RoElementAction != nil:
		return "roElementAction"
	case e.RoReadyToAir != nil:
		return "roReadyToAir"
	case e.MosReqObjList != nil:
		return "mosReqObjList"
	case e.MosObjList != nil:
		return "mosObjList"
	case e.RoStorySend != nil:
		return "roStorySend"
	case e.RoReqAll != nil:
		return "roReqAll"
	case e.RoListAll != nil:
		return "roListAll"
	default:
		return ""
	}
}

// Profile returns the MOS profile number a message body belongs to. It is a
// best-effort classification used to select the correct TCP port for a peer.
func (e *Envelope) Profile() int {
	switch e.BodyName() {
	case "heartbeat", "reqMachInfo", "listMachInfo":
		return 0
	case "mosAck", "mosReqObj", "mosObj", "mosReqAll", "mosListAll":
		return 1
	case "roCreate", "roReplace", "roDelete", "roAck", "roElementAction", "roReadyToAir":
		return 2
	case "mosReqObjList", "mosObjList":
		return 3
	case "roStorySend", "roReqAll", "roListAll":
		return 4
	default:
		return -1
	}
}

// Marshal renders the envelope as a complete XML document including the XML
// declaration, suitable for writing to a MOS socket.
func (e *Envelope) Marshal() ([]byte, error) {
	body, err := xml.MarshalIndent(e, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal mos envelope: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(body)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// Unmarshal parses a single <mos> document into an Envelope.
func Unmarshal(data []byte) (*Envelope, error) {
	var e Envelope
	if err := xml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("unmarshal mos envelope: %w", err)
	}
	return &e, nil
}
