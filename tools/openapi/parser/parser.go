// Package parser extracts structured endpoint definitions from Fleet's
// canonical REST API Markdown reference (docs/REST API/rest-api.md).
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Param struct {
	Name        string
	Type        string
	In          string // query, path, or body ("json"/"form" in the docs normalize to body)
	Description string
	Required    bool
}

type Response struct {
	Status  int
	Example any // decoded JSON example payload, nil if the docs show none
}

type Endpoint struct {
	Section     string // nearest "## " heading, used as the OpenAPI tag
	Heading     string // the "### " heading text
	Description string
	Method      string
	Path        string // normalized: ":id" becomes "{id}"
	Params      []Param
	Responses   []Response
}

type Skip struct {
	Heading string
	Reason  string
}

type Result struct {
	Endpoints []Endpoint
	Skipped   []Skip
}

// ReasonNoRequestLine is the Skip.Reason recorded for a "### " section with
// no request line. Such a section documents something other than an
// endpoint (for example "Retrieve your API token", which walks through UI
// steps), so it's a tolerated skip rather than a parse failure. Any other
// Skip.Reason means a section that looks like an endpoint failed to parse,
// which HardSkips treats as fatal.
const ReasonNoRequestLine = "no request line found"

// HardSkips returns the subset of res.Skipped that are real parse failures,
// excluding sections that were never endpoints in the first place (no
// request line). Callers should treat a non-empty result as fatal.
func HardSkips(res *Result) []Skip {
	var hard []Skip
	for _, s := range res.Skipped {
		if s.Reason == ReasonNoRequestLine {
			continue
		}
		hard = append(hard, s)
	}
	return hard
}

var requestLineRe = regexp.MustCompile("^`(GET|POST|PUT|PATCH|DELETE|HEAD) (/[^` ]*)`\\s*$")
var statusLineRe = regexp.MustCompile("^`Status: ([0-9]{3})[^`]*`\\s*$")

// requiredRe matches the docs' "**Required**" / "**Required.**" / "**required**"
// markers. The closing "**" must immediately follow "Required" (with at most a
// trailing period), so conditional prose like "**Required if platform is
// Android**" does not match: "if platform is Android" sits before the closing
// "**", not right after "Required".
var requiredRe = regexp.MustCompile(`\*\*[Rr]equired\.?\*\*`)

// Parse walks the entire document. Sections that don't parse as endpoints are
// recorded in Skipped with a reason; they are never fatal here. Callers decide
// which skips matter: see HardSkips.
func Parse(md string) *Result {
	res := &Result{}
	lines := strings.Split(md, "\n")

	section := ""
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if h, ok := strings.CutPrefix(line, "## "); ok && !strings.HasPrefix(line, "###") {
			section = strings.TrimSpace(h)
			continue
		}
		h, ok := strings.CutPrefix(line, "### ")
		if !ok || strings.HasPrefix(line, "#### ") {
			continue
		}
		heading := strings.TrimSpace(h)
		end := i + 1
		for end < len(lines) && !strings.HasPrefix(lines[end], "## ") && !strings.HasPrefix(lines[end], "### ") {
			end++
		}
		ep, err := parseSection(section, heading, lines[i+1:end])
		if err != nil {
			res.Skipped = append(res.Skipped, Skip{Heading: heading, Reason: err.Error()})
		} else {
			res.Endpoints = append(res.Endpoints, *ep)
		}
		i = end - 1
	}
	return res
}

func parseSection(section, heading string, body []string) (*Endpoint, error) {
	ep := &Endpoint{Section: section, Heading: heading}

	// The request line must appear before the first "#### " sub-heading.
	reqIdx := -1
	for i, line := range body {
		if strings.HasPrefix(line, "#### ") {
			break
		}
		if m := requestLineRe.FindStringSubmatch(line); m != nil {
			ep.Method, ep.Path = m[1], normalizePath(m[2])
			reqIdx = i
			break
		}
	}
	if reqIdx == -1 {
		return nil, errors.New(ReasonNoRequestLine)
	}
	ep.Description = descriptionAbove(body[:reqIdx])

	var err error
	ep.Params, err = parseParams(body)
	if err != nil {
		return nil, err
	}
	ep.Responses, err = parseResponses(body)
	if err != nil {
		return nil, err
	}
	return ep, nil
}

// descriptionAbove joins the prose lines above the request line, skipping
// blockquotes (deprecation notes) and blank lines at the edges.
func descriptionAbove(lines []string) string {
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, ">") {
			continue
		}
		out = append(out, t)
	}
	return strings.Join(out, " ")
}

func normalizePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") {
			segs[i] = "{" + s[1:] + "}"
		}
	}
	return strings.Join(segs, "/")
}

func parseParams(body []string) ([]Param, error) {
	start := -1
	for i, l := range body {
		if strings.TrimSpace(l) == "#### Parameters" {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return nil, nil
	}

	var params []Param
	inTable := false
	rowNum := 0
	for _, l := range body[start:] {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "|") {
			if inTable {
				break
			}
			if strings.HasPrefix(t, "#") {
				break // hit the next sub-heading without ever seeing a table
			}
			continue
		}
		inTable = true
		rowNum++
		if rowNum <= 2 {
			continue // header and separator rows
		}
		cells := splitRow(t)
		if len(cells) != 4 {
			return nil, fmt.Errorf("parameters table row has %d columns, want 4: %q", len(cells), t)
		}
		in := strings.ToLower(cells[2])
		if in == "json" || in == "form" {
			in = "body"
		}
		params = append(params, Param{
			Name:        cells[0],
			Type:        strings.ToLower(cells[1]),
			In:          in,
			Description: cells[3],
			Required:    requiredRe.MatchString(cells[3]) || in == "path",
		})
	}
	return params, nil
}

func splitRow(row string) []string {
	row = strings.Trim(row, "|")
	parts := strings.Split(row, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// parseResponses extracts the "##### Default response" block: a Status line
// and an optional fenced json example. Only the default response is modeled;
// error responses share Fleet's standard error envelope and are out of the
// pilot's scope.
func parseResponses(body []string) ([]Response, error) {
	for i, l := range body {
		if strings.TrimSpace(l) != "##### Default response" {
			continue
		}
		resp := Response{Status: 200}
		found := false
		for j := i + 1; j < len(body); j++ {
			t := strings.TrimSpace(body[j])
			if strings.HasPrefix(t, "#") {
				break
			}
			if m := statusLineRe.FindStringSubmatch(t); m != nil {
				resp.Status, _ = strconv.Atoi(m[1])
				found = true
				continue
			}
			if t == "```json" {
				var buf []string
				for k := j + 1; k < len(body); k++ {
					if strings.TrimSpace(body[k]) == "```" {
						break
					}
					buf = append(buf, body[k])
				}
				var example any
				if err := json.Unmarshal(stripLineComments([]byte(strings.Join(buf, "\n"))), &example); err != nil {
					return nil, fmt.Errorf("invalid JSON in default response example: %v", err)
				}
				resp.Example = example
				break
			}
		}
		if !found && resp.Example == nil {
			return nil, fmt.Errorf("default response block has no Status line or example")
		}
		return []Response{resp}, nil
	}
	return nil, fmt.Errorf("no default response block found")
}

// stripLineComments removes "// comment" annotations the docs use inside
// otherwise-valid JSON examples (for example, "// Fleet Premium only"),
// leaving "//" inside string literals (URLs, etc.) untouched.
func stripLineComments(b []byte) []byte {
	var out []byte
	inString := false
	escaped := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inString {
			out = append(out, c)
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(b) && b[i+1] == '/' {
			for i < len(b) && b[i] != '\n' {
				i++
			}
			if i < len(b) {
				out = append(out, b[i]) // keep the newline
			}
			continue
		}
		out = append(out, c)
	}
	return out
}
