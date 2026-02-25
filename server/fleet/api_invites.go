package fleet

type CreateInviteRequest struct {
	InvitePayload
}

type CreateInviteResponse struct {
	Invite *Invite `json:"invite,omitempty"`
	Err    error   `json:"error,omitempty"`
}

func (r CreateInviteResponse) Error() error { return r.Err }

type ListInvitesRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type ListInvitesResponse struct {
	Invites []Invite `json:"invites"`
	Err     error    `json:"error,omitempty"`
}

func (r ListInvitesResponse) Error() error { return r.Err }

type UpdateInviteRequest struct {
	ID uint `url:"id"`
	InvitePayload
}

type UpdateInviteResponse struct {
	Invite *Invite `json:"invite"`
	Err    error   `json:"error,omitempty"`
}

func (r UpdateInviteResponse) Error() error { return r.Err }

type DeleteInviteRequest struct {
	ID uint `url:"id"`
}

type DeleteInviteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteInviteResponse) Error() error { return r.Err }

type VerifyInviteRequest struct {
	Token string `url:"token"`
}

type VerifyInviteResponse struct {
	Invite *Invite `json:"invite"`
	Err    error   `json:"error,omitempty"`
}

func (r VerifyInviteResponse) Error() error { return r.Err }
