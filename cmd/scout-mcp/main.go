// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Command scout-mcp is an MCP server over stdio that exposes scout's
// diagnostics as tools. An MCP host starts it as a child process:
//
//	{"command": "scout-mcp", "args": ["--allow", ".internal.example.com"]}
//
// It evaluates only loopback endpoints unless --allow (or SCOUT_MCP_ALLOW)
// names more hosts, sends no credentials, and runs the scout program found
// on PATH or at --scout.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sebastienrousseau/scout-mcp/internal/runner"
	"github.com/sebastienrousseau/scout-mcp/internal/server"
)

// Version is stamped by the release build; a local build says dev.
var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "scout-mcp:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, in io.Reader, out, errw io.Writer) error {
	fs := flag.NewFlagSet("scout-mcp", flag.ContinueOnError)
	fs.SetOutput(errw)
	allow := fs.String("allow", os.Getenv("SCOUT_MCP_ALLOW"), "hosts scout may be pointed at besides loopback, comma-separated; a leading dot allows subdomains")
	scoutPath := fs.String("scout", os.Getenv("SCOUT_MCP_SCOUT"), "path of the scout program; empty means scout on PATH")
	version := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version {
		_, err := fmt.Fprintln(out, "scout-mcp", Version)
		return err
	}
	s := &server.Server{
		Version: Version,
		Runner:  runner.Exec{Path: *scoutPath},
		Allow:   server.ParseAllowlist(*allow),
	}
	return s.Serve(ctx, in, out)
}
