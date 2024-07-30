package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/version"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/urfave/cli/v2"
)

var ErrGeneric = errors.New(`Something's gone wrong. Please try again. If this keeps happening please file an issue:
https://github.com/fleetdm/fleet/issues/new/choose`)

func unauthenticatedClientFromCLI(c *cli.Context) (*service.Client, error) {
	cc, err := clientConfigFromCLI(c)
	if err != nil {
		return nil, err
	}

	return unauthenticatedClientFromConfig(cc, getDebug(c), c.App.Writer, c.App.ErrWriter)
}

func clientFromCLI(c *cli.Context) (*service.Client, error) {
	fleetClient, err := unauthenticatedClientFromCLI(c)
	if err != nil {
		return nil, err
	}

	configPath, context := c.String("config"), c.String("context")

	// if a config file is explicitly provided, do not set an invalid arbitrary token
	if !c.IsSet("config") && flag.Lookup("test.v") != nil {
		fleetClient.SetToken("AAAA")
		return fleetClient, nil
	}

	// Add authentication token
	t, err := getConfigValue(configPath, context, "token")
	if err != nil {
		return nil, fmt.Errorf("error getting token from the config: %w", err)
	}

	token, ok := t.(string)
	if !ok {
		fmt.Fprintln(os.Stderr, "Token invalid. Please log in with: fleetctl login")
		return nil, fmt.Errorf("token config value expected type %T, got %T: %+v", "", t, t)
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "Token missing. Please log in with: fleetctl login")
		return nil, errors.New("token config value missing")
	}
	fleetClient.SetToken(token)

	// Check if version matches fleet server. Also ensures that the token is valid.
	clientInfo := version.Version()

	serverInfo, err := fleetClient.Version()
	if err != nil {
		if errors.Is(err, service.ErrUnauthenticated) {
			fmt.Fprintln(os.Stderr, "Token invalid or session expired. Please log in with: fleetctl login")
		}
		return nil, err
	}

	if clientInfo.Version != serverInfo.Version {
		fmt.Fprintf(
			os.Stderr,
			"Warning: Version mismatch.\nClient Version:   %s\nServer Version:  %s\n",
			clientInfo.Version, serverInfo.Version,
		)
		// This is just a warning, continue ...
	}

	// check that AppConfig's Apple BM terms are not expired.
	var sce kithttp.StatusCoder
	switch appCfg, err := fleetClient.GetAppConfig(); {
	case err == nil:
		if appCfg.MDM.AppleBMTermsExpired {
			fleet.WriteAppleBMTermsExpiredBanner(os.Stderr)
			// This is just a warning, continue ...
		}
	case errors.As(err, &sce) && sce.StatusCode() == http.StatusForbidden:
		// OK, could be a user without permissions to read app config (e.g. gitops).
	default:
		return nil, err
	}

	return fleetClient, nil
}

func unauthenticatedClientFromConfig(cc Context, debug bool, outputWriter io.Writer, errWriter io.Writer) (*service.Client, error) {
	options := []service.ClientOption{
		service.SetClientOutputWriter(outputWriter),
		service.SetClientErrorWriter(errWriter),
	}

	if len(cc.CustomHeaders) > 0 {
		options = append(options, service.WithCustomHeaders(cc.CustomHeaders))
	}

	if flag.Lookup("test.v") != nil {
		return service.NewClient(
			os.Getenv("FLEET_SERVER_ADDRESS"), true, "", "", options...)
	}

	if cc.Address == "" {
		return nil, errors.New("set the Fleet API address with: fleetctl config set --address https://localhost:8080")
	}

	if runtime.GOOS == "windows" && cc.RootCA == "" && !cc.TLSSkipVerify {
		return nil, errors.New("Windows clients must configure rootca (secure) or tls-skip-verify (insecure)")
	}

	if debug {
		options = append(options, service.EnableClientDebug())
	}

	fleet, err := service.NewClient(
		cc.Address,
		cc.TLSSkipVerify,
		cc.RootCA,
		cc.URLPrefix,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Fleet API client handler: %w", err)
	}

	return fleet, nil
}

// returns an HTTP client and the parsed URL for the configured server's
// address. The reason why this exists instead of using
// unauthenticatedClientFromConfig is because this doesn't apply the same rules
// around TLS config - in particular, it only sets a root CA if one is
// explicitly configured.
func rawHTTPClientFromConfig(cc Context) (*http.Client, *url.URL, error) {
	if flag.Lookup("test.v") != nil {
		cc.Address = os.Getenv("FLEET_SERVER_ADDRESS")
	}
	baseURL, err := url.Parse(cc.Address)
	if err != nil {
		return nil, nil, fmt.Errorf("parse address: %w", err)
	}

	var rootCA *x509.CertPool
	if cc.RootCA != "" {
		rootCA = x509.NewCertPool()
		// read in the root cert file specified in the context
		certs, err := os.ReadFile(cc.RootCA)
		if err != nil {
			return nil, nil, fmt.Errorf("reading root CA: %w", err)
		}

		// add certs to pool
		if ok := rootCA.AppendCertsFromPEM(certs); !ok {
			return nil, nil, errors.New("failed to add certificates to root CA pool")
		}
	}

	cli := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: cc.TLSSkipVerify,
		RootCAs:            rootCA,
	}))
	return cli, baseURL, nil
}

func clientConfigFromCLI(c *cli.Context) (Context, error) {
	// if a config file is explicitly provided, do not return a default context,
	// just override the address and skip verify before returning.
	if !c.IsSet("config") && flag.Lookup("test.v") != nil {
		return Context{
			Address:       os.Getenv("FLEET_SERVER_ADDRESS"),
			TLSSkipVerify: true,
		}, nil
	}

	var zeroCtx Context

	if err := makeConfigIfNotExists(c.String("config")); err != nil {
		return zeroCtx, fmt.Errorf("error verifying that config exists at %s: %w", c.String("config"), err)
	}

	config, err := readConfig(c.String("config"))
	if err != nil {
		return zeroCtx, err
	}

	cc, ok := config.Contexts[c.String("context")]
	if !ok {
		return zeroCtx, fmt.Errorf("context %q is not found", c.String("context"))
	}
	if flag.Lookup("test.v") != nil {
		cc.Address = os.Getenv("FLEET_SERVER_ADDRESS")
		cc.TLSSkipVerify = true
	}
	return cc, nil
}

// apiCommand fleetctl api [options] uri
// -F, --field <key=value>
// Add a typed parameter in key=value format
// -H, --header <key:value>
// Add a HTTP request header in key:value format
// -X, --method <string> (default "GET")
// The HTTP method for the request
func apiCommand() *cli.Command {
	var (
		flField  []string
		flHeader []string
		flMethod string
	)
	return &cli.Command{
		Name:      "api",
		Usage:     "Run an api command by uri",
		UsageText: `fleetctl api [options] [url]`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "F",
				Aliases: []string{"field"},
				Usage:   "Add a typed parameter in key=value format",
			},
			&cli.StringSliceFlag{
				Name:    "H",
				Aliases: []string{"header"},
				Usage:   "Add a HTTP request header in key:value format",
			},
			&cli.StringFlag{
				Name:        "X",
				Value:       "GET",
				Destination: &flMethod,
				Usage:       "The HTTP method for the request",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			uriString := c.Args().First()
			params := url.Values{}
			method := "GET"
			// TODO add param for body for POST etc

			if uriString == "" {
				return errors.New("must provide uri first argument")
			}

			flField = c.StringSlice("F")
			flHeader = c.StringSlice("H")

			if len(flField) > 0 {
				for _, each := range flField {
					k, v, found := strings.Cut(each, "=")
					if !found {
						continue
					}
					params.Add(k, v)
				}
			}

			headers := map[string]string{}
			if len(flHeader) > 0 {
				for _, each := range flHeader {
					k, v, found := strings.Cut(each, ":")
					if !found {
						continue
					}
					headers[k] = v
				}
			}

			if flMethod != "" {
				method = flMethod
			}

			if !strings.HasPrefix(uriString, "/") {
				uriString = fmt.Sprintf("/%s", uriString)
			}

			if !strings.HasPrefix(uriString, "/api/v1/fleet") {
				uriString = fmt.Sprintf("/api/v1/fleet%s", uriString)
			}

			fleetClient, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			resp, err := fleetClient.AuthenticatedDoCustomHeaders(method, uriString, params.Encode(), nil, headers)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				_, err := io.Copy(c.App.Writer, resp.Body)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("Got non 2XX return of %d", resp.StatusCode)
			}

			return nil
		},
	}
}
