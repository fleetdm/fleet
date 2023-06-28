package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"

	"github.com/go-xmlfmt/xmlfmt"
	"github.com/gorilla/mux"
)

// Code forked from https://github.com/oscartbeaumont/windows_mdm
// Global config, populated via Command line flags
var (
	domain            string
	deepLinkUserEmail string
	authPolicy        string
	profileDir        string
	staticDir         string
	verbose           bool
)

func main() {
	fmt.Println("Starting Windows MDM Demo Server")

	// Parse CMD flags. This populates the varibles defined above
	flag.StringVar(&domain, "domain", "mdmwindows.com", "Your servers primary domain")
	flag.StringVar(&deepLinkUserEmail, "dl-user-email", "demo@mdmwindows.com", "An email of the enrolling user when using the Deeplink ('/deeplink')")
	flag.StringVar(&authPolicy, "auth-policy", "OnPremise", "An email of the enrolling user when using the Deeplink ('/deeplink')")
	flag.StringVar(&profileDir, "mdm-profile-dir", "./profile", "The MDM policy directory contains the SyncML MDM profile commmands to enforce to enrolled devices")
	flag.StringVar(&staticDir, "static-dir", "./static", "The directory to serve static files")
	flag.BoolVar(&verbose, "verbose", false, "HTTP traffic dump")
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

	// MS-MDE and MS-MDM endpoints
	r.Path("/EnrollmentServer/Discovery.svc").Methods("GET", "POST").HandlerFunc(DiscoveryHandler)
	r.Path("/EnrollmentServer/Policy.svc").Methods("POST").HandlerFunc(PolicyHandler)
	r.Path("/EnrollmentServer/Enrollment.svc").Methods("POST").HandlerFunc(EnrollHandler)
	r.Path("/ManagementServer/MDM.svc").Methods("POST").HandlerFunc(ManageHandler)

	// Static root endpoint
	r.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		w.Write([]byte(`<center><h1>FleetDM Windows MDM Demo Server<br></h1>.<center>`))
		w.Write([]byte(`<br><center><img src="https://fleetdm.com/images/press-kit/fleet-logo-dark-rgb.png"></center>`))
	})

	// Static file serve
	fileServer := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/").Handler(http.StripPrefix("/static", fileServer))

	// Start HTTPS Server
	fmt.Println("HTTPS server listening on port 443")
	err = http.ListenAndServeTLS(":443", "./certs/dev_cert_mdmwindows_com_cert.pem", "./certs/dev_cert_mdmwindows_com.key", globalHandler(r))
	if err != nil {
		panic(err)
	}
}

// drainBody reads all of bytes to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadClosers have identical error-matching behavior.
func drainBody(b io.ReadCloser) (r1, r2 io.ReadCloser, body []byte, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, http.NoBody, nil, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, nil, err
	}
	if err = b.Close(); err != nil {
		return nil, b, nil, err
	}
	return io.NopCloser(&buf), io.NopCloser(bytes.NewReader(buf.Bytes())), buf.Bytes(), nil
}

// global HTTP handler to log input and output https traffic
func globalHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if verbose {
			// grabbing Input Header and Body
			reqHeader, err := httputil.DumpRequest(r, false)
			if err != nil {
				panic(err)
			}

			var bodyBytes []byte
			reqBodySave := r.Body
			if r.Body != nil {
				reqBodySave, r.Body, bodyBytes, err = drainBody(r.Body)
				if err != nil {
					panic(err)
				}
			}
			r.Body = reqBodySave

			var beautifiedReqBody string
			if len(bodyBytes) > 0 {
				beautifiedReqBody = xmlfmt.FormatXML(string(bodyBytes), " ", "  ")
			}

			fmt.Printf("\n\n============================= Input Request =============================\n")
			fmt.Println("----------- Input Header -----------\n", string(reqHeader))
			if len(beautifiedReqBody) > 0 {
				fmt.Println("----------- Input Body -----------\n", string(beautifiedReqBody))
			} else {
				fmt.Printf("----------- Empty Input Body -----------\n")
			}
			fmt.Printf("=========================================================================\n\n\n")
		}

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)

		if verbose {
			// grabbing Output Header and Body
			var beautifiedResponseBody string
			responseBody := rec.Body.Bytes()
			if len(responseBody) > 0 {
				beautifiedResponseBody = xmlfmt.FormatXML(string(responseBody), " ", "  ")
			}

			responseHeader, err := httputil.DumpResponse(rec.Result(), false)
			if err != nil {
				panic(err)
			}

			fmt.Printf("\n\n============================= Output Response =============================\n")
			fmt.Println("----------- Response Header -----------\n", string(responseHeader))
			if len(beautifiedResponseBody) > 0 {
				fmt.Println("----------- Response Body -----------\n", string(beautifiedResponseBody))
			} else {
				fmt.Printf("----------- Empty Response Body -----------\n")
			}
			fmt.Printf("=========================================================================\n\n\n")
		}

		// we copy the captured response headers to our new response
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.Write(rec.Body.Bytes())
	})
}
