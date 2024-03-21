package scepserver_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/gorilla/mux"

	kitlog "github.com/go-kit/kit/log"
)

func TestCACaps(t *testing.T) {
	server, _, teardown := newServer(t)
	defer teardown()
	url := server.URL + "/scep?operation=GetCACaps"
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("expected", http.StatusOK, "got", resp.StatusCode)
	}
}

func TestEncodePKCSReq_Request(t *testing.T) {
	pkcsreq := loadTestFile(t, "../scep/testdata/PKCSReq.der")
	msg := scepserver.SCEPRequest{
		Operation: "PKIOperation",
		Message:   pkcsreq,
	}
	methods := []string{"POST", "GET"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			r := httptest.NewRequest(method, "http://acme.co/scep", nil)
			rr := *r
			if err := scepserver.EncodeSCEPRequest(context.Background(), &rr, msg); err != nil {
				t.Fatal(err)
			}

			q := r.URL.Query()
			if have, want := q.Get("operation"), msg.Operation; have != want {
				t.Errorf("have %s, want %s", have, want)
			}

			if method == "POST" {
				if have, want := rr.ContentLength, int64(len(msg.Message)); have != want {
					t.Errorf("have %d, want %d", have, want)
				}
			}

			if method == "GET" {
				if q.Get("message") == "" {
					t.Errorf("expected GET PKIOperation to have a non-empty message field")
				}
			}
		})
	}
}

func TestGetCACertMessage(t *testing.T) {
	testMsg := "testMsg"
	sr := scepserver.SCEPRequest{Operation: "GetCACert", Message: []byte(testMsg)}
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080/scep", nil)
	if err != nil {
		t.Fatal(err)
	}
	err = scepserver.EncodeSCEPRequest(context.Background(), req, sr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(req.URL.RawQuery, "message="+testMsg) {
		t.Fatal("RawQuery does not contain message")
	}
}

func TestPKIOperation(t *testing.T) {
	server, _, teardown := newServer(t)
	defer teardown()
	pkcsreq := loadTestFile(t, "../scep/testdata/PKCSReq.der")
	body := bytes.NewReader(pkcsreq)
	url := server.URL + "/scep?operation=PKIOperation"
	resp, err := http.Post(url, "", body) //nolint:gosec
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("expected", http.StatusOK, "got", resp.StatusCode)
	}
}

func TestPKIOperationGET(t *testing.T) {
	server, _, teardown := newServer(t)
	defer teardown()
	pkcsreq := loadTestFile(t, "../scep/testdata/PKCSReq.der")
	message := base64.StdEncoding.EncodeToString(pkcsreq)
	req, err := http.NewRequest("GET", server.URL+"/scep", nil)
	if err != nil {
		t.Fatal(err)
	}
	params := req.URL.Query()
	params.Set("operation", "PKIOperation")
	params.Set("message", message)
	req.URL.RawQuery = params.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("expected", http.StatusOK, "got", resp.StatusCode)
	}
}

func TestInvalidReqs(t *testing.T) {
	server, _, teardown := newServer(t)
	defer teardown()
	// Check that invalid requests return status 400.
	req, err := http.NewRequest("GET", server.URL+"/scep", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Error("expected", http.StatusBadRequest, "got", resp.StatusCode)
	}

	req, err = http.NewRequest("GET", server.URL+"/scep", nil)
	if err != nil {
		t.Fatal(err)
	}

	params := req.URL.Query()
	params.Set("operation", "PKIOperation")
	params.Set("message", "")
	req.URL.RawQuery = params.Encode()

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Error("expected", http.StatusBadRequest, "got", resp.StatusCode)
	}

	params = req.URL.Query()
	params.Set("operation", "InvalidOperation")
	req.URL.RawQuery = params.Encode()

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Error("expected", http.StatusBadRequest, "got", resp.StatusCode)
	}

	postReq, err := http.NewRequest("POST", server.URL+"/scep", nil)
	if err != nil {
		t.Fatal(err)
	}

	params = req.URL.Query()
	params.Set("operation", "PKIOperation")
	req.URL.RawQuery = params.Encode()

	resp, err = http.DefaultClient.Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Error("expected", http.StatusBadRequest, "got", resp.StatusCode)
	}

	params = req.URL.Query()
	params.Set("operation", "InvalidOperation")
	req.URL.RawQuery = params.Encode()

	resp, err = http.DefaultClient.Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Error("expected", http.StatusBadRequest, "got", resp.StatusCode)
	}
}

func newServer(t *testing.T, opts ...scepserver.ServiceOption) (*httptest.Server, scepserver.Service, func()) {
	var err error
	var depot depot.Depot // cert storage
	{
		depot, err = filedepot.NewFileDepot("../scep/testdata/testca")
		if err != nil {
			t.Fatal(err)
		}
		depot = &noopDepot{depot}
	}
	crt, key, err := depot.CA([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	var svc scepserver.Service // scep service
	{
		svc, err = scepserver.NewService(crt[0], key, scepserver.NopCSRSigner())
		if err != nil {
			t.Fatal(err)
		}
	}
	logger := kitlog.NewNopLogger()
	e := scepserver.MakeServerEndpoints(svc)
	scepHandler := scepserver.MakeHTTPHandler(e, svc, logger)
	r := mux.NewRouter()
	r.Handle("/scep", scepHandler)
	server := httptest.NewServer(r)
	teardown := func() {
		server.Close()
		os.Remove("../scep/testdata/testca/serial")
		os.Remove("../scep/testdata/testca/index.txt")
	}
	return server, svc, teardown
}

type noopDepot struct{ depot.Depot }

func (d *noopDepot) Put(name string, crt *x509.Certificate) error {
	return nil
}

/* helpers */

func loadTestFile(t *testing.T, path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
