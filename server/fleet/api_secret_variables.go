package fleet

/////////////////////////////////////////////////////////////////////////////////
// Create secret variables (spec)
//////////////////////////////////////////////////////////////////////////////////

type CreateSecretVariablesRequest struct {
	DryRun          bool             `json:"dry_run"`
	SecretVariables []SecretVariable `json:"secrets"`
}

type CreateSecretVariablesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r CreateSecretVariablesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Create secret variable
//////////////////////////////////////////////////////////////////////////////////

type CreateSecretVariableRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CreateSecretVariableResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`

	Err error `json:"error,omitempty"`
}

func (r CreateSecretVariableResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// List secret variables
//////////////////////////////////////////////////////////////////////////////////

type ListSecretVariablesRequest struct {
	ListOptions ListOptions `url:"list_options"`
}

type ListSecretVariablesResponse struct {
	CustomVariables []SecretVariableIdentifier `json:"custom_variables"`
	Meta            *PaginationMetadata        `json:"meta"`
	Count           int                        `json:"count"`

	Err error `json:"error,omitempty"`
}

func (r ListSecretVariablesResponse) Error() error { return r.Err }

/////////////////////////////////////////////////////////////////////////////////
// Delete secret variable
//////////////////////////////////////////////////////////////////////////////////

type DeleteSecretVariableRequest struct {
	ID uint `url:"id"`
}

type DeleteSecretVariableResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteSecretVariableResponse) Error() error { return r.Err }
