package messages

import (
	"strings"
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	id := Identity{MosID: "bridge.mos", NcsID: "news.ncs"}

	cases := []struct {
		name     string
		build    func() *Envelope
		wantBody string
	}{
		{"heartbeat", id.Heartbeat, "heartbeat"},
		{"reqMachInfo", id.ReqMachInfo, "reqMachInfo"},
		{
			"roCreate",
			func() *Envelope {
				e := id.New()
				e.RoCreate = &RoCreate{
					RoID:   "RO123",
					RoSlug: "Morning Show",
					Stories: []Story{{
						StoryID:   "S1",
						StorySlug: "Lead",
						Items:     []Item{{ItemID: "I1", ObjID: "OBJ9", ItemEdDur: "00:00:30:00"}},
					}},
				}
				return e
			},
			"roCreate",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := tc.build()
			data, err := env.Marshal()
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(string(data), "<"+tc.wantBody+">") {
				t.Fatalf("marshaled XML missing <%s>:\n%s", tc.wantBody, data)
			}
			got, err := Unmarshal(data)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.MosID != env.MosID || got.NcsID != env.NcsID {
				t.Fatalf("identity mismatch: got %s/%s", got.MosID, got.NcsID)
			}
			if got.BodyName() != tc.wantBody {
				t.Fatalf("BodyName = %q, want %q", got.BodyName(), tc.wantBody)
			}
		})
	}
}

func TestRoCreateFieldFidelity(t *testing.T) {
	xml := `<?xml version="1.0"?>
<mos>
  <mosID>a.mos</mosID>
  <ncsID>b.ncs</ncsID>
  <messageID>42</messageID>
  <roCreate>
    <roID>RO1</roID>
    <roSlug>Show</roSlug>
    <story>
      <storyID>S1</storyID>
      <item><itemID>I1</itemID><objID>O1</objID></item>
      <item><itemID>I2</itemID><objID>O2</objID></item>
    </story>
    <unknownFutureTag>ignored</unknownFutureTag>
  </roCreate>
</mos>`
	env, err := Unmarshal([]byte(xml))
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.RoCreate == nil {
		t.Fatal("roCreate not parsed")
	}
	if env.MessageID != "42" {
		t.Fatalf("messageID = %q, want 42", env.MessageID)
	}
	if len(env.RoCreate.Stories) != 1 || len(env.RoCreate.Stories[0].Items) != 2 {
		t.Fatalf("story/item counts wrong: %+v", env.RoCreate.Stories)
	}
	if env.RoCreate.Stories[0].Items[1].ObjID != "O2" {
		t.Fatalf("second item objID = %q, want O2", env.RoCreate.Stories[0].Items[1].ObjID)
	}
}

func TestProfileClassification(t *testing.T) {
	id := Identity{}
	if p := id.Heartbeat().Profile(); p != 0 {
		t.Fatalf("heartbeat profile = %d, want 0", p)
	}
	e := id.New()
	e.MosReqObjList = &MosReqObjList{}
	if p := e.Profile(); p != 3 {
		t.Fatalf("mosReqObjList profile = %d, want 3", p)
	}
}
