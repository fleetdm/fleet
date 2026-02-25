package fleet

// TranslatorRequest is the request type for the translator endpoint.
type TranslatorRequest struct {
	List []TranslatePayload `json:"list"`
}

// TranslatorResponse is the response type for the translator endpoint.
type TranslatorResponse struct {
	List []TranslatePayload `json:"list"`
	Err  error              `json:"error,omitempty"`
}

func (r TranslatorResponse) Error() error { return r.Err }
