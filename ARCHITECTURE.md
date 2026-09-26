<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Architecture

scout-mcp is an MCP server over stdio that gives an agent three
read-only tools: evaluate an MCP server with
[scout](https://github.com/sebastienrousseau/scout), check a scout
attestation offline, and report versions. It holds no diagnostic logic:
evaluation runs the scout program, and attestation checks use the
Apache-2.0 verifier from
[scout-reporting](https://github.com/sebastienrousseau/scout-reporting).

## The two decisions

**Run the scout program; do not link its engine.** scout's engine lives
in its `internal` packages, which cannot be imported, and linking a copy
would tie this server to one build of it. scout-mcp runs the same binary
an operator runs by hand and reads the JSON it prints. The version the
server reports is whatever `scout version` says, and every safety
property scout has carries over, because the server can only ask for
what scout's own flags allow.

**The agent chooses the endpoint, so nothing of the operator's travels
with it.** Every run is fixed three ways, whatever the caller asks:

- **No configuration.** `SCOUT_CONFIG` points at an empty file, so no
  profile the operator wrote for their own use can switch on mutations,
  add credentials or redirect the report.
- **No credentials.** `--auth none`, always.
- **Read-only.** No flag that allows a mutating or destructive tool is
  ever passed.

On top of that, an allowlist decides where scout may be pointed:
loopback only by default, widened with `--allow` or `SCOUT_MCP_ALLOW`.

## Flow of one call

```text
MCP client ──stdio, JSON-RPC──► internal/server
                                  │ initialize / server/discover / ping
                                  │ tools/list → three tools, all readOnlyHint
                                  │ tools/call:
                                  │   scout_check   ─► allowlist ─┬─► internal/runner
                                  │   scout_version ──────────────┘         │
                                  │                                       ▼
                                  │                         scout check <endpoint>
                                  │                           --auth none --output json
                                  │                           SCOUT_CONFIG=<empty file>
                                  │   ◄──── score, failing checks, guidance ─┘
                                  │
                                  │   scout_verify_attestation ─► scout-reporting/attestation
                                  │                                 Parse, Validate, Covers
                                  ◄──── verdict, in process, no network
```

`scout_check` refuses an endpoint the allowlist does not cover before
scout starts. Requests are answered in turn; a scout run can take a
minute, and a client that wants two at once can start two servers.

## Packages

| Path | Role |
| :--- | :--- |
| `cmd/scout-mcp` | Flags (`--allow`, `--scout`, `--version`) and the stdio loop |
| `internal/server` | JSON-RPC handling, the handshake revisions and `server/discover`, the tool definitions, the allowlist, and attestation checks through scout-reporting's verifier |
| `internal/runner` | Runs the scout program with the fixed flags and environment, and parses its report |

Stdout is the MCP transport, so nothing else is written there; logs go
to stderr.

## Protocol revisions

The server speaks the handshake revisions 2025-11-25, 2025-06-18 and
2025-03-26 through `initialize`, and the stateless 2026-07-28 revision
through `server/discover`, which carries the server's identity in
`_meta`.

## Distribution

Releases ship binaries and a container image built on scout's own image,
pinned by digest, so the image carries the scout it runs. The listing in
the MCP Registry is `io.github.sebastienrousseau/scout-mcp`. Releases
follow scout's in lockstep: scout's release dispatch opens a sync pull
request that moves the image digest, `SCOUT_VERSION`, the install lines
and `server.json` together.
