package fleetctl

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/ghodss/yaml"
	zlog "github.com/rs/zerolog/log"
)

// fleetNameLookupTimeout bounds the whole lookup: the API client has no timeout of its own, and a wrong --fleet-url
// must not stall the package build.
var fleetNameLookupTimeout = 10 * time.Second

// fleetNameForPackage returns the name of the fleet that enrollSecret belongs to, asking the server through the
// fleetctl config contexts logged in to fleetURL. It returns "" whenever the name can't be found, which never fails the
// build: Fleet Free, a global (Unassigned) secret, a role that can't read enroll secrets (Fleet masks them), no matching
// context, or any error. Only an invalid or expired token prints a note, to errWriter.
func fleetNameForPackage(configPath, fleetURL, enrollSecret string, errWriter io.Writer) string {
	target, ok := normalizeServerURL(fleetURL)
	if !ok || enrollSecret == "" {
		return ""
	}
	// Read the config directly: `package` must not create ~/.fleet/config as a side effect.
	b, err := os.ReadFile(configPath)
	if err != nil {
		zlog.Debug().Err(err).Msg("fleet name lookup: read fleetctl config")
		return ""
	}
	var config configFile
	if err := yaml.Unmarshal(b, &config); err != nil {
		zlog.Debug().Err(err).Msg("fleet name lookup: parse fleetctl config")
		return ""
	}

	var names []string
	for name, cc := range config.Contexts {
		if address, ok := normalizeServerURL(joinURLPrefix(cc.Address, cc.URLPrefix)); ok && address == target && cc.Token != "" {
			names = append(names, name)
		}
	}
	slices.SortFunc(names, func(a, b string) int {
		switch {
		case a == b:
			return 0
		case a == "default":
			return -1
		case b == "default":
			return 1
		}
		return strings.Compare(a, b)
	})

	deadline := time.Now().Add(fleetNameLookupTimeout)
	listed, unauthenticated := false, false
	for _, name := range names {
		timeout := time.Until(deadline)
		if timeout <= 0 {
			break
		}
		fleetName, err := fleetNameFromServer(config.Contexts[name], enrollSecret, timeout)
		switch {
		case err == nil && fleetName != "":
			return fleetName
		case err == nil:
			listed = true
		case errors.Is(err, service.ErrUnauthenticated):
			unauthenticated = true
		}
		zlog.Debug().Err(err).Str("context", name).Msg("fleet name lookup: no fleet name")
	}
	if unauthenticated && !listed {
		fmt.Fprintf(errWriter, "Note: left the fleet name out of the file name because the fleetctl token for %s is invalid or expired. "+
			"To add it, log in with fleetctl login.\n", fleetURL)
	}
	return ""
}

// fleetNameFromServer lists the fleets visible to the context's user and returns the name of the one with enrollSecret,
// or "" if none has it.
func fleetNameFromServer(cc Context, enrollSecret string, timeout time.Duration) (string, error) {
	options := []service.ClientOption{service.SetClientErrorWriter(io.Discard)}
	if len(cc.CustomHeaders) > 0 {
		options = append(options, service.WithCustomHeaders(cc.CustomHeaders))
	}
	client, err := service.NewClient(cc.Address, cc.TLSSkipVerify, cc.RootCA, cc.URLPrefix, options...)
	if err != nil {
		return "", err
	}
	httpClient, ok := client.HTTP.(*http.Client)
	if !ok {
		return "", errors.New("unexpected HTTP client type")
	}
	httpClient.Timeout = timeout
	client.SetToken(cc.Token)

	fleets, err := client.ListTeams("")
	if err != nil {
		return "", err
	}
	for _, f := range fleets {
		for _, s := range f.Secrets {
			if s != nil && s.Secret == enrollSecret {
				return f.Name, nil
			}
		}
	}
	return "", nil
}

func joinURLPrefix(address, prefix string) string {
	if prefix = strings.Trim(prefix, "/"); prefix == "" {
		return address
	}
	return strings.TrimRight(address, "/") + "/" + prefix
}

// normalizeServerURL returns rawURL as scheme://host[:port][/path], lowercased, without a default port or a trailing
// slash, so that equivalent spellings of a server URL compare equal. A URL without a scheme is taken as https, since
// --fleet-url's help says "host:port".
func normalizeServerURL(rawURL string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", false
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return "", false
	}
	scheme, host, port := strings.ToLower(u.Scheme), strings.ToLower(u.Hostname()), u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" || strings.Contains(host, ":") {
		host = net.JoinHostPort(host, port)
		host = strings.TrimSuffix(host, ":")
	}
	return scheme + "://" + host + strings.TrimRight(u.Path, "/"), true
}
