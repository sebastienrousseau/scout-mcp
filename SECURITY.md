<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Security Policy

scout-mcp lets an agent run scout, a program that makes real requests to
the endpoint it is given. The agent's choice of endpoint comes from a
prompt, which is untrusted. Its security posture is about three things:
scout is pointed only where the operator allowed, it carries none of the
operator's credentials, and it changes nothing on the server it evaluates.

## Reporting a Vulnerability

Report security issues through [GitHub's private vulnerability reporting](https://github.com/sebastienrousseau/scout-mcp/security/advisories/new). Do not open a public issue.

You will receive an acknowledgement within **72 hours**. A confirmed
vulnerability is fixed and released within **90 days** of the report, or
sooner when a fix is straightforward; if the window cannot be met you will
be told why and given a revised date.

A way to make `scout_check` reach a host that is not on the allowlist, to
make scout send a credential, or to make it invoke a tool that does not
declare `readOnlyHint` is a vulnerability, whatever the prompt that did it.

## Supported Versions

Only the latest release is supported. The version is scout's; see the
lockstep rule in [CHANGELOG.md](CHANGELOG.md).

## Security Measures

Each item names the test that enforces it, so the claim can be checked
rather than taken on trust.

- **Allowlisted endpoints only, loopback by default.** The endpoint is
  checked before scout runs; a URL that is not http or https, or that
  carries credentials, is refused, and a leading-dot entry allows
  subdomains but not the apex. `TestCheckRefusals` in
  `internal/server/server_test.go`.
- **No credentials, no operator configuration.** Every run is
  `scout check <endpoint> --auth none`, with `SCOUT_CONFIG` pointing at an
  empty file, so no profile can add a credential or switch on mutations.
  `TestCheckFixesTheSafetySettings` in `internal/runner/runner_test.go`
  asserts both, and that `--allow-mutations`, `--allow-destructive` and
  `--token` are never passed.
- **Read-only tools.** Every tool is annotated `readOnlyHint: true` and
  `destructiveHint: false`. `TestHandshakeAndList`.
- **Unknown arguments are refused**, so an argument a later version might
  honour is not silently accepted by this one. `TestCheckRefusals`.
- **Bounded results.** At most 25 failures per result, and scout's error
  output cut to its last 400 characters.
- **Attestations are verified offline.** `scout_verify_attestation` never
  fetches anything; it checks structure and integrity with
  [scout-reporting](https://github.com/sebastienrousseau/scout-reporting).
  Who signed a statement is the envelope's job, not this tool's.
- **Supply chain.** One direct dependency, scout-reporting, which has none.
  CI runs `govulncheck` on every push. Release checksums and the container
  image are signed with cosign keyless, and the checksums carry SLSA build
  provenance. The image's base is scout's release image, pinned by digest.
