package fleet

type GetTeamScheduleRequest struct {
	TeamID      uint        `url:"fleet_id"`
	ListOptions ListOptions `url:"list_options"`
}

type GetTeamScheduleResponse struct {
	Scheduled []ScheduledQueryResponse `json:"scheduled"`
	Err       error                    `json:"error,omitempty"`
}

func (r GetTeamScheduleResponse) Error() error { return r.Err }

type TeamScheduleQueryRequest struct {
	TeamID uint `url:"fleet_id"`
	ScheduledQueryPayload
}

type TeamScheduleQueryResponse struct {
	Scheduled *ScheduledQuery `json:"scheduled,omitempty"`
	Err       error           `json:"error,omitempty"`
}

func (r TeamScheduleQueryResponse) Error() error { return r.Err }

type ModifyTeamScheduleRequest struct {
	TeamID           uint `url:"fleet_id"`
	ScheduledQueryID uint `url:"report_id"`
	ScheduledQueryPayload
}

type ModifyTeamScheduleResponse struct {
	Scheduled *ScheduledQuery `json:"scheduled,omitempty"`
	Err       error           `json:"error,omitempty"`
}

func (r ModifyTeamScheduleResponse) Error() error { return r.Err }

type DeleteTeamScheduleRequest struct {
	TeamID           uint `url:"fleet_id"`
	ScheduledQueryID uint `url:"report_id"`
}

type DeleteTeamScheduleResponse struct {
	Scheduled *ScheduledQuery `json:"scheduled,omitempty"`
	Err       error           `json:"error,omitempty"`
}

func (r DeleteTeamScheduleResponse) Error() error { return r.Err }
