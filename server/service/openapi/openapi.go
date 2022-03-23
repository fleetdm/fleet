package openapi

import (
	"encoding/json"
	"io"

	"github.com/invopop/jsonschema"
)

type endpoint struct {
	name string
	verb string
	path string
	req  interface{}
}

type Document struct {
	// This string MUST be the semantic version number of the OpenAPI
	// Specification version that the OpenAPI document uses.
	OpenAPI string `json:"openapi"`

	// Provides metadata about the API.
	Info Info `json:"info"`

	Servers []*Server      `json:"servers,omitempty"`
	Paths   []*PathPattern `json:"paths,omitempty"`
	// Components
	// Security
	// Tags
	// ExternalDocs

	endpoints []*endpoint
}

func (d *Document) Render(w io.Writer) error {
}

func (d *Document) RegisterEndpoint(name, verb, path string, req interface{}) {
	d.endpoints = append(d.endpoints, &endpoint{name, verb, path, req})
}

type Info struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version"`
}

type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	// Variables
}

type PathPattern struct {
	Pattern string
	Item    PathItem
}

func (p PathPattern) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]PathItem{p.Pattern: p.Item})
}

type PathItem struct {
	Ref         string `json:"$ref,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`

	GET     *Operation `json:"get,omitempty"`
	PUT     *Operation `json:"put,omitempty"`
	POST    *Operation `json:"post,omitempty"`
	DELETE  *Operation `json:"delete,omitempty"`
	OPTIONS *Operation `json:"options,omitempty"`
	HEAD    *Operation `json:"head,omitempty"`
	PATCH   *Operation `json:"patch,omitempty"`
	TRACE   *Operation `json:"trace,omitempty"`

	Servers    []*Server    `json:"servers,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"` // must be "query", "header", "path" or "cookie"
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	// AllowEmptyValue (use is NOT RECOMMENDED, likely to be removed from spec)

	// next are variable fields - used to specify serialization of the parameter

	// Style
	// Explode
	// AllowReserved
	Schema *Schema `json:"schema,omitempty"`
	// Example | Examples
	// Content
}

type Operation struct {
	Tags        []string `json:"tags,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	// ExternalDocs
	OperationID string             `json:"operationId,omitempty"`
	Parameters  []*Parameter       `json:"parameters,omitempty"`
	RequestBody *RequestBody       `json:"requestBody,omitempty"`
	Responses   []*ResponsePattern `json:"responses,omitempty"`
	// Callbacks
	Deprecated bool `json:"deprecated,omitempty"`
	// Security
	Servers []*Server `json:"servers,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required"`
}

type ResponsePattern struct {
	StatusCode string // can be "default" or an HTTP status code
	Response   Response
}

func (p ResponsePattern) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]Response{p.StatusCode: p.Response})
}

type Response struct {
	Description string               `json:"description"`
	Headers     map[string]Parameter `json:"headers,omitempty"` // Same as Parameter, but no Name (comes from map) nor In (implicit).
	Content     map[string]MediaType `json:"content,omitempty"`
	// Links
}

type MediaType struct {
	Schema Schema `json:"schema"`
	// Example | Examples
	// Encoding
}

type Schema struct {
	// This object is an extended subset of the JSON Schema Specification Wright
	// Draft 00 (https://json-schema.org/).
	jsonschema.Schema

	Nullable bool `json:"nullable,omitempty"`
	// Discriminator
	// ReadOnly
	// WriteOnly
	// XML
	// ExternalDocs
	// Example
	Deprecated bool `json:"deprecated,omitempty"`
}
