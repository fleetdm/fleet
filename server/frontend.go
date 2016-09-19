package server

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
)

type binaryFileSystem struct {
	fs *assetfs.AssetFS
}

func (b *binaryFileSystem) Open(name string) (http.File, error) {
	return b.fs.Open(name)
}

func (b *binaryFileSystem) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		if _, err := b.fs.Open(p); err != nil {
			return false
		}
		return true
	}
	return false
}

func newBinaryFileSystem(root string) *binaryFileSystem {
	return &binaryFileSystem{
		fs: &assetfs.AssetFS{
			Asset:     Asset,
			AssetDir:  AssetDir,
			AssetInfo: AssetInfo,
			Prefix:    root,
		},
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
