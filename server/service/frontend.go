package service

import (
	"html/template"
	"io/ioutil"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

func newBinaryFileSystem(root string) *assetfs.AssetFS {
	return &assetfs.AssetFS{
		Asset:     Asset,
		AssetDir:  AssetDir,
		AssetInfo: AssetInfo,
		Prefix:    root,
	}
}

func ServeFrontend() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs := newBinaryFileSystem("/frontend")
		file, err := fs.Open("templates/react.tmpl")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t, err := template.New("react").Parse(string(data))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func ServeStaticAssets(path string) http.Handler {
	return http.StripPrefix(path, http.FileServer(newBinaryFileSystem("/assets")))
}
