package fleetctl

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
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
// Add a parameter to the query string in key=value format
// -B, --body-field <key=value>
// Add a value to the body in key=value format. If the value is in the format of "@filename", upload
// that file as a MIME multipart upload. If the value is in the format of "<filename", use the
// contents of that file as the value of this key.
// -H, --header <key:value>
// Add a HTTP request header in key:value format
// -X, --method <string> (default "GET")
// The HTTP method for the request
func apiCommand() *cli.Command {
	var (
		flQuery  []string
		flBody   []string
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
				Usage:   "Add a parameter to the query string in key=value format",
			},
			&cli.StringSliceFlag{
				Name:    "B",
				Aliases: []string{"body-field"},
				Usage: `Add a value to the body in key=value format. If the value is in the ` +
					`format of "@filename", upload that file using a MIME multipart upload. If ` +
					`the value is in the format of "<filename", use the contents of that file as ` +
					`the value of this key.`,
			},
			&cli.StringSliceFlag{
				Name:    "H",
				Aliases: []string{"header"},
				Usage:   "Add a HTTP request header in key:value format",
			},
			&cli.StringFlag{
				Name:        "X",
				Value:       "",
				Destination: &flMethod,
				Usage: "The HTTP method for the request. Defaults to GET, or POST when -B " +
					"arguments are present.",
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
			} else if c.Args().Len() > 1 {
				return fmt.Errorf("extra arguments: %s\nEnsure any flags are before the URL",
					strings.Join(c.Args().Slice()[1:], " "))
			}

			flQuery = c.StringSlice("F")
			flHeader = c.StringSlice("H")
			flBody = c.StringSlice("B")

			if len(flQuery) > 0 {
				for _, each := range flQuery {
					k, v, found := strings.Cut(each, "=")
					if !found {
						continue
					}
					params.Add(k, v)
				}
			}

			body, headers, err := parseBodyFlags(flBody)
			if err != nil {
				return err
			}
			if body != nil {
				// If a body is present, change the default method to POST. It can
				// still be overridden with -X. If the user attempts to send a body
				// with a GET request, an error is returned.
				method = "POST"
			}

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

			if body != nil && method == "GET" {
				return fmt.Errorf("GET requests may not include a body")
			}

			if !strings.HasPrefix(uriString, "/") {
				uriString = fmt.Sprintf("/%s", uriString)
			}

			if !strings.HasPrefix(uriString, "/api/v1/fleet") && !strings.HasPrefix(uriString, "/api/latest/fleet") {
				uriString = fmt.Sprintf("/api/v1/fleet%s", uriString)
			}

			fleetClient, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			resp, err := fleetClient.AuthenticatedDoCustomHeaders(method, uriString, params.Encode(), body, headers)
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

// parseBodyFlags parses the value of the "-B" (request body data field) and returns the body,
// headers and any fatal errors encountered.
func parseBodyFlags(flBody []string) (body any, headers map[string]string, err error) {
	headers = make(map[string]string)
	if len(flBody) == 0 {
		return
	}

	// Populate jsonBody and multipartWriter in parallel. If we encounter a file upload,
	jsonBody := make(map[string]any)
	isMultiPart := false
	multipartBuffer := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(multipartBuffer)

	for _, each := range flBody {
		k, v, found := strings.Cut(each, "=")
		if !found {
			// if no "=", set body field to true
			multipartWriter.WriteField(k, "true")
			jsonBody[k] = true
			continue
		}
		if len(v) > 0 && (v[0] == '<' || v[0] == '@') {
			st, err := os.Stat(v[1:])
			switch {
			case err != nil:
				fmt.Fprintf(os.Stderr,
					"warning: encountered argument value %q but unable to stat file %q, the value "+
						"will be sent as a literal string",
					v, v[1:])
			case !st.Mode().IsRegular():
				fmt.Fprintf(os.Stderr,
					"warning: encountered argument value %q but file %q is not a regular file, "+
						"the value will be sent as a literal string",
					v, v[1:])
			default:
				// the file is validated to exist and be a regular file
				switch v[0] {
				case '<':
					// use contents of the file as the value of this field
					if contents, err := os.ReadFile(v[1:]); err == nil {
						v = string(contents)
					} else {
						fmt.Fprintf(os.Stderr,
							"warning: while processing body field %s: unable to read file %q: %v; "+
								"the field will be sent as the literal string %q",
							k, v[1:], err, v)
					}
				case '@':
					// do a multipart file upload
					if fileWriter, err := multipartWriter.CreateFormFile(k, path.Base(v[1:])); err == nil {
						if fp, err := os.Open(v[1:]); err == nil {
							defer fp.Close()
							io.Copy(fileWriter, fp)
							// from here on out, this is definitely a multipart upload
							isMultiPart = true
							// skip the rest of the loop so we don't attempt to write as a regular
							// form field
							continue
						} else {
							fmt.Fprintf(os.Stderr,
								"warning: while processing body field %s: error opening file %q "+
									"for upload: %v; the field will be sent as the literal string %q",
								k, v[1:], err, v)
						}
					} else {
						fmt.Fprintf(os.Stderr,
							"warning: while processing body field %s: error creating form file "+
								"field: %v; the field will be sent as the literal string %q",
							k, err, v)
					}
				}
			}
		}
		var jsonValue any
		multipartWriter.WriteField(k, v)
		if err := json.Unmarshal([]byte(v), &jsonValue); err == nil {
			jsonBody[k] = jsonValue
		} else {
			jsonBody[k] = v
		}
	}

	if isMultiPart {
		multipartWriter.Close()
		body = multipartBuffer
		headers["Content-Type"] = multipartWriter.FormDataContentType()
		headers["Content-Length"] = fmt.Sprintf("%d", multipartBuffer.Len())
	} else {
		body, err = json.Marshal(jsonBody)
		if err != nil {
			return
		}

		headers["Content-Type"] = "application/json"
		headers["Content-Length"] = fmt.Sprintf("%d", len(body.([]byte)))
	}

	return
}
