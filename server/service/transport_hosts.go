package service

import (
	"context"
	"net/http"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func decodeGetHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return getHostRequest{ID: id}, nil
}

func decodeHostByIdentifierRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	identifier, err := nameFromRequest(r, "identifier")
	if err != nil {
		return nil, err
	}
	return hostByIdentifierRequest{Identifier: identifier}, nil
}

func decodeDeleteHostRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id, err := idFromRequest(r, "id")
	if err != nil {
		return nil, err
	}
	return deleteHostRequest{ID: id}, nil
}

func decodeListHostsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	opt, err := listOptionsFromRequest(r)
	hopt := kolide.HostListOptions{ListOptions: opt}
	status := r.URL.Query().Get("status")
	switch status {
	case "new", "online", "offline", "mia":
		hopt.StatusFilter = kolide.HostStatus(status)
	case "":
		// No error when unset
	default:
		return nil, errors.Errorf("invalid status %s", status)

	}
	if err != nil {
		return nil, err
	}
	return listHostsRequest{ListOptions: hopt}, nil
}
