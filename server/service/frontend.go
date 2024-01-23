package service

import (
	"fmt"
	"html/template"
	"io"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/go-kit/kit/log"
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
		if r.Method == http.MethodPost {
			// log the request details if a post is made to the root route
			details := []interface{}{"err", "invalid method", "uri", r.RequestURI, "method", r.Method, "proto", r.Proto, "remote", r.RemoteAddr, "host", r.Host, "user-agent", r.UserAgent(), "referer", r.Referer()}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Log("err", "read request body: "+err.Error())
			} else {
				details = append(details, "body", string(body))
				for k, v := range r.Header {
					details = append(details, k, fmt.Sprintf("%v", v))
				}
				logger.Log(details...)
			}
		}
		writeBrowserSecurityHeaders(w)
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

func ServeStaticAssets(path string) http.Handler {
	return http.StripPrefix(path, http.FileServer(newBinaryFileSystem("/assets")))
}
