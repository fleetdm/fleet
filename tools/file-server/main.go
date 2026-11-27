package main

import (
	"net/http"
	"os"
)

func main() {
	fs := http.FileServer(http.FS(os.DirFS(os.Args[2])))
	// TUF target paths are reused across versions, so proxies in front of this server (e.g. a
	// Cloudflare tunnel, which caches .exe by default) must not serve stale targets.
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fs.ServeHTTP(w, r)
	}))
	//nolint:gosec // G114: file server used for testing purposes only.
	err := http.ListenAndServe("0.0.0.0:"+os.Args[1], nil)
	if err != nil {
		panic(err)
	}
}
