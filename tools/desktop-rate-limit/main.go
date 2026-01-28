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

	endpoints := [][2]string{
		{"GET", "/api/latest/fleet/device/%s/desktop"},
		{"GET", "/api/latest/fleet/device/%s/transparency"},
		{"POST", "/api/latest/fleet/device/%s/refetch"},
	}

	start := time.Now()
	for i := 1; ; i++ {
		token := uuid.NewString()
		endpoint := endpoints[0]
		if *fleetDesktopToken != "" && (i%500 == 0) {
			log.Print("Attempting good token")
			token = *fleetDesktopToken
		} else {
			endpoint = endpoints[i%len(endpoints)]
		}
		req, err := http.NewRequest(endpoint[0], *fleetURL+fmt.Sprintf(endpoint[1], token), nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("X-Forwarded-For", "127.0.0.1")
		res, err := c.Do(req)
		if err != nil {
			panic(err)
		}
		_ = res.Body.Close()
		log.Printf("%d: %s %s: %d\n", i, req.Method, req.URL.Path, res.StatusCode)
		if res.StatusCode == http.StatusTooManyRequests {
			log.Printf("Rate limited: %s\n", time.Since(start))
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
