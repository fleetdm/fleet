package fleet

type PackResponse struct {
	Pack
	QueryCount uint `json:"query_count" renameto:"report_count"`

	// All current hosts in the pack. Hosts which are selected explicty and
	// hosts which are part of a label.
	TotalHostsCount uint `json:"total_hosts_count"`

	// IDs of hosts which were explicitly selected.
	HostIDs  []uint `json:"host_ids"`
	LabelIDs []uint `json:"label_ids"`
	TeamIDs  []uint `json:"team_ids" renameto:"fleet_ids"`
}

type GetPackRequest struct {
	ID uint `url:"id"`
}

type GetPackResponse struct {
	Pack PackResponse `json:"pack"`
	Err  error        `json:"error,omitempty"`
}

func (r GetPackResponse) Error() error { return r.Err }

type CreatePackRequest struct {
	PackPayload
}

type CreatePackResponse struct {
	Pack PackResponse `json:"pack"`
	Err  error        `json:"error,omitempty"`
}

func (r CreatePackResponse) Error() error { return r.Err }

type ModifyPackRequest struct {
	ID uint `json:"-" url:"id"`
	PackPayload
}

type ModifyPackResponse struct {
	Pack PackResponse `json:"pack"`
	Err  error        `json:"error,omitempty"`
}

func (r ModifyPackResponse) Error() error { return r.Err }

type ListPacksRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type ListPacksResponse struct {
	Packs []PackResponse `json:"packs"`
	Err   error          `json:"error,omitempty"`
}

func (r ListPacksResponse) Error() error { return r.Err }

type DeletePackRequest struct {
	Name string `url:"name"`
}

type DeletePackResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeletePackResponse) Error() error { return r.Err }

type DeletePackByIDRequest struct {
	ID uint `url:"id"`
}

type DeletePackByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeletePackByIDResponse) Error() error { return r.Err }

type ApplyPackSpecsRequest struct {
	Specs []*PackSpec `json:"specs"`
}

type ApplyPackSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyPackSpecsResponse) Error() error { return r.Err }

type GetPackSpecsResponse struct {
	Specs []*PackSpec `json:"specs"`
	Err   error       `json:"error,omitempty"`
}

func (r GetPackSpecsResponse) Error() error { return r.Err }

type GetPackSpecResponse struct {
	Spec *PackSpec `json:"specs,omitempty"`
	Err  error     `json:"error,omitempty"`
}

func (r GetPackSpecResponse) Error() error { return r.Err }
