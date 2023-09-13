package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		var detail string
		body, err := io.ReadAll(request.Body)
		if err != nil {
			detail = fmt.Sprintf("| ERROR: reading request body: %s", err)
		} else if len(body) != 0 {
			detail = fmt.Sprintf("| BODY: %s", string(body))
		}
		log.Printf("%s %s %s\n", request.Method, request.URL.Path, detail)
		if err := request.Write(writer); err != nil {
			log.Fatalf(err.Error())
		}
	})

	port := ":4648"
	if p := os.Getenv("SERVER_PORT"); p != "" {
		port = p
	}

	fmt.Printf("Server running at http://localhost%s\n", port)
	if err := http.ListenAndServe(""+port, nil); err != nil {
		log.Fatalf(err.Error())
	}
}
