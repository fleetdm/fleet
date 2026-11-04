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

// packageFleet is what the package file name needs about the fleet an enroll secret belongs to: its name, and its ID
// with the other fleets' names, to tell apart fleets whose names clean up the same.
type packageFleet struct {
	Name       string
	ID         uint
	OtherNames []string
}

// fleetForPackage finds the fleet that enrollSecret belongs to, asking the server through the fleetctl config contexts
// logged in to fleetURL. It returns the zero value whenever the fleet can't be found, which never fails the build: Fleet
// Free, a global (Unassigned) secret, a role that can't read enroll secrets (Fleet masks them), no matching context, or
// any error. Only an invalid or expired token prints a note, to errWriter.
func fleetForPackage(configPath, fleetURL, enrollSecret string, errWriter io.Writer) packageFleet {
	target, ok := normalizeServerURL(fleetURL)
	if !ok || enrollSecret == "" {
		return packageFleet{}
	}
	// Read the config directly: `package` must not create ~/.fleet/config as a side effect.
	b, err := os.ReadFile(configPath)
	if err != nil {
		zlog.Debug().Err(err).Msg("fleet name lookup: read fleetctl config")
		return packageFleet{}
	}
	var config configFile
	if err := yaml.Unmarshal(b, &config); err != nil {
		zlog.Debug().Err(err).Msg("fleet name lookup: parse fleetctl config")
		return packageFleet{}
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
		fleet, err := fleetFromServer(config.Contexts[name], enrollSecret, timeout)
		switch {
		case err == nil && fleet.Name != "":
			return fleet
		case err == nil:
			listed = true
			zlog.Debug().Str("context", name).Msg("fleet name lookup: no fleet visible to this user has the enroll secret")
		default:
			unauthenticated = unauthenticated || errors.Is(err, service.ErrUnauthenticated)
			zlog.Debug().Err(err).Str("context", name).Msg("fleet name lookup: list fleets")
		}
	}
	if unauthenticated && !listed {
		fmt.Fprintf(errWriter, "Note: the fleet's name was left out of the package's filename because the API token for %s is invalid or "+
			"expired. To add it, log in with: fleetctl login\n", fleetURL)
	}
	return packageFleet{}
}

// fleetFromServer lists the fleets visible to the context's user and returns the one with enrollSecret, or the zero
// value if none has it.
func fleetFromServer(cc Context, enrollSecret string, timeout time.Duration) (packageFleet, error) {
	options := []service.ClientOption{service.SetClientErrorWriter(io.Discard)}
	if len(cc.CustomHeaders) > 0 {
		options = append(options, service.WithCustomHeaders(cc.CustomHeaders))
	}
	client, err := service.NewClient(cc.Address, cc.TLSSkipVerify, cc.RootCA, cc.URLPrefix, options...)
	if err != nil {
		return packageFleet{}, err
	}
	httpClient, ok := client.HTTP.(*http.Client)
	if !ok {
		return packageFleet{}, errors.New("unexpected HTTP client type")
	}
	httpClient.Timeout = timeout
	client.SetToken(cc.Token)

	fleets, err := client.ListTeams("")
	if err != nil {
		return packageFleet{}, err
	}
	for _, f := range fleets {
		for _, s := range f.Secrets {
			if s != nil && s.Secret == enrollSecret {
				match := packageFleet{Name: f.Name, ID: f.ID}
				for _, other := range fleets {
					if other.ID != f.ID {
						match.OtherNames = append(match.OtherNames, other.Name)
					}
				}
				return match, nil
			}
		}
	}
	return packageFleet{}, nil
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
