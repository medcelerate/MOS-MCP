// Package mcpserver builds the Model Context Protocol server that exposes the
// MOS bridge as a set of tools. Tools are registered conditionally based on
// which MOS profiles the configuration enables.
package mcpserver

import (
	"context"
	"fmt"

	"github.com/medcelerate/MOS-MCP/internal/config"
	"github.com/medcelerate/MOS-MCP/internal/mos"
	"github.com/medcelerate/MOS-MCP/internal/mos/messages"
	"github.com/medcelerate/MOS-MCP/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// deps carries the shared dependencies every tool handler needs.
type deps struct {
	mgr *mos.Manager
	cfg *config.Config
}

// New builds an MCP server with tools appropriate to the enabled profiles.
func New(mgr *mos.Manager, cfg *config.Config) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "mos-mcp",
		Version: version.Version,
	}, nil)

	d := &deps{mgr: mgr, cfg: cfg}

	// Connection management + Profile 0 are always available.
	registerConnTools(s, d)

	if cfg.HasProfile(1) || cfg.HasProfile(3) {
		registerObjectTools(s, d)
	}
	if cfg.HasProfile(2) || cfg.HasProfile(4) {
		registerRoTools(s, d)
	}
	return s
}

// roundtrip sends env to a peer and returns the reply plus its XML rendering.
func (d *deps) roundtrip(ctx context.Context, peer string, env *messages.Envelope) (*messages.Envelope, string, error) {
	if peer == "" {
		return nil, "", fmt.Errorf("peer is required")
	}
	reply, err := d.mgr.Send(ctx, peer, env)
	if err != nil {
		return nil, "", err
	}
	xmlBytes, _ := reply.Marshal()
	return reply, string(xmlBytes), nil
}

// textResult wraps a human-readable summary as tool content. The typed Out
// value supplied alongside it becomes the structured content.
func textResult(summary string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: summary}},
	}
}

func boolPtr(b bool) *bool { return &b }

// annRead builds annotations for a read-only tool (does not modify anything).
func annRead(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true}
}

// annWrite builds annotations for a tool that makes additive, non-destructive
// changes.
func annWrite(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, DestructiveHint: boolPtr(false)}
}

// annDestructive builds annotations for a tool that may overwrite or remove
// existing state.
func annDestructive(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, DestructiveHint: boolPtr(true)}
}
