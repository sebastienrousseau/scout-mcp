// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sebastienrousseau/scout-reporting/attestation"
)

// checkTimeout bounds one scout run. A full run at scout's default pacing
// takes well under a minute against a responsive server; this is the
// ceiling for one that is not.
const checkTimeout = 5 * time.Minute

// phases are scout's phase names, which scout_check accepts to narrow a run.
var phases = []string{"net", "discovery", "auth", "handshake", "protocol", "catalog", "execution", "performance", "resilience"}

// maxFailures bounds the failures listed in one result, so an agent's
// context is not filled by a server that fails everything.
const maxFailures = 25

type result struct {
	Content           []content `json:"content"`
	StructuredContent any       `json:"structuredContent,omitempty"`
	IsError           bool      `json:"isError,omitempty"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func textResult(text string, structured any) result {
	return result{Content: []content{{Type: "text", Text: text}}, StructuredContent: structured}
}

func errorResult(text string) result {
	return result{Content: []content{{Type: "text", Text: text}}, IsError: true}
}

type toolFunc func(ctx context.Context, s *Server, args json.RawMessage) result

var toolFuncs = map[string]toolFunc{
	"scout_check":              callCheck,
	"scout_verify_attestation": callVerify,
	"scout_version":            callVersion,
}

func readOnly(openWorld bool) map[string]any {
	return map[string]any{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": openWorld}
}

func tools() []map[string]any {
	return []map[string]any{
		{
			"name":  "scout_check",
			"title": "Evaluate an MCP server with scout",
			"description": "Evaluate an MCP server's Streamable HTTP endpoint with scout's diagnostic and return its score, grade, counts and " +
				"every failing check with what it means and a link to the fix. Use it to find out whether a server is ready for agents " +
				"before relying on it. The endpoint must be on this server's allowlist (loopback addresses by default). No credentials " +
				"leave this machine, and scout calls only the server's tools that declare readOnlyHint. An evaluation takes up to a minute.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"endpoint": map[string]any{"type": "string", "description": "The server's Streamable HTTP URL, such as http://127.0.0.1:3000/mcp."},
					"phases": map[string]any{
						"type": "array", "items": map[string]any{"type": "string", "enum": phases},
						"description": "Evaluate only these phases; omit for all nine.",
					},
				},
				"required":             []string{"endpoint"},
				"additionalProperties": false,
			},
			"outputSchema": checkOutputSchema(),
			"annotations":  readOnly(true),
		},
		{
			"name":  "scout_verify_attestation",
			"title": "Verify a scout attestation",
			"description": "Check a scout attestation, the in-toto statement scout produces about one MCP server, offline: its structure, " +
				"its subject digest, and optionally that it is about a given endpoint. Returns whether it is valid, what it says, and why " +
				"not when it is not. Nothing is fetched; pass the statement's JSON text.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"statement": map[string]any{"type": "string", "description": "The attestation's JSON text."},
					"endpoint":  map[string]any{"type": "string", "description": "When given, the endpoint the statement must be about."},
				},
				"required":             []string{"statement"},
				"additionalProperties": false,
			},
			"outputSchema": verifyOutputSchema(),
			"annotations":  readOnly(false),
		},
		{
			"name":        "scout_version",
			"title":       "Report the versions in use",
			"description": "Return scout-mcp's version and the version of the scout program it uses, for a bug report or to confirm a check exists.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false},
			"outputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scoutMcp": map[string]any{"type": "string"},
					"scout":    map[string]any{"type": "string"},
				},
				"required": []string{"scoutMcp", "scout"},
			},
			"annotations": readOnly(false),
		},
	}
}

func checkOutputSchema() map[string]any {
	failure := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "string"}, "phase": map[string]any{"type": "string"},
			"severity": map[string]any{"type": "string"}, "title": map[string]any{"type": "string"},
			"detail": map[string]any{"type": "string"}, "doc": map[string]any{"type": "string"},
		},
		"required": []string{"id", "phase", "detail"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"endpoint": map[string]any{"type": "string"},
			"score":    map[string]any{"type": "number"},
			"grade":    map[string]any{"type": "string"},
			"counts": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pass": map[string]any{"type": "integer"}, "warn": map[string]any{"type": "integer"},
					"fail": map[string]any{"type": "integer"}, "skip": map[string]any{"type": "integer"}, "info": map[string]any{"type": "integer"},
				},
			},
			"blocked":   map[string]any{"type": "string"},
			"failures":  map[string]any{"type": "array", "items": failure},
			"truncated": map[string]any{"type": "boolean"},
		},
		"required": []string{"endpoint", "score", "grade", "counts", "failures"},
	}
}

func verifyOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"valid":    map[string]any{"type": "boolean"},
			"error":    map[string]any{"type": "string"},
			"covers":   map[string]any{"type": "boolean"},
			"endpoint": map[string]any{"type": "string"},
			"score":    map[string]any{"type": "number"},
			"grade":    map[string]any{"type": "string"},
			"failing":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"valid"},
	}
}

type checkFailure struct {
	ID       string `json:"id"`
	Phase    string `json:"phase"`
	Severity string `json:"severity,omitempty"`
	Title    string `json:"title,omitempty"`
	Detail   string `json:"detail"`
	Doc      string `json:"doc,omitempty"`
}

type checkOutput struct {
	Endpoint  string         `json:"endpoint"`
	Score     float64        `json:"score"`
	Grade     string         `json:"grade"`
	Counts    map[string]int `json:"counts"`
	Blocked   string         `json:"blocked,omitempty"`
	Failures  []checkFailure `json:"failures"`
	Truncated bool           `json:"truncated,omitempty"`
}

func callCheck(ctx context.Context, s *Server, raw json.RawMessage) result {
	var args struct {
		Endpoint string   `json:"endpoint"`
		Phases   []string `json:"phases"`
	}
	if err := strictDecode(raw, &args); err != nil {
		return errorResult("scout_check takes {\"endpoint\": \"http://…/mcp\", \"phases\": [optional]}: " + err.Error())
	}
	if strings.TrimSpace(args.Endpoint) == "" {
		return errorResult("scout_check needs an endpoint: the server's Streamable HTTP URL, such as http://127.0.0.1:3000/mcp")
	}
	for _, p := range args.Phases {
		if !slices.Contains(phases, p) {
			return errorResult(fmt.Sprintf("%q is not a scout phase; the phases are %s", p, strings.Join(phases, ", ")))
		}
	}
	if err := s.Allow.Check(args.Endpoint); err != nil {
		return errorResult(err.Error())
	}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	report, err := s.Runner.Check(ctx, args.Endpoint, args.Phases)
	if err != nil {
		return errorResult("scout could not evaluate " + args.Endpoint + ": " + err.Error())
	}
	out := checkOutput{
		Endpoint: args.Endpoint, Score: report.Score.Total, Grade: report.Score.Grade, Blocked: report.Blocked,
		Counts:   map[string]int{"pass": report.Counts.Pass, "warn": report.Counts.Warn, "fail": report.Counts.Fail, "skip": report.Counts.Skip, "info": report.Counts.Info},
		Failures: []checkFailure{},
	}
	for _, f := range report.Failures() {
		if len(out.Failures) == maxFailures {
			out.Truncated = true
			break
		}
		out.Failures = append(out.Failures, checkFailure{ID: f.ID, Phase: f.Phase, Severity: f.Severity, Title: f.Title, Detail: f.Detail, Doc: f.DocURL})
	}
	return textResult(summarise(out), out)
}

func summarise(o checkOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s scored %.0f/100 (%s): %d pass, %d warn, %d fail, %d skipped.\n",
		o.Endpoint, o.Score, o.Grade, o.Counts["pass"], o.Counts["warn"], o.Counts["fail"], o.Counts["skip"])
	if o.Blocked != "" {
		fmt.Fprintf(&b, "The run stopped early: %s.\n", o.Blocked)
	}
	for _, f := range o.Failures {
		fmt.Fprintf(&b, "- %s", f.ID)
		if f.Severity != "" {
			fmt.Fprintf(&b, " (%s)", f.Severity)
		}
		fmt.Fprintf(&b, ": %s", f.Detail)
		if f.Doc != "" {
			fmt.Fprintf(&b, " See %s", f.Doc)
		}
		b.WriteString("\n")
	}
	if o.Truncated {
		fmt.Fprintf(&b, "Only the first %d failures are listed.\n", maxFailures)
	}
	return strings.TrimRight(b.String(), "\n")
}

type verifyOutput struct {
	Valid    bool     `json:"valid"`
	Error    string   `json:"error,omitempty"`
	Covers   *bool    `json:"covers,omitempty"`
	Endpoint string   `json:"endpoint,omitempty"`
	Score    *float64 `json:"score,omitempty"`
	Grade    string   `json:"grade,omitempty"`
	Failing  []string `json:"failing,omitempty"`
}

func callVerify(_ context.Context, _ *Server, raw json.RawMessage) result {
	var args struct {
		Statement string `json:"statement"`
		Endpoint  string `json:"endpoint"`
	}
	if err := strictDecode(raw, &args); err != nil {
		return errorResult("scout_verify_attestation takes {\"statement\": \"<JSON text>\", \"endpoint\": optional}: " + err.Error())
	}
	if strings.TrimSpace(args.Statement) == "" {
		return errorResult("scout_verify_attestation needs the statement's JSON text in \"statement\"")
	}
	st, err := attestation.Parse([]byte(args.Statement))
	if err == nil {
		err = st.Validate()
	}
	if err != nil {
		// A statement that does not verify is an answer, not a failure of
		// the tool: the agent asked whether it was valid.
		out := verifyOutput{Valid: false, Error: err.Error()}
		return textResult("Not a valid scout attestation: "+err.Error(), out)
	}
	p := st.Predicate
	out := verifyOutput{Valid: true, Endpoint: p.Target.Endpoint}
	if p.Score != nil {
		out.Score, out.Grade = &p.Score.Total, p.Score.Grade
	}
	for _, v := range p.Verdicts {
		if v.Status == "fail" && len(out.Failing) < maxFailures {
			out.Failing = append(out.Failing, v.ID)
		}
	}
	text := fmt.Sprintf("Valid scout attestation about %s %s", p.Target.Transport, p.Target.Endpoint)
	if out.Score != nil {
		text += fmt.Sprintf(", scored %.0f/100 (%s)", *out.Score, out.Grade)
	}
	text += fmt.Sprintf(" by %s %s at %s.", p.Instrument.Name, p.Instrument.Version, p.RanAt.UTC().Format(time.RFC3339))
	if args.Endpoint != "" {
		covers := st.Covers("http", args.Endpoint)
		out.Covers = &covers
		if covers {
			text += " It is about " + args.Endpoint + "."
		} else {
			text += " It is not about " + args.Endpoint + ", so it is no evidence about that server."
		}
	}
	if len(out.Failing) > 0 {
		text += " Failing: " + strings.Join(out.Failing, ", ") + "."
	}
	text += " Validity is structure and integrity; who signed the statement is checked with its envelope, such as cosign or gh attestation verify."
	return textResult(text, out)
}

func callVersion(ctx context.Context, s *Server, raw json.RawMessage) result {
	var args struct{}
	if err := strictDecode(raw, &args); err != nil {
		return errorResult("scout_version takes no arguments")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	v, err := s.Runner.Version(ctx)
	if err != nil {
		return errorResult("scout-mcp " + s.Version + " could not run scout: " + err.Error() + "; install scout, or pass its path with --scout")
	}
	out := map[string]string{"scoutMcp": s.Version, "scout": v}
	return textResult("scout-mcp "+s.Version+", running "+v, out)
}

// strictDecode refuses unknown fields, so a misspelt argument is reported
// rather than silently ignored.
func strictDecode(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
