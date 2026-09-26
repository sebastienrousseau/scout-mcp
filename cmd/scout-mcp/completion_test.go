// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCompletionCoversEveryFlagInEveryShell(t *testing.T) {
	for _, shell := range completionShells {
		var out, errw bytes.Buffer
		if err := run(context.Background(), []string{"--completion", shell}, strings.NewReader(""), &out, &errw); err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		script := out.String()
		for _, flag := range []string{"allow", "scout", "version", "completion"} {
			if !strings.Contains(script, flag) {
				t.Errorf("%s completion does not mention --%s:\n%s", shell, flag, script)
			}
		}
		for _, want := range map[string][]string{
			"bash": {"complete -F _scout_mcp scout-mcp", "--scout) COMPREPLY=($(compgen -f", `compgen -W "bash fish zsh"`},
			"zsh":  {"#compdef scout-mcp", "--scout[", ":path:_files", ":shell:(bash fish zsh)"},
			"fish": {"complete -c scout-mcp -l scout", "-r -F", `-x -a "bash fish zsh"`},
		}[shell] {
			if !strings.Contains(script, want) {
				t.Errorf("%s completion lacks %q:\n%s", shell, want, script)
			}
		}
	}
}

func TestCompletionRefusesAnUnknownShell(t *testing.T) {
	var out, errw bytes.Buffer
	err := run(context.Background(), []string{"--completion", "powershell"}, strings.NewReader(""), &out, &errw)
	if err == nil || !strings.Contains(err.Error(), "bash, fish, zsh") {
		t.Errorf("err = %v, want a refusal naming the supported shells", err)
	}
	if out.Len() != 0 {
		t.Errorf("wrote %q for an unknown shell", out.String())
	}
}

func TestZshEscape(t *testing.T) {
	if got := zshEscape("a [b]: it's"); got != `a \[b\]\: it'\''s` {
		t.Errorf("zshEscape = %q", got)
	}
}

func TestIsBool(t *testing.T) {
	var out, errw bytes.Buffer
	_ = run(context.Background(), []string{"--completion", "fish"}, strings.NewReader(""), &out, &errw)
	if !strings.Contains(out.String(), "-l version -d \"print the version and exit\"\n") {
		t.Errorf("a boolean flag was given a value requirement:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "-l allow -d") || !strings.Contains(out.String(), "subdomains\" -x\n") {
		t.Errorf("a value flag was not marked as taking one:\n%s", out.String())
	}
}
