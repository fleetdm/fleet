package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	calendartest "github.com/fleetdm/fleet/v4/ee/server/calendar/load_test"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	port := flag.Uint("port", 8083, "Port to listen on")
	dbFileName := flag.String("db", "./calendar.db", "SQLite db file name")
	flag.Parse()

	handler, err := calendartest.Configure(*dbFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer calendartest.Close()

	listenAddr := fmt.Sprintf(":%d", *port)
	errLogger := log.New(os.Stderr, "", log.LstdFlags)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ErrorLog:     errLogger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Start the HTTP server
	err = server.ListenAndServe()
	if err != nil {
		log.Print(err)
	}
}
