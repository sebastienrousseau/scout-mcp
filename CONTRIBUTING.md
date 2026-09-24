<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Contributing

scout-mcp puts scout in an agent's hands. The agent's arguments come from a
prompt, which is the thing to keep in mind for every change here: the
server must stay safe whatever it is asked.

## Getting started

1. Fork and clone the repository.
2. Install **Go** at the version the `go` directive in `go.mod` names,
   **make**, and [scout](https://github.com/sebastienrousseau/scout) to try
   `scout_check` by hand.
3. Create a branch from `main` and open the pull request against `main`.
   Every workflow filters on `pull_request: branches: [main]`, so a PR
   aimed elsewhere runs no CI.
4. Make the change.
5. Verify:

   ```bash
   make            # format, vet, lint, headers, tests, stdio smoke test
   make test-race
   ```

## Commits

**Sign your commits cryptographically and add a DCO sign-off trailer.**
Both are required and both are enforced: signing proves who authored the
commit; the sign-off (`git commit -s`) certifies the
[Developer Certificate of Origin](https://developercertificate.org).
Merge commits are exempt from the DCO check.

Use [Conventional Commits](https://www.conventionalcommits.org/) with an
imperative subject: `feat(server): report the phase a run stopped in`,
not `Added phase`.

## What a change needs

- **A test that fails without it.** The fake runner in
  `internal/server/server_test.go` and the stand-in scout program in
  `internal/runner/runner_test.go` reach every branch without a live
  server; never point a test at one.
- **85% statement coverage in every package** but `cmd/scout-mcp`.
- **An entry under `## [Unreleased]` in `CHANGELOG.md`.**
- **A tracking issue, open for at least a week**, when the change widens
  what an agent can make scout do: the allowlist's default, a new scout
  flag passed through, a credential of any kind, a tool that is not
  read-only. Most of those are declined.
- **A reason in the commit for any new dependency.** The module has one.

## Pull request checklist

- [ ] `make` and `make test-race` pass
- [ ] The change is covered by a test that fails without it
- [ ] `CHANGELOG.md` has an entry under `## [Unreleased]`
- [ ] Commits are signed and carry a DCO sign-off
