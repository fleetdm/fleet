package fleet

type ConditionalAccessMicrosoftCreateRequest struct {
	// MicrosoftTenantID holds the Entra tenant ID.
	MicrosoftTenantID string `json:"microsoft_tenant_id"`
}

type ConditionalAccessMicrosoftCreateResponse struct {
	// MicrosoftAuthenticationURL holds the URL to redirect the admin to consent access
	// to the tenant to Fleet's multi-tenant application.
	MicrosoftAuthenticationURL string `json:"microsoft_authentication_url"`
	Err                        error  `json:"error,omitempty"`
}

func (r ConditionalAccessMicrosoftCreateResponse) Error() error { return r.Err }

type ConditionalAccessMicrosoftConfirmRequest struct{}

type ConditionalAccessMicrosoftConfirmResponse struct {
	ConfigurationCompleted bool   `json:"configuration_completed"`
	SetupError             string `json:"setup_error"`
	Err                    error  `json:"error,omitempty"`
}

func (r ConditionalAccessMicrosoftConfirmResponse) Error() error { return r.Err }

type ConditionalAccessMicrosoftDeleteRequest struct{}

type ConditionalAccessMicrosoftDeleteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ConditionalAccessMicrosoftDeleteResponse) Error() error { return r.Err }
