package fleet

type GetGlobalScheduleRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type GetGlobalScheduleResponse struct {
	GlobalSchedule []*ScheduledQuery `json:"global_schedule"`
	Err            error             `json:"error,omitempty"`
}

func (r GetGlobalScheduleResponse) Error() error { return r.Err }

type GlobalScheduleQueryRequest struct {
	QueryID  uint    `json:"query_id" renameto:"report_id"`
	Interval uint    `json:"interval"`
	Snapshot *bool   `json:"snapshot"`
	Removed  *bool   `json:"removed"`
	Platform *string `json:"platform"`
	Version  *string `json:"version"`
	Shard    *uint   `json:"shard"`
}

type GlobalScheduleQueryResponse struct {
	Scheduled *ScheduledQuery `json:"scheduled,omitempty"`
	Err       error           `json:"error,omitempty"`
}

func (r GlobalScheduleQueryResponse) Error() error { return r.Err }

type ModifyGlobalScheduleRequest struct {
	ID uint `json:"-" url:"id"`
	ScheduledQueryPayload
}

type ModifyGlobalScheduleResponse struct {
	Scheduled *ScheduledQuery `json:"scheduled,omitempty"`
	Err       error           `json:"error,omitempty"`
}

func (r ModifyGlobalScheduleResponse) Error() error { return r.Err }

type DeleteGlobalScheduleRequest struct {
	ID uint `url:"id"`
}

type DeleteGlobalScheduleResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteGlobalScheduleResponse) Error() error { return r.Err }
