// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

// Package runner runs the scout binary on the server's behalf.
//
// scout-mcp does not link scout's engine: that lives in scout's internal
// packages and would tie this server to one build of it. It runs the scout
// program instead, the same artefact an operator runs by hand, and reads the
// JSON report it prints. That keeps the version the server reports honest —
// it is whatever `scout version` says — and keeps every safety property scout
// has, because the server can only ask for what scout's own flags allow.
//
// Three things are fixed for every run, whatever the caller asks:
//
//   - No configuration file. SCOUT_CONFIG points at an empty one, so no
//     profile or defaults block the operator wrote for their own use can
//     switch on mutations, add credentials or redirect the report.
//   - No credentials. --auth none, always. An agent chooses the endpoint;
//     the operator's secrets must not travel to wherever it points.
//   - Read-only. Nothing that allows a mutating or destructive tool is ever
//     passed; scout's default is to invoke only tools declaring readOnlyHint.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Finding is one check's outcome, as the report carries it.
type Finding struct {
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Status   string `json:"status"`
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	DocURL   string `json:"doc_url,omitempty"`
}

// Report is the part of scout's JSON report the server uses.
type Report struct {
	Target struct {
		Endpoint string `json:"endpoint"`
	} `json:"target"`
	Blocked string `json:"blocked"`
	Counts  struct {
		Pass int `json:"pass"`
		Warn int `json:"warn"`
		Fail int `json:"fail"`
		Skip int `json:"skip"`
		Info int `json:"info"`
	} `json:"counts"`
	Score struct {
		Total float64 `json:"total"`
		Grade string  `json:"grade"`
	} `json:"score"`
	Phases []struct {
		Name     string    `json:"name"`
		Findings []Finding `json:"findings"`
	} `json:"phases"`
}

// Failures returns every failing finding, in report order.
func (r Report) Failures() []Finding {
	var out []Finding
	for _, p := range r.Phases {
		for _, f := range p.Findings {
			if f.Status == "fail" {
				out = append(out, f)
			}
		}
	}
	return out
}

// Runner runs scout.
type Runner interface {
	// Check evaluates endpoint, limited to phases when any are given.
	Check(ctx context.Context, endpoint string, phases []string) (Report, error)
	// Version is what `scout version` prints.
	Version(ctx context.Context) (string, error)
}

// Exec runs the scout program at Path.
type Exec struct {
	// Path is the scout binary. Empty means "scout" on PATH.
	Path string
}

// ErrNoReport means scout exited without a report: it could not run at all.
var ErrNoReport = errors.New("scout produced no report")

func (e Exec) bin() string {
	if e.Path != "" {
		return e.Path
	}
	return "scout"
}

// Check runs `scout check` with the fixed safety settings and parses the
// report. Exit 0 and 2 both carry a report; 2 means a check failed, which is
// a result, not an error.
func (e Exec) Check(ctx context.Context, endpoint string, phases []string) (Report, error) {
	dir, err := os.MkdirTemp("", "scout-mcp-")
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	cfg := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfg, []byte("{}\n"), 0o600); err != nil {
		return Report{}, err
	}
	args := []string{"check", endpoint, "--auth", "none", "--output", "json", "--no-color"}
	if len(phases) > 0 {
		args = append(args, "--phases", strings.Join(phases, ","))
	}
	cmd := exec.CommandContext(ctx, e.bin(), args...) // #nosec G204 -- the arguments are built here, the endpoint is allowlisted by the caller
	cmd.Env = append(os.Environ(), "SCOUT_CONFIG="+cfg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	// Exit 2 means a check failed and the report is on stdout: a result.
	var exit *exec.ExitError
	failedChecks := errors.As(runErr, &exit) && exit.ExitCode() == 2
	if runErr != nil && !failedChecks {
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		if stdout.Len() == 0 {
			return Report{}, fmt.Errorf("%w: %w: %s", ErrNoReport, runErr, msg)
		}
	}
	var r Report
	if err := json.Unmarshal(stdout.Bytes(), &r); err != nil {
		return Report{}, fmt.Errorf("%w: the output was not a scout report: %w", ErrNoReport, err)
	}
	return r, nil
}

// Version runs `scout version`.
func (e Exec) Version(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, e.bin(), "version").Output() // #nosec G204 -- a fixed argument
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
