package main

import (
	"fmt"
	"net/http"
)

const addr = "localhost:9337"

func main() {
	idp := getIDP()
	mux := http.NewServeMux()

	// Essentially these two endpoints would be added to the Fleet server.
	mux.HandleFunc("/metadata", idp.ServeMetadata)
	mux.HandleFunc("/sso", idp.ServeSSO)
	mux.HandleFunc("/ssowithidentifier", idp.ServeSSOWithIdentifier)

	mux.Handle("/img/", http.StripPrefix("/img", http.FileServer(http.Dir("./img"))))

	fmt.Println(hostsForUser("zach@fleetdm.com"))
	panic(http.ListenAndServe(addr, mux))
}
