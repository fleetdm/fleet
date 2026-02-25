package fleet

type GetScheduledQueriesInPackRequest struct {
	ID          uint        `url:"id"`
	ListOptions ListOptions `url:"list_options"`
}

type ScheduledQueryResponse struct {
	ScheduledQuery
}

type GetScheduledQueriesInPackResponse struct {
	Scheduled []ScheduledQueryResponse `json:"scheduled"`
	Err       error                    `json:"error,omitempty"`
}

func (r GetScheduledQueriesInPackResponse) Error() error { return r.Err }

type ScheduleQueryRequest struct {
	PackID   uint    `json:"pack_id"`
	QueryID  uint    `json:"query_id" renameto:"report_id"`
	Interval uint    `json:"interval"`
	Snapshot *bool   `json:"snapshot"`
	Removed  *bool   `json:"removed"`
	Platform *string `json:"platform"`
	Version  *string `json:"version"`
	Shard    *uint   `json:"shard"`
}

type ScheduleQueryResponse struct {
	Scheduled *ScheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r ScheduleQueryResponse) Error() error { return r.Err }

type GetScheduledQueryRequest struct {
	ID uint `url:"id"`
}

type GetScheduledQueryResponse struct {
	Scheduled *ScheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r GetScheduledQueryResponse) Error() error { return r.Err }

type ModifyScheduledQueryRequest struct {
	ID uint `json:"-" url:"id"`
	ScheduledQueryPayload
}

type ModifyScheduledQueryResponse struct {
	Scheduled *ScheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r ModifyScheduledQueryResponse) Error() error { return r.Err }

type DeleteScheduledQueryRequest struct {
	ID uint `url:"id"`
}

type DeleteScheduledQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteScheduledQueryResponse) Error() error { return r.Err }
