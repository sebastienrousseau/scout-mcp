# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: GPL-3.0-only

.PHONY: all build test test-race coverage vet lint format spdx-check smoke digest \
        server-json image lockstep family help

# Every gate CI runs that needs no network, in the order the cheap ones fail
# first.
all: format vet lint spdx-check test smoke

build:
	CGO_ENABLED=0 go build -trimpath -o build/scout-mcp ./cmd/scout-mcp

test:
	go test ./... -cover

test-race:
	go test -race -shuffle=on -count=1 ./...

# The gate is 85% statement coverage in every package with statements,
# except cmd/scout-mcp; ci.yml says why.
coverage:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	gofmt -l -w .

spdx-check:
	go run ./scripts/spdx_sweep.go

# The server answers the handshake and lists its three tools over stdio.
smoke:
	@mkdir -p build
	@printf '%s\n' \
	  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"make","version":"0"}}}' \
	  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
	  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
	  | go run ./cmd/scout-mcp > build/smoke.out
	@grep -q '"protocolVersion":"2025-11-25"' build/smoke.out
	@for t in scout_check scout_verify_attestation scout_version; do grep -q "\"name\":\"$$t\"" build/smoke.out || { echo "smoke: $$t missing from tools/list"; exit 1; }; done
	@echo "smoke: initialize and tools/list answered with all three tools"

# server.json against the registry schema it names. Needs the network and
# check-jsonschema (through uvx, or pipx on a CI runner).
CHECK_JSONSCHEMA ?= $(shell command -v uvx >/dev/null 2>&1 && echo 'uvx --from check-jsonschema check-jsonschema' || echo 'pipx run check-jsonschema')
server-json:
	@mkdir -p build
	curl -fsSL "$$(python3 -c 'import json; print(json.load(open("server.json"))["$$schema"])')" -o build/server.schema.json
	$(CHECK_JSONSCHEMA) --schemafile build/server.schema.json server.json

# The container image for this machine's architecture, built the way
# goreleaser lays out its context, without goreleaser.
IMAGE ?= scout-mcp:dev
ARCH ?= $(shell go env GOARCH)
image:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -trimpath -ldflags "-s -w -X main.Version=dev" \
	  -o build/image/linux/$(ARCH)/scout-mcp ./cmd/scout-mcp
	cp Dockerfile build/image/Dockerfile
	docker build --platform linux/$(ARCH) -t $(IMAGE) build/image

# This repository carries scout's version. See docs/ecosystem.md in scout.
# The Dockerfile's FROM digest is scout's image for its SCOUT_VERSION.
digest:
	scripts/verify-digest.sh

lockstep:
	scripts/lockstep.sh

# The family manifest in scout is the single source of what this repository is.
family:
	scripts/family.sh

help:
	@printf '%s\n' "targets: all build test test-race coverage vet lint format spdx-check smoke server-json image lockstep family"
