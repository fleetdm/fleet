package service

import (
	"html/template"
	"io/ioutil"
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

func ServeFrontend(urlPrefix string, logger log.Logger) http.Handler {
	herr := func(w http.ResponseWriter, err string) {
		logger.Log("err", err)
		http.Error(w, err, http.StatusInternalServerError)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs := newBinaryFileSystem("/frontend")
		file, err := fs.Open("templates/react.tmpl")
		if err != nil {
			herr(w, "load react template: "+err.Error())
			return
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			herr(w, "read bindata file: "+err.Error())
			return
		}
		t, err := template.New("react").Parse(string(data))
		if err != nil {
			herr(w, "create react template: "+err.Error())
			return
		}
		if err := t.Execute(w, struct{ URLPrefix string }{urlPrefix}); err != nil {
			herr(w, "execute react template: "+err.Error())
			return
		}
	})
}

func ServeStaticAssets(path string) http.Handler {
	return http.StripPrefix(path, http.FileServer(newBinaryFileSystem("/assets")))
}
