// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package server

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Allowlist decides which endpoints an agent may point scout at.
//
// It is on by default and narrow: with no entries, only loopback addresses
// are allowed, so a server running on the operator's own machine can be
// evaluated and nothing else. scout makes real requests to the endpoint it is
// given, and an agent choosing that endpoint from a prompt is exactly the
// situation in which a request should not go anywhere the operator did not
// name.
type Allowlist struct {
	// Hosts are exact host names, or suffixes when they start with a dot:
	// ".example.com" allows every subdomain of example.com but not
	// example.com itself.
	Hosts []string
}

// ParseAllowlist reads a comma-separated list of hosts.
func ParseAllowlist(s string) Allowlist {
	var a Allowlist
	for _, h := range strings.Split(s, ",") {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			a.Hosts = append(a.Hosts, h)
		}
	}
	return a
}

// Check returns why endpoint is refused, or nil.
func (a Allowlist) Check(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return fmt.Errorf("%q is not an http or https URL; pass the server's Streamable HTTP endpoint, such as http://127.0.0.1:3000/mcp", endpoint)
	}
	if u.User != nil {
		return fmt.Errorf("%q carries credentials in the URL; scout-mcp never sends credentials, so pass the endpoint without them", endpoint)
	}
	host := strings.ToLower(u.Hostname())
	if loopback(host) {
		return nil
	}
	for _, h := range a.Hosts {
		if host == h || (strings.HasPrefix(h, ".") && strings.HasSuffix(host, h)) {
			return nil
		}
	}
	return fmt.Errorf("%s is not on this server's allowlist, which permits loopback addresses%s; the operator widens it with --allow or SCOUT_MCP_ALLOW", host, a.describe())
}

func (a Allowlist) describe() string {
	if len(a.Hosts) == 0 {
		return " only"
	}
	return " and " + strings.Join(a.Hosts, ", ")
}

func loopback(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
