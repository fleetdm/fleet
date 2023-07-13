package commonmdm

import (
	"net/url"
	"path"
)

// ResolveURL resolves a relative path to a server URL (typically the Fleet
// server's). If cleanQuery is true, the query string part is cleared.
func ResolveURL(serverURL, relPath string, cleanQuery bool) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, relPath)
	if cleanQuery {
		u.RawQuery = ""
	}
	return u.String(), nil
}
