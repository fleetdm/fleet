package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"

	"github.com/go-xmlfmt/xmlfmt"
)

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

		shouldLog := strings.HasPrefix(r.URL.Path, "/EnrollmentServer") || strings.HasPrefix(r.URL.Path, "/ManagementServer")

		if !shouldLog {
			// Skip logging, call next handler
			h.ServeHTTP(w, r)
			return
		}

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

			responseBody := rec.Body.Bytes()

			var beautifiedResponseBody string

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
