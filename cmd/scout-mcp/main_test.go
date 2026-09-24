// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	var out, errw bytes.Buffer
	if err := run(context.Background(), []string{"--version"}, strings.NewReader(""), &out, &errw); err != nil || !strings.HasPrefix(out.String(), "scout-mcp ") {
		t.Errorf("--version: %v %q", err, out.String())
	}
	out.Reset()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	if err := run(context.Background(), []string{"--allow", "a.example"}, in, &out, &errw); err != nil || !strings.Contains(out.String(), `"id":1`) {
		t.Errorf("serve: %v %q", err, out.String())
	}
	if err := run(context.Background(), []string{"--nope"}, strings.NewReader(""), &out, &errw); err == nil {
		t.Error("an unknown flag was accepted")
	}
}
