package digicert

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// REST client for https://one.digicert.com/mpki/docs/swagger-ui/index.html

// Timeout is the timeout for requests. It is exportable for testing.
var Timeout = ptr.Duration(20 * time.Second)

func VerifyProfileID(ctx context.Context, logger kitlog.Logger, config fleet.DigiCertIntegration) error {

	client := fleethttp.NewClient(fleethttp.WithTimeout(*Timeout))

	config.URL = strings.TrimRight(config.URL, "/")
	req, err := http.NewRequest("GET", config.URL+"/mpki/api/v2/profile/"+url.PathEscape(config.ProfileID), nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating DigiCert request")
	}
	req.Header.Set("X-API-key", config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sending DigiCert request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ctxerr.Errorf(ctx, "unexpected DigiCert status code: %d", resp.StatusCode)
	}

	type profile struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	var p profile
	err = json.UnmarshalRead(resp.Body, &p)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshaling DigiCert response")
	}
	level.Debug(logger).Log("msg", "DigiCert profile verified", "id", p.ID, "name", p.Name, "status", p.Status)
	return nil
}
