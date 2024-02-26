package main

import (
	"fmt"
	"net/http"
)

const addr = "localhost:9337"

func main() {
	idp := getIDP()
	mux := http.NewServeMux()

	mux.HandleFunc("/metadata", idp.ServeMetadata)
	mux.HandleFunc("/sso", idp.ServeSSO)
	fmt.Println(hostsForUser("zach@fleetdm.com"))
	panic(http.ListenAndServe(addr, mux))
}
