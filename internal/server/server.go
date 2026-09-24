// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package server is an MCP server over stdio that exposes scout's
// diagnostics as tools, so an agent can evaluate an MCP server, or check an
// attestation about one, from inside the editor.
//
// It speaks newline-delimited JSON-RPC 2.0 on stdin and stdout, the stdio
// transport of the Model Context Protocol, and implements the handshake,
// ping and the tools methods. Every tool is read-only. Logs go to stderr,
// because stdout is the transport.
package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/sebastienrousseau/scout-mcp/internal/runner"
)

// ProtocolVersions are the handshake revisions this server speaks, newest
// first. The 2026-07-28 revision has no handshake: a client on it calls
// server/discover instead and sends its context with every request, which
// this server needs nothing from.
var ProtocolVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

// StatelessVersion is the revision that replaced initialize with
// server/discover.
const StatelessVersion = "2026-07-28"

// metaServerInfo is where 2026-07-28 carries the server's identity.
const metaServerInfo = "io.modelcontextprotocol/serverInfo"

// Server answers MCP requests.
type Server struct {
	// Version is scout-mcp's own version, reported in serverInfo.
	Version string
	// Runner runs scout.
	Runner runner.Runner
	// Allow decides which endpoints scout may be pointed at.
	Allow Allowlist

	mu  sync.Mutex
	out io.Writer
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// The JSON-RPC error codes this server returns.
const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeNoMethod       = -32601
	codeInvalidParams  = -32602
)

// maxLine bounds one message. A tools/call carrying an attestation is the
// largest thing a client sends, and statements are tens of kilobytes.
const maxLine = 4 << 20

// Serve reads requests from in until it closes or ctx ends, answering each on
// out. Requests are answered in turn; a scout run can take a minute, and a
// client that wants two at once can start two servers.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	s.out = out
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64<<10), maxLine)
	for sc.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handle(ctx, line)
	}
	if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (s *Server) handle(ctx context.Context, line []byte) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		s.send(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{codeParse, "parse error: " + err.Error()}})
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		if len(req.ID) > 0 {
			s.send(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{codeInvalidRequest, "not a JSON-RPC 2.0 request"}})
		}
		return
	}
	if len(req.ID) == 0 {
		return // a notification; none needs an answer
	}
	result, rerr := s.dispatch(ctx, req)
	if rerr != nil {
		s.send(response{JSONRPC: "2.0", ID: req.ID, Error: rerr})
		return
	}
	s.send(response{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (s *Server) dispatch(ctx context.Context, req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := ProtocolVersions[0]
		if slices.Contains(ProtocolVersions, p.ProtocolVersion) {
			version = p.ProtocolVersion
		}
		return map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "scout-mcp", "version": s.Version},
			"instructions":    instructions,
		}, nil
	case "server/discover":
		return map[string]any{
			"supportedVersions": append([]string{StatelessVersion}, ProtocolVersions...),
			"capabilities":      map[string]any{"tools": map[string]any{"listChanged": false}},
			"instructions":      instructions,
			"_meta":             map[string]any{metaServerInfo: map[string]any{"name": "scout-mcp", "version": s.Version}},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": tools()}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{codeInvalidParams, "tools/call needs a name and arguments: " + err.Error()}
		}
		call, ok := toolFuncs[p.Name]
		if !ok {
			return nil, &rpcError{codeInvalidParams, fmt.Sprintf("no such tool %q; tools/list names the three this server has", p.Name)}
		}
		if len(p.Arguments) == 0 {
			p.Arguments = json.RawMessage("{}")
		}
		return call(ctx, s, p.Arguments), nil
	default:
		return nil, &rpcError{codeNoMethod, "method not found: " + req.Method}
	}
}

func (s *Server) send(r response) {
	b, err := json.Marshal(r)
	if err != nil {
		b, _ = json.Marshal(response{JSONRPC: "2.0", ID: r.ID, Error: &rpcError{-32603, "internal error: " + err.Error()}})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.out.Write(append(b, '\n'))
}

const instructions = "Evaluates MCP servers with scout: scout_check evaluates an allowlisted endpoint " +
	"and returns the score and every failing check with its guidance; scout_verify_attestation checks a scout " +
	"attestation offline. Every tool is read-only, and scout never calls a mutating tool on the server it evaluates."
