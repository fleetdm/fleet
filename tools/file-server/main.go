package main

import (
	"net/http"
	"os"
)

func main() {
	fs := http.FileServer(http.FS(os.DirFS(os.Args[2])))
	http.Handle("/", fs)
	err := http.ListenAndServe(":"+os.Args[1], nil)
	if err != nil {
		panic(err)
	}
}
