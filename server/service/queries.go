package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Get Query
////////////////////////////////////////////////////////////////////////////////

type getQueryRequest struct {
	ID uint `url:"id"`
}

type getQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r getQueryResponse) error() error { return r.Err }

func getQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getQueryRequest)
	query, err := svc.GetQuery(ctx, req.ID)
	if err != nil {
		return getQueryResponse{Err: err}, nil
	}
	return getQueryResponse{query, nil}, nil
}

func (svc *Service) GetQuery(ctx context.Context, id uint) (*fleet.Query, error) {
	// Load query first to get its teamID.
	query, err := svc.ds.Query(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get query from datastore")
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionRead); err != nil {
		return nil, err
	}
	return query, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Queries
////////////////////////////////////////////////////////////////////////////////

type listQueriesRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
	// TeamID url argument set to 0 means global.
	TeamID         uint `query:"team_id,optional"`
	MergeInherited bool `query:"merge_inherited,optional"`
	// only return queries targeted to run on this platform
	Platform string `query:"platform,optional"`
}

type listQueriesResponse struct {
	Queries []fleet.Query             `json:"queries"`
	Count   int                       `json:"count"`
	Meta    *fleet.PaginationMetadata `json:"meta"`
	Err     error                     `json:"error,omitempty"`
}

func (r listQueriesResponse) error() error { return r.Err }

func listQueriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listQueriesRequest)

	var teamID *uint
	if req.TeamID != 0 {
		teamID = &req.TeamID
	}

	var urlPlatform *string
	if req.Platform != "" {
		urlPlatform = &req.Platform
	}

	queries, count, meta, err := svc.ListQueries(ctx, req.ListOptions, teamID, nil, req.MergeInherited, urlPlatform)
	if err != nil {
		return listQueriesResponse{Err: err}, nil
	}

	respQueries := make([]fleet.Query, 0, len(queries))
	for _, query := range queries {
		respQueries = append(respQueries, *query)
	}

	return listQueriesResponse{
		Queries: respQueries,
		Count:   count,
		Meta:    meta,
	}, nil
}

func (svc *Service) ListQueries(ctx context.Context, opt fleet.ListOptions, teamID *uint, scheduled *bool, mergeInherited bool, urlPlatform *string) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
	// Check the user is allowed to list queries on the given team.
	if err := svc.authz.Authorize(ctx, &fleet.Query{
		TeamID: teamID,
	}, fleet.ActionRead); err != nil {
		return nil, 0, nil, err
	}

	// always include metadata for queries
	opt.IncludeMetadata = true

	var dbPlatform *string
	if urlPlatform != nil {
		// validate platform filter
		if *urlPlatform == "macos" {
			// More user-friendly API param "macos" is called "darwin" in the datastore
			dbPlatform = ptr.String("darwin")
		} else {
			dbPlatform = urlPlatform
		}
		if strings.Contains(*urlPlatform, ",") {
			return nil, 0, nil, &fleet.BadRequestError{Message: "queries can only be filtered by one platform at a time"}
		}
		targetableDBPlatforms := []string{"darwin", "windows", "linux"}
		if !slices.Contains(targetableDBPlatforms, *dbPlatform) {
			return nil, 0, nil, &fleet.BadRequestError{Message: "provided platform param cannot be a scheduled query target"}
		}
	}

	queries, count, meta, err := svc.ds.ListQueries(ctx, fleet.ListQueryOptions{
		ListOptions:    opt,
		TeamID:         teamID,
		IsScheduled:    scheduled,
		MergeInherited: mergeInherited,
		Platform:       dbPlatform,
	})
	if err != nil {
		return nil, 0, nil, err
	}

	return queries, count, meta, nil
}

////////////////////////////////////////////////////////////////////////////////
// Query Reports
////////////////////////////////////////////////////////////////////////////////

type getQueryReportRequest struct {
	ID uint `url:"id"`
}

type getQueryReportResponse struct {
	QueryID       uint                       `json:"query_id"`
	Results       []fleet.HostQueryResultRow `json:"results"`
	ReportClipped bool                       `json:"report_clipped"`
	Err           error                      `json:"error,omitempty"`
}

func (r getQueryReportResponse) error() error { return r.Err }

func getQueryReportEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getQueryReportRequest)
	queryReportResults, reportClipped, err := svc.GetQueryReportResults(ctx, req.ID)
	if err != nil {
		return listQueriesResponse{Err: err}, nil
	}
	// Return an empty array if there are no results stored.
	results := []fleet.HostQueryResultRow{}
	if len(queryReportResults) > 0 {
		results = queryReportResults
	}
	return getQueryReportResponse{
		QueryID:       req.ID,
		Results:       results,
		ReportClipped: reportClipped,
	}, nil
}

func (svc *Service) GetQueryReportResults(ctx context.Context, id uint) ([]fleet.HostQueryResultRow, bool, error) {
	// Load query first to get its teamID.
	query, err := svc.ds.Query(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, false, ctxerr.Wrap(ctx, err, "get query from datastore")
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionRead); err != nil {
		return nil, false, err
	}

	if query.DiscardData {
		return nil, false, nil
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, false, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	queryReportResultRows, err := svc.ds.QueryResultRows(ctx, id, filter)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "get query report results")
	}
	queryReportResults, err := fleet.MapQueryReportResultsToRows(queryReportResultRows)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "map db rows to results")
	}
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "get app config")
	}
	reportClipped, err := svc.QueryReportIsClipped(ctx, id, appConfig.ServerSettings.GetQueryReportCap())
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "check query report is clipped")
	}
	return queryReportResults, reportClipped, nil
}

func (svc *Service) QueryReportIsClipped(ctx context.Context, queryID uint, maxQueryReportRows int) (bool, error) {
	query, err := svc.ds.Query(ctx, queryID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return false, ctxerr.Wrap(ctx, err, "get query from datastore")
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionRead); err != nil {
		return false, err
	}

	count, err := svc.ds.ResultCountForQuery(ctx, queryID)
	if err != nil {
		return false, err
	}
	return count >= maxQueryReportRows, nil
}

////////////////////////////////////////////////////////////////////////////////
// Create Query
////////////////////////////////////////////////////////////////////////////////

type createQueryRequest struct {
	fleet.QueryPayload
}

type createQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r createQueryResponse) error() error { return r.Err }

func createQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createQueryRequest)
	query, err := svc.NewQuery(ctx, req.QueryPayload)
	if err != nil {
		return createQueryResponse{Err: err}, nil
	}
	return createQueryResponse{query, nil}, nil
}

func (svc *Service) NewQuery(ctx context.Context, p fleet.QueryPayload) (*fleet.Query, error) {
	// Check the user is allowed to create a new query on the team.
	if err := svc.authz.Authorize(ctx, fleet.Query{
		TeamID: p.TeamID,
	}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if p.Logging == nil || (p.Logging != nil && *p.Logging == "") {
		p.Logging = ptr.String(fleet.LoggingSnapshot)
	}

	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("query payload verification: %s", err),
		})
	}

	query := &fleet.Query{
		Saved: true,

		TeamID: p.TeamID,
	}

	if p.Name != nil {
		query.Name = *p.Name
	}
	if p.Description != nil {
		query.Description = *p.Description
	}
	if p.Query != nil {
		query.Query = *p.Query
	}
	if p.Interval != nil {
		query.Interval = *p.Interval
	}
	if p.Platform != nil {
		query.Platform = *p.Platform
	}
	if p.MinOsqueryVersion != nil {
		query.MinOsqueryVersion = *p.MinOsqueryVersion
	}
	if p.AutomationsEnabled != nil {
		query.AutomationsEnabled = *p.AutomationsEnabled
	}
	if p.Logging != nil {
		query.Logging = *p.Logging
	}
	if p.ObserverCanRun != nil {
		query.ObserverCanRun = *p.ObserverCanRun
	}
	if p.DiscardData != nil {
		query.DiscardData = *p.DiscardData
	}

	logging.WithExtras(ctx, "name", query.Name, "sql", query.Query)

	vc, ok := viewer.FromContext(ctx)
	if ok {
		query.AuthorID = ptr.Uint(vc.UserID())
		query.AuthorName = vc.FullName()
		query.AuthorEmail = vc.Email()
	}

	query, err := svc.ds.NewQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeCreatedSavedQuery{
			ID:   query.ID,
			Name: query.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for query creation")
	}

	return query, nil
}

////////////////////////////////////////////////////////////////////////////////
// Modify Query
////////////////////////////////////////////////////////////////////////////////

type modifyQueryRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.QueryPayload
}

type modifyQueryResponse struct {
	Query *fleet.Query `json:"query,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r modifyQueryResponse) error() error { return r.Err }

func modifyQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyQueryRequest)
	query, err := svc.ModifyQuery(ctx, req.ID, req.QueryPayload)
	if err != nil {
		return modifyQueryResponse{Err: err}, nil
	}
	return modifyQueryResponse{query, nil}, nil
}

func (svc *Service) ModifyQuery(ctx context.Context, id uint, p fleet.QueryPayload) (*fleet.Query, error) {
	// Load query first to determine if the user can modify it.
	query, err := svc.ds.Query(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, err
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if p.Logging != nil && *p.Logging == "" {
		p.Logging = ptr.String(fleet.LoggingSnapshot)
	}

	if err := p.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf("query payload verification: %s", err),
		})
	}

	shouldDiscardQueryResults, shouldDeleteStats := false, false

	if p.Name != nil {
		query.Name = *p.Name
	}
	if p.Description != nil {
		query.Description = *p.Description
	}
	if p.Query != nil {
		if query.Query != *p.Query {
			shouldDiscardQueryResults = true
			shouldDeleteStats = true
		}
		query.Query = *p.Query
	}
	if p.Interval != nil {
		query.Interval = *p.Interval
	}
	if p.Platform != nil {
		if !comparePlatforms(query.Platform, *p.Platform) {
			shouldDiscardQueryResults = true
		}
		query.Platform = *p.Platform
	}
	if p.MinOsqueryVersion != nil {
		if query.MinOsqueryVersion != *p.MinOsqueryVersion {
			shouldDiscardQueryResults = true
		}
		query.MinOsqueryVersion = *p.MinOsqueryVersion
	}
	if p.AutomationsEnabled != nil {
		query.AutomationsEnabled = *p.AutomationsEnabled
	}
	if p.Logging != nil {
		if query.Logging != *p.Logging && *p.Logging != fleet.LoggingSnapshot {
			shouldDiscardQueryResults = true
		}
		query.Logging = *p.Logging
	}
	if p.ObserverCanRun != nil {
		query.ObserverCanRun = *p.ObserverCanRun
	}
	if p.DiscardData != nil {
		if *p.DiscardData && *p.DiscardData != query.DiscardData {
			shouldDiscardQueryResults = true
		}
		query.DiscardData = *p.DiscardData
	}

	logging.WithExtras(ctx, "name", query.Name, "sql", query.Query)

	if err := svc.ds.SaveQuery(ctx, query, shouldDiscardQueryResults, shouldDeleteStats); err != nil {
		return nil, err
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeEditedSavedQuery{
			ID:   query.ID,
			Name: query.Name,
		},
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for query modification")
	}

	return query, nil
}

func comparePlatforms(platform1, platform2 string) bool {
	if platform1 == platform2 {
		return true
	}
	p1s := strings.Split(platform1, ",")
	slices.Sort(p1s)
	p2s := strings.Split(platform2, ",")
	slices.Sort(p2s)
	return slices.Compare(p1s, p2s) == 0
}

////////////////////////////////////////////////////////////////////////////////
// Delete Query
////////////////////////////////////////////////////////////////////////////////

type deleteQueryRequest struct {
	Name string `url:"name"`
	// TeamID if not set is assumed to be 0 (global).
	TeamID uint `url:"team_id,optional"`
}

type deleteQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryResponse) error() error { return r.Err }

func deleteQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteQueryRequest)
	var teamID *uint
	if req.TeamID != 0 {
		teamID = &req.TeamID
	}
	err := svc.DeleteQuery(ctx, teamID, req.Name)
	if err != nil {
		return deleteQueryResponse{Err: err}, nil
	}
	return deleteQueryResponse{}, nil
}

func (svc *Service) DeleteQuery(ctx context.Context, teamID *uint, name string) error {
	// Load query first to determine if the user can delete it.
	query, err := svc.ds.QueryByName(ctx, teamID, name)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return err
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteQuery(ctx, teamID, name); err != nil {
		return err
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedSavedQuery{
			Name: name,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for query deletion")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Query By ID
////////////////////////////////////////////////////////////////////////////////

type deleteQueryByIDRequest struct {
	ID uint `url:"id"`
}

type deleteQueryByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteQueryByIDResponse) error() error { return r.Err }

func deleteQueryByIDEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteQueryByIDRequest)
	err := svc.DeleteQueryByID(ctx, req.ID)
	if err != nil {
		return deleteQueryByIDResponse{Err: err}, nil
	}
	return deleteQueryByIDResponse{}, nil
}

func (svc *Service) DeleteQueryByID(ctx context.Context, id uint) error {
	// Load query first to determine if the user can delete it.
	query, err := svc.ds.Query(ctx, id)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return ctxerr.Wrap(ctx, err, "lookup query by ID")
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteQuery(ctx, query.TeamID, query.Name); err != nil {
		return ctxerr.Wrap(ctx, err, "delete query")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedSavedQuery{
			Name: query.Name,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for query deletion by id")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Queries
////////////////////////////////////////////////////////////////////////////////

type deleteQueriesRequest struct {
	IDs []uint `json:"ids"`
}

type deleteQueriesResponse struct {
	Deleted uint  `json:"deleted"`
	Err     error `json:"error,omitempty"`
}

func (r deleteQueriesResponse) error() error { return r.Err }

func deleteQueriesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteQueriesRequest)
	deleted, err := svc.DeleteQueries(ctx, req.IDs)
	if err != nil {
		return deleteQueriesResponse{Err: err}, nil
	}
	return deleteQueriesResponse{Deleted: deleted}, nil
}

func (svc *Service) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	// Verify that the user is allowed to delete all the requested queries.
	for _, id := range ids {
		query, err := svc.ds.Query(ctx, id)
		if err != nil {
			setAuthCheckedOnPreAuthErr(ctx)
			return 0, ctxerr.Wrap(ctx, err, "lookup query by ID")
		}
		if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
			return 0, err
		}
	}

	n, err := svc.ds.DeleteQueries(ctx, ids)
	if err != nil {
		return n, err
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeDeletedMultipleSavedQuery{
			IDs: ids,
		},
	); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "create activity for query deletions")
	}
	return n, nil
}

////////////////////////////////////////////////////////////////////////////////
// Apply Query Specs
////////////////////////////////////////////////////////////////////////////////

type applyQuerySpecsRequest struct {
	Specs []*fleet.QuerySpec `json:"specs"`
}

type applyQuerySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyQuerySpecsResponse) error() error { return r.Err }

func applyQuerySpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*applyQuerySpecsRequest)
	err := svc.ApplyQuerySpecs(ctx, req.Specs)
	if err != nil {
		return applyQuerySpecsResponse{Err: err}, nil
	}
	return applyQuerySpecsResponse{}, nil
}

func (svc *Service) ApplyQuerySpecs(ctx context.Context, specs []*fleet.QuerySpec) error {
	// 1. Turn specs into queries.
	queries := []*fleet.Query{}
	for _, spec := range specs {
		query, err := svc.queryFromSpec(ctx, spec)
		if err != nil {
			setAuthCheckedOnPreAuthErr(ctx)
			return ctxerr.Wrap(ctx, err, "creating query from spec")
		}
		queries = append(queries, query)
	}
	// 2. Run authorization checks and verify their fields.
	for _, query := range queries {
		if err := svc.authz.Authorize(ctx, query, fleet.ActionWrite); err != nil {
			return err
		}
		if err := query.Verify(); err != nil {
			return ctxerr.Wrap(ctx, &fleet.BadRequestError{
				Message: fmt.Sprintf("query payload verification: %s", err),
			})
		}
	}
	// 3. Apply the queries.

	// first, find out if we should delete query results
	queriesToDiscardResults := make(map[uint]struct{})
	for _, query := range queries {
		dbQuery, err := svc.ds.QueryByName(ctx, query.TeamID, query.Name)
		if err != nil && !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "fetching saved query")
		}

		if dbQuery == nil {
			// then we're creating a new query, so move on.
			continue
		}

		if (query.DiscardData && query.DiscardData != dbQuery.DiscardData) ||
			(query.Logging != dbQuery.Logging && query.Logging != fleet.LoggingSnapshot) ||
			query.Query != dbQuery.Query ||
			query.MinOsqueryVersion != dbQuery.MinOsqueryVersion ||
			!comparePlatforms(query.Platform, dbQuery.Platform) {
			queriesToDiscardResults[dbQuery.ID] = struct{}{}
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return ctxerr.New(ctx, "user must be authenticated to apply queries")
	}
	err := svc.ds.ApplyQueries(ctx, vc.UserID(), queries, queriesToDiscardResults)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "applying queries")
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeAppliedSpecSavedQuery{
			Specs: specs,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for query spec")
	}
	return nil
}

func (svc *Service) queryFromSpec(ctx context.Context, spec *fleet.QuerySpec) (*fleet.Query, error) {
	var teamID *uint
	if spec.TeamName != "" {
		team, err := svc.ds.TeamByName(ctx, spec.TeamName)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team by name")
		}
		teamID = &team.ID
	}
	logging := spec.Logging
	if logging == "" {
		logging = fleet.LoggingSnapshot
	}
	return &fleet.Query{
		Name:        spec.Name,
		Description: spec.Description,
		Query:       spec.Query,

		TeamID:             teamID,
		Interval:           spec.Interval,
		ObserverCanRun:     spec.ObserverCanRun,
		Platform:           spec.Platform,
		MinOsqueryVersion:  spec.MinOsqueryVersion,
		AutomationsEnabled: spec.AutomationsEnabled,
		Logging:            logging,
		DiscardData:        spec.DiscardData,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Specs
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecsResponse struct {
	Specs []*fleet.QuerySpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

type getQuerySpecsRequest struct {
	TeamID uint `url:"team_id,optional"`
}

func (r getQuerySpecsResponse) error() error { return r.Err }

func getQuerySpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getQuerySpecsRequest)
	var teamID *uint
	if req.TeamID != 0 {
		teamID = &req.TeamID
	}
	specs, err := svc.GetQuerySpecs(ctx, teamID)
	if err != nil {
		return getQuerySpecsResponse{Err: err}, nil
	}
	return getQuerySpecsResponse{Specs: specs}, nil
}

func (svc *Service) GetQuerySpecs(ctx context.Context, teamID *uint) ([]*fleet.QuerySpec, error) {
	queries, _, _, err := svc.ListQueries(ctx, fleet.ListOptions{}, teamID, nil, false, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting queries")
	}

	// Turn queries into specs.
	var specs []*fleet.QuerySpec
	for _, query := range queries {
		spec, err := svc.specFromQuery(ctx, query)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create spec from query")
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func (svc *Service) specFromQuery(ctx context.Context, query *fleet.Query) (*fleet.QuerySpec, error) {
	var teamName string
	if query.TeamID != nil {
		team, err := svc.ds.Team(ctx, *query.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get team from id")
		}
		teamName = team.Name
	}
	return &fleet.QuerySpec{
		Name:        query.Name,
		Description: query.Description,
		Query:       query.Query,

		TeamName:           teamName,
		Interval:           query.Interval,
		ObserverCanRun:     query.ObserverCanRun,
		Platform:           query.Platform,
		MinOsqueryVersion:  query.MinOsqueryVersion,
		AutomationsEnabled: query.AutomationsEnabled,
		Logging:            query.Logging,
		DiscardData:        query.DiscardData,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Spec
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecResponse struct {
	Spec *fleet.QuerySpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

type getQuerySpecRequest struct {
	Name   string `url:"name"`
	TeamID uint   `query:"team_id,optional"`
}

func (r getQuerySpecResponse) error() error { return r.Err }

func getQuerySpecEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getQuerySpecRequest)
	var teamID *uint
	if req.TeamID != 0 {
		teamID = &req.TeamID
	}
	spec, err := svc.GetQuerySpec(ctx, teamID, req.Name)
	if err != nil {
		return getQuerySpecResponse{Err: err}, nil
	}
	return getQuerySpecResponse{Spec: spec}, nil
}

func (svc *Service) GetQuerySpec(ctx context.Context, teamID *uint, name string) (*fleet.QuerySpec, error) {
	// Check the user is allowed to get the query on the requested team.
	if err := svc.authz.Authorize(ctx, &fleet.Query{
		TeamID: teamID,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	query, err := svc.ds.QueryByName(ctx, teamID, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get query by name")
	}
	spec, err := svc.specFromQuery(ctx, query)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create spec from query")
	}
	return spec, nil
}
