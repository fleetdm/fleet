package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const DELAY = 10 * time.Second // adjust this to simulate slow webhook response

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

		time.Sleep(DELAY)
		if _, err := writer.Write(nil); err != nil {
			log.Printf("error writing response %s", err.Error())
		}
	})

	port := ":4648"
	if p := os.Getenv("SERVER_PORT"); p != "" {
		port = p
	}

	fmt.Printf("Server running at http://localhost%s\n", port)
	server := &http.Server{
		Addr:              port,
		ReadHeaderTimeout: 30 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("server error %s", err.Error())
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig // block on signal
}
