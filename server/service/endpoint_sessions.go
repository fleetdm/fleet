package service

import (
	"bytes"
	"context"
	"html/template"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type initiateSSORequest struct {
	RelayURL string `json:"relay_url"`
}

type initiateSSOResponse struct {
	URL string `json:"url,omitempty"`
	Err error  `json:"error,omitempty"`
}

func (r initiateSSOResponse) error() error { return r.Err }

func makeInitiateSSOEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(initiateSSORequest)
		idProviderURL, err := svc.InitiateSSO(ctx, req.RelayURL)
		if err != nil {
			return initiateSSOResponse{Err: err}, nil
		}
		return initiateSSOResponse{URL: idProviderURL}, nil
	}
}

type callbackSSOResponse struct {
	content string
	Err     error `json:"error,omitempty"`
}

func (r callbackSSOResponse) error() error { return r.Err }

// If html is present we return a web page
func (r callbackSSOResponse) html() string { return r.content }

func makeCallbackSSOEndpoint(svc fleet.Service, urlPrefix string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authResponse := request.(fleet.Auth)
		session, err := svc.CallbackSSO(ctx, authResponse)
		var resp callbackSSOResponse
		if err != nil {
			// redirect to login page on front end if there was some problem,
			// errors should still be logged
			session = &fleet.SSOSession{
				RedirectURL: urlPrefix + "/login",
				Token:       "",
			}
			resp.Err = err
		}
		relayStateLoadPage := ` <html>
     <script type='text/javascript'>
     var redirectURL = {{ .RedirectURL }};
     window.localStorage.setItem('FLEET::auth_token', '{{ .Token }}');
     window.location = redirectURL;
     </script>
     <body>
     Redirecting to Fleet at {{ .RedirectURL }} ...
     </body>
     </html>
    `
		tmpl, err := template.New("relayStateLoader").Parse(relayStateLoadPage)
		if err != nil {
			return nil, err
		}
		var writer bytes.Buffer
		err = tmpl.Execute(&writer, session)
		if err != nil {
			return nil, err
		}
		resp.content = writer.String()
		return resp, nil
	}
}

type ssoSettingsResponse struct {
	Settings *fleet.SessionSSOSettings `json:"settings,omitempty"`
	Err      error                     `json:"error,omitempty"`
}

func (r ssoSettingsResponse) error() error { return r.Err }

func makeSSOSettingsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, unused interface{}) (interface{}, error) {
		settings, err := svc.SSOSettings(ctx)
		if err != nil {
			return ssoSettingsResponse{Err: err}, nil
		}
		return ssoSettingsResponse{Settings: settings}, nil
	}
}
