package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/gorilla/mux"
)

// errBadRoute is used for mux errors
var errBadRoute = errors.New("bad route")

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

	if render, ok := response.(renderHijacker); ok {
		render.hijackRender(ctx, w)
		return nil
	}

	if e, ok := response.(statuser); ok {
		w.WriteHeader(e.Status())
		if e.Status() == http.StatusNoContent {
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
	Status() int
}

// loads a html page
type htmlPage interface {
	html() string
	error() error
}

// renderHijacker can be implemented by response values to take control of
// their own rendering.
type renderHijacker interface {
	hijackRender(ctx context.Context, w http.ResponseWriter)
}

func uintFromRequest(r *http.Request, name string) (uint64, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return 0, errBadRoute
	}
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, ctxerr.Wrap(r.Context(), err, "uintFromRequest")
	}
	return u, nil
}

func intFromRequest(r *http.Request, name string) (int64, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return 0, errBadRoute
	}
	u, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ctxerr.Wrap(r.Context(), err, "intFromRequest")
	}
	return u, nil
}

func stringFromRequest(r *http.Request, name string) (string, error) {
	vars := mux.Vars(r)
	s, ok := vars[name]
	if !ok {
		return "", errBadRoute
	}
	unescaped, err := url.PathUnescape(s)
	if err != nil {
		return "", ctxerr.Wrap(r.Context(), err, "unescape value in path")
	}
	return unescaped, nil
}

// default number of items to include per page
const defaultPerPage = 20

// listOptionsFromRequest parses the list options from the request parameters
func listOptionsFromRequest(r *http.Request) (fleet.ListOptions, error) {
	var err error

	pageString := r.URL.Query().Get("page")
	perPageString := r.URL.Query().Get("per_page")
	orderKey := r.URL.Query().Get("order_key")
	orderDirectionString := r.URL.Query().Get("order_direction")
	afterString := r.URL.Query().Get("after")

	var page int
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			return fleet.ListOptions{}, ctxerr.New(r.Context(), "non-int page value")
		}
		if page < 0 {
			return fleet.ListOptions{}, ctxerr.New(r.Context(), "negative page value")
		}
	}

	// We default to 0 for per_page so that not specifying any paging
	// information gets all results
	var perPage int
	if perPageString != "" {
		perPage, err = strconv.Atoi(perPageString)
		if err != nil {
			return fleet.ListOptions{}, ctxerr.New(r.Context(), "non-int per_page value")
		}
		if perPage <= 0 {
			return fleet.ListOptions{}, ctxerr.New(r.Context(), "invalid per_page value")
		}
	}

	if perPage == 0 && pageString != "" {
		// We explicitly set a non-zero default if a page is specified
		// (because the client probably intended for paging, and
		// leaving the 0 would turn that off)
		perPage = defaultPerPage
	}

	if orderKey == "" && orderDirectionString != "" {
		return fleet.ListOptions{},
			ctxerr.New(r.Context(), "order_key must be specified with order_direction")
	}

	var orderDirection fleet.OrderDirection
	switch orderDirectionString {
	case "desc":
		orderDirection = fleet.OrderDescending
	case "asc":
		orderDirection = fleet.OrderAscending
	case "":
		orderDirection = fleet.OrderAscending
	default:
		return fleet.ListOptions{},
			ctxerr.New(r.Context(), "unknown order_direction: "+orderDirectionString)

	}

	query := r.URL.Query().Get("query")

	return fleet.ListOptions{
		Page:           uint(page),
		PerPage:        uint(perPage),
		OrderKey:       orderKey,
		OrderDirection: orderDirection,
		MatchQuery:     query,
		After:          afterString,
	}, nil
}

func hostListOptionsFromRequest(r *http.Request) (fleet.HostListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return fleet.HostListOptions{}, err
	}

	hopt := fleet.HostListOptions{ListOptions: opt}

	status := r.URL.Query().Get("status")
	switch fleet.HostStatus(status) {
	case fleet.StatusNew, fleet.StatusOnline, fleet.StatusOffline, fleet.StatusMIA:
		hopt.StatusFilter = fleet.HostStatus(status)
	case "":
		// No error when unset
	default:
		return hopt, ctxerr.Errorf(r.Context(), "invalid status %s", status)

	}
	if err != nil {
		return hopt, err
	}

	additionalInfoFiltersString := r.URL.Query().Get("additional_info_filters")
	if additionalInfoFiltersString != "" {
		hopt.AdditionalFilters = strings.Split(additionalInfoFiltersString, ",")
	}

	teamID := r.URL.Query().Get("team_id")
	if teamID != "" {
		id, err := strconv.Atoi(teamID)
		if err != nil {
			return hopt, err
		}
		tid := uint(id)
		hopt.TeamFilter = &tid
	}

	policyID := r.URL.Query().Get("policy_id")
	if policyID != "" {
		id, err := strconv.Atoi(policyID)
		if err != nil {
			return hopt, err
		}
		pid := uint(id)
		hopt.PolicyIDFilter = &pid
	}

	policyResponse := r.URL.Query().Get("policy_response")
	if policyResponse != "" {
		var v *bool
		switch policyResponse {
		case "passing":
			v = ptr.Bool(true)
		case "failing":
			v = ptr.Bool(false)
		}
		hopt.PolicyResponseFilter = v
	}

	softwareID := r.URL.Query().Get("software_id")
	if softwareID != "" {
		id, err := strconv.Atoi(softwareID)
		if err != nil {
			return hopt, err
		}
		sid := uint(id)
		hopt.SoftwareIDFilter = &sid
	}

	disableFailingPolicies := r.URL.Query().Get("disable_failing_policies")
	if disableFailingPolicies != "" {
		boolVal, err := strconv.ParseBool(disableFailingPolicies)
		if err != nil {
			return hopt, err
		}
		hopt.DisableFailingPolicies = boolVal
	}

	return hopt, nil
}

func carveListOptionsFromRequest(r *http.Request) (fleet.CarveListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return fleet.CarveListOptions{}, err
	}

	copt := fleet.CarveListOptions{ListOptions: opt}

	expired := r.URL.Query().Get("expired")
	// TODO(mna): allow the same bool encodings as strconv.ParseBool and use it?
	switch expired {
	case "1", "true":
		copt.Expired = true
	case "0", "":
		copt.Expired = false
	default:
		return copt, ctxerr.Errorf(r.Context(), "invalid expired value %s", expired)
	}
	return copt, nil
}

func userListOptionsFromRequest(r *http.Request) (fleet.UserListOptions, error) {
	opt, err := listOptionsFromRequest(r)
	if err != nil {
		return fleet.UserListOptions{}, err
	}

	uopt := fleet.UserListOptions{ListOptions: opt}

	if tid := r.URL.Query().Get("team_id"); tid != "" {
		teamID, err := strconv.ParseUint(tid, 10, 64)
		if err != nil {
			return uopt, ctxerr.Wrap(r.Context(), err, "parse team_id as int")
		}
		uopt.TeamID = uint(teamID)
	}

	return uopt, nil
}

type getGenericSpecRequest struct {
	Name string `url:"name"`
}
