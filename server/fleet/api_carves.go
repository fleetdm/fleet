package fleet

import ()

type ListCarvesRequest struct {
	ListOptions CarveListOptions `url:"carve_options"`
}

type ListCarvesResponse struct {
	Carves []CarveMetadata `json:"carves"`
	Err    error           `json:"error,omitempty"`
}

func (r ListCarvesResponse) Error() error { return r.Err }

type GetCarveRequest struct {
	ID int64 `url:"id"`
}

type GetCarveResponse struct {
	Carve CarveMetadata `json:"carve"`
	Err   error         `json:"error,omitempty"`
}

func (r GetCarveResponse) Error() error { return r.Err }

type GetCarveBlockRequest struct {
	ID      int64 `url:"id"`
	BlockId int64 `url:"block_id"`
}

type GetCarveBlockResponse struct {
	Data []byte `json:"data"`
	Err  error  `json:"error,omitempty"`
}

func (r GetCarveBlockResponse) Error() error { return r.Err }

type CarveBeginRequest struct {
	NodeKey    string `json:"node_key"`
	BlockCount int64  `json:"block_count"`
	BlockSize  int64  `json:"block_size"`
	CarveSize  int64  `json:"carve_size"`
	CarveId    string `json:"carve_id"`
	RequestId  string `json:"request_id"`
}

func (r *CarveBeginRequest) hostNodeKey() string {
	return r.NodeKey
}

type CarveBeginResponse struct {
	SessionId string `json:"session_id"`
	Success   bool   `json:"success,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r CarveBeginResponse) Error() error { return r.Err }

type CarveBlockRequest struct {
	BlockId   int64  `json:"block_id"`
	SessionId string `json:"session_id"`
	RequestId string `json:"request_id"`
	Data      []byte `json:"data"`
}

type CarveBlockResponse struct {
	Success bool  `json:"success,omitempty"`
	Err     error `json:"error,omitempty"`
}

func (r CarveBlockResponse) Error() error { return r.Err }
