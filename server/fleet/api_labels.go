package fleet

////////////////////////////////////////////////////////////////////////////////
// Create Label
////////////////////////////////////////////////////////////////////////////////

type CreateLabelRequest struct {
	LabelPayload
}

type CreateLabelResponse struct {
	Label LabelResponse `json:"label"`
	Err   error         `json:"error,omitempty"`
}

func (r CreateLabelResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Modify Label
////////////////////////////////////////////////////////////////////////////////

type ModifyLabelRequest struct {
	ID uint `json:"-" url:"id"`
	ModifyLabelPayload
}

type ModifyLabelResponse struct {
	Label LabelWithTeamNameResponse `json:"label"`
	Err   error                     `json:"error,omitempty"`
}

func (r ModifyLabelResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Label
////////////////////////////////////////////////////////////////////////////////

type GetLabelRequest struct {
	ID uint `url:"id"`
}

type LabelWithTeamNameResponse struct {
	LabelWithTeamName
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
	HostIDs     []uint `json:"host_ids,omitempty"`
}

type LabelResponse struct {
	Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
	HostIDs     []uint `json:"host_ids,omitempty"`
}

type GetLabelResponse struct {
	Label LabelWithTeamNameResponse `json:"label"`
	Err   error                     `json:"error,omitempty"`
}

func (r GetLabelResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// List Labels
////////////////////////////////////////////////////////////////////////////////

type ListLabelsRequest struct {
	ListOptions       ListOptions `url:"label_list_options"`
	TeamID            *string     `query:"team_id,optional" renameto:"fleet_id"` // string because it's an int or "global"
	IncludeHostCounts *bool       `query:"include_host_counts,optional"`
}

type ListLabelsResponse struct {
	Labels []LabelResponse `json:"labels"`
	Err    error           `json:"error,omitempty"`
}

func (r ListLabelsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Labels Summary
////////////////////////////////////////////////////////////////////////////////

type GetLabelsSummaryRequest struct {
	TeamID *string `query:"team_id,optional" renameto:"fleet_id"` // string because it's an int or "global"
}

type GetLabelsSummaryResponse struct {
	Labels []*LabelSummary `json:"labels"`
	Err    error           `json:"error,omitempty"`
}

func (r GetLabelsSummaryResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// List Hosts in Label
////////////////////////////////////////////////////////////////////////////////

type ListHostsInLabelRequest struct {
	ID          uint            `url:"id"`
	ListOptions HostListOptions `url:"host_options"`
}

////////////////////////////////////////////////////////////////////////////////
// Delete Label
////////////////////////////////////////////////////////////////////////////////

type DeleteLabelRequest struct {
	Name string `url:"name"`
}

type DeleteLabelResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteLabelResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Delete Label By ID
////////////////////////////////////////////////////////////////////////////////

type DeleteLabelByIDRequest struct {
	ID uint `url:"id"`
}

type DeleteLabelByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteLabelByIDResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Apply Label Specs
////////////////////////////////////////////////////////////////////////////////

type ApplyLabelSpecsRequest struct {
	Specs       []*LabelSpec `json:"specs"`
	TeamID      *uint        `json:"-" query:"team_id,optional" renameto:"fleet_id"`
	NamesToMove []string     `json:"names_to_move,omitempty"`
}

type ApplyLabelSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyLabelSpecsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Label Specs
////////////////////////////////////////////////////////////////////////////////

type GetLabelSpecsRequest struct {
	TeamID *uint `query:"team_id,optional" renameto:"fleet_id"`
}

type GetLabelSpecsResponse struct {
	Specs []*LabelSpec `json:"specs"`
	Err   error        `json:"error,omitempty"`
}

func (r GetLabelSpecsResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Label Spec
////////////////////////////////////////////////////////////////////////////////

type GetLabelSpecResponse struct {
	Spec *LabelSpec `json:"specs,omitempty"`
	Err  error      `json:"error,omitempty"`
}

func (r GetLabelSpecResponse) Error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Remove Labels From Host
////////////////////////////////////////////////////////////////////////////////

type RemoveLabelsFromHostRequest struct {
	ID     uint     `url:"id"`
	Labels []string `json:"labels"`
}

type RemoveLabelsFromHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r RemoveLabelsFromHostResponse) Error() error { return r.Err }
