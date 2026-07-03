package fleet

//////////////////////////////////////////////////////////////////////////////////
// List custom host vitals
//////////////////////////////////////////////////////////////////////////////////

type ListCustomHostVitalsRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type ListCustomHostVitalsResponse struct {
	CustomHostVitals []CustomHostVital   `json:"custom_host_vitals"`
	Meta             *PaginationMetadata `json:"meta"`
	Count            int                 `json:"count"`

	Err error `json:"error,omitempty"`
}

func (r ListCustomHostVitalsResponse) Error() error { return r.Err }

//////////////////////////////////////////////////////////////////////////////////
// Create custom host vital
//////////////////////////////////////////////////////////////////////////////////

type CreateCustomHostVitalRequest struct {
	Name string `json:"name"`
}

type CreateCustomHostVitalResponse struct {
	CustomHostVital *CustomHostVital `json:"custom_host_vital,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r CreateCustomHostVitalResponse) Error() error { return r.Err }

//////////////////////////////////////////////////////////////////////////////////
// Update (rename) custom host vital
//////////////////////////////////////////////////////////////////////////////////

type UpdateCustomHostVitalRequest struct {
	ID   uint   `url:"id"`
	Name string `json:"name"`
}

type UpdateCustomHostVitalResponse struct {
	CustomHostVital *CustomHostVital `json:"custom_host_vital,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r UpdateCustomHostVitalResponse) Error() error { return r.Err }

//////////////////////////////////////////////////////////////////////////////////
// Delete custom host vital
//////////////////////////////////////////////////////////////////////////////////

type DeleteCustomHostVitalRequest struct {
	ID uint `url:"id"`
}

type DeleteCustomHostVitalResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteCustomHostVitalResponse) Error() error { return r.Err }

//////////////////////////////////////////////////////////////////////////////////
// Set host custom host vital value
//////////////////////////////////////////////////////////////////////////////////

type SetHostCustomHostVitalValueRequest struct {
	HostID uint   `url:"host_id"`
	ID     uint   `url:"id"`
	Value  string `json:"value"`
}

type SetHostCustomHostVitalValueResponse struct {
	Err error `json:"error,omitempty"`
}

func (r SetHostCustomHostVitalValueResponse) Error() error { return r.Err }
