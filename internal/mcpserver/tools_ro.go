package mcpserver

import (
	"context"
	"fmt"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerRoTools registers Profile 2 (running order) and Profile 4 (stories)
// tools, each gated on its own profile.
func registerRoTools(s *mcp.Server, d *deps) {
	if d.cfg.HasProfile(2) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_create_running_order",
			Description: "Create a new running order (rundown/playlist) on a peer with the given stories and items (Profile 2, roCreate).",
			Annotations: annWrite("Create running order"),
		}, d.createRunningOrder)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_replace_running_order",
			Description: "Replace the entire contents of an existing running order on a peer (Profile 2, roReplace).",
			Annotations: annDestructive("Replace running order"),
		}, d.replaceRunningOrder)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_delete_running_order",
			Description: "Delete a running order from a peer by ID (Profile 2, roDelete).",
			Annotations: annDestructive("Delete running order"),
		}, d.deleteRunningOrder)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_element_action",
			Description: "Modify a running order's elements: INSERT, REPLACE, MOVE, SWAP or DELETE stories/items (Profile 2, roElementAction).",
			Annotations: annDestructive("Modify running-order elements"),
		}, d.elementAction)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_ready_to_air",
			Description: "Signal whether a running order is ready to air (Profile 2, roReadyToAir).",
			Annotations: annWrite("Set ready-to-air"),
		}, d.readyToAir)
	}

	if d.cfg.HasProfile(4) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_send_story",
			Description: "Send the full body of a single story, including narrative paragraphs and item references, to a peer (Profile 4, roStorySend).",
			Annotations: annWrite("Send story"),
		}, d.sendStory)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_request_all_running_orders",
			Description: "Request descriptors for all running orders known to a peer (Profile 4, roReqAll).",
			Annotations: annRead("List running orders"),
		}, d.requestAllRunningOrders)
	}
}

// itemInput is a simplified item description used by the RO tools.
type itemInput struct {
	ItemSlug string `json:"itemSlug,omitempty" jsonschema:"human-readable item name"`
	ObjID    string `json:"objID,omitempty" jsonschema:"the MOS object this item references"`
	MosID    string `json:"mosID,omitempty" jsonschema:"the mosID of the device owning the object"`
	Duration string `json:"duration,omitempty" jsonschema:"editorial duration, e.g. 00:00:30:00"`
}

// storyInput is a simplified story description used by the RO tools.
type storyInput struct {
	StoryID   string      `json:"storyID,omitempty" jsonschema:"unique story ID"`
	StorySlug string      `json:"storySlug,omitempty" jsonschema:"human-readable story name"`
	Items     []itemInput `json:"items,omitempty" jsonschema:"items within the story"`
}

func toStories(in []storyInput) []messages.Story {
	out := make([]messages.Story, 0, len(in))
	for _, s := range in {
		story := messages.Story{StoryID: s.StoryID, StorySlug: s.StorySlug}
		for _, it := range s.Items {
			story.Items = append(story.Items, messages.Item{
				ItemSlug:  it.ItemSlug,
				ObjID:     it.ObjID,
				MosID:     it.MosID,
				ItemEdDur: it.Duration,
			})
		}
		out = append(out, story)
	}
	return out
}

type roReplyOut struct {
	Peer      string `json:"peer"`
	RoID      string `json:"roID"`
	RoStatus  string `json:"roStatus,omitempty"`
	ReplyType string `json:"replyType"`
	ReplyXML  string `json:"replyXML"`
}

func (d *deps) roReply(peer, roID string, reply *messages.Envelope, xmlStr string) roReplyOut {
	out := roReplyOut{Peer: peer, RoID: roID, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	if reply.RoAck != nil {
		out.RoStatus = reply.RoAck.RoStatus
		if reply.RoAck.RoID != "" {
			out.RoID = reply.RoAck.RoID
		}
	}
	return out
}

// --- mos_create_running_order ---

type createRoIn struct {
	Peer    string       `json:"peer" jsonschema:"name of the peer"`
	RoID    string       `json:"roID" jsonschema:"unique running-order ID"`
	RoSlug  string       `json:"roSlug,omitempty" jsonschema:"human-readable running-order name"`
	Stories []storyInput `json:"stories,omitempty" jsonschema:"stories in the running order, in order"`
}

func (d *deps) createRunningOrder(ctx context.Context, _ *mcp.CallToolRequest, in createRoIn) (*mcp.CallToolResult, roReplyOut, error) {
	if in.RoID == "" {
		return nil, roReplyOut{}, fmt.Errorf("roID is required")
	}
	env := d.mgr.Identity().New()
	env.RoCreate = &messages.RoCreate{RoID: in.RoID, RoSlug: in.RoSlug, Stories: toStories(in.Stories)}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, roReplyOut{}, err
	}
	out := d.roReply(in.Peer, in.RoID, reply, xmlStr)
	return textResult(fmt.Sprintf("Created running order %q on %q; status %q.", in.RoID, in.Peer, out.RoStatus)), out, nil
}

// --- mos_replace_running_order ---

func (d *deps) replaceRunningOrder(ctx context.Context, _ *mcp.CallToolRequest, in createRoIn) (*mcp.CallToolResult, roReplyOut, error) {
	if in.RoID == "" {
		return nil, roReplyOut{}, fmt.Errorf("roID is required")
	}
	env := d.mgr.Identity().New()
	env.RoReplace = &messages.RoReplace{RoID: in.RoID, RoSlug: in.RoSlug, Stories: toStories(in.Stories)}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, roReplyOut{}, err
	}
	out := d.roReply(in.Peer, in.RoID, reply, xmlStr)
	return textResult(fmt.Sprintf("Replaced running order %q on %q; status %q.", in.RoID, in.Peer, out.RoStatus)), out, nil
}

// --- mos_delete_running_order ---

type deleteRoIn struct {
	Peer string `json:"peer" jsonschema:"name of the peer"`
	RoID string `json:"roID" jsonschema:"the running-order ID to delete"`
}

func (d *deps) deleteRunningOrder(ctx context.Context, _ *mcp.CallToolRequest, in deleteRoIn) (*mcp.CallToolResult, roReplyOut, error) {
	if in.RoID == "" {
		return nil, roReplyOut{}, fmt.Errorf("roID is required")
	}
	env := d.mgr.Identity().New()
	env.RoDelete = &messages.RoDelete{RoID: in.RoID}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, roReplyOut{}, err
	}
	out := d.roReply(in.Peer, in.RoID, reply, xmlStr)
	return textResult(fmt.Sprintf("Deleted running order %q on %q; status %q.", in.RoID, in.Peer, out.RoStatus)), out, nil
}

// --- mos_element_action ---

type elementActionIn struct {
	Peer           string   `json:"peer" jsonschema:"name of the peer"`
	RoID           string   `json:"roID" jsonschema:"the running-order ID"`
	Operation      string   `json:"operation" jsonschema:"one of INSERT, REPLACE, MOVE, SWAP, DELETE"`
	TargetStoryID  string   `json:"targetStoryID,omitempty" jsonschema:"story ID the action targets"`
	TargetItemID   string   `json:"targetItemID,omitempty" jsonschema:"item ID the action targets"`
	SourceStoryIDs []string `json:"sourceStoryIDs,omitempty" jsonschema:"source story IDs (for MOVE/SWAP/DELETE)"`
	SourceItemIDs  []string `json:"sourceItemIDs,omitempty" jsonschema:"source item IDs (for MOVE/SWAP/DELETE)"`
}

func (d *deps) elementAction(ctx context.Context, _ *mcp.CallToolRequest, in elementActionIn) (*mcp.CallToolResult, roReplyOut, error) {
	if in.RoID == "" || in.Operation == "" {
		return nil, roReplyOut{}, fmt.Errorf("roID and operation are required")
	}
	act := &messages.RoElementAction{Operation: in.Operation, RoID: in.RoID}
	if in.TargetStoryID != "" || in.TargetItemID != "" {
		act.ElementTarget = &messages.ElementTarget{StoryID: in.TargetStoryID, ItemID: in.TargetItemID}
	}
	if len(in.SourceStoryIDs) > 0 || len(in.SourceItemIDs) > 0 {
		act.ElementSource = &messages.ElementSource{StoryID: in.SourceStoryIDs, ItemID: in.SourceItemIDs}
	}
	env := d.mgr.Identity().New()
	env.RoElementAction = act
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, roReplyOut{}, err
	}
	out := d.roReply(in.Peer, in.RoID, reply, xmlStr)
	return textResult(fmt.Sprintf("%s on running order %q (%q); status %q.", in.Operation, in.RoID, in.Peer, out.RoStatus)), out, nil
}

// --- mos_ready_to_air ---

type readyToAirIn struct {
	Peer  string `json:"peer" jsonschema:"name of the peer"`
	RoID  string `json:"roID" jsonschema:"the running-order ID"`
	Ready bool   `json:"ready" jsonschema:"true for READY, false for NOT READY"`
}

func (d *deps) readyToAir(ctx context.Context, _ *mcp.CallToolRequest, in readyToAirIn) (*mcp.CallToolResult, roReplyOut, error) {
	if in.RoID == "" {
		return nil, roReplyOut{}, fmt.Errorf("roID is required")
	}
	air := "NOT READY"
	if in.Ready {
		air = "READY"
	}
	env := d.mgr.Identity().New()
	env.RoReadyToAir = &messages.RoReadyToAir{RoID: in.RoID, RoAir: air}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, roReplyOut{}, err
	}
	out := d.roReply(in.Peer, in.RoID, reply, xmlStr)
	return textResult(fmt.Sprintf("Running order %q on %q marked %s; status %q.", in.RoID, in.Peer, air, out.RoStatus)), out, nil
}

// --- mos_send_story ---

type sendStoryIn struct {
	Peer       string   `json:"peer" jsonschema:"name of the peer"`
	RoID       string   `json:"roID" jsonschema:"the running-order ID the story belongs to"`
	StoryID    string   `json:"storyID" jsonschema:"the story ID"`
	StorySlug  string   `json:"storySlug,omitempty" jsonschema:"human-readable story name"`
	Paragraphs []string `json:"paragraphs,omitempty" jsonschema:"narrative body paragraphs"`
}

func (d *deps) sendStory(ctx context.Context, _ *mcp.CallToolRequest, in sendStoryIn) (*mcp.CallToolResult, roReplyOut, error) {
	if in.RoID == "" || in.StoryID == "" {
		return nil, roReplyOut{}, fmt.Errorf("roID and storyID are required")
	}
	body := &messages.StoryBody{}
	for _, p := range in.Paragraphs {
		body.Paragraphs = append(body.Paragraphs, messages.Paragraph{Text: p})
	}
	env := d.mgr.Identity().New()
	env.RoStorySend = &messages.RoStorySend{RoID: in.RoID, StoryID: in.StoryID, StorySlug: in.StorySlug, StoryBody: body}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, roReplyOut{}, err
	}
	out := d.roReply(in.Peer, in.RoID, reply, xmlStr)
	return textResult(fmt.Sprintf("Sent story %q to %q; status %q.", in.StoryID, in.Peer, out.RoStatus)), out, nil
}

// --- mos_request_all_running_orders ---

func (d *deps) requestAllRunningOrders(ctx context.Context, _ *mcp.CallToolRequest, in peerOnlyIn) (*mcp.CallToolResult, replyOut, error) {
	env := d.mgr.Identity().New()
	env.RoReqAll = &messages.RoReqAll{}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, replyOut{}, err
	}
	out := replyOut{Peer: in.Peer, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	return textResult(fmt.Sprintf("Requested all running orders from %q; reply %q.", in.Peer, reply.BodyName())), out, nil
}
