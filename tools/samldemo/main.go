package main

import (
	"fmt"
	"net/http"
)

const addr = "localhost:9337"

func main() {
	idp := getIDP()
	mux := http.NewServeMux()

	// IdP
	// Essentially these endpoints would be added to the Fleet server.
	mux.HandleFunc("/metadata", idp.ServeMetadata)
	mux.HandleFunc("/sso", idp.ServeSSO)
	mux.HandleFunc("/ssowithidentifier", idp.ServeSSOWithIdentifier)
	mux.HandleFunc("/saml/acs", idp.ServeMFAResponse)

	mux.Handle("/img/", http.StripPrefix("/img", http.FileServer(http.Dir("./img"))))

	// SP (used for MFA)
	m := NewSPMiddleware()
	mux.Handle("/", m)

	mux.Handle("/testsp", m.RequireAccount(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	// fmt.Println(hostsForUser("zach@fleetdm.com"))
	fmt.Println("serving")
	panic(http.ListenAndServe(addr, mux))
}
