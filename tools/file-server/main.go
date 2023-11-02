package main

import (
	"net/http"
	"os"
)

func main() {
	fs := http.FileServer(http.FS(os.DirFS(os.Args[2])))
	http.Handle("/", fs)
	//nolint:gosec // G114: file server used for testing purposes only.
	err := http.ListenAndServe("0.0.0.0:"+os.Args[1], nil)
	if err != nil {
		panic(err)
	}
}
