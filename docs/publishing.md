<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: GPL-3.0-only -->

# Publishing the registry listing

[`server.json`](../server.json) lists scout-mcp in the
[official MCP Registry](https://registry.modelcontextprotocol.io) as
`io.github.sebastienrousseau/scout-mcp`, pointing at the container image.
It is published by hand, after the release, because it needs the
Maintainer's GitHub login.

## Before

- The release for the version in `server.json` is out, and
  `ghcr.io/sebastienrousseau/scout-mcp:X.Y.Z` is public.
- The image carries the ownership label the registry checks. It must equal
  the `name` in `server.json`:

  ```sh
  docker buildx imagetools inspect ghcr.io/sebastienrousseau/scout-mcp:X.Y.Z \
    --format '{{ json (index .Image "linux/amd64").Config.Labels }}'
  ```

  and the output must map `io.modelcontextprotocol.server.name` to
  `io.github.sebastienrousseau/scout-mcp`.
- `make server-json` passes: the file validates against the schema it
  names.

## Publish

Install `mcp-publisher` (`brew install mcp-publisher`, or the binary from
the [registry's releases](https://github.com/modelcontextprotocol/registry/releases)),
then, from the repository root:

```sh
mcp-publisher login github
mcp-publisher publish
```

`login github` opens a device-code flow; the `io.github.sebastienrousseau/`
namespace is granted to whoever authenticates as `sebastienrousseau`.

## After

Read the listing back from the registry rather than trusting the command's
output:

```sh
curl -fsSL "https://registry.modelcontextprotocol.io/v0.1/servers?search=io.github.sebastienrousseau/scout-mcp" -o listing.json
jq '.servers[].server | {name, version}' listing.json
```

The version must be the one just released.
