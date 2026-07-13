package service

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
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

func ServeFrontend(urlPrefix string, sandbox bool, logger *slog.Logger, serveCSP bool) http.Handler {
	herr := func(ctx context.Context, w http.ResponseWriter, err string) {
		logger.ErrorContext(ctx, err)
		http.Error(w, err, http.StatusInternalServerError)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		nonce, err := endpointer.WriteBrowserSecurityHeaders(w, serveCSP, serveCSP)
		if err != nil {
			herr(ctx, w, "write browser security headers err: "+err.Error())
			return
		}

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
			herr(ctx, w, "load react template: "+err.Error())
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			herr(ctx, w, "read bindata file: "+err.Error())
			return
		}
		t, err := template.New("react").Parse(string(data))
		if err != nil {
			herr(ctx, w, "create react template: "+err.Error())
			return
		}
		serverType := "on-premise"
		if sandbox {
			serverType = "sandbox"
		}
		if err := t.Execute(w, struct {
			URLPrefix  string
			ServerType string
			CSPNonce   string
		}{
			URLPrefix:  urlPrefix,
			ServerType: serverType,
			CSPNonce:   nonce,
		}); err != nil {
			herr(ctx, w, "execute react template: "+err.Error())
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
	logger *slog.Logger,
	serveCSP bool,
) http.Handler {
	herr := func(ctx context.Context, w http.ResponseWriter, err string) {
		logger.ErrorContext(ctx, err)
		http.Error(w, err, http.StatusInternalServerError)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce, err := endpointer.WriteBrowserSecurityHeaders(w, serveCSP, serveCSP)
		if err != nil {
			herr(r.Context(), w, "write browser security headers err: "+err.Error())
			return
		}
		ctx := r.Context()
		setupRequired, err := svc.SetupRequired(ctx)
		if err != nil {
			herr(ctx, w, "setup required err: "+err.Error())
			return
		}
		if setupRequired {
			herr(ctx, w, "fleet instance not setup")
			return
		}

		appCfg, err := ds.AppConfig(r.Context())
		if err != nil {
			herr(ctx, w, "load appconfig err: "+err.Error())
			return
		}

		errorMsg := r.URL.Query().Get("error")
		if errorMsg != "" {
			if err := renderEnrollPage(w, appCfg, urlPrefix, "", errorMsg, nonce, ""); err != nil {
				herr(ctx, w, err.Error())
			}
			return
		}

		enrollSecret := r.URL.Query().Get("enroll_secret")
		if enrollSecret == "" {
			if err := renderEnrollPage(w, appCfg, urlPrefix, "", "This URL is invalid. : Enroll secret is invalid. Please contact your IT admin.", nonce, ""); err != nil {
				herr(ctx, w, err.Error())
			}
			return
		}

		authRequired, err := shared_mdm.RequiresEnrollOTAAuthentication(r.Context(), ds,
			enrollSecret, appCfg.MDM.MacOSSetup.EnableEndUserAuthentication)
		if err != nil {
			herr(ctx, w, "check if authentication is required err: "+err.Error())
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
					herr(ctx, w, "initiate IdP SSO authentication err: "+err.Error())
					return
				}
				return
			}
		}

		// if we get here, IdP SSO authentication is either not required, or has
		// been successfully completed (we have a cookie with the IdP account
		// reference).

		// Clear the BYOD IdP cookie now that we are about to render the enrollment page.
		var idpUUID string
		fullyManaged := r.URL.Query().Get("fully_managed")
		if authRequired && (fullyManaged == "true" || fullyManaged == "1") {
			idpUUID = r.URL.Query().Get("enrollment_reference")
			http.SetCookie(w, &http.Cookie{
				Name:     shared_mdm.BYODIdpCookieName,
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				Secure:   cookieSecure,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		if err := renderEnrollPage(w, appCfg, urlPrefix, enrollSecret, "", nonce, idpUUID); err != nil {
			herr(ctx, w, err.Error())
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

func renderEnrollPage(w io.Writer, appCfg *fleet.AppConfig, urlPrefix, enrollSecret, errorMessage, nonce, idpUUID string) error {
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
		CSPNonce              string
		IdpUUID               string
	}{
		URLPrefix:             urlPrefix,
		EnrollURL:             enrollURL,
		ErrorMessage:          errorMessage,
		AndroidMDMEnabled:     appCfg.MDM.AndroidEnabledAndConfigured,
		MacMDMEnabled:         appCfg.MDM.EnabledAndConfigured,
		AndroidFeatureEnabled: true,
		CSPNonce:              nonce,
		IdpUUID:               idpUUID,
	}); err != nil {
		return fmt.Errorf("execute react template: %w", err)
	}
	return nil
}

func initiateOTAEnrollSSO(svc fleet.Service, w http.ResponseWriter, r *http.Request, enrollSecret string) error {
	requestURL := "/enroll?enroll_secret=" + url.QueryEscape(enrollSecret)
	// pass the fully_managed parameter for Android enrollments so that it is returned after the callback, else the
	// user won't get the android fully managed page
	if r.URL.Query().Get("fully_managed") == "true" {
		requestURL += "&fully_managed=true"
	}
	// Same with the byod modifier parameter for Apple enrollments
	if r.URL.Query().Get("byod") == "true" {
		requestURL += "&byod=true"
	}
	ssnID, ssnDurationSecs, idpURL, err := svc.InitiateMDMSSO(r.Context(), fleet.SSOInitiatorOTAEnroll, requestURL, "")
	if err != nil {
		return err
	}
	setSSOCookie(w, ssnID, ssnDurationSecs)
	http.Redirect(w, r, idpURL, http.StatusSeeOther)
	return nil
}

// hashedAssetRe matches build-output filenames that embed a content hash, e.g.
// "bundle-3ccf015bc0fac64b4ce8.js" or "logo@1a2b3c4d.png". A content change
// produces a new hash and therefore a new URL, so these are safe to cache
// forever. Unhashed names (dev builds like "bundle.js") must keep revalidating.
var hashedAssetRe = regexp.MustCompile(`[-@][0-9a-f]{8,}\.[a-z0-9]+$`)

func ServeStaticAssets(path string, serveCSP bool) http.Handler {
	contentTypes := []string{"text/javascript", "text/css"}
	staticAssetsServer := endpointer.BrowserSecurityHeadersHandler(serveCSP, http.FileServer(newBinaryFileSystem("/assets")))
	withoutGzip := http.StripPrefix(path, assetCacheControl(staticAssetsServer))

	withOpts, err := gzhttp.NewWrapper(gzhttp.ContentTypes(contentTypes))
	if err != nil { // fall back to serving without gzip if serving with gzip somehow fails
		return withoutGzip
	}

	return withOpts(withoutGzip)
}

func assetCacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&cacheControlResponseWriter{ResponseWriter: w, path: r.URL.Path}, r)
	})
}

type cacheControlResponseWriter struct {
	http.ResponseWriter
	path        string
	wroteHeader bool
}

// WriteHeader decides Cache-Control from the final status so the long-lived
// immutable cache is applied only to successful responses for hashed assets.
// Caching a transient 404/500 would otherwise pin a broken asset at the browser/CDN.
func (w *cacheControlResponseWriter) WriteHeader(status int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if (status == http.StatusOK || status == http.StatusNotModified) && hashedAssetRe.MatchString(w.path) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *cacheControlResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *cacheControlResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
