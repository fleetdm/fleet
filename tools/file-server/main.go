package main

import (
	"log"
	"net/http"
	"os"
)

func loggingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do a simple log of the path and method being hit
		log.Println(r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}

func main() {
	fs := loggingHandler(http.FileServer(http.FS(os.DirFS(os.Args[2]))))
	http.Handle("/", fs)
	//nolint:gosec // G114: file server used for testing purposes only.
	err := http.ListenAndServe("0.0.0.0:"+os.Args[1], nil)
	if err != nil {
		panic(err)
	}
}
