package client

// setOrbitNodeKeyer is implemented by orbit request structs that carry a node key.
type setOrbitNodeKeyer interface {
	setOrbitNodeKey(nodeKey string)
}

// EnrollOrbitResponse is the response returned by the orbit enrollment endpoint.
type EnrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

func (r EnrollOrbitResponse) Error() error { return r.Err }
