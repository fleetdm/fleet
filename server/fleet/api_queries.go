package fleet

type GetQueryRequest struct {
	ID uint `url:"id"`
}

type GetQueryResponse struct {
	Query *Query `json:"query,omitempty" renameto:"report"`
	Err   error  `json:"error,omitempty"`
}

func (r GetQueryResponse) Error() error { return r.Err }

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

type CreateQueryRequest struct {
	QueryPayload
}

type CreateQueryResponse struct {
	Query *Query `json:"query,omitempty" renameto:"report"`
	Err   error  `json:"error,omitempty"`
}

func (r CreateQueryResponse) Error() error { return r.Err }

type ModifyQueryRequest struct {
	ID uint `json:"-" url:"id"`
	QueryPayload
}

type ModifyQueryResponse struct {
	Query *Query `json:"query,omitempty" renameto:"report"`
	Err   error  `json:"error,omitempty"`
}

func (r ModifyQueryResponse) Error() error { return r.Err }

type DeleteQueryRequest struct {
	Name string `url:"name"`
	// TeamID if not set is assumed to be 0 (global).
	TeamID uint `url:"fleet_id,optional"`
}

type DeleteQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteQueryResponse) Error() error { return r.Err }

type DeleteQueryByIDRequest struct {
	ID uint `url:"id"`
}

type DeleteQueryByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteQueryByIDResponse) Error() error { return r.Err }

type DeleteQueriesRequest struct {
	IDs []uint `json:"ids"`
}

type DeleteQueriesResponse struct {
	Deleted uint  `json:"deleted"`
	Err     error `json:"error,omitempty"`
}

func (r DeleteQueriesResponse) Error() error { return r.Err }

type ApplyQuerySpecsRequest struct {
	Specs []*QuerySpec `json:"specs"`
}

type ApplyQuerySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyQuerySpecsResponse) Error() error { return r.Err }

type GetQuerySpecsResponse struct {
	Specs []*QuerySpec `json:"specs"`
	Err   error        `json:"error,omitempty"`
}

func (r GetQuerySpecsResponse) Error() error { return r.Err }

type GetQuerySpecsRequest struct {
	TeamID uint `url:"fleet_id,optional"`
}

type GetQuerySpecResponse struct {
	Spec *QuerySpec `json:"specs,omitempty"`
	Err  error      `json:"error,omitempty"`
}

func (r GetQuerySpecResponse) Error() error { return r.Err }

type GetQuerySpecRequest struct {
	Name   string `url:"name"`
	TeamID uint   `query:"team_id,optional" renameto:"fleet_id"`
}
