package messages

// Profile 2 — Running order (content list) construction and control.

// Item is a single playable element within a story (a reference to a MOS
// object plus timing and trigger metadata).
type Item struct {
	ItemID            string `xml:"itemID,omitempty" json:"itemID,omitempty"`
	ItemSlug          string `xml:"itemSlug,omitempty" json:"itemSlug,omitempty"`
	ObjID             string `xml:"objID,omitempty" json:"objID,omitempty"`
	MosID             string `xml:"mosID,omitempty" json:"mosID,omitempty"`
	MosPlugInID       string `xml:"mosPlugInID,omitempty" json:"mosPlugInID,omitempty"`
	MosAbstract       string `xml:"mosAbstract,omitempty" json:"mosAbstract,omitempty"`
	ObjSlug           string `xml:"objSlug,omitempty" json:"objSlug,omitempty"`
	ObjDur            string `xml:"objDur,omitempty" json:"objDur,omitempty"`
	ObjTB             string `xml:"objTB,omitempty" json:"objTB,omitempty"`
	ItemEdStart       string `xml:"itemEdStart,omitempty" json:"itemEdStart,omitempty"`
	ItemEdDur         string `xml:"itemEdDur,omitempty" json:"itemEdDur,omitempty"`
	ItemUserTimingDur string `xml:"itemUserTimingDur,omitempty" json:"itemUserTimingDur,omitempty"`
	ItemTrigger       string `xml:"itemTrigger,omitempty" json:"itemTrigger,omitempty"`
	MacroIn           string `xml:"macroIn,omitempty" json:"macroIn,omitempty"`
	MacroOut          string `xml:"macroOut,omitempty" json:"macroOut,omitempty"`
}

// Story is a group of items within a running order.
type Story struct {
	StoryID   string `xml:"storyID,omitempty" json:"storyID,omitempty"`
	StorySlug string `xml:"storySlug,omitempty" json:"storySlug,omitempty"`
	StoryNum  string `xml:"storyNum,omitempty" json:"storyNum,omitempty"`
	Items     []Item `xml:"item" json:"items,omitempty"`
}

// RoCreate creates a new running order (playlist) on the receiving device.
type RoCreate struct {
	RoID      string  `xml:"roID"`
	RoSlug    string  `xml:"roSlug,omitempty"`
	RoChannel string  `xml:"roChannel,omitempty"`
	RoEdStart string  `xml:"roEdStart,omitempty"`
	RoEdDur   string  `xml:"roEdDur,omitempty"`
	RoTrigger string  `xml:"roTrigger,omitempty"`
	Stories   []Story `xml:"story"`
}

// RoReplace replaces the entire contents of an existing running order. It uses
// the same structure as RoCreate.
type RoReplace struct {
	RoID      string  `xml:"roID"`
	RoSlug    string  `xml:"roSlug,omitempty"`
	RoChannel string  `xml:"roChannel,omitempty"`
	RoEdStart string  `xml:"roEdStart,omitempty"`
	RoEdDur   string  `xml:"roEdDur,omitempty"`
	RoTrigger string  `xml:"roTrigger,omitempty"`
	Stories   []Story `xml:"story"`
}

// RoDelete removes a running order from the device.
type RoDelete struct {
	RoID string `xml:"roID"`
}

// RoAck acknowledges a running-order message on the upper port.
type RoAck struct {
	RoID     string `xml:"roID,omitempty" json:"roID,omitempty"`
	RoStatus string `xml:"roStatus,omitempty" json:"roStatus,omitempty"`
}

// Element-action operation values for RoElementAction.
const (
	ElementActionInsert  = "INSERT"
	ElementActionReplace = "REPLACE"
	ElementActionMove    = "MOVE"
	ElementActionSwap    = "SWAP"
	ElementActionDelete  = "DELETE"
)

// ElementSource carries the stories or items an action operates with.
type ElementSource struct {
	StoryID []string `xml:"storyID,omitempty"`
	ItemID  []string `xml:"itemID,omitempty"`
	Stories []Story  `xml:"story,omitempty"`
	Items   []Item   `xml:"item,omitempty"`
}

// ElementTarget identifies the position an action applies to.
type ElementTarget struct {
	StoryID string `xml:"storyID,omitempty"`
	ItemID  string `xml:"itemID,omitempty"`
}

// RoElementAction modifies the elements of an existing running order. The
// Operation attribute selects INSERT, REPLACE, MOVE, SWAP or DELETE.
type RoElementAction struct {
	Operation     string         `xml:"operation,attr"`
	RoID          string         `xml:"roID"`
	ElementTarget *ElementTarget `xml:"element_target,omitempty"`
	ElementSource *ElementSource `xml:"element_source,omitempty"`
}

// RoReadyToAir signals whether a running order is cleared to air.
type RoReadyToAir struct {
	RoID  string `xml:"roID"`
	RoAir string `xml:"roAir"` // "READY" or "NOT READY"
}
