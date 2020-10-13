package service

import (
	"context"
	"net/http"
	"strings"

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
	switch kolide.HostStatus(status) {
	case kolide.StatusNew, kolide.StatusOnline, kolide.StatusOffline, kolide.StatusMIA:
		hopt.StatusFilter = kolide.HostStatus(status)
	case "":
		// No error when unset
	default:
		return nil, errors.Errorf("invalid status %s", status)

	}
	if err != nil {
		return nil, err
	}

	additionalInfoFiltersString := r.URL.Query().Get("additional_info_filters")
	if additionalInfoFiltersString != "" {
		hopt.AdditionalFilters = strings.Split(additionalInfoFiltersString, ",")
	}
	return listHostsRequest{ListOptions: hopt}, nil
}
