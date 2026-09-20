package mcpserver

import (
	"context"
	"fmt"

	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerObjectTools registers Profile 1 (object) and Profile 3 (search)
// tools, each gated on its own profile.
func registerObjectTools(s *mcp.Server, d *deps) {
	if d.cfg.HasProfile(1) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_request_object",
			Description: "Request the description of a single media object from a peer by object ID (Profile 1).",
			Annotations: annRead("Request media object"),
		}, d.requestObject)

		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_request_all_objects",
			Description: "Request descriptions of all media objects from a peer (Profile 1, mosReqAll). Returns a mosListAll of objects.",
			Annotations: annRead("Request all objects"),
		}, d.requestAllObjects)
	}

	if d.cfg.HasProfile(3) {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "mos_search_objects",
			Description: "Search a peer's object database (Profile 3, mosReqObjList) using a general text query with optional paging. Returns matching object descriptions.",
			Annotations: annRead("Search media objects"),
		}, d.searchObjects)
	}
}

// --- mos_request_object ---

type requestObjectIn struct {
	Peer  string `json:"peer" jsonschema:"name of the peer"`
	ObjID string `json:"objID" jsonschema:"the object ID to fetch"`
}

type objectOut struct {
	Peer      string            `json:"peer"`
	Object    *messages.MosObj  `json:"object,omitempty"`
	Objects   []messages.MosObj `json:"objects,omitempty"`
	ReplyType string            `json:"replyType"`
	ReplyXML  string            `json:"replyXML"`
}

func (d *deps) requestObject(ctx context.Context, _ *mcp.CallToolRequest, in requestObjectIn) (*mcp.CallToolResult, objectOut, error) {
	if in.ObjID == "" {
		return nil, objectOut{}, fmt.Errorf("objID is required")
	}
	env := d.mgr.Identity().New()
	env.MosReqObj = &messages.MosReqObj{ObjID: in.ObjID}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, objectOut{}, err
	}
	out := objectOut{Peer: in.Peer, Object: reply.MosObj, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	summary := fmt.Sprintf("Requested object %q from %q; reply %q.", in.ObjID, in.Peer, reply.BodyName())
	if reply.MosObj != nil {
		summary = fmt.Sprintf("Object %q: slug=%q type=%q dur=%q", reply.MosObj.ObjID, reply.MosObj.ObjSlug, reply.MosObj.ObjType, reply.MosObj.ObjDur)
	}
	return textResult(summary), out, nil
}

// --- mos_request_all_objects ---

func (d *deps) requestAllObjects(ctx context.Context, _ *mcp.CallToolRequest, in peerOnlyIn) (*mcp.CallToolResult, objectOut, error) {
	env := d.mgr.Identity().New()
	env.MosReqAll = &messages.MosReqAll{Pause: 0}
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, objectOut{}, err
	}
	out := objectOut{Peer: in.Peer, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	if reply.MosListAll != nil {
		out.Objects = reply.MosListAll.MosObj
	}
	return textResult(fmt.Sprintf("Requested all objects from %q; reply %q with %d object(s).", in.Peer, reply.BodyName(), len(out.Objects))), out, nil
}

// --- mos_search_objects ---

type searchIn struct {
	Peer          string `json:"peer" jsonschema:"name of the peer"`
	GeneralSearch string `json:"generalSearch,omitempty" jsonschema:"free-text search terms"`
	Start         int    `json:"start,omitempty" jsonschema:"first result index to return (paging)"`
	End           int    `json:"end,omitempty" jsonschema:"last result index to return (paging)"`
	Username      string `json:"username,omitempty" jsonschema:"optional username to attribute the query"`
}

type searchOut struct {
	Peer      string            `json:"peer"`
	Objects   []messages.MosObj `json:"objects,omitempty"`
	Total     string            `json:"total,omitempty"`
	Status    string            `json:"status,omitempty"`
	ReplyType string            `json:"replyType"`
	ReplyXML  string            `json:"replyXML"`
}

func (d *deps) searchObjects(ctx context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, searchOut, error) {
	req := &messages.MosReqObjList{
		Username:      in.Username,
		QueryID:       "1",
		GeneralSearch: in.GeneralSearch,
	}
	if in.Start > 0 {
		req.ListReturnStart = fmt.Sprintf("%d", in.Start)
	}
	if in.End > 0 {
		req.ListReturnEnd = fmt.Sprintf("%d", in.End)
	}
	env := d.mgr.Identity().New()
	env.MosReqObjList = req
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, searchOut{}, err
	}
	out := searchOut{Peer: in.Peer, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	if reply.MosObjList != nil {
		out.Total = reply.MosObjList.ListReturnTotal
		out.Status = reply.MosObjList.ListReturnStatus
		if reply.MosObjList.List != nil {
			out.Objects = reply.MosObjList.List.MosObj
		}
	}
	return textResult(fmt.Sprintf("Search on %q returned %d object(s) (total=%s).", in.Peer, len(out.Objects), out.Total)), out, nil
}
