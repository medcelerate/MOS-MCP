package messages

// Profile 1 — Basic object workflow: media object metadata exchange.

// Common MOS acknowledgement status values.
const (
	StatusAck  = "ACK"
	StatusNack = "NACK"
)

// MosAck acknowledges an object message on the lower port.
type MosAck struct {
	ObjID             string `xml:"objID,omitempty"`
	ObjRev            string `xml:"objRev,omitempty"`
	Status            string `xml:"status,omitempty"`
	StatusDescription string `xml:"statusDescription,omitempty"`
}

// MosReqObj requests the description of a single object by ID.
type MosReqObj struct {
	ObjID string `xml:"objID"`
}

// ObjPath describes one representation (proxy, path or metadata) of an object.
type ObjPath struct {
	TechDescription string `xml:"techDescription,attr,omitempty"`
	Value           string `xml:",chardata"`
}

// ObjPaths groups the different paths available for an object.
type ObjPaths struct {
	ObjProxyPath    []ObjPath `xml:"objProxyPath,omitempty"`
	ObjMetadataPath []ObjPath `xml:"objMetadataPath,omitempty"`
	ObjPath         []ObjPath `xml:"objPath,omitempty"`
}

// MosObj is the description of a single media object.
type MosObj struct {
	ObjID       string    `xml:"objID,omitempty"`
	ObjSlug     string    `xml:"objSlug,omitempty"`
	MosAbstract string    `xml:"mosAbstract,omitempty"`
	ObjGroup    string    `xml:"objGroup,omitempty"`
	ObjType     string    `xml:"objType,omitempty"`
	ObjTB       string    `xml:"objTB,omitempty"`
	ObjRev      string    `xml:"objRev,omitempty"`
	ObjDur      string    `xml:"objDur,omitempty"`
	Status      string    `xml:"status,omitempty"`
	ObjAir      string    `xml:"objAir,omitempty"`
	ObjPaths    *ObjPaths `xml:"objPaths,omitempty"`
	CreatedBy   string    `xml:"createdBy,omitempty"`
	Created     string    `xml:"created,omitempty"`
	ChangedBy   string    `xml:"changedBy,omitempty"`
	Changed     string    `xml:"changed,omitempty"`
	Description string    `xml:"description,omitempty"`
}

// MosReqAll requests that the device send descriptions of all its objects. The
// Pause field (seconds) requests a delay between successive mosObj messages;
// 0 requests a single mosListAll response instead.
type MosReqAll struct {
	Pause int `xml:"pause"`
}

// MosListAll returns all object descriptions in a single message.
type MosListAll struct {
	MosObj []MosObj `xml:"mosObj"`
}
