<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

<p align="center">
  <img src="https://raw.githubusercontent.com/sebastienrousseau/scout/main/.github/logo.svg" alt="scout-mcp logo" width="128" />
</p>

<h1 align="center">scout-mcp</h1>

<p align="center">
  scout, the Model Context Protocol server diagnostic, as MCP tools — so an agent can evaluate a server, or check an attestation about one, from inside the editor. Read-only, allowlisted, and it never sends a credential.
</p>

<p align="center">
  <a href="https://github.com/sebastienrousseau/scout-mcp/actions"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/scout-mcp/ci.yml?style=for-the-badge&logo=github" alt="Build Status" /></a>
  <a href="https://github.com/sebastienrousseau/scout-mcp/pkgs/container/scout-mcp"><img src="https://img.shields.io/badge/ghcr.io-scout--mcp-fc8d62?style=for-the-badge&logo=docker&logoColor=white" alt="Container image" /></a>
  <a href="https://pkg.go.dev/github.com/sebastienrousseau/scout-mcp"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=for-the-badge&logo=go&logoColor=white" alt="Go Reference" /></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/scout-mcp"><img src="https://img.shields.io/ossf-scorecard/github.com/sebastienrousseau/scout-mcp?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-GPL--3.0--only-blue?style=for-the-badge" alt="License: GPL-3.0-only" /></a>
  <a href="#requirements"><img src="https://img.shields.io/github/go-mod/go-version/sebastienrousseau/scout-mcp?style=for-the-badge&logo=go&logoColor=white&label=Go" alt="Minimum Go version" /></a>
</p>

---

## Contents

**Getting started**

- [Install](#install) — `go install`, the container image, and an MCP host configuration
- [Requirements](#requirements) — scout itself, and the Go floor to build from source
- [Quick Start](#quick-start) — ask the agent to evaluate a local server

**The scout-mcp ecosystem**

- [The scout-mcp ecosystem](#the-scout-mcp-ecosystem) — `scout`, `scout-reporting`, `scout-action`, `scout-mcp`, `scout-lsp`, `scout-census` at a glance

**Reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the three tools and the protocol surface
- [Ecosystem comparison](#ecosystem-comparison) — beside running scout yourself
- [Benchmarks](#benchmarks) — what the server adds to a run
- [Features](#features) — the allowlist, no credentials, read-only
- [Configuration](#configuration) — two flags and their environment variables
- [Examples](#examples) — tool calls and results

**Operational**

- [When not to use scout-mcp](#when-not-to-use-scout-mcp) — limitations
- [Development](#development) — make targets, CI
- [Security](#security) — what an agent can and cannot make it do
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — tool names, arguments and results
- [License](#license)

---

## Install

`scout_check` runs the `scout` program. **scout must be installed and on
`PATH`, or named with `--scout`**, for that tool to work; the container
image carries it. `scout_verify_attestation` needs nothing but scout-mcp.

### As a Go program

```sh
go install github.com/sebastienrousseau/scout-mcp/cmd/scout-mcp@v0.0.6
go install github.com/sebastienrousseau/scout/cmd/scout@v0.0.6
```

Release binaries for Linux, macOS and Windows on amd64 and arm64 are on
the [releases page](https://github.com/sebastienrousseau/scout-mcp/releases),
with signed checksums and SLSA provenance.

### As a container image

```sh
docker pull ghcr.io/sebastienrousseau/scout-mcp:0.0.6
```

The image is scout's own release image with scout-mcp added: distroless,
non-root, linux/amd64 and linux/arm64, with scout at `/usr/local/bin/scout`
and scout-mcp started with `--scout` pointing at it.

### In an MCP host

scout-mcp speaks MCP over stdio, so a host starts it as a child process.
For Claude Desktop (`claude_desktop_config.json`), Claude Code (`.mcp.json`)
and other hosts that read an `mcpServers` block:

```json
{
  "mcpServers": {
    "scout": {
      "command": "scout-mcp",
      "args": ["--allow", ".internal.example.com"]
    }
  }
}
```

With the container image instead:

```json
{
  "mcpServers": {
    "scout": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "ghcr.io/sebastienrousseau/scout-mcp:0.0.6"]
    }
  }
}
```

Inside a container, loopback is the container itself. To evaluate a server
on the host, run the container with `--network host` on Linux, or point at
`host.docker.internal` and add `-e SCOUT_MCP_ALLOW=host.docker.internal`
to the arguments.

The server is listed in the official MCP Registry as
`io.github.sebastienrousseau/scout-mcp`; [`server.json`](server.json) is
that listing.

---

## Requirements

| Requirement | Floor | Enforced by |
|---|---|---|
| scout | on `PATH`, or `--scout`; the version this one is in lockstep with | CI builds scout at that tag and evaluates this server with it |
| Go (building from source) | the `go` directive in [`go.mod`](go.mod) | CI tests on that version and on latest stable, on Linux, macOS and Windows |
| An MCP host | any that starts stdio servers and speaks revision 2025-03-26 or later | the handshake negotiates 2025-11-25, 2025-06-18 or 2025-03-26, and 2026-07-28 through `server/discover` |
| The server under test | a Streamable HTTP endpoint on the allowlist | `scout_check` refuses any other URL before scout runs |

The Go floor is raised only when a release needs a language feature, on a
patch release like everything else pre-1.0, and the changelog says so.

---

## Quick Start

```sh
go install github.com/sebastienrousseau/scout-mcp/cmd/scout-mcp@v0.0.6
go install github.com/sebastienrousseau/scout/cmd/scout@v0.0.6
claude mcp add scout -- scout-mcp
```

Then, with an MCP server of your own listening on
`http://127.0.0.1:3000/mcp`, ask the agent:

> Evaluate my MCP server at <http://127.0.0.1:3000/mcp> with scout and fix what fails.

The agent calls `scout_check`, which runs
`scout check http://127.0.0.1:3000/mcp --auth none --output json` and
returns the score, the grade and every failing check with its detail and a
link to the fix. Loopback needs no configuration; any other host must be
named with `--allow` first.

---

## The scout-mcp ecosystem

One engine, three surfaces, five satellites. This repository is the
distribution surface: its deliverable is a registry listing, so scout is
where agents look for tools.

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [`scout`](https://github.com/sebastienrousseau/scout) | The engine, every check, and the CLI, TUI and web surfaces (GPL-3.0-only) | Evaluate a server and write the statement |
| [`scout-reporting`](https://github.com/sebastienrousseau/scout-reporting) | The attestation format, its schema and the offline verifier (Apache-2.0) | Gate on a statement in a gateway, registry or pipeline |
| [`scout-action`](https://github.com/sebastienrousseau/scout-action) | The GitHub Action and GitLab template wrapping the published image by digest (Apache-2.0) | Run scout in CI without installing it |
| **`scout-mcp`** | scout's diagnostics as read-only MCP tools (GPL-3.0-only) | Evaluate a server from inside an editor |
| `scout-lsp` | A language server over MCP artefacts (planned) | Hover a check id for its remediation |
| `scout-census` | The published reliability census (planned) | Reproduce the numbers |

The family manifest lives in scout at
[`docs/ecosystem.md`](https://github.com/sebastienrousseau/scout/blob/main/docs/ecosystem.md);
`make family` checks this repository's row against it. Every lockstep
repository carries scout's version; this one wraps scout's release, so its
version is scout's latest, exactly.

---

## Capabilities at a glance

| Area | Capability | Status |
| :--- | :--- | :--- |
| Evaluate | `scout_check`: score, grade, counts and up to 25 failing checks for an allowlisted Streamable HTTP endpoint, optionally narrowed to some of scout's nine phases | Stable |
| Verify | `scout_verify_attestation`: structure, subject digest and target of a scout attestation, offline | Stable |
| Identify | `scout_version`: scout-mcp's version and the scout it runs | Stable |
| Results | Text for the agent, plus `structuredContent` matching each tool's `outputSchema` | Stable |
| Protocol | stdio; handshake 2025-11-25, 2025-06-18, 2025-03-26; `server/discover` for 2026-07-28; `ping` | Stable |
| Servers that are programs (`--stdio`) | not through a tool; run `scout check --stdio` yourself | Out of scope |
| Credentials | never sent; every run is `--auth none` | Out of scope by design |

---

## Ecosystem comparison

The alternative is running scout in a terminal and pasting the report into
the conversation, or poking the server by hand in an inspector. scout-mcp
is the first with the decisions made for an agent: which hosts it may
reach, that no credential travels, and a result sized for a context
window.

| Approach | An agent can call it | Targets limited by the operator | Scored, with remediation links |
| :--- | :---: | :---: | :---: |
| **scout-mcp** | yes | yes — loopback unless `--allow` | yes |
| `scout check` in a terminal | no | the operator types the URL | yes |
| [MCP Inspector](https://github.com/modelcontextprotocol/inspector) | no — a UI for a person | the operator types the URL | no |

---

## Benchmarks

The server adds a process start and a JSON round trip to a run; the run
itself is scout's, and bounded at five minutes. Measured with
[hyperfine](https://github.com/sharkdp/hyperfine) on the binaries built
from this tree.

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| Start, `initialize`, `tools/list`, `scout_version`, exit | 16 ms mean | Apple A18 Pro, Go 1.27.1, 2026-09-24 |
| scout's full stdio evaluation of scout-mcp | 94 ms mean | same |
| `scout_check` against a server | scout's own timings, in its report | the target server |

---

## Features

**An allowlist, on by default and narrow.** With no configuration,
`scout_check` evaluates loopback addresses only: `localhost`, any
`*.localhost` name, and loopback IPs. `--allow` names more hosts, exactly
or as `.example.com` for every subdomain (not the apex). scout makes real
requests to the endpoint it is given, and an agent choosing that endpoint
from a prompt is exactly the situation in which a request should not go
anywhere the operator did not name. A URL that is not http or https, or
that carries credentials in it, is refused before scout runs.

**No credentials.** Every run is `scout check <endpoint> --auth none`, with
`SCOUT_CONFIG` pointing at an empty configuration file, so no profile or
defaults block the operator wrote for their own use can add a credential,
switch on mutations or redirect the report. The operator's secrets do not
travel to wherever an agent points scout.

**Read-only, twice.** Every tool here is annotated `readOnlyHint: true`.
And no flag that allows a mutating tool is ever passed to scout, so scout
invokes only the evaluated server's tools that declare `readOnlyHint`.

**scout, not a copy of it.** scout-mcp runs the scout program and reads
the JSON report it prints; it does not link scout's engine. The version
`scout_version` reports is whatever `scout version` says, and every safety
property scout has holds, because the server can only ask for what scout's
own flags allow.

**Answers an agent can act on.** A failing check comes with its id,
severity, detail and a documentation link. A statement that does not
verify is a result saying why, not a tool error. A server that fails
everything returns the first 25 failures and says it truncated.

---

## Configuration

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `--allow` | `SCOUT_MCP_ALLOW` | empty: loopback only | Hosts `scout_check` may evaluate besides loopback, comma-separated; a leading dot allows subdomains |
| `--scout` | `SCOUT_MCP_SCOUT` | `scout` on `PATH` | Path of the scout program |
| `--version` | — | — | Print the version and exit |
| `--completion` | — | — | Print a completion script for `bash`, `zsh` or `fish`, and exit |

A flag overrides its environment variable. Nothing else is read: scout
itself runs with an empty configuration file, whatever the operator's own
scout configuration says.

Shell completions come from the flag set, so they list every flag:

```sh
scout-mcp --completion bash > /etc/bash_completion.d/scout-mcp
scout-mcp --completion zsh > "${fpath[1]}/_scout-mcp"
scout-mcp --completion fish > ~/.config/fish/completions/scout-mcp.fish
```

---

## Examples

A `tools/call` for `scout_check`:

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"scout_check","arguments":{"endpoint":"http://127.0.0.1:3000/mcp","phases":["handshake","catalog"]}}}
```

`arguments` for the other two:

```json
{"statement": "<the attestation's JSON text>", "endpoint": "https://mcp.example.com/mcp"}
```

```json
{}
```

The first is `scout_verify_attestation`, with `endpoint` optional; the
second is `scout_version`, which takes no arguments. An unknown argument
is refused rather than ignored, so a misspelling is reported.

To see the handshake and the tool list without a host:

```sh
make smoke
```

---

## When not to use scout-mcp

- **For a server that is a program.** `scout_check` takes a Streamable
  HTTP URL. An agent starting arbitrary programs is a different trust
  decision; run `scout check --stdio -- <command>` yourself.
- **For a server that needs credentials.** Every run is `--auth none`, so
  the evaluation is what an unauthenticated client sees. Run scout in a
  terminal with the credentials you were given for the rest.
- **As a CI gate.** Use [scout-action](https://github.com/sebastienrousseau/scout-action),
  which keeps the report and fails the job.
- **To check who signed an attestation.** `scout_verify_attestation`
  checks structure and integrity; the signature is the envelope's, and
  `cosign` or `gh attestation verify` checks it.
- **For many evaluations at once.** Requests are answered in turn; a host
  that wants two at once starts two servers.

---

## Development

```bash
make            # format, vet, lint, headers, tests, stdio smoke test
make test-race  # race detector, randomised order
make image      # the container image for this machine, without goreleaser
make family     # this repository's row in scout's family manifest
make lockstep   # the version is scout's latest release
```

Every gate CI runs has a local form; [DEVELOPMENT.md](DEVELOPMENT.md) maps
them. CI also builds scout at the lockstep version and has it evaluate this
server over stdio, failing below 90 or on any failing check but
`supply.provenance`.

---

## Security

The endpoint an agent passes is checked against the allowlist before scout
runs, and a refused endpoint never reaches scout. scout runs with
`--auth none` and an empty configuration file, and with no flag that
allows a mutating tool; each of those is a test in `internal/runner` and
`internal/server`. What reaches the agent is bounded: at most 25 failures
per result, and scout's error output cut to its last 400 characters. CI
runs `govulncheck` on every push, and releases are signed with cosign
keyless and carry SLSA build provenance.

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

The four entry points, identical across every repo in the family:

- **[User Manual](https://scoutmcp.io/manual/)** — scout's rendered manual: the phases, the checks, the report
- **[API reference](https://pkg.go.dev/github.com/sebastienrousseau/scout-mcp)** — this module's packages
- **[Developer docs](DEVELOPMENT.md)** — toolchain, task map, reproducing every CI gate locally
- **[Ecosystem map](https://github.com/sebastienrousseau/scout/blob/main/docs/ecosystem.md)** — the family, the published artefacts, the lockstep version rule

| Document | Covers |
|---|---|
| [`docs/publishing.md`](docs/publishing.md) | Publishing the registry listing |
| [`docs/adr/`](docs/adr/README.md) | Decision records for this repository |
| [`server.json`](server.json) | The MCP Registry listing |
| [`SECURITY.md`](SECURITY.md) | Disclosure policy, supported versions, what is guaranteed |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Signed-commit and DCO policy, what a change needs |
| [`CHANGELOG.md`](CHANGELOG.md) | Per-release notes, and the lockstep version rule |
| [`SUPPORT.md`](SUPPORT.md) | Where to ask, and what to expect |

---

## Stability guarantees

scout-mcp is pre-1.0, carries scout's version, and follows SemVer with the
patch digit moving for everything until 1.0.

**The breaking axis is what an agent or a host relies on.** These are
breaking:

- Removing or renaming a tool, an argument or a structured result field
- Making an optional argument required
- Widening the default allowlist, sending a credential, or passing scout a
  flag that allows a mutating tool
- Changing a flag's or an environment variable's meaning

Added tools, optional arguments and result fields, and a new scout release
underneath are **not** breaking. What a run reports is scout's, and scout's
own stability rule governs it.

**Deprecation window.** A deprecated tool or argument keeps working for at
least one release after the release that announces it.

---

## License

Licensed under the **[GNU General Public License v3.0 only](LICENSE)**.

scout-mcp is GPL-3.0-only like [scout](https://github.com/sebastienrousseau/scout),
the program it runs. It uses the Apache-2.0
[scout-reporting](https://github.com/sebastienrousseau/scout-reporting)
verifier for attestations.

<p align="right"><a href="#contents">Back to Top</a></p>
