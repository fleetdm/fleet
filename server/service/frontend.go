package service

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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

		enrollSecret := r.URL.Query().Get("enroll_secret")
		authRequired, err := requiresEnrollOTAAuthentication(r.Context(), ds, enrollSecret)
		if err != nil {
			herr(w, "check if authentication is required err: "+err.Error())
			return
		}

		if authRequired {
		}

		fs := newBinaryFileSystem("/frontend")
		file, err := fs.Open("templates/enroll-ota.html")
		if err != nil {
			herr(w, "load enroll ota template: "+err.Error())
			return
		}

		data, err := io.ReadAll(file)
		if err != nil {
			herr(w, "read bindata file: "+err.Error())
			return
		}

		t, err := template.New("enroll-ota").Parse(string(data))
		if err != nil {
			herr(w, "create react template: "+err.Error())
			return
		}

		enrollURL, err := generateEnrollOTAURL(urlPrefix, enrollSecret)
		if err != nil {
			herr(w, "generate enroll ota url: "+err.Error())
			return
		}
		if err := t.Execute(w, struct {
			EnrollURL             string
			URLPrefix             string
			AndroidMDMEnabled     bool
			MacMDMEnabled         bool
			AndroidFeatureEnabled bool
		}{
			URLPrefix:             urlPrefix,
			EnrollURL:             enrollURL,
			AndroidMDMEnabled:     appCfg.MDM.AndroidEnabledAndConfigured,
			MacMDMEnabled:         appCfg.MDM.EnabledAndConfigured,
			AndroidFeatureEnabled: true,
		}); err != nil {
			herr(w, "execute react template: "+err.Error())
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

func requiresEnrollOTAAuthentication(ctx context.Context, ds fleet.Datastore, enrollSecret string) (bool, error) {
	secret, err := ds.VerifyEnrollSecret(ctx, enrollSecret)
	if err != nil && !fleet.IsNotFound(err) {
		return false, ctxerr.Wrap(ctx, err, "verify enroll secret")
	}

	if secret == nil {
		// enroll secret is invalid, check if any team has IdP enabled for setup experience
		ids, err := ds.TeamIDsWithSetupExperienceIdPEnabled(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get team IDs with setup experience IdP enabled")
		}
		return len(ids) > 0, nil
	}

	if secret.TeamID == nil { // enroll in "no team"
		ac, err := ds.AppConfig(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get app config for no-team settings")
		}
		return ac.MDM.MacOSSetup.EnableEndUserAuthentication, nil
	}

	tm, err := ds.Team(ctx, *secret.TeamID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "get team for settings")
	}
	return tm.Config.MDM.MacOSSetup.EnableEndUserAuthentication, nil
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
