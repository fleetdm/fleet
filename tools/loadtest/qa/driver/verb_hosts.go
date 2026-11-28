package main

import (
	"context"
	"fmt"
	"net/http"
)

// hostsIDP is an identity-provider integration setting the user on a host (PUT /hosts/{id}/device_mapping with
// source idp, Premium). Emails come from a fixed range, so most calls change the host's user and write an
// edited_host_idp_data activity, as the profile counts them.
type hostsIDP struct {
	users int
}

func newHostsIDP(p *params) (verb, error) {
	n, err := p.integer("users", 5000)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, fmt.Errorf("with.users %d", n)
	}
	return &hostsIDP{users: n}, nil
}

func (v *hostsIDP) setup(context.Context, *env) error { return nil }

func (v *hostsIDP) run(ctx context.Context, e *env) result {
	hs := e.hosts.pick(1, false)
	if len(hs) == 0 {
		return result{skip: "no hosts"}
	}
	h := hs[0]
	body := map[string]string{"email": fmt.Sprintf("idp-user-%05d@example.com", randIntN(v.users)), "source": "idp"}
	status, err := e.api.do(ctx, http.MethodPut, fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", h.ID), nil, body, nil)
	return result{status: status, err: err, detail: map[string]any{"host_id": h.ID}}
}
