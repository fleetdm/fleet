// Package parser extracts structured endpoint definitions from Fleet's
// canonical REST API Markdown reference (docs/REST API/rest-api.md).
package parser

import (
	"fmt"
	"regexp"
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

var requestLineRe = regexp.MustCompile("^`(GET|POST|PUT|PATCH|DELETE|HEAD) (/[^` ]*)`\\s*$")

// Parse walks the entire document. Sections that don't parse as endpoints are
// recorded in Skipped with a reason; they are never fatal here. Callers decide
// which skips matter (allowlisted endpoints missing is the caller's error).
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
		return nil, fmt.Errorf("no request line found")
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

func parseParams(body []string) ([]Param, error)       { return nil, nil }
func parseResponses(body []string) ([]Response, error) { return nil, nil }
