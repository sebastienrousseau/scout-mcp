// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sebastienrousseau/scout-mcp/internal/runner"
	"github.com/sebastienrousseau/scout-reporting/attestation"
)

type fakeRunner struct {
	report   runner.Report
	err      error
	version  string
	verErr   error
	endpoint string
	phases   []string
}

func (f *fakeRunner) Check(_ context.Context, endpoint string, phases []string) (runner.Report, error) {
	f.endpoint, f.phases = endpoint, phases
	return f.report, f.err
}

func (f *fakeRunner) Version(context.Context) (string, error) { return f.version, f.verErr }

// exchange sends lines to a server and returns its responses by id.
func exchange(t *testing.T, s *Server, lines ...string) map[string]map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := s.Serve(context.Background(), strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]any{}
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("not JSON on stdout: %q", l)
		}
		id, _ := json.Marshal(m["id"])
		got[string(id)] = m
	}
	return got
}

func call(name string, args any) string {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 7, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
	return string(b)
}

func resultOf(t *testing.T, m map[string]any) (text string, isError bool, structured map[string]any) {
	t.Helper()
	r, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", m)
	}
	c := r["content"].([]any)[0].(map[string]any)
	isError, _ = r["isError"].(bool)
	structured, _ = r["structuredContent"].(map[string]any)
	return c["text"].(string), isError, structured
}

func TestHandshakeAndList(t *testing.T) {
	s := &Server{Version: "0.0.5", Runner: &fakeRunner{}}
	got := exchange(t, s,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`,
	)
	init := got["1"]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-06-18" || init["serverInfo"].(map[string]any)["version"] != "0.0.5" {
		t.Errorf("initialize = %v", init)
	}
	if got["4"]["result"].(map[string]any)["protocolVersion"] != ProtocolVersions[0] {
		t.Errorf("an unknown revision should get the newest: %v", got["4"])
	}
	if _, ok := got["2"]["result"]; !ok {
		t.Errorf("ping = %v", got["2"])
	}
	list := got["3"]["result"].(map[string]any)["tools"].([]any)
	if len(list) != 3 {
		t.Fatalf("%d tools", len(list))
	}
	for _, tl := range list {
		tool := tl.(map[string]any)
		ann := tool["annotations"].(map[string]any)
		if ann["readOnlyHint"] != true || ann["destructiveHint"] != false {
			t.Errorf("%s is not read-only: %v", tool["name"], ann)
		}
		if tool["outputSchema"] == nil || tool["description"] == "" {
			t.Errorf("%s lacks an output schema or a description", tool["name"])
		}
	}
	if len(got) != 4 {
		t.Errorf("the notification was answered: %d responses", len(got))
	}
}

func TestProtocolErrors(t *testing.T) {
	s := &Server{Runner: &fakeRunner{}}
	got := exchange(t, s,
		`not json`,
		`{"jsonrpc":"1.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope"}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":"x"}`,
		``,
	)
	code := func(id string) float64 { return got[id]["error"].(map[string]any)["code"].(float64) }
	for id, want := range map[string]float64{"null": codeParse, "2": codeInvalidRequest, "3": codeNoMethod, "4": codeInvalidParams, "5": codeInvalidParams} {
		if code(id) != want {
			t.Errorf("id %s: code %v, want %v", id, code(id), want)
		}
	}
}

func TestCheck(t *testing.T) {
	var report runner.Report
	if err := json.Unmarshal([]byte(`{"target":{"endpoint":"http://127.0.0.1:9/mcp"},"blocked":"",
		"counts":{"pass":10,"warn":1,"fail":2,"skip":3,"info":4},"score":{"total":71,"grade":"C"},
		"phases":[{"name":"auth","findings":[{"id":"auth.unauthenticated_tools","phase":"auth","status":"fail","severity":"critical","title":"t","detail":"anyone can call delete","doc_url":"https://scoutmcp.io/c"}]},
		{"name":"protocol","findings":null},
		{"name":"catalog","findings":[{"id":"catalog.x","phase":"catalog","status":"pass","detail":"ok"},{"id":"catalog.y","phase":"catalog","status":"fail","detail":"bad"}]}]}`), &report); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRunner{report: report}
	s := &Server{Runner: fr}
	text, isErr, st := resultOf(t, exchange(t, s, call("scout_check", map[string]any{"endpoint": "http://127.0.0.1:9/mcp", "phases": []string{"auth"}}))["7"])
	if isErr || st["score"].(float64) != 71 || st["grade"] != "C" || len(st["failures"].([]any)) != 2 {
		t.Fatalf("result = %v %v", isErr, st)
	}
	if !strings.Contains(text, "71/100 (C)") || !strings.Contains(text, "auth.unauthenticated_tools (critical): anyone can call delete See https://scoutmcp.io/c") {
		t.Errorf("text = %q", text)
	}
	if fr.endpoint != "http://127.0.0.1:9/mcp" || len(fr.phases) != 1 {
		t.Errorf("runner got %q %v", fr.endpoint, fr.phases)
	}
}

func TestCheckTruncatesAndReportsBlocked(t *testing.T) {
	var r runner.Report
	r.Blocked = "endpoint is not reachable"
	r.Phases = append(r.Phases, struct {
		Name     string           `json:"name"`
		Findings []runner.Finding `json:"findings"`
	}{Name: "catalog"})
	for range maxFailures + 5 {
		r.Phases[0].Findings = append(r.Phases[0].Findings, runner.Finding{ID: "catalog.x", Status: "fail", Detail: "d"})
	}
	s := &Server{Runner: &fakeRunner{report: r}}
	text, _, st := resultOf(t, exchange(t, s, call("scout_check", map[string]any{"endpoint": "http://localhost:1/mcp"}))["7"])
	if st["truncated"] != true || len(st["failures"].([]any)) != maxFailures {
		t.Errorf("not truncated: %v", st)
	}
	if !strings.Contains(text, "stopped early: endpoint is not reachable") || !strings.Contains(text, "Only the first") {
		t.Errorf("text = %q", text)
	}
}

func TestCheckRefusals(t *testing.T) {
	fr := &fakeRunner{err: errors.New("boom")}
	s := &Server{Runner: fr, Allow: ParseAllowlist(".corp.example, mcp.example.com")}
	cases := map[string]struct {
		args any
		want string
	}{
		"not allowed":       {map[string]any{"endpoint": "https://evil.example.net/mcp"}, "not on this server's allowlist"},
		"not a url":         {map[string]any{"endpoint": "ftp://x/mcp"}, "not an http or https URL"},
		"credentials":       {map[string]any{"endpoint": "http://u:p@127.0.0.1/mcp"}, "carries credentials"},
		"empty":             {map[string]any{"endpoint": " "}, "needs an endpoint"},
		"unknown field":     {map[string]any{"endpoint": "http://127.0.0.1/mcp", "token": "x"}, "takes"},
		"bad phase":         {map[string]any{"endpoint": "http://127.0.0.1/mcp", "phases": []string{"exploit"}}, "not a scout phase"},
		"runner fails":      {map[string]any{"endpoint": "https://a.corp.example/mcp"}, "could not evaluate https://a.corp.example/mcp: boom"},
		"exact host passes": {map[string]any{"endpoint": "https://mcp.example.com/mcp"}, "could not evaluate"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			text, isErr, _ := resultOf(t, exchange(t, s, call("scout_check", tc.args))["7"])
			if !isErr || !strings.Contains(text, tc.want) {
				t.Errorf("%v %q", isErr, text)
			}
		})
	}
	if err := (Allowlist{Hosts: []string{".corp.example"}}).Check("https://corp.example/mcp"); err == nil {
		t.Error("a leading dot allowed the parent domain")
	}
	for _, ok := range []string{"http://localhost:3000/mcp", "http://127.0.0.2/mcp", "http://[::1]/mcp", "http://app.localhost/mcp"} {
		if err := (Allowlist{}).Check(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
}

func signedStatement(t *testing.T) string {
	t.Helper()
	target := attestation.Target{Transport: "http", Endpoint: "https://mcp.example.com/mcp"}
	st := &attestation.Statement{
		Type: attestation.StatementType, PredicateType: attestation.PredicateType,
		Subject: []attestation.Subject{attestation.SubjectFor(target)},
		Predicate: attestation.Evaluation{
			SubjectKind: attestation.SubjectKindDescriptor, Target: target,
			JudgedAgainst: attestation.Basis{SpecRevision: "2026-07-28", Rubric: "1", CheckInventory: "1"},
			Instrument:    attestation.Instrument{Name: "scout", Version: "0.0.5", SchemaVersion: 1},
			RanAt:         time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), Took: "1s",
			Verdicts: []attestation.Verdict{{ID: "protocol.origin", Phase: "protocol", Status: "fail", Severity: "major"}},
			Counts:   attestation.Counts{Fail: 1},
			Score:    &attestation.Score{Total: 88, Grade: "B", Assessed: 6, Of: 6},
		},
	}
	b, err := st.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestVerifyAttestation(t *testing.T) {
	s := &Server{Runner: &fakeRunner{}}
	good := signedStatement(t)

	text, isErr, st := resultOf(t, exchange(t, s, call("scout_verify_attestation", map[string]any{"statement": good, "endpoint": "https://mcp.example.com/mcp"}))["7"])
	if isErr || st["valid"] != true || st["covers"] != true || st["score"].(float64) != 88 || !strings.Contains(text, "Failing: protocol.origin") {
		t.Errorf("good: %v %q %v", isErr, text, st)
	}
	_, _, st = resultOf(t, exchange(t, s, call("scout_verify_attestation", map[string]any{"statement": good, "endpoint": "https://other.example/mcp"}))["7"])
	if st["covers"] != false {
		t.Errorf("other endpoint: %v", st)
	}
	tampered := strings.Replace(good, "https://mcp.example.com/mcp", "https://evil.example/mcp", 1)
	text, isErr, st = resultOf(t, exchange(t, s, call("scout_verify_attestation", map[string]any{"statement": tampered}))["7"])
	if isErr || st["valid"] != false || !strings.Contains(text, "Not a valid") {
		t.Errorf("tampered: %v %q %v", isErr, text, st)
	}
	for _, args := range []any{map[string]any{"statement": ""}, map[string]any{"x": 1}} {
		if _, isErr, _ := resultOf(t, exchange(t, s, call("scout_verify_attestation", args))["7"]); !isErr {
			t.Errorf("%v was not refused", args)
		}
	}
}

func TestVersion(t *testing.T) {
	s := &Server{Version: "0.0.5", Runner: &fakeRunner{version: "scout 0.0.5"}}
	text, isErr, st := resultOf(t, exchange(t, s, call("scout_version", map[string]any{}))["7"])
	if isErr || st["scout"] != "scout 0.0.5" || !strings.Contains(text, "scout-mcp 0.0.5") {
		t.Errorf("%v %q %v", isErr, text, st)
	}
	s.Runner = &fakeRunner{verErr: errors.New("not found")}
	if text, isErr, _ := resultOf(t, exchange(t, s, call("scout_version", nil))["7"]); !isErr || !strings.Contains(text, "--scout") {
		t.Errorf("%v %q", isErr, text)
	}
	if _, isErr, _ := resultOf(t, exchange(t, s, call("scout_version", map[string]any{"x": 1}))["7"]); !isErr {
		t.Error("an argument was accepted")
	}
}

func TestServeStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &Server{Runner: &fakeRunner{}}
	if err := s.Serve(ctx, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`+"\n"), &bytes.Buffer{}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

// The 2026-07-28 revision has no handshake: a client learns the server from
// server/discover, which carries the identity in _meta, and then calls tools
// with no initialize first.
func TestStatelessRevision(t *testing.T) {
	s := &Server{Version: "0.0.5", Runner: &fakeRunner{version: "scout 0.0.5"}}
	got := exchange(t, s,
		`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"scout_version","arguments":{},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
	)
	d := got["1"]["result"].(map[string]any)
	versions := d["supportedVersions"].([]any)
	if versions[0] != StatelessVersion || len(versions) != 1+len(ProtocolVersions) {
		t.Errorf("supportedVersions = %v", versions)
	}
	info := d["_meta"].(map[string]any)[metaServerInfo].(map[string]any)
	if info["name"] != "scout-mcp" || info["version"] != "0.0.5" || d["instructions"] == "" {
		t.Errorf("discover = %v", d)
	}
	if _, isErr, _ := resultOf(t, got["7"]); isErr {
		t.Errorf("a tool call without initialize failed: %v", got["7"])
	}
}
