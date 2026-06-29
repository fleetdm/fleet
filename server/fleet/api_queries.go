package fleet

////////////////////////////////////////////////////////////////////////////////
// Get Query
////////////////////////////////////////////////////////////////////////////////

type GetQueryRequest struct {
	ID uint `url:"id"`
}

type GetQueryResponse struct {
	// Because `fleet.Query` has a `query` field that we don't want to rename,
	// it's simpler to just duplicate the query in the response struct rather than
	// relying on the `renameto` tag here.
	// TODO - In Fleet 5, remove the extra field.
	Query  *Query `json:"query,omitempty"`
	Report *Query `json:"report,omitempty"`
	Err    error  `json:"error,omitempty"`
}

func (r GetQueryResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// List Queries
////////////////////////////////////////////////////////////////////////////////

type ListQueriesRequest struct {
	ListOptions ListOptions `url:"list_options"`
	// TeamID url argument set to 0 means global.
	TeamID         uint `query:"team_id,optional" renameto:"fleet_id"`
	MergeInherited bool `query:"merge_inherited,optional"`
	// only return queries targeted to run on this platform
	Platform string `query:"platform,optional"`
}

type ListQueriesResponse struct {
	Queries             []Query             `json:"queries" renameto:"reports"`
	Count               int                 `json:"count"`
	InheritedQueryCount int                 `json:"inherited_query_count" renameto:"inherited_report_count"`
	Meta                *PaginationMetadata `json:"meta"`
	Err                 error               `json:"error,omitempty"`
}

func (r ListQueriesResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Query Reports
////////////////////////////////////////////////////////////////////////////////

type GetQueryReportRequest struct {
	ID     uint  `url:"id"`
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetQueryReportResponse struct {
	QueryID       uint                 `json:"query_id" renameto:"report_id"`
	Results       []HostQueryResultRow `json:"results"`
	ReportClipped bool                 `json:"report_clipped"`
	Err           error                `json:"error,omitempty"`
}

func (r GetQueryReportResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Create Query
////////////////////////////////////////////////////////////////////////////////

type CreateQueryRequest struct {
	QueryPayload
}

type CreateQueryResponse struct {
	// Because `fleet.Query` has a `query` field that we don't want to rename,
	// it's simpler to just duplicate the query in the response struct rather than
	// relying on the `renameto` tag here.
	// TODO - In Fleet 5, remove the extra field.
	Query  *Query `json:"query,omitempty"`
	Report *Query `json:"report,omitempty"`
	Err    error  `json:"error,omitempty"`
}

func (r CreateQueryResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Modify Query
////////////////////////////////////////////////////////////////////////////////

type ModifyQueryRequest struct {
	ID uint `json:"-" url:"id"`
	QueryPayload
}

type ModifyQueryResponse struct {
	// Because `fleet.Query` has a `query` field that we don't want to rename,
	// it's simpler to just duplicate the query in the response struct rather than
	// relying on the `renameto` tag here.
	// TODO - In Fleet 5, remove the extra field.
	Query  *Query `json:"query,omitempty"`
	Report *Query `json:"report,omitempty"`
	Err    error  `json:"error,omitempty"`
}

func (r ModifyQueryResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Delete Query
////////////////////////////////////////////////////////////////////////////////

type DeleteQueryRequest struct {
	Name string `url:"name"`
	// TeamID if not set is assumed to be 0 (global).
	TeamID uint `url:"fleet_id,optional"`
}

type DeleteQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteQueryResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Delete Query By ID
////////////////////////////////////////////////////////////////////////////////

type DeleteQueryByIDRequest struct {
	ID uint `url:"id"`
}

type DeleteQueryByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteQueryByIDResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Delete Queries
////////////////////////////////////////////////////////////////////////////////

type DeleteQueriesRequest struct {
	IDs []uint `json:"ids"`
}

type DeleteQueriesResponse struct {
	Deleted uint  `json:"deleted"`
	Err     error `json:"error,omitempty"`
}

func (r DeleteQueriesResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Apply Query Specs
////////////////////////////////////////////////////////////////////////////////

type ApplyQuerySpecsRequest struct {
	Specs []*QuerySpec `json:"specs"`
}

type ApplyQuerySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyQuerySpecsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Query Specs
////////////////////////////////////////////////////////////////////////////////

type GetQuerySpecsRequest struct {
	TeamID uint `url:"fleet_id,optional"`
}

type GetQuerySpecsResponse struct {
	Specs []*QuerySpec `json:"specs"`
	Err   error        `json:"error,omitempty"`
}

func (r GetQuerySpecsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Query Spec
////////////////////////////////////////////////////////////////////////////////

type GetQuerySpecRequest struct {
	Name   string `url:"name"`
	TeamID uint   `query:"team_id,optional" renameto:"fleet_id"`
}

type GetQuerySpecResponse struct {
	Spec *QuerySpec `json:"specs,omitempty"`
	Err  error      `json:"error,omitempty"`
}

func (r GetQuerySpecResponse) Error() error { return r.Err }
