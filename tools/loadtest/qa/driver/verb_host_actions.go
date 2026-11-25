package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"
)

// lookups times the reads a verb makes to find what to act on (a host's software, its profiles), so the event can
// report them apart from the action's own request.
type lookups struct {
	n    int
	took time.Duration
	max  time.Duration
}

func (l *lookups) do(ctx context.Context, e *env, method, path string, q url.Values, out any) (int, error) {
	start := time.Now()
	status, err := e.api.do(ctx, method, path, q, nil, out)
	d := time.Since(start)
	l.n++
	l.took += d
	l.max = max(l.max, d)
	return status, err
}

func (l *lookups) add(detail map[string]any) map[string]any {
	if l.n > 0 {
		detail["lookups"], detail["lookup_ms"], detail["lookup_max_ms"] = l.n, l.took.Milliseconds(), l.max.Milliseconds()
	}
	return detail
}

// result is the action's own request; took is its time alone.
func (l *lookups) result(status int, err error, took time.Duration, detail map[string]any) result {
	return result{status: status, err: err, took: took, detail: l.add(detail)}
}

// failed is a lookup that failed: the action never ran, and the lookup's status is what broke.
func (l *lookups) failed(status int, err error, detail map[string]any) result {
	return result{status: status, err: fmt.Errorf("lookup: %w", err), detail: l.add(detail)}
}

func (l *lookups) skipped(why string) result {
	return result{skip: why, detail: l.add(map[string]any{})}
}

// softwareUninstall uninstalls a Fleet-managed package from an online host where it's installed
// (POST /hosts/{id}/software/{title_id}/uninstall). The host's uninstall result writes uninstalled_software; an
// install automation may put it back later, as it would on a real host.
type softwareUninstall struct {
	tries int
}

func newSoftwareUninstall(p *params) (verb, error) {
	n, err := p.integer("tries", 10)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, fmt.Errorf("with.tries %d", n)
	}
	return &softwareUninstall{tries: n}, nil
}

func (v *softwareUninstall) setup(context.Context, *env) error { return nil }

func (v *softwareUninstall) run(ctx context.Context, e *env) result {
	var look lookups
	for _, h := range e.hosts.pick(v.tries, true) {
		var resp struct {
			Software []struct {
				ID              uint   `json:"id"`
				Status          string `json:"status"`
				SoftwarePackage *struct {
					Name string `json:"name"`
				} `json:"software_package"`
			} `json:"software"`
		}
		q := url.Values{"available_for_install": {"true"}, "per_page": {"100"}}
		status, err := look.do(ctx, e, http.MethodGet, fmt.Sprintf("/api/latest/fleet/hosts/%d/software", h.ID), q, &resp)
		if err != nil {
			return look.failed(status, err, map[string]any{"host_id": h.ID})
		}
		var installed []uint
		for _, s := range resp.Software {
			if s.Status == "installed" && s.SoftwarePackage != nil {
				installed = append(installed, s.ID)
			}
		}
		if len(installed) == 0 {
			continue
		}
		title := installed[randIntN(len(installed))]
		start := time.Now()
		status, err = e.api.do(ctx, http.MethodPost, fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", h.ID, title), nil, nil, nil)
		return look.result(status, err, time.Since(start), map[string]any{"host_id": h.ID, "title_id": title})
	}
	return look.skipped(fmt.Sprintf("no installed package on %d online hosts", v.tries))
}

// profilesResend resends one configuration profile to an online host
// (POST /hosts/{id}/configuration_profiles/{uuid}/resend), the way an admin retries a profile from the host page.
type profilesResend struct {
	platforms []string
	tries     int
}

func newProfilesResend(p *params) (verb, error) {
	plat, err := p.str("platform", "darwin")
	if err != nil {
		return nil, err
	}
	n, err := p.integer("tries", 10)
	if err != nil {
		return nil, err
	}
	if n < 1 {
		return nil, fmt.Errorf("with.tries %d", n)
	}
	return &profilesResend{platforms: []string{plat}, tries: n}, nil
}

func (v *profilesResend) setup(context.Context, *env) error { return nil }

func (v *profilesResend) run(ctx context.Context, e *env) result {
	var look lookups
	for _, h := range e.hosts.pick(4*v.tries, true) {
		if !slices.Contains(v.platforms, h.Platform) {
			continue
		}
		var resp struct {
			Host struct {
				MDM struct {
					Profiles []struct {
						ProfileUUID string `json:"profile_uuid"`
						Status      string `json:"status"`
					} `json:"profiles"`
				} `json:"mdm"`
			} `json:"host"`
		}
		status, err := look.do(ctx, e, http.MethodGet, fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), url.Values{"exclude_software": {"true"}}, &resp)
		if err != nil {
			return look.failed(status, err, map[string]any{"host_id": h.ID})
		}
		var uuids []string
		for _, p := range resp.Host.MDM.Profiles {
			if p.ProfileUUID != "" && p.Status != "pending" { // a pending profile can't be resent
				uuids = append(uuids, p.ProfileUUID)
			}
		}
		if len(uuids) == 0 {
			continue
		}
		u := uuids[randIntN(len(uuids))]
		start := time.Now()
		status, err = e.api.do(ctx, http.MethodPost, fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", h.ID, url.PathEscape(u)), nil, nil, nil)
		return look.result(status, err, time.Since(start), map[string]any{"host_id": h.ID, "profile_uuid": u})
	}
	return look.skipped("no online host with a delivered profile")
}

// hostsTransfer moves a few hosts from one fleet to another (POST /hosts/transfer), so they pick up the other
// fleet's profiles, software and policies. Only fleets listed in `fleets` take part, so the moves stay where the
// population can absorb them.
type hostsTransfer struct {
	names    []string
	min, max int
	fleetIDs map[uint]string
}

func newHostsTransfer(p *params) (verb, error) {
	v := &hostsTransfer{}
	var err error
	if v.min, v.max, err = p.intRange("hosts", 1, 20); err != nil {
		return nil, err
	}
	raw, ok := p.get("fleets")
	if !ok {
		return nil, errors.New("with.fleets: list the fleets hosts may move between")
	}
	list, ok := raw.([]any)
	if !ok || len(list) < 2 {
		return nil, errors.New("with.fleets: want a list of at least two fleet names")
	}
	for _, n := range list {
		s, ok := n.(string)
		if !ok {
			return nil, fmt.Errorf("with.fleets: %v isn't a name", n)
		}
		v.names = append(v.names, s)
	}
	return v, nil
}

func (v *hostsTransfer) setup(ctx context.Context, e *env) error {
	var resp struct {
		Fleets []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"fleets"`
	}
	if _, err := e.api.do(ctx, http.MethodGet, "/api/latest/fleet/fleets", nil, nil, &resp); err != nil {
		return err
	}
	v.fleetIDs = map[uint]string{}
	for _, f := range resp.Fleets {
		if slices.Contains(v.names, f.Name) {
			v.fleetIDs[f.ID] = f.Name
		}
	}
	if len(v.fleetIDs) != len(v.names) {
		return fmt.Errorf("found %d of the fleets %v", len(v.fleetIDs), v.names)
	}
	return nil
}

func (v *hostsTransfer) run(ctx context.Context, e *env) result {
	ids := make([]uint, 0, len(v.fleetIDs))
	for id := range v.fleetIDs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	from, to := ids[randIntN(len(ids))], ids[randIntN(len(ids))]
	for to == from {
		to = ids[randIntN(len(ids))]
	}
	n := v.min + randIntN(v.max-v.min+1)
	var hosts []uint
	for _, h := range e.hosts.pick(50*n, false) {
		if h.FleetID != nil && *h.FleetID == from {
			hosts = append(hosts, h.ID)
			if len(hosts) == n {
				break
			}
		}
	}
	detail := map[string]any{"from": v.fleetIDs[from], "to": v.fleetIDs[to], "hosts": len(hosts)}
	if len(hosts) == 0 {
		return result{skip: "no hosts in " + v.fleetIDs[from], detail: detail}
	}
	status, err := e.api.do(ctx, http.MethodPost, "/api/latest/fleet/hosts/transfer", nil,
		map[string]any{"fleet_id": to, "hosts": hosts}, nil)
	if err == nil {
		e.hosts.moved(hosts, to)
	}
	return result{status: status, err: err, detail: detail}
}
