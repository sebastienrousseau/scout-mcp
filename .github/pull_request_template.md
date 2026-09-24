<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->
<!--
Thanks for contributing to scout-mcp.

Branch from `main` and target `main`. An agent's arguments reach this
server from a prompt, so a change to what scout can be pointed at, what it
carries, or what it may invoke is the change to describe most carefully.
-->

## What this changes

<!-- One or two sentences. What is different after this merges? -->

## Why

<!-- The problem, not the patch. If it fixes an issue, link it: Fixes #123 -->

## How it was verified

- [ ] `make` passes
- [ ] `make test-race` passes
- [ ] A change to a tool's arguments or results is covered by a test that fails without it
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`

## Risk

<!--
Name anything that widens what an agent can make scout do: the allowlist,
a flag passed through, a credential, a tool that is not read-only.
-->

---

- [ ] Commits are signed and carry a DCO `Signed-off-by` trailer (`git commit -s -S`)
