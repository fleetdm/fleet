package spec

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/tools/openapi/parser"
	"gopkg.in/yaml.v3"
)

type AllowedEndpoint struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
}

type Allowlist struct {
	Endpoints []AllowedEndpoint `yaml:"endpoints"`
}

func LoadAllowlist(path string) (Allowlist, error) {
	var a Allowlist
	b, err := os.ReadFile(path)
	if err != nil {
		return a, err
	}
	if err := yaml.Unmarshal(b, &a); err != nil {
		return a, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(a.Endpoints) == 0 {
		return a, fmt.Errorf("%s lists no endpoints", path)
	}
	return a, nil
}

type Info struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

type Server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
}

type Tag struct {
	Name string `yaml:"name"`
}

type Parameter struct {
	Name        string         `yaml:"name"`
	In          string         `yaml:"in"`
	Description string         `yaml:"description,omitempty"`
	Required    bool           `yaml:"required,omitempty"`
	Schema      map[string]any `yaml:"schema"`
}

type MediaType struct {
	Schema  map[string]any `yaml:"schema"`
	Example any            `yaml:"example,omitempty"`
}

type RequestBody struct {
	Required bool                 `yaml:"required"`
	Content  map[string]MediaType `yaml:"content"`
}

type ResponseObj struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content,omitempty"`
}

type Operation struct {
	Summary     string                 `yaml:"summary"`
	Description string                 `yaml:"description,omitempty"`
	OperationID string                 `yaml:"operationId"`
	Tags        []string               `yaml:"tags"`
	Parameters  []Parameter            `yaml:"parameters,omitempty"`
	RequestBody *RequestBody           `yaml:"requestBody,omitempty"`
	Responses   map[string]ResponseObj `yaml:"responses"`
}

type Components struct {
	SecuritySchemes map[string]map[string]string `yaml:"securitySchemes"`
}

type Document struct {
	OpenAPI    string                           `yaml:"openapi"`
	Info       Info                             `yaml:"info"`
	Servers    []Server                         `yaml:"servers"`
	Tags       []Tag                            `yaml:"tags"`
	Paths      map[string]map[string]*Operation `yaml:"paths"`
	Components Components                       `yaml:"components"`
	Security   []map[string][]string            `yaml:"security"`
}

func Build(res *parser.Result, allow Allowlist) (*Document, error) {
	byKey := make(map[string]parser.Endpoint, len(res.Endpoints))
	for _, e := range res.Endpoints {
		byKey[e.Method+" "+e.Path] = e
	}

	doc := &Document{
		OpenAPI: "3.1.0",
		Info: Info{
			Title: "Fleet REST API (pilot)",
			Description: "Pilot OpenAPI specification generated from Fleet's canonical " +
				"REST API reference (docs/REST API/rest-api.md). Covers a subset of " +
				"endpoints. The Markdown reference remains the source of truth.",
			Version: "main",
		},
		Servers: []Server{{
			URL:         "https://fleet.example.com",
			Description: "Your Fleet server URL.",
		}},
		Paths: map[string]map[string]*Operation{},
		Components: Components{
			SecuritySchemes: map[string]map[string]string{
				"bearerAuth": {"type": "http", "scheme": "bearer"},
			},
		},
		Security: []map[string][]string{{"bearerAuth": {}}},
	}

	tags := map[string]bool{}
	opIDs := map[string]string{}
	var missing []string
	for _, ae := range allow.Endpoints {
		e, ok := byKey[ae.Method+" "+ae.Path]
		if !ok {
			missing = append(missing, ae.Method+" "+ae.Path)
			continue
		}
		op := buildOperation(e)
		if prev, dup := opIDs[op.OperationID]; dup {
			return nil, fmt.Errorf("duplicate operationId %q from headings %q and %q", op.OperationID, prev, e.Heading)
		}
		opIDs[op.OperationID] = e.Heading
		if doc.Paths[e.Path] == nil {
			doc.Paths[e.Path] = map[string]*Operation{}
		}
		doc.Paths[e.Path][strings.ToLower(e.Method)] = op
		tags[e.Section] = true
	}
	if len(missing) > 0 {
		msg := fmt.Sprintf("allowlisted endpoint(s) not found in Markdown: %s", strings.Join(missing, ", "))
		for _, s := range res.Skipped {
			msg += fmt.Sprintf("\n  skipped section %q: %s", s.Heading, s.Reason)
		}
		return nil, fmt.Errorf("%s", msg)
	}

	for t := range tags {
		doc.Tags = append(doc.Tags, Tag{Name: t})
	}
	sort.Slice(doc.Tags, func(i, j int) bool { return doc.Tags[i].Name < doc.Tags[j].Name })
	return doc, nil
}

func buildOperation(e parser.Endpoint) *Operation {
	op := &Operation{
		Summary:     e.Heading,
		Description: e.Description,
		OperationID: operationID(e.Heading),
		Tags:        []string{e.Section},
		Responses:   map[string]ResponseObj{},
	}

	var bodyProps map[string]any
	var bodyRequired []string
	for _, p := range e.Params {
		switch p.In {
		case "query", "path":
			op.Parameters = append(op.Parameters, Parameter{
				Name: p.Name, In: p.In, Description: p.Description,
				Required: p.Required, Schema: typeSchema(p.Type),
			})
		case "body":
			if bodyProps == nil {
				bodyProps = map[string]any{}
			}
			s := typeSchema(p.Type)
			s["description"] = p.Description
			bodyProps[p.Name] = s
			if p.Required {
				bodyRequired = append(bodyRequired, p.Name)
			}
		}
	}
	if bodyProps != nil {
		schema := map[string]any{"type": "object", "properties": bodyProps}
		if len(bodyRequired) > 0 {
			sort.Strings(bodyRequired)
			schema["required"] = bodyRequired
		}
		op.RequestBody = &RequestBody{
			Required: len(bodyRequired) > 0,
			Content:  map[string]MediaType{"application/json": {Schema: schema}},
		}
	}

	for _, r := range e.Responses {
		obj := ResponseObj{Description: "Default response."}
		if r.Example != nil {
			obj.Content = map[string]MediaType{
				"application/json": {Schema: Infer(r.Example), Example: r.Example},
			}
		}
		op.Responses[fmt.Sprintf("%d", r.Status)] = obj
	}
	return op
}

// operationID turns "List MDM commands" into "listMDMCommands": first word
// lowercased, later words get an uppercased first letter with inner caps kept.
func operationID(heading string) string {
	words := strings.Fields(heading)
	var b strings.Builder
	for i, w := range words {
		w = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, w)
		if w == "" {
			continue
		}
		if i == 0 {
			b.WriteString(strings.ToLower(w))
		} else {
			b.WriteString(strings.ToUpper(w[:1]) + w[1:])
		}
	}
	return b.String()
}

func typeSchema(docType string) map[string]any {
	switch docType {
	case "integer":
		return map[string]any{"type": "integer"}
	case "boolean":
		return map[string]any{"type": "boolean"}
	case "number", "float":
		return map[string]any{"type": "number"}
	case "array", "list":
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	case "object":
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string"}
	}
}

func (d *Document) Render() ([]byte, error) {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(d); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}
