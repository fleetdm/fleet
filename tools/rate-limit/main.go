package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/uuid"
)

func main() {
	fleetURL := flag.String("fleet_url", "", "URL (with protocol and port of Fleet server)")
	fleetDesktopToken := flag.String("fleet_desktop_token", "", "Valid \"Fleet Desktop\" token")

	flag.Parse()

	if *fleetURL == "" {
		log.Fatal("missing fleet_url argument")
	}

	c := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		// Ignoring "G402: TLS InsecureSkipVerify set true", this is only used for automated testing.
		InsecureSkipVerify: true, //nolint:gosec
	}))

	start := time.Now()
	for i := 1; ; i++ {
		token := uuid.NewString()
		if *fleetDesktopToken != "" && (i%500 == 0) {
			log.Print("Attempting good token")
			token = *fleetDesktopToken
		}
		req, err := http.NewRequest("GET", *fleetURL+fmt.Sprintf("/api/latest/fleet/device/%s/desktop", token), nil)
		req.Header.Add("X-Forwarded-For", "127.0.0.1")
		if err != nil {
			panic(err)
		}
		res, err := c.Do(req)
		if err != nil {
			panic(err)
		}
		log.Printf(": %d: %d\n", i, res.StatusCode)
		if res.StatusCode == http.StatusTooManyRequests {
			log.Printf("Rate limited: %s\n", time.Since(start))
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
