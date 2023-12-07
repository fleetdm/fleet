package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// Code forked from https://github.com/oscartbeaumont/windows_mdm

var domain string
var deepLinkUserEmail string
var authPolicy string
var profileDir string
var staticDir string
var verbose bool

func main() {
	fmt.Println("Starting Windows MDM Demo Server")

	// Parse CMD flags. This populates the varibles defined above
	flag.StringVar(&domain, "domain", "mdmwindows.com", "Your servers primary domain")
	flag.StringVar(&deepLinkUserEmail, "dl-user-email", "demo@mdmwindows.com", "An email of the enrolling user when using the Deeplink ('/deeplink')")
	flag.StringVar(&authPolicy, "auth-policy", "Federated", "An email of the enrolling user when using the Deeplink ('/deeplink')")
	flag.StringVar(&profileDir, "mdm-profile-dir", "./profile", "The MDM policy directory contains the SyncML MDM profile commmands to enforce to enrolled devices")
	flag.StringVar(&staticDir, "static-dir", "./static", "The directory to serve static files")
	flag.BoolVar(&verbose, "verbose", true, "HTTP traffic dump")
	flag.Parse()

	// Verify authPolicy is valid
	if authPolicy != "Federated" && authPolicy != "OnPremise" {
		panic("unsupported authpolicy")
	}

	// Checking if profile directory exists
	_, err := os.Stat(profileDir)
	if err != nil {
		if os.IsNotExist(err) {
			panic("profile directory does not exists")
		} else {
			panic(err)
		}
	}

	// Checking if static directory exists
	_, err = os.Stat(staticDir)
	if err != nil {
		if os.IsNotExist(err) {
			panic("static directory does not exists")
		} else {
			panic(err)
		}
	}

	// Create HTTP request router
	r := mux.NewRouter()

	//MS-MDE and MS-MDM endpoints
	r.Path("/EnrollmentServer/Discovery.svc").Methods("GET", "POST", "PUT").HandlerFunc(DiscoveryHandler)
	r.Path("/EnrollmentServer/Policy.svc").Methods("POST").HandlerFunc(PolicyHandler)
	r.Path("/EnrollmentServer/Enrollment.svc").Methods("POST").HandlerFunc(EnrollHandler)
	r.Path("/ManagementServer/MDM.svc").Methods("POST").HandlerFunc(ManageHandler)
	r.Path("/EnrollmentServer/Auth.svc").Methods("GET", "POST").HandlerFunc(TokenHandler)
	r.Path("/TOS.svc").Methods("GET", "POST").HandlerFunc(TOSHandler)

	//Static root endpoint
	r.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(`<center><h1>Windows MDM Demo Server<br></h1>.<center>`))
	})

	//Static file serve
	fileServer := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/").Handler(http.StripPrefix("/static", fileServer))

	// Start HTTPS Server
	fmt.Println("HTTPS server listening on port 443")
	err = http.ListenAndServeTLS(":443", "./certs/dev_cert_mdmwindows_com_cert.pem", "./certs/dev_cert_mdmwindows_com.key", globalHandler(r))
	if err != nil {
		panic(err)
	}
}
