package e2e

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test/enrollment"
)

// Doer executes an HTTP request.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type api struct {
	doer Doer
}

func (a *api) RawCommandEnqueue(ctx context.Context, ids []string, cmd *mdm.Command, nopush bool) error {
	r, err := test.PlistReader(cmd)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(enqueueURL, "/") {
		return errors.New("missing trailing slash of enqueue URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, enqueueURL+strings.Join(ids, ","), r)
	if err != nil {
		return err
	}

	v := req.URL.Query()
	if nopush {
		v.Set("nopush", "1")
	}
	req.URL.RawQuery = v.Encode()

	resp, err := a.doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return enrollment.HTTPErrors(resp)
}
