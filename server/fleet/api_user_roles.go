package fleet

type ApplyUserRoleSpecsRequest struct {
	Spec *UsersRoleSpec `json:"spec"`
}

type ApplyUserRoleSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyUserRoleSpecsResponse) Error() error { return r.Err }
