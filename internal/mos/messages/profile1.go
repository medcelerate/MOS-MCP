package messages

// Profile 1 — Basic object workflow: media object metadata exchange.

// Common MOS acknowledgement status values.
const (
	StatusAck  = "ACK"
	StatusNack = "NACK"
)

// MosAck acknowledges an object message on the lower port.
type MosAck struct {
	ObjID             string `xml:"objID,omitempty" json:"objID,omitempty"`
	ObjRev            string `xml:"objRev,omitempty" json:"objRev,omitempty"`
	Status            string `xml:"status,omitempty" json:"status,omitempty"`
	StatusDescription string `xml:"statusDescription,omitempty" json:"statusDescription,omitempty"`
}

// MosReqObj requests the description of a single object by ID.
type MosReqObj struct {
	ObjID string `xml:"objID" json:"objID"`
}

// ObjPath describes one representation (proxy, path or metadata) of an object.
type ObjPath struct {
	TechDescription string `xml:"techDescription,attr,omitempty" json:"techDescription,omitempty"`
	Value           string `xml:",chardata" json:"value,omitempty"`
}

// ObjPaths groups the different paths available for an object.
type ObjPaths struct {
	ObjProxyPath    []ObjPath `xml:"objProxyPath,omitempty" json:"objProxyPath,omitempty"`
	ObjMetadataPath []ObjPath `xml:"objMetadataPath,omitempty" json:"objMetadataPath,omitempty"`
	ObjPath         []ObjPath `xml:"objPath,omitempty" json:"objPath,omitempty"`
}

// MosObj is the description of a single media object.
type MosObj struct {
	ObjID       string    `xml:"objID,omitempty" json:"objID,omitempty"`
	ObjSlug     string    `xml:"objSlug,omitempty" json:"objSlug,omitempty"`
	MosAbstract string    `xml:"mosAbstract,omitempty" json:"mosAbstract,omitempty"`
	ObjGroup    string    `xml:"objGroup,omitempty" json:"objGroup,omitempty"`
	ObjType     string    `xml:"objType,omitempty" json:"objType,omitempty"`
	ObjTB       string    `xml:"objTB,omitempty" json:"objTB,omitempty"`
	ObjRev      string    `xml:"objRev,omitempty" json:"objRev,omitempty"`
	ObjDur      string    `xml:"objDur,omitempty" json:"objDur,omitempty"`
	Status      string    `xml:"status,omitempty" json:"status,omitempty"`
	ObjAir      string    `xml:"objAir,omitempty" json:"objAir,omitempty"`
	ObjPaths    *ObjPaths `xml:"objPaths,omitempty" json:"objPaths,omitempty"`
	CreatedBy   string    `xml:"createdBy,omitempty" json:"createdBy,omitempty"`
	Created     string    `xml:"created,omitempty" json:"created,omitempty"`
	ChangedBy   string    `xml:"changedBy,omitempty" json:"changedBy,omitempty"`
	Changed     string    `xml:"changed,omitempty" json:"changed,omitempty"`
	Description string    `xml:"description,omitempty" json:"description,omitempty"`
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
