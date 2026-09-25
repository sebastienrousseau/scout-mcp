<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Development

The single entry point for working on scout-mcp: toolchain, how to
reproduce every CI gate locally, and how a release is cut. If a gate fails
in CI and you cannot reproduce it from this file, that is a bug in this
file.

## Toolchain

| Tool | Version | Why |
|---|---|---|
| Go | as pinned by the `go` directive in `go.mod` | `GOTOOLCHAIN=auto` downloads it; CI never pins a version separately |
| make | any | Task runner for everything below |
| scout | the version in `CHANGELOG.md` | `scout_check` runs it; `go install github.com/sebastienrousseau/scout/cmd/scout@vX.Y.Z` |

Optional, only for the gate that uses it: `golangci-lint` (`make lint`),
`markdownlint-cli2`, `codespell` and `lychee` (the Docs Lint workflow and
`pre-commit`), `curl` and `python3` (`make family`, `make lockstep`,
`make server-json`), `uvx` or `pipx` (`make server-json`), `jq` (the
dogfood job), `docker` (`make image`), `goreleaser` (`goreleaser check`).

## Reproducing every CI gate

| CI job | Local command |
|---|---|
| Test (three OSes × two Go versions) | `make test` |
| Race & Shuffled Tests | `make test-race` |
| Coverage Gate (85% per package but `cmd/scout-mcp`) | `make coverage` |
| Lint | `gofmt -l .` and `make lint` |
| Vulnerability Scan | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Repository Checks | `make smoke server-json family lockstep` |
| scout evaluates scout-mcp | `scout check --stdio --rps 0 --no-color --output json -- ./build/scout-mcp --scout "$(command -v scout)"` after `make build` |
| Licence Headers | `make spdx-check` |
| Markdown & Spelling | `markdownlint-cli2 '**/*.md'` and `codespell` |
| Link Check | `lychee --offline --include-fragments '**/*.md'` |
| DCO check | `git log --format=%B origin/main.. \| grep Signed-off-by` |

`make` with no target runs the gates that need no network, in the order
they fail fastest.

The dogfood job fails below a score of 90 or on any failing check other
than `supply.provenance`, which judges how the binary was built rather
than how the server behaves. A local build from a dirty tree fails it;
that is expected.

## Test layout

`internal/server/server_test.go` drives the server through its stdio loop
with a fake runner: the handshake, every tool, every refusal. The
allowlist cases are there. `internal/runner/runner_test.go` stands a shell
script in for scout and asserts the arguments and environment every run
gets; it is `//go:build !windows`, so on Windows that package reports no
test files. `cmd/scout-mcp/main_test.go` covers `run`; `main` itself is
exempt from the coverage gate.

## Generated artefacts

None are committed. Release archives, checksums and the image are built by
goreleaser into `dist/`; `make build`, `make smoke` and `make image` write
to `build/`. Both are ignored.

## Release model

The version is scout's latest release, exactly, and a release is cut after
scout's, on a `feat/vX.Y.Z` branch:

1. Move the `## [Unreleased]` entries under a `## [X.Y.Z] — date` heading
   in `CHANGELOG.md`, and update the version in the README's install
   lines and in `server.json` (both `version` and the image tag).
2. Point the `Dockerfile`'s `FROM` at scout X.Y.Z's image digest
   (`docker buildx imagetools inspect ghcr.io/sebastienrousseau/scout:X.Y.Z`)
   and set `SCOUT_VERSION=X.Y.Z`.
3. `make lockstep` and `scripts/verify-release-versions.sh vX.Y.Z`.
4. `goreleaser check`, and optionally the Release workflow's dry run.
5. Push a signed annotated tag `vX.Y.Z` with the message
   `scout-mcp vX.Y.Z`. The Release workflow builds the binaries and the
   image, signs them, and attests the checksums.
6. Read the tag, the release page and the image's labels back before
   calling it done, then publish the registry listing:
   [docs/publishing.md](docs/publishing.md).

Steps 1 and 2 are what `.github/workflows/sync.yml` does on scout's
release dispatch: it opens a pull request that makes both edits. With a
`SYNC_TOKEN` secret, a fine-grained token with `pull-requests: write` on
this repository, that pull request's checks start on their own; without
it, close and reopen the pull request to start them, because one opened
with `GITHUB_TOKEN` triggers no workflows.

## Conventions

- Stdout is the MCP transport; nothing else is written there.
- Every exported identifier is documented.
- Anything a tool argument or a scout report carries is untrusted input.
