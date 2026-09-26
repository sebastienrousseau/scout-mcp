<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Changelog

All notable changes to scout-mcp are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions are
[Semantic Versioning](https://semver.org/) shaped.

**This repository carries scout's version.** It is in lockstep with
[scout](https://github.com/sebastienrousseau/scout): it wraps scout's
release — the container image is built on scout's image for the same
version — so its version is scout's latest release, exactly, and a release
here with nothing in it is the version rule working.

## [Unreleased]

### Changed

- **A manual, an architecture page and a template README.** The docs
  are built with MkDocs from scout's hash-locked requirements, strictly on
  every pull request, and deployed to GitHub Pages from main.
  `ARCHITECTURE.md` explains why the server runs the scout program instead
  of linking it, and what is fixed on every run. The README follows the
  portfolio template, which `scripts/readme-check.sh` now enforces in CI.
- **The sync pull request can start its own checks.** A pull request
  opened with `GITHUB_TOKEN` triggers no workflows, so every sync pull
  request needed a manual close and reopen before CI ran. With a
  `SYNC_TOKEN` secret, a fine-grained token with `pull-requests: write` on
  this repository, the workflow opens the pull request with it and the
  checks start. The commits are still made with `GITHUB_TOKEN`, so they
  stay GitHub-signed and exempt from the DCO check as the bot's. Without
  the secret nothing changes, and the run log says to close and reopen.

## [0.0.6] — 2026-09-25

### Changed

- **In lockstep with scout 0.0.6.** The image builds on `ghcr.io/sebastienrousseau/scout@sha256:d7b69bd815514e1dd86bb06b6eeffaf4b66d3b9e8e89ea4edb84ffa5c1b81b39`, the multi-arch image scout's release published for 0.0.6, and every install line and the registry listing name the release. Opened by the sync workflow on the release's dispatch; scout's own changelog says what changed in the diagnostic.

## [0.0.5] — 2026-09-24

The first release, in lockstep with scout 0.0.5.

### Added

- **An MCP server over stdio that exposes scout as tools.** An MCP host
  starts `scout-mcp` as a child process and speaks newline-delimited
  JSON-RPC 2.0 to it. It implements the handshake for protocol revisions
  2025-11-25, 2025-06-18 and 2025-03-26, `server/discover` for 2026-07-28,
  `ping`, `tools/list` and `tools/call`. Logs go to stderr; stdout is the
  transport.

- **Three tools, every one annotated `readOnlyHint: true`.**
  - `scout_check` runs the `scout` program against a server's Streamable
    HTTP endpoint and returns the score, the grade, the counts and every
    failing check with its detail and documentation link — at most 25, so
    a server that fails everything does not fill the agent's context. It
    can be narrowed to any of scout's nine phases. One run is bounded at
    five minutes.
  - `scout_verify_attestation` checks a scout attestation offline, with
    [scout-reporting](https://github.com/sebastienrousseau/scout-reporting)'s
    verifier: its structure, its subject digest, and optionally that it
    is about a given endpoint. A statement that does not verify is an
    answer, not a tool error. Signatures are the envelope's job and are
    not checked.
  - `scout_version` reports scout-mcp's version and the scout it runs.

- **An allowlist, on by default and narrow.** `scout_check` evaluates
  loopback addresses only — `localhost`, `*.localhost` and loopback IPs —
  until the operator names more hosts with `--allow` or `SCOUT_MCP_ALLOW`,
  as exact names or as `.example.com` for every subdomain. scout makes
  real requests to the endpoint it is given, and an agent choosing that
  endpoint from a prompt is exactly when a request should not go anywhere
  the operator did not name. A URL that carries credentials is refused.

- **No credentials, and read-only, whatever the caller asks.** Every run
  is `scout check <endpoint> --auth none`, with `SCOUT_CONFIG` pointing at
  an empty configuration file so no profile the operator wrote can add
  credentials, switch on mutations or redirect the report. No flag that
  allows a mutating tool is ever passed, so scout invokes only the
  evaluated server's tools that declare `readOnlyHint`.

- **`--scout` and `SCOUT_MCP_SCOUT`** name the scout program; otherwise
  `scout` on `PATH` is used. `--version` prints the version the release
  build stamped.

- **A container image, `ghcr.io/sebastienrousseau/scout-mcp`**, built on
  scout 0.0.5's distroless image pinned by digest, so it carries the scout
  it runs; multi-arch, non-root, signed with cosign keyless. Release
  binaries for Linux, macOS and Windows on amd64 and arm64, with signed
  checksums and SLSA build provenance.

- **`server.json`** for the official MCP Registry, as
  `io.github.sebastienrousseau/scout-mcp`, pointing at the image.

### Checked

- **scout scored it 95/100 (A)** with
  `scout check --stdio -- scout-mcp`. The one failure was
  `supply.provenance`, because that binary was built from an uncommitted
  tree; CI now runs the same evaluation on every push and fails below 90
  or on any other failing check.

[Unreleased]: https://github.com/sebastienrousseau/scout-mcp/compare/v0.0.6...HEAD
[0.0.6]: https://github.com/sebastienrousseau/scout-mcp/releases/tag/v0.0.6
[0.0.5]: https://github.com/sebastienrousseau/scout-mcp/releases/tag/v0.0.5
