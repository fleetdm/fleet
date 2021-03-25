package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

var (
	// errBadRoute is used for mux errors
	errBadRoute = errors.New("bad route")
)

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	// The has to happen first, if an error happens we'll redirect to an error
	// page and the error will be logged
	if page, ok := response.(htmlPage); ok {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		_, err := io.WriteString(w, page.html())
		return err
	}

	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}

	if e, ok := response.(statuser); ok {
		w.WriteHeader(e.status())
		if e.status() == http.StatusNoContent {
			return nil
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(response)
}

// statuser allows response types to implement a custom
// http success status - default is 200 OK
type statuser interface {
	status() int
}

// loads a html page
type htmlPage interface {
	html() string
	error() error
}

func idFromRequest(r *http.Request, name string) (uint, error) {
	vars := mux.Vars(r)
	id, ok := vars[name]
	if !ok {
		return 0, errBadRoute
	}
	uid, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}
	return uint(uid), nil
}

func nameFromRequest(r *http.Request, varName string) (string, error) {
	vars := mux.Vars(r)
	name, ok := vars[varName]
	if !ok {
		return "", errBadRoute
	}
	unescaped, err := url.PathUnescape(name)
	if err != nil {
		return "", errors.Wrap(err, "unescape name in path")
	}
	return unescaped, nil
}

// default number of items to include per page
const defaultPerPage = 20

// listOptionsFromRequest parses the list options from the request parameters
func listOptionsFromRequest(r *http.Request) (kolide.ListOptions, error) {
	var err error

	pageString := r.URL.Query().Get("page")
	perPageString := r.URL.Query().Get("per_page")
	orderKey := r.URL.Query().Get("order_key")
	orderDirectionString := r.URL.Query().Get("order_direction")

	var page int = 0
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			return kolide.ListOptions{}, errors.New("non-int page value")
		}
		if page < 0 {
			return kolide.ListOptions{}, errors.New("negative page value")
		}
	}

	// We default to 0 for per_page so that not specifying any paging
	// information gets all results
	var perPage int = 0
	if perPageString != "" {
		perPage, err = strconv.Atoi(perPageString)
		if err != nil {
			return kolide.ListOptions{}, errors.New("non-int per_page value")
		}
		if perPage <= 0 {
			return kolide.ListOptions{}, errors.New("invalid per_page value")
		}
	}

	if perPage == 0 && pageString != "" {
		// We explicitly set a non-zero default if a page is specified
		// (because the client probably intended for paging, and
		// leaving the 0 would turn that off)
		perPage = defaultPerPage
	}

	if orderKey == "" && orderDirectionString != "" {
		return kolide.ListOptions{},
			errors.New("order_key must be specified with order_direction")
	}

	var orderDirection kolide.OrderDirection
	switch orderDirectionString {
	case "desc":
		orderDirection = kolide.OrderDescending
	case "asc":
		orderDirection = kolide.OrderAscending
	case "":
		orderDirection = kolide.OrderAscending
	default:
		return kolide.ListOptions{},
			errors.New("unknown order_direction: " + orderDirectionString)

	}

	// Special some keys so that the frontend can use consistent names.
	// TODO #317 remove special cases
	switch orderKey {
	case "hostname":
		orderKey = "host_name"
	case "memory":
		orderKey = "physical_memory"
	case "detail_updated_at":
		orderKey = "detail_update_time"
	}

	return kolide.ListOptions{
		Page:           uint(page),
		PerPage:        uint(perPage),
		OrderKey:       orderKey,
		OrderDirection: orderDirection,
	}, nil
}

func hostListOptionsFromRequest(r *http.Request) (kolide.HostListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return kolide.HostListOptions{}, err
	}

	hopt := kolide.HostListOptions{ListOptions: opt}

	status := r.URL.Query().Get("status")
	switch kolide.HostStatus(status) {
	case kolide.StatusNew, kolide.StatusOnline, kolide.StatusOffline, kolide.StatusMIA:
		hopt.StatusFilter = kolide.HostStatus(status)
	case "":
		// No error when unset
	default:
		return hopt, errors.Errorf("invalid status %s", status)

	}
	if err != nil {
		return hopt, err
	}

	additionalInfoFiltersString := r.URL.Query().Get("additional_info_filters")
	if additionalInfoFiltersString != "" {
		hopt.AdditionalFilters = strings.Split(additionalInfoFiltersString, ",")
	}

	query := r.URL.Query().Get("query")
	hopt.MatchQuery = query

	return hopt, nil
}

func decodeNoParamsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

type getGenericSpecRequest struct {
	Name string
}

func decodeGetGenericSpecRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	name, err := nameFromRequest(r, "name")
	if err != nil {
		return nil, err
	}
	var req getGenericSpecRequest
	req.Name = name
	return req, nil
}
