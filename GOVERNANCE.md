<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Governance

scout-mcp is one repository in the scout family and is governed the way
scout is. This document says what is specific to this repository and points
at scout's for the rest.

## Roles

**Maintainer:** Sebastien Rousseau (<sebastian.rousseau@gmail.com>,
GitHub `@sebastienrousseau`), with commit access, responsible for the
server's safety defaults, its releases, its registry listing and its
security response.

**Contributor:** anyone who opens an issue or a pull request. Mechanics
are in [CONTRIBUTING.md](CONTRIBUTING.md).

## What is decided here, and what is not

This repository decides **what an agent can ask for**: the tools, their
arguments and results, the allowlist's default, and the settings every
scout run is fixed to. A change that widens any of those follows the rule
in [CONTRIBUTING.md](CONTRIBUTING.md): a tracking issue open for at least
a week before it lands, and a record in [docs/adr/](docs/adr/README.md)
when the decision will be questioned later.

It does not decide **the verdicts**: which checks scout runs and how it
scores them are scout's, recorded in scout's ADRs.

## Version and release

The version is scout's. This repository never chooses its own; see the
lockstep rule in [CHANGELOG.md](CHANGELOG.md). Releases are signed tags,
cut by the Maintainer, and the registry listing is published by the
Maintainer after the release; see [docs/publishing.md](docs/publishing.md).

## Continuity

The single-Maintainer model is a real bus-factor risk, stated rather than
hidden. The succession procedure — hand-off, community fork after six
months of unresponsiveness, and compromise response — is scout's, in
[scout's GOVERNANCE.md](https://github.com/sebastienrousseau/scout/blob/main/GOVERNANCE.md),
and applies to this repository as one of the family. GPL-3.0-only lets
anyone fork under the same licence without further permission.

The family manifest names a kill criterion for this repository: if the
registry listings produce no measurable referrals across two quarters, it
is archived.

## Changes to this document

Through the usual pull request process, with the same one-week notice as
a substantial code change.
