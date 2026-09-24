<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Architecture Decision Records

Decisions about this repository that will be questioned later, with the
reasoning that produced them. Records are immutable once merged; a
decision that changes gets a new record superseding the old one.

Decisions about the checks, the score and the attestation format are
scout's, in [scout's ADRs](https://github.com/sebastienrousseau/scout/blob/main/docs/adr/README.md).
This directory records what is decided here.

There are no records yet. The two decisions made so far are documented
where they are enforced: running scout as a program rather than linking
its engine, with the settings every run is fixed to, in the package
documentation of `internal/runner`; and the loopback-only default
allowlist, in `internal/server/allow.go`. The first change to either
starts a record here.

| # | Decision | Status |
|---|---|---|
| — | none yet | — |
