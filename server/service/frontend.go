package service

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/go-kit/log"
	"github.com/klauspost/compress/gzhttp"
)

func newBinaryFileSystem(root string) *assetfs.AssetFS {
	return &assetfs.AssetFS{
		Asset:     bindata.Asset,
		AssetDir:  bindata.AssetDir,
		AssetInfo: bindata.AssetInfo,
		Prefix:    root,
	}
}

func ServeFrontend(urlPrefix string, sandbox bool, logger log.Logger) http.Handler {
	herr := func(w http.ResponseWriter, err string) {
		logger.Log("err", err)
		http.Error(w, err, http.StatusInternalServerError)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint_utils.WriteBrowserSecurityHeaders(w)

		// The following check is to prevent a misconfigured osquery from submitting
		// data to the root endpoint (the osquery remote API uses POST for all its endpoints).
		// See https://github.com/fleetdm/fleet/issues/16182.
		if r.Method == "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		fs := newBinaryFileSystem("/frontend")
		file, err := fs.Open("templates/react.tmpl")
		if err != nil {
			herr(w, "load react template: "+err.Error())
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			herr(w, "read bindata file: "+err.Error())
			return
		}
		t, err := template.New("react").Parse(string(data))
		if err != nil {
			herr(w, "create react template: "+err.Error())
			return
		}
		serverType := "on-premise"
		if sandbox {
			serverType = "sandbox"
		}
		if err := t.Execute(w, struct {
			URLPrefix  string
			ServerType string
		}{
			URLPrefix:  urlPrefix,
			ServerType: serverType,
		}); err != nil {
			herr(w, "execute react template: "+err.Error())
			return
		}
	})
}

// ServeEndUserEnrollOTA implements the entrypoint handler for the /enroll
// path, used to add hosts in "BYOD" mode (currently, iPhone/iPad/Android).
func ServeEndUserEnrollOTA(
	svc fleet.Service,
	urlPrefix string,
	ds fleet.Datastore,
	logger log.Logger,
) http.Handler {
	herr := func(w http.ResponseWriter, err string) {
		logger.Log("err", err)
		http.Error(w, err, http.StatusInternalServerError)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint_utils.WriteBrowserSecurityHeaders(w)
		setupRequired, err := svc.SetupRequired(r.Context())
		if err != nil {
			herr(w, "setup required err: "+err.Error())
			return
		}
		if setupRequired {
			herr(w, "fleet instance not setup")
			return
		}

		appCfg, err := ds.AppConfig(r.Context())
		if err != nil {
			herr(w, "load appconfig err: "+err.Error())
			return
		}

		errorMsg := r.URL.Query().Get("error")
		if errorMsg != "" {
			if err := renderEnrollPage(w, appCfg, urlPrefix, "", errorMsg); err != nil {
				herr(w, err.Error())
			}
			return
		}

		enrollSecret := r.URL.Query().Get("enroll_secret")
		if enrollSecret == "" {
			if err := renderEnrollPage(w, appCfg, urlPrefix, "", "This URL is invalid. : Enroll secret is invalid. Please contact your IT admin."); err != nil {
				herr(w, err.Error())
			}
			return
		}

		authRequired, err := shared_mdm.RequiresEnrollOTAAuthentication(r.Context(), ds,
			enrollSecret, appCfg.MDM.MacOSSetup.EnableEndUserAuthentication)
		if err != nil {
			herr(w, "check if authentication is required err: "+err.Error())
			return
		}

		if authRequired {
			// check if authentication cookie is present, in which case we go ahead with
			// offering the enrollment profile to download.
			var cookieIdPRef string
			if byodCookie, _ := r.Cookie(shared_mdm.BYODIdpCookieName); byodCookie != nil {
				cookieIdPRef = byodCookie.Value

				// if the cookie is present, we should also receive a (matching) enroll reference
				if cookieIdPRef != "" {
					enrollRef := r.URL.Query().Get("enrollment_reference")
					if cookieIdPRef != enrollRef {
						cookieIdPRef = "" // cookie does not match the enroll reference, so we ignore it and require authentication
					}
				}
			}

			if cookieIdPRef == "" {
				// IdP authentication has not been completed yet, initiate it by
				// redirecting to the configured IdP provider.
				if err := initiateOTAEnrollSSO(svc, w, r, enrollSecret); err != nil {
					herr(w, "initiate IdP SSO authentication err: "+err.Error())
					return
				}
				return
			}
		}

		// if we get here, IdP SSO authentication is either not required, or has
		// been successfully completed (we have a cookie with the IdP account
		// reference).
		if err := renderEnrollPage(w, appCfg, urlPrefix, enrollSecret, ""); err != nil {
			herr(w, err.Error())
			return
		}
	})
}

func generateEnrollOTAURL(fleetURL string, enrollSecret string) (string, error) {
	path, err := url.JoinPath(fleetURL, "/api/v1/fleet/enrollment_profiles/ota")
	if err != nil {
		return "", fmt.Errorf("creating path for end user ota enrollment url: %w", err)
	}

	enrollURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("parsing end user ota enrollment url: %w", err)
	}

	q := enrollURL.Query()
	q.Set("enroll_secret", enrollSecret)
	enrollURL.RawQuery = q.Encode()
	return enrollURL.String(), nil
}

func renderEnrollPage(w io.Writer, appCfg *fleet.AppConfig, urlPrefix, enrollSecret, errorMessage string) error {
	fs := newBinaryFileSystem("/frontend")
	file, err := fs.Open("templates/enroll-ota.html")
	if err != nil {
		return fmt.Errorf("load enroll ota template: %w", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read bindata file: %w", err)
	}

	t, err := template.New("enroll-ota").Parse(string(data))
	if err != nil {
		return fmt.Errorf("create react template: %w", err)
	}

	enrollURL, err := generateEnrollOTAURL(urlPrefix, enrollSecret)
	if err != nil {
		return fmt.Errorf("generate enroll ota url: %w", err)
	}
	if err := t.Execute(w, struct {
		EnrollURL             string
		URLPrefix             string
		ErrorMessage          string
		AndroidMDMEnabled     bool
		MacMDMEnabled         bool
		AndroidFeatureEnabled bool
	}{
		URLPrefix:             urlPrefix,
		EnrollURL:             enrollURL,
		ErrorMessage:          errorMessage,
		AndroidMDMEnabled:     appCfg.MDM.AndroidEnabledAndConfigured,
		MacMDMEnabled:         appCfg.MDM.EnabledAndConfigured,
		AndroidFeatureEnabled: true,
	}); err != nil {
		return fmt.Errorf("execute react template: %w", err)
	}
	return nil
}

func initiateOTAEnrollSSO(svc fleet.Service, w http.ResponseWriter, r *http.Request, enrollSecret string) error {
	ssnID, ssnDurationSecs, idpURL, err := svc.InitiateMDMSSO(r.Context(), "ota_enroll", "/enroll?enroll_secret="+url.QueryEscape(enrollSecret), "")
	if err != nil {
		return err
	}
	setSSOCookie(w, ssnID, ssnDurationSecs)
	http.Redirect(w, r, idpURL, http.StatusSeeOther)
	return nil
}

func ServeStaticAssets(path string) http.Handler {
	contentTypes := []string{"text/javascript", "text/css"}
	withoutGzip := http.StripPrefix(path, http.FileServer(newBinaryFileSystem("/assets")))

	withOpts, err := gzhttp.NewWrapper(gzhttp.ContentTypes(contentTypes))
	if err != nil { // fall back to serving without gzip if serving with gzip somehow fails
		return withoutGzip
	}

	return withOpts(withoutGzip)
}
