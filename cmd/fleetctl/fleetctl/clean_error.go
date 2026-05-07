package fleetctl

import (
	"errors"
	"regexp"

	"github.com/fleetdm/fleet/v4/client"
)

// statusCodeErrPrefixRE matches the "<VERB> <path> received status <code> "
// prefix that BaseClient.ParseResponse adds when wrapping a *StatusCodeErr.
// VERB is uppercase (GET, POST, PATCH, ...) and path starts with "/".
var statusCodeErrPrefixRE = regexp.MustCompile(`[A-Z]+ /\S* received status \d+ `)

// CleanStatusCodeErr returns a copy of err suitable for display in fleetctl
// CLI output. If the chain contains a *client.StatusCodeErr, the verbose
// "VERB /path received status N " prefix added by the HTTP client is stripped
// so users see only the server-provided reason. If no StatusCodeErr is in the
// chain, err is returned unchanged.
//
// The returned value is for printing only — it does not preserve the original
// error chain, so callers that need to inspect err with errors.Is / errors.As
// should do so against the original error before passing it here.
func CleanStatusCodeErr(err error) error {
	if err == nil {
		return nil
	}
	var sce *client.StatusCodeErr
	if !errors.As(err, &sce) {
		return err
	}
	return errors.New(statusCodeErrPrefixRE.ReplaceAllString(err.Error(), ""))
}
