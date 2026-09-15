package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/medcelerate/MOS-MCP/internal/mos"
	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerConnTools registers connection-management and Profile 0 tools. These
// are always available regardless of enabled profiles.
func registerConnTools(s *mcp.Server, d *deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_status",
		Description: "Report the bridge's MOS identity, configured peers and their connection state, whether the device listener is active, and the inbox size.",
	}, d.status)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_add_peer",
		Description: "Register an outbound MOS peer (a newsroom system or media device) to connect to. Ports default to the MOS standard 10540/10541/10542. The peer is added in memory for this session; add it to the config file or the web console to persist it.",
	}, d.addPeer)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_remove_peer",
		Description: "Unregister a previously added MOS peer and close its connections.",
	}, d.removePeer)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_heartbeat",
		Description: "Send a MOS heartbeat to a peer to verify the connection is alive. Returns the peer's echoed heartbeat.",
	}, d.heartbeat)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_request_machine_info",
		Description: "Send reqMachInfo to a peer and return its listMachInfo response: manufacturer, model, revisions and supported MOS profiles. This is the closest thing MOS has to capability discovery.",
	}, d.requestMachineInfo)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_inbox",
		Description: "Return the most recent MOS messages received from newsroom systems while the bridge is acting as a MOS device (newest first).",
	}, d.inbox)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mos_send_raw",
		Description: "Send a raw MOS XML message (a full <mos>...</mos> document, or just the inner body element) to a peer and return the raw XML reply. Use for message types not covered by a dedicated tool, or for debugging.",
	}, d.sendRaw)
}

// --- mos_status ---

type statusOut struct {
	Identity     messages.Identity `json:"identity"`
	Peers        []mos.PeerStatus  `json:"peers"`
	DeviceActive bool              `json:"deviceActive"`
	InboxCount   int               `json:"inboxCount"`
	Profiles     string            `json:"enabledProfiles"`
}

func (d *deps) status(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, statusOut, error) {
	st := d.mgr.Status()
	out := statusOut{
		Identity:     st.Identity,
		Peers:        st.Peers,
		DeviceActive: st.DeviceActive,
		InboxCount:   st.InboxCount,
		Profiles:     d.cfg.EnabledProfilesString(),
	}
	var b strings.Builder
	fmt.Fprintf(&b, "MOS identity: mosID=%s ncsID=%s\n", st.Identity.MosID, st.Identity.NcsID)
	fmt.Fprintf(&b, "Enabled profiles: %s\n", out.Profiles)
	fmt.Fprintf(&b, "Device listener active: %v\n", st.DeviceActive)
	fmt.Fprintf(&b, "Inbox messages: %d\n", st.InboxCount)
	if len(st.Peers) == 0 {
		b.WriteString("No peers configured.\n")
	} else {
		b.WriteString("Peers:\n")
		for _, p := range st.Peers {
			fmt.Fprintf(&b, "  - %s (%s) lower=%v upper=%v query=%v\n",
				p.Name, p.Host, p.LowerConnected, p.UpperConnected, p.QueryConnected)
		}
	}
	return textResult(b.String()), out, nil
}

// --- mos_add_peer ---

type addPeerIn struct {
	Name      string `json:"name" jsonschema:"unique name for this peer"`
	Host      string `json:"host" jsonschema:"hostname or IP address of the MOS peer"`
	LowerPort int    `json:"lowerPort,omitempty" jsonschema:"lower/object port (default 10540)"`
	UpperPort int    `json:"upperPort,omitempty" jsonschema:"upper/running-order port (default 10541)"`
	QueryPort int    `json:"queryPort,omitempty" jsonschema:"query/search port (default 10542)"`
}

type okOut struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (d *deps) addPeer(ctx context.Context, _ *mcp.CallToolRequest, in addPeerIn) (*mcp.CallToolResult, okOut, error) {
	err := d.mgr.AddPeer(mos.PeerConfig{
		Name:      in.Name,
		Host:      in.Host,
		LowerPort: in.LowerPort,
		UpperPort: in.UpperPort,
		QueryPort: in.QueryPort,
	})
	if err != nil {
		return nil, okOut{}, err
	}
	msg := fmt.Sprintf("Added peer %q (%s).", in.Name, in.Host)
	return textResult(msg), okOut{OK: true, Message: msg}, nil
}

// --- mos_remove_peer ---

type peerOnlyIn struct {
	Peer string `json:"peer" jsonschema:"name of the peer"`
}

func (d *deps) removePeer(ctx context.Context, _ *mcp.CallToolRequest, in peerOnlyIn) (*mcp.CallToolResult, okOut, error) {
	if err := d.mgr.RemovePeer(in.Peer); err != nil {
		return nil, okOut{}, err
	}
	msg := fmt.Sprintf("Removed peer %q.", in.Peer)
	return textResult(msg), okOut{OK: true, Message: msg}, nil
}

// --- mos_heartbeat ---

type replyOut struct {
	Peer      string `json:"peer"`
	ReplyType string `json:"replyType"`
	ReplyXML  string `json:"replyXML"`
}

func (d *deps) heartbeat(ctx context.Context, _ *mcp.CallToolRequest, in peerOnlyIn) (*mcp.CallToolResult, replyOut, error) {
	env := d.mgr.Identity().Heartbeat()
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, replyOut{}, err
	}
	out := replyOut{Peer: in.Peer, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	return textResult(fmt.Sprintf("Heartbeat to %q returned %q.", in.Peer, reply.BodyName())), out, nil
}

// --- mos_request_machine_info ---

type machInfoOut struct {
	Peer      string                 `json:"peer"`
	MachInfo  *messages.ListMachInfo `json:"machineInfo,omitempty"`
	ReplyType string                 `json:"replyType"`
	ReplyXML  string                 `json:"replyXML"`
}

func (d *deps) requestMachineInfo(ctx context.Context, _ *mcp.CallToolRequest, in peerOnlyIn) (*mcp.CallToolResult, machInfoOut, error) {
	env := d.mgr.Identity().ReqMachInfo()
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, machInfoOut{}, err
	}
	out := machInfoOut{Peer: in.Peer, MachInfo: reply.ListMachInfo, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	summary := fmt.Sprintf("Machine info from %q (%s).", in.Peer, reply.BodyName())
	if reply.ListMachInfo != nil {
		summary = fmt.Sprintf("%s manufacturer=%s model=%s mosRev=%s",
			summary, reply.ListMachInfo.Manufacturer, reply.ListMachInfo.Model, reply.ListMachInfo.MosRev)
	}
	return textResult(summary), out, nil
}

// --- mos_inbox ---

type inboxIn struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of messages to return (default all retained)"`
}

type inboxOut struct {
	Entries []mos.InboxEntry `json:"entries"`
}

func (d *deps) inbox(ctx context.Context, _ *mcp.CallToolRequest, in inboxIn) (*mcp.CallToolResult, inboxOut, error) {
	entries := d.mgr.Inbox().List(in.Limit)
	summary := fmt.Sprintf("%d inbound message(s).", len(entries))
	return textResult(summary), inboxOut{Entries: entries}, nil
}

// --- mos_send_raw ---

type sendRawIn struct {
	Peer string `json:"peer" jsonschema:"name of the peer"`
	XML  string `json:"xml" jsonschema:"raw MOS XML: a full <mos> document or just the inner body element"`
}

func (d *deps) sendRaw(ctx context.Context, _ *mcp.CallToolRequest, in sendRawIn) (*mcp.CallToolResult, replyOut, error) {
	trimmed := strings.TrimSpace(in.XML)
	if trimmed == "" {
		return nil, replyOut{}, fmt.Errorf("xml is required")
	}
	var env *messages.Envelope
	if strings.Contains(trimmed, "<mos>") || strings.HasPrefix(trimmed, "<?xml") {
		parsed, err := messages.Unmarshal([]byte(trimmed))
		if err != nil {
			return nil, replyOut{}, fmt.Errorf("parse xml: %w", err)
		}
		env = parsed
	} else {
		// Wrap a bare body element in an envelope.
		wrapped := "<mos>" + trimmed + "</mos>"
		parsed, err := messages.Unmarshal([]byte(wrapped))
		if err != nil {
			return nil, replyOut{}, fmt.Errorf("parse xml body: %w", err)
		}
		env = parsed
	}
	// Always stamp our identity.
	env.MosID = d.mgr.Identity().MosID
	env.NcsID = d.mgr.Identity().NcsID
	reply, xmlStr, err := d.roundtrip(ctx, in.Peer, env)
	if err != nil {
		return nil, replyOut{}, err
	}
	out := replyOut{Peer: in.Peer, ReplyType: reply.BodyName(), ReplyXML: xmlStr}
	return textResult(fmt.Sprintf("Sent raw message to %q, reply %q.", in.Peer, reply.BodyName())), out, nil
}
