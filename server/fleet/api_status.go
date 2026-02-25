package fleet

type StatusResponse struct {
	Err error `json:"error,omitempty"`
}

func (m StatusResponse) Error() error { return m.Err }
