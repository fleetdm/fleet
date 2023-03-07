package externalsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

var ErrInvalidGrant = errors.New("the credentials provided were invalid")

type Okta struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
}

// oktaError is the response body for requests with errors coming from Okta.
type oktaError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// ROPLogin performs a login using the "Resource Owner Password Flow" as
// specified by RFC 6749 and described in
// https://developer.okta.com/docs/guides/implement-grant-type/ropassword/main/
func (o *Okta) ROPLogin(ctx context.Context, username, password string) error {
	params := url.Values{
		"username":   []string{username},
		"password":   []string{password},
		"scope":      []string{"openid"},
		"grant_type": []string{"password"},
	}
	req, err := http.NewRequestWithContext(
		ctx, "POST", o.BaseURL,
		strings.NewReader(params.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := o.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrInvalidGrant
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %s", err)
	}

	var oktaErr oktaError
	if err := json.Unmarshal(body, &oktaErr); err != nil {
		return fmt.Errorf("decoding okta response: %s", err)
	}
	return fmt.Errorf("okta error: %s: %s", oktaErr.Error, oktaErr.Description)
}

func (o *Okta) do(req *http.Request) (*http.Response, error) {
	client := fleethttp.NewClient()
	req.SetBasicAuth(o.ClientID, o.ClientSecret)
	return client.Do(req)
}
