package app

import (
	"html/template"
	"net/http"
	"strings"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/contrib/renders/multitemplate"
)

type BinaryFileSystem struct {
	fs http.FileSystem
}

func (b *BinaryFileSystem) Open(name string) (http.File, error) {
	return b.fs.Open(name)
}

func (b *BinaryFileSystem) Exists(prefix string, filepath string) bool {

	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		if _, err := b.fs.Open(p); err != nil {
			return false
		}
		return true
	}
	return false
}

func NewBinaryFileSystem(root string) *BinaryFileSystem {
	return &BinaryFileSystem{
		fs: &assetfs.AssetFS{
			Asset:     Asset,
			AssetDir:  AssetDir,
			AssetInfo: AssetInfo,
			Prefix:    root,
		},
	}
}

func loadTemplates(list ...string) multitemplate.Render {
	r := multitemplate.New()
	for _, x := range list {
		templateString, err := Asset("frontend/templates/" + x)
		if err != nil {
			panic(err)
		}
		tmplMessage, err := template.New(x).Parse(string(templateString))
		if err != nil {
			panic(err)
		}
		r.Add(x, tmplMessage)
	}
	return r
}
