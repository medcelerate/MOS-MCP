# Privacy Policy

_Last updated: 2026-09-20_

`mos-mcp` is a self-hosted bridge that you run on your own machine or network. It
is designed to keep your data on your infrastructure.

## What data it handles

- **MOS traffic.** The bridge exchanges MOS protocol messages (media object
  metadata, running orders, stories, machine info) with the MOS peers **you
  configure** — newsroom systems and media devices on your own network.
- **Configuration.** Your settings (identity, roles, profiles, peer hosts and
  ports) are read from and written to a local YAML config file that you control.
- **Inbox.** When acting as a MOS device, recently received messages are kept in
  an in-memory ring buffer so you can inspect them. This buffer is not persisted
  and is cleared when the process stops.

## What it does not do

- It does **not** send your data to the author, to Anthropic, to OpenAI, or to
  any other third party.
- It contains **no telemetry, analytics, or tracking**.
- It only makes network connections to the MOS peers you explicitly configure,
  and it binds its admin console and optional HTTP endpoint to `127.0.0.1`
  (loopback) by default.

## Data retention

The bridge stores no data beyond your local configuration file and the
in-memory inbox described above. Uninstalling the connector or deleting your
config file removes everything it retains.

## Contact

Questions or issues: <https://github.com/medcelerate/MOS-MCP/issues>
