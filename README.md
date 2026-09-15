# mos-mcp

A cross-platform bridge between the **MOS (Media Object Server) protocol** used
by broadcast newsroom systems and the **Model Context Protocol (MCP)**, so an AI
client can drive newsroom workflows: query media objects, build and send running
orders, search object databases, and feed or read from a live newsroom computer
system (NCS).

Written in Go. Ships as a single self-contained binary (the admin web console is
embedded) for macOS, Linux, and Windows on amd64 and arm64.

> Implements the message set of [MOS Protocol v2.8.5](https://mosprotocol.com).
> MCP is provided by the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

---

## What it does

MOS is a TCP/XML protocol spoken between an **NCS** (newsroom computer system —
ENPS, iNEWS, …) and a **MOS device** (video server, graphics, prompter, …) over
three ports: `10540` (lower / object metadata), `10541` (upper / running orders)
and `10542` (query / search).

`mos-mcp` can act as **either side, or both at once**:

- **client role** — dials out to MOS peers and issues requests on your behalf.
- **device role** — listens for an NCS to connect and answers with acks, while
  recording everything it receives to an inbox you can inspect.

It exposes these capabilities as MCP tools, and only registers the tools for the
MOS **profiles** you enable.

### A note on discovery

MOS has **no network auto-discovery** — there is no broadcast, mDNS, or registry.
Peers are always configured by hand (host, IDs, ports). The only capability query
is Profile 0's `reqMachInfo`/`listMachInfo`, sent *after* connecting; the admin
console's **Test** button uses it to confirm a peer you entered is reachable and
to show which profiles it reports.

---

## Install

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/medcelerate/MOS-MCP/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/medcelerate/MOS-MCP/main/scripts/install.ps1 | iex
```

Or download a binary for your platform from the
[Releases](https://github.com/medcelerate/MOS-MCP/releases) page.

### From source

```bash
go install github.com/medcelerate/MOS-MCP/cmd/mos-mcp@latest
```

---

## Quick start

```bash
cp config.example.yaml config.yaml   # edit to taste
mos-mcp --config config.yaml
```

By default this serves MCP over **stdio**, starts the **device listener**, and
opens the **admin console** at <http://127.0.0.1:8088>.

### Connecting an MCP client

**Claude Desktop / Claude Code (stdio):** add to your MCP client config:

```json
{
  "mcpServers": {
    "mos": {
      "command": "mos-mcp",
      "args": ["--config", "/path/to/config.yaml"]
    }
  }
}
```

**Networked clients (Streamable HTTP):** set `mcp.transport: http` (or `both`)
in the config; the endpoint is served at `mcp.http.addr` (default
`127.0.0.1:8080`).

> Logs go to **stderr** so they never corrupt the stdio MCP stream on stdout.

---

## Configuration

See [`config.example.yaml`](config.example.yaml) for the full annotated file.
Key fields:

| Field | Meaning |
|-------|---------|
| `mosID` / `ncsID` | This bridge's identity on the MOS network |
| `role` | `client`, `device`, or `both` |
| `profiles` | Which MOS profiles to enable: `[0,1,2,3,4]` |
| `peers` | Remote MOS systems to dial (client role) |
| `listen` | Ports the device listener binds (device role) |
| `mcp.transport` | `stdio`, `http`, or `both` |
| `web.enabled` / `web.addr` | Admin console (loopback by default) |

Selected fields can be overridden via environment variables: `MOSMCP_MOSID`,
`MOSMCP_NCSID`, `MOSMCP_ROLE`, `MOSMCP_MCP_TRANSPORT`, `MOSMCP_MCP_HTTP_ADDR`,
`MOSMCP_WEB_ADDR`, `MOSMCP_WEB_ENABLED`, `MOSMCP_LOG_LEVEL`, `MOSMCP_CONFIG`.

The admin console reads and writes the same config file; changes made in the UI
are persisted and applied live.

---

## MCP tools

Always available (connection management + Profile 0):

| Tool | Description |
|------|-------------|
| `mos_status` | Identity, peers, connection state, inbox size |
| `mos_add_peer` / `mos_remove_peer` | Register / unregister an outbound peer |
| `mos_heartbeat` | Verify a peer is alive |
| `mos_request_machine_info` | `reqMachInfo` → device identity + supported profiles |
| `mos_inbox` | Recent messages received in device role |
| `mos_send_raw` | Send arbitrary MOS XML (debug / uncovered messages) |

Profile 1 (objects): `mos_request_object`, `mos_request_all_objects`
Profile 3 (search): `mos_search_objects`
Profile 2 (running orders): `mos_create_running_order`, `mos_replace_running_order`,
`mos_delete_running_order`, `mos_element_action`, `mos_ready_to_air`
Profile 4 (stories): `mos_send_story`, `mos_request_all_running_orders`

Tools are registered only for the profiles enabled in `profiles`.

---

## Admin console

At `web.addr` (default <http://127.0.0.1:8088>) you can:

- View bridge identity, enabled profiles, and device-listener state
- Add, edit, and remove peers (persisted to the config file)
- **Test** a peer (heartbeat + `reqMachInfo`) and see its reported capabilities
- Watch the inbox of messages received from a connected NCS

The console is a single embedded HTML page — no separate server or build step.

---

## Development

```bash
go build ./...          # build
go test ./...           # unit + integration tests
go vet ./...            # static checks
```

Cross-platform archives are produced by [GoReleaser](https://goreleaser.com):

```bash
goreleaser check                        # validate config
goreleaser release --snapshot --clean   # build local snapshot into ./dist
```

### CI

- **CI** (`.github/workflows/ci.yml`) runs vet, race tests, and build on Linux,
  macOS, and Windows for every push and PR, plus golangci-lint.
- **Release** (`.github/workflows/release.yml`) runs GoReleaser on any `v*` tag,
  cross-compiling all targets and publishing a GitHub Release with checksums.

To cut a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

---

## Project layout

```
cmd/mos-mcp        entrypoint (flags, wiring, transport selection)
internal/config    YAML config: load, validate, save, env overrides
internal/mos       MOS TCP layer: framing, connections, peers, device server, manager
  └ messages       encoding/xml structs for MOS profiles 0–4
internal/mcpserver MCP server and tool handlers (gated by profile)
internal/web       embedded admin console + JSON API
```

## License

MIT — see [LICENSE](LICENSE).
