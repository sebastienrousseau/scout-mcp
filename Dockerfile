# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only
#
# Runtime image for scout-mcp.
#
# The binary is built by goreleaser, not here: the image ships the same
# artefact the release archive does. dockers_v2 lays the build context out
# by platform (linux/amd64/scout-mcp, linux/arm64/scout-mcp) and buildx
# sets TARGETPLATFORM for each architecture.
#
# The base is scout's own release image, pinned by digest: distroless
# nonroot with scout at /usr/local/bin/scout. scout_check runs that program,
# so the image carries the scout release this one is in lockstep with and
# needs nothing else. SCOUT_VERSION names that release, and
# scripts/verify-release-versions.sh refuses a release where it and the
# tag disagree; move it and the digest together. To resolve the digest:
#   docker buildx imagetools inspect ghcr.io/sebastienrousseau/scout:<version>
ARG SCOUT_VERSION=0.0.6
FROM ghcr.io/sebastienrousseau/scout@sha256:d7b69bd815514e1dd86bb06b6eeffaf4b66d3b9e8e89ea4edb84ffa5c1b81b39

ARG SCOUT_VERSION
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/scout-mcp /usr/local/bin/scout-mcp

# io.modelcontextprotocol.server.name is how the MCP Registry verifies that
# whoever publishes server.json owns this image; it must equal server.json's
# name exactly.
LABEL io.modelcontextprotocol.server.name="io.github.sebastienrousseau/scout-mcp" \
      io.github.sebastienrousseau.scout.version="${SCOUT_VERSION}" \
      org.opencontainers.image.source="https://github.com/sebastienrousseau/scout-mcp" \
      org.opencontainers.image.url="https://github.com/sebastienrousseau/scout-mcp" \
      org.opencontainers.image.description="scout-mcp: scout's MCP server diagnostics as read-only MCP tools, over stdio" \
      org.opencontainers.image.licenses="GPL-3.0-only" \
      org.opencontainers.image.title="scout-mcp" \
      org.opencontainers.image.vendor="Sebastien Rousseau"

# distroless nonroot is uid/gid 65532. Numeric, so Kubernetes runAsNonRoot
# can verify it.
USER 65532:65532
WORKDIR /home/nonroot

# An MCP host runs this with `docker run -i --rm`; stdin and stdout are the
# transport. The path of scout is fixed so PATH inside the image does not
# matter.
ENTRYPOINT ["/usr/local/bin/scout-mcp", "--scout", "/usr/local/bin/scout"]
CMD []
