package messages

// Profiles 3 and 4 — object search and advanced running-order workflow.

// SearchField is a single field constraint within a search group.
type SearchField struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

// SearchGroup groups search field constraints for mosReqObjList.
type SearchGroup struct {
	SearchField []SearchField `xml:"searchField,omitempty"`
}

// MosReqObjList queries the device's object database (Profile 3, query port).
type MosReqObjList struct {
	Username        string        `xml:"username,omitempty"`
	QueryID         string        `xml:"queryID,omitempty"`
	ListReturnStart string        `xml:"listReturnStart,omitempty"`
	ListReturnEnd   string        `xml:"listReturnEnd,omitempty"`
	GeneralSearch   string        `xml:"generalSearch,omitempty"`
	MosSchema       string        `xml:"mosSchema,omitempty"`
	SearchGroups    []SearchGroup `xml:"searchGroup,omitempty"`
}

// MosObjList returns a page of object descriptions matching a query.
type MosObjList struct {
	QueryID          string   `xml:"queryID,omitempty"`
	ListReturnStart  string   `xml:"listReturnStart,omitempty"`
	ListReturnEnd    string   `xml:"listReturnEnd,omitempty"`
	ListReturnTotal  string   `xml:"listReturnTotal,omitempty"`
	ListReturnStatus string   `xml:"listReturnStatus,omitempty"`
	List             *ObjList `xml:"list,omitempty"`
}

// ObjList wraps the objects returned in a mosObjList response.
type ObjList struct {
	MosObj []MosObj `xml:"mosObj"`
}

// StoryBodyItem is an inline item reference within a story body.
type StoryBodyItem struct {
	Item
}

// Paragraph is a text paragraph within a story body.
type Paragraph struct {
	Text string `xml:",chardata"`
}

// StoryBody holds the ordered mix of paragraphs and item references that make
// up the narrative body of a story.
type StoryBody struct {
	Items      []Item      `xml:"storyItem,omitempty"`
	Paragraphs []Paragraph `xml:"p,omitempty"`
}

// RoStorySend transmits the full body of a single story (Profile 4).
type RoStorySend struct {
	RoID      string     `xml:"roID"`
	StoryID   string     `xml:"storyID"`
	StorySlug string     `xml:"storySlug,omitempty"`
	StoryNum  string     `xml:"storyNum,omitempty"`
	StoryBody *StoryBody `xml:"storyBody,omitempty"`
}

// RoReqAll requests descriptions of all running orders known to the peer. It
// carries no fields.
type RoReqAll struct{}

// RoListDescription is a lightweight running-order descriptor.
type RoListDescription struct {
	RoID   string `xml:"roID"`
	RoSlug string `xml:"roSlug,omitempty"`
}

// RoListAll returns descriptors for all known running orders.
type RoListAll struct {
	Ro []RoListDescription `xml:"ro"`
}
