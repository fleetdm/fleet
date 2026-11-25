package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// fleetsAgentOptions edits one fleet's agent options through the API (POST /fleets/{id}/agent_options), the way an
// admin does in the UI: one option alternates between two values. It's the route that writes edited_agent_options
// for a fleet (a GitOps fleet file changes them silently). On a GitOps-owned fleet the next GitOps run puts the
// file's value back, as it would for a customer.
type fleetsAgentOptions struct {
	option string
	values [2]json.RawMessage

	fleets []uint
}

func newFleetsAgentOptions(p *params) (verb, error) {
	v := &fleetsAgentOptions{}
	var err error
	if v.option, err = p.str("option", "distributed_tls_max_attempts"); err != nil {
		return nil, err
	}
	vals, err := p.str("values", "3,4")
	if err != nil {
		return nil, err
	}
	a, b, ok := strings.Cut(vals, ",")
	if !ok || strings.TrimSpace(a) == strings.TrimSpace(b) {
		return nil, fmt.Errorf("with.values %q: want two different JSON values, e.g. 3,4", vals)
	}
	for i, s := range []string{a, b} {
		s = strings.TrimSpace(s)
		if !json.Valid([]byte(s)) {
			return nil, fmt.Errorf("with.values: %q isn't a JSON value", s)
		}
		v.values[i] = json.RawMessage(s)
	}
	return v, nil
}

func (v *fleetsAgentOptions) setup(ctx context.Context, e *env) error {
	var resp struct {
		Fleets []struct {
			ID uint `json:"id"`
		} `json:"fleets"`
	}
	if _, err := e.api.do(ctx, http.MethodGet, "/api/latest/fleet/fleets", nil, nil, &resp); err != nil {
		return err
	}
	for _, f := range resp.Fleets {
		v.fleets = append(v.fleets, f.ID)
	}
	if len(v.fleets) == 0 {
		return errors.New("no fleets")
	}
	return nil
}

func (v *fleetsAgentOptions) run(ctx context.Context, e *env) result {
	id := v.fleets[randIntN(len(v.fleets))]
	detail := map[string]any{"fleet_id": id}
	var resp struct {
		Fleet *struct {
			AgentOptions json.RawMessage `json:"agent_options"`
		} `json:"fleet"`
	}
	if status, err := e.api.do(ctx, http.MethodGet, fmt.Sprintf("/api/latest/fleet/fleets/%d", id), nil, nil, &resp); err != nil {
		return result{status: status, err: err, detail: detail}
	}
	if resp.Fleet == nil || len(resp.Fleet.AgentOptions) == 0 || string(resp.Fleet.AgentOptions) == "null" {
		return result{skip: "fleet has no agent options of its own", detail: detail}
	}
	var ao map[string]json.RawMessage
	if err := json.Unmarshal(resp.Fleet.AgentOptions, &ao); err != nil {
		return result{err: fmt.Errorf("agent options: %w", err), detail: detail}
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(ao["config"], &cfg); err != nil || cfg == nil {
		return result{skip: "agent options have no config object", detail: detail}
	}
	var opts map[string]json.RawMessage
	if err := json.Unmarshal(cfg["options"], &opts); err != nil || opts == nil {
		opts = map[string]json.RawMessage{}
	}
	next := v.values[0]
	if string(opts[v.option]) == string(v.values[0]) {
		next = v.values[1]
	}
	opts[v.option] = next
	var err error
	if cfg["options"], err = json.Marshal(opts); err != nil {
		return result{err: err, detail: detail}
	}
	if ao["config"], err = json.Marshal(cfg); err != nil {
		return result{err: err, detail: detail}
	}
	detail[v.option] = next
	status, err := e.api.do(ctx, http.MethodPost, fmt.Sprintf("/api/latest/fleet/fleets/%d/agent_options", id), nil, ao, nil)
	return result{status: status, err: err, detail: detail}
}
