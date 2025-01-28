package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags)

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("failed to read body: %s", err)
			return
		}

		var v interface{}
		if err := json.Unmarshal(body, &v); err != nil {
			log.Printf("failed to parse JSON body: %s", err)
			return
		}
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			panic(err)
		}
		log.Printf("%s", b)

		w.WriteHeader(http.StatusOK)
	}))
	//nolint:gosec // G114: file server used for testing purposes only.
	err := http.ListenAndServe("0.0.0.0:"+os.Args[1], nil)
	if err != nil {
		panic(err)
	}
}
