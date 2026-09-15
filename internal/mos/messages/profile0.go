package messages

// Profile 0 — Basic connection and machine-information messages.

// Heartbeat verifies that a connection between two MOS peers is alive. It is
// sent periodically and echoed back by the receiver.
type Heartbeat struct {
	Time string `xml:"time,omitempty"`
}

// ReqMachInfo requests the receiving device's machine information. It carries
// no fields.
type ReqMachInfo struct{}

// MosProfile reports support for a single MOS profile within a
// SupportedProfiles block, e.g. <mosProfile number="0">YES</mosProfile>.
type MosProfile struct {
	Number    int    `xml:"number,attr"`
	Supported string `xml:",chardata"` // "YES" or "NO"
}

// SupportedProfiles lists which MOS profiles a device supports.
type SupportedProfiles struct {
	DeviceType string       `xml:"deviceType,attr,omitempty"`
	Profiles   []MosProfile `xml:"mosProfile"`
}

// ListMachInfo is the response to reqMachInfo and describes the device.
type ListMachInfo struct {
	Manufacturer      string             `xml:"manufacturer,omitempty"`
	Model             string             `xml:"model,omitempty"`
	HwRev             string             `xml:"hwRev,omitempty"`
	SwRev             string             `xml:"swRev,omitempty"`
	DOM               string             `xml:"DOM,omitempty"`
	SN                string             `xml:"SN,omitempty"`
	ID                string             `xml:"ID,omitempty"`
	Time              string             `xml:"time,omitempty"`
	OpTime            string             `xml:"opTime,omitempty"`
	MosRev            string             `xml:"mosRev,omitempty"`
	SupportedProfiles *SupportedProfiles `xml:"supportedProfiles,omitempty"`
}
