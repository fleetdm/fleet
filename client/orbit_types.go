package client

// EnrollOrbitResponse is the response returned by the orbit enrollment endpoint.
type EnrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

func (r EnrollOrbitResponse) Error() error { return r.Err }
