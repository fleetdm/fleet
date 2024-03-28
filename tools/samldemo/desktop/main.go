package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	// When integrated in Fleet desktop, this would actually listen on a localhost port. We would
	// probably want to pick a set of localhost addresses to try binding to, in cases where some
	// ports are already used on the system. The client should then be prepared to fallback to those
	// other ports.
	addr = "localhost:9339"

	// Once integrated this should be loaded from environment (FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH) like in Fleet Desktop
	identifierPath = "/opt/orbit/identifier"
)

type identifierResponse struct {
	Err        string `json:"err,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

func main() {
	mux := http.NewServeMux()

	// This code would be added to Fleet Desktop
	mux.HandleFunc("/identifier", func(w http.ResponseWriter, r *http.Request) {
		// Once the IdP is integrated into the Fleet server, this should be set to the URL of the
		// Fleet server (which is set in the Desktop environment via FLEET_DESKTOP_FLEET_URL)
		w.Header().Set("Access-Control-Allow-Origin", "*")

		var response identifierResponse
		identifier, err := ioutil.ReadFile(identifierPath)
		if err != nil {
			response.Err = "no identifier found at " + identifierPath
		} else {
			response.Identifier = string(identifier)
		}
		bytes, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "failed to marshal json", http.StatusInternalServerError)
		}

		w.Write(bytes)
	})

	panic(http.ListenAndServe(addr, mux))
}
