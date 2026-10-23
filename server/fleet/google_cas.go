package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// GoogleCASCA is what Fleet needs to request certificates from a Google Cloud Certificate Authority Service (CAS) CA pool. The service
// account key is the JSON file Google Cloud issues for a service account with the CA Service Certificate Manager role.
type GoogleCASCA struct {
	ID         uint             `json:"-"`
	Name       string           `json:"name"`
	Project    string           `json:"project"`
	Location   string           `json:"location"`
	CAPool     string           `json:"ca_pool"`
	APIKeyJSON GoogleCASKeyJSON `json:"api_key_json"`
}

// GoogleCASKeyJSON is a Google service account key file. It decodes from either a string or the file's JSON object: GitOps documents
// Google keys as a variable holding the key file, and YAML parses the expanded file as a map.
type GoogleCASKeyJSON string

func (k *GoogleCASKeyJSON) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case bytes.Equal(b, []byte("null")):
		*k = ""
	case len(b) > 0 && b[0] == '"':
		var key string
		if err := json.Unmarshal(b, &key); err != nil {
			return err
		}
		*k = GoogleCASKeyJSON(key)
	case len(b) > 0 && b[0] == '{':
		var compact bytes.Buffer
		if err := json.Compact(&compact, b); err != nil {
			return err
		}
		*k = GoogleCASKeyJSON(compact.String())
	default:
		return errors.New("api_key_json must be the service account key file's JSON, as an object or a string")
	}
	return nil
}

// Equals reports whether two Google CAS CAs have the same settings, including the key, ignoring their IDs.
func (g *GoogleCASCA) Equals(other *GoogleCASCA) bool {
	return g.Name == other.Name && g.Project == other.Project && g.Location == other.Location && g.CAPool == other.CAPool &&
		g.APIKeyJSON.Equals(other.APIKeyJSON)
}

// Equals compares the keys as JSON values. A key pasted in the UI keeps Google's field order and formatting, while GitOps re-encodes it
// with sorted fields, so the text differs for the same key.
func (k GoogleCASKeyJSON) Equals(other GoogleCASKeyJSON) bool {
	if k == other {
		return true
	}
	var a, b any
	if json.Unmarshal([]byte(k), &a) != nil || json.Unmarshal([]byte(other), &b) != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func (g *GoogleCASCA) Preprocess() {
	g.Name = Preprocess(g.Name)
	g.Project = Preprocess(g.Project)
	g.Location = Preprocess(g.Location)
	g.CAPool = Preprocess(g.CAPool)
}

// CAPoolResourceName returns the CA pool in the resource name format the Google CAS API takes.
func (g *GoogleCASCA) CAPoolResourceName() string {
	return fmt.Sprintf("projects/%s/locations/%s/caPools/%s", g.Project, g.Location, g.CAPool)
}

// GoogleCASCAUpdatePayload changes the fields it sets and leaves the others as they are.
type GoogleCASCAUpdatePayload struct {
	Name       *string           `json:"name"`
	Project    *string           `json:"project"`
	Location   *string           `json:"location"`
	CAPool     *string           `json:"ca_pool"`
	APIKeyJSON *GoogleCASKeyJSON `json:"api_key_json"`
}

func (p *GoogleCASCAUpdatePayload) Preprocess() {
	for _, field := range []*string{p.Name, p.Project, p.Location, p.CAPool} {
		if field != nil {
			*field = Preprocess(*field)
		}
	}
}

type GoogleCASService interface {
	// VerifyCAPool checks that the service account key is valid and can read the configured CA pool.
	VerifyCAPool(ctx context.Context, ca GoogleCASCA) error
}
