package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// labelsMembership adds a host to, or removes one from, a manual label through the host labels API
// (POST/DELETE /hosts/{id}/labels), the way an integration keeps a group in sync. Each call writes an edited_label
// activity. The driver owns its labels (prefix), and keeps each one's size near target.
//
// When GitOps owns global labels, these labels must be listed in default.yml without a hosts key, so GitOps keeps
// them and leaves their membership to this verb.
type labelsMembership struct {
	count  int
	prefix string
	target int
	band   int

	mu     sync.Mutex
	labels []*ownedLabel
}

type ownedLabel struct {
	id      uint
	name    string
	members map[uint]struct{}
}

func newLabelsMembership(p *params) (verb, error) {
	v := &labelsMembership{}
	route, err := p.str("route", "api")
	if err != nil {
		return nil, err
	}
	if route != "api" {
		return nil, fmt.Errorf("with.route %q: only api here; the GitOps route is part of gitops.apply", route)
	}
	if v.count, err = p.integer("labels", 10); err != nil {
		return nil, err
	}
	if v.prefix, err = p.str("prefix", "driver-api-label-"); err != nil {
		return nil, err
	}
	if v.target, err = p.integer("hosts", 200); err != nil {
		return nil, err
	}
	if v.band, err = p.integer("band", 50); err != nil {
		return nil, err
	}
	if v.count < 1 || v.target < 1 || v.band < 0 || v.prefix == "" {
		return nil, errors.New("with: labels and hosts must be >= 1, band >= 0, prefix not empty")
	}
	return v, nil
}

func (v *labelsMembership) setup(ctx context.Context, e *env) error {
	var list struct {
		Labels []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"labels"`
	}
	if _, err := e.api.do(ctx, http.MethodGet, "/api/latest/fleet/labels", nil, nil, &list); err != nil {
		return err
	}
	ids := make(map[string]uint, len(list.Labels))
	for _, l := range list.Labels {
		ids[l.Name] = l.ID
	}
	created := 0
	for i := 1; i <= v.count; i++ {
		name := fmt.Sprintf("%s%02d", v.prefix, i)
		id, ok := ids[name]
		if !ok {
			var cr struct {
				Label struct {
					ID uint `json:"id"`
				} `json:"label"`
			}
			body := map[string]any{
				"name":        name,
				"description": "Manual label the load-test activity driver edits through the host labels API.",
			}
			if _, err := e.api.do(ctx, http.MethodPost, "/api/latest/fleet/labels", nil, body, &cr); err != nil {
				return fmt.Errorf("create label %s: %w", name, err)
			}
			id = cr.Label.ID
			created++
		}
		members, err := labelMembers(ctx, e, id)
		if err != nil {
			return fmt.Errorf("label %s members: %w", name, err)
		}
		v.labels = append(v.labels, &ownedLabel{id: id, name: name, members: members})
	}
	e.logger.InfoContext(ctx, "labels ready", "prefix", v.prefix, "labels", len(v.labels), "created", created)
	return nil
}

func labelMembers(ctx context.Context, e *env, id uint) (map[uint]struct{}, error) {
	members := map[uint]struct{}{}
	for page := 0; ; page++ {
		var resp struct {
			Hosts []struct {
				ID uint `json:"id"`
			} `json:"hosts"`
		}
		q := url.Values{"page": {strconv.Itoa(page)}, "per_page": {"500"}, "disable_issues": {"true"}}
		if _, err := e.api.do(ctx, http.MethodGet, fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", id), q, nil, &resp); err != nil {
			return nil, err
		}
		for _, h := range resp.Hosts {
			members[h.ID] = struct{}{}
		}
		if len(resp.Hosts) < 500 {
			return members, nil
		}
	}
}

func (v *labelsMembership) run(ctx context.Context, e *env) result {
	v.mu.Lock()
	defer v.mu.Unlock()
	l := v.labels[randIntN(len(v.labels))]
	n := len(l.members)
	add := n < v.target-v.band || (n <= v.target+v.band && randIntN(2) == 0)
	detail := map[string]any{"label": l.name, "route": "api"}

	var hostID uint
	if add {
		for _, h := range e.hosts.pick(10, false) {
			if _, in := l.members[h.ID]; !in {
				hostID = h.ID
				break
			}
		}
	} else {
		for id := range l.members { // map order is random enough
			hostID = id
			break
		}
	}
	if hostID == 0 {
		return result{skip: "no host to add or remove", detail: detail}
	}
	method := http.MethodPost
	if !add {
		method = http.MethodDelete
	}
	detail["op"], detail["host_id"] = map[bool]string{true: "add", false: "remove"}[add], hostID
	status, err := e.api.do(ctx, method, fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", hostID), nil,
		map[string][]string{"labels": {l.name}}, nil)
	switch {
	case err == nil && add:
		l.members[hostID] = struct{}{}
	case err == nil, status == http.StatusNotFound: // a deleted host leaves the label too
		delete(l.members, hostID)
	}
	return result{status: status, err: err, detail: detail}
}
