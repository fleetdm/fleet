package fleet

type TriggerRequest struct {
	Name string `query:"name,optional"`
}

type TriggerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r TriggerResponse) Error() error { return r.Err }
