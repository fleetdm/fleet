package main

import (
	"archive/zip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/micromdm/micromdm/pkg/crypto/mdmcertutil"
)

const (
	vendorCertEnvName = "VENDOR_CERT_PEM"
	vendorKeyEnvName  = "VENDOR_KEY_PEM"
	vendorPassEnvName = "VENDOR_KEY_PASSPHRASE"

	keySize                   = 2048
	rsaPrivateKeyPEMBlockType = "RSA PRIVATE KEY"
)

func main() {
	vendorCert := os.Getenv(vendorCertEnvName)
	if vendorCert == "" {
		log.Fatalf("vendor cert must be set in %s", vendorCertEnvName)
	}
	vendorKey := os.Getenv(vendorKeyEnvName)
	if vendorKey == "" {
		log.Fatalf("vendor key must be set in %s", vendorKeyEnvName)
	}
	vendorKeyPassphrase := os.Getenv(vendorPassEnvName)
	if vendorKeyPassphrase == "" {
		log.Fatalf("vendor key passphrase must be set in %s", vendorPassEnvName)
	}

	email := flag.String("email", "", "Customer email for generated certificate")
	country := flag.String("country", "US", "Country for generated certificate")
	outfile := flag.String("out", "", "Path to output file")

	flag.Parse()

	if *email == "" {
		log.Fatalf("--email must be provided")
	}
	if *outfile == "" {
		log.Fatalf("--out must be provided")
	}

	archive, err := os.Create(*outfile)
	if err != nil {
		log.Fatalf("open file for output: %v", err)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	// Private key
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		log.Fatalf("generate rsa key: %v", err)
	}
	w, err := zipWriter.Create("key.pem")
	if err != nil {
		log.Fatalf("create key.pem: %v", err)
	}
	privPem := &pem.Block{
		Type:  rsaPrivateKeyPEMBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	if err := pem.Encode(w, privPem); err != nil {
		log.Fatalf("write key.pem: %v", err)
	}

	// Generate Push CSR
	cname := fmt.Sprintf("Fleet MDM (%s)", *email)
	derBytes, err := mdmcertutil.NewCSR(key, *email, *country, cname)
	if err != nil {
		log.Fatalf("generate csr: %v", err)
	}
	pushCSR := mdmcertutil.PemCSR(derBytes)

	// Sign Push CSR (with Fleet's vendor certificate)
	req, err := signCSR(vendorCert, vendorKey, vendorKeyPassphrase, pushCSR)
	if err != nil {
		log.Fatalf("sign csr: %v", err)
	}
	encodedReq, err := req.Encode()
	if err != nil {
		log.Fatalf("encode csr: %v", err)
	}
	w, err = zipWriter.Create("push.csr")
	if err != nil {
		log.Fatalf("create push.csr: %v", err)
	}
	if _, err := w.Write(encodedReq); err != nil {
		log.Fatalf("write push.csr: %v", err)
	}

	zipWriter.Close()
}

func signCSR(vendorCert, vendorKey, vendorKeyPass string, pushCSR []byte) (*mdmcertutil.PushCertificateRequest, error) {
	workdir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("create workdir: %w", err)
	}
	defer os.RemoveAll(workdir)

	// Set up files as expected by mdmcertutil.CreatePushCertificateRequest
	pKeyPath := filepath.Join(workdir, "vendor.key")
	if err := ioutil.WriteFile(pKeyPath, []byte(vendorKey), 0600); err != nil {
		return nil, fmt.Errorf("write priv.key: %w", err)
	}

	// Convert vendor cert PEM to DER
	block, _ := pem.Decode([]byte(vendorCert))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}
	mdmCertPath := filepath.Join(workdir, "vendor.cert")
	if err := ioutil.WriteFile(mdmCertPath, []byte(cert.Raw), 0600); err != nil {
		return nil, fmt.Errorf("write mdm.cert: %w", err)
	}

	pushCSRPath := filepath.Join(workdir, "push.csr")
	if err := ioutil.WriteFile(pushCSRPath, []byte(pushCSR), 0600); err != nil {
		return nil, fmt.Errorf("write push.csr: %w", err)
	}

	req, err := mdmcertutil.CreatePushCertificateRequest(mdmCertPath, pushCSRPath, pKeyPath, []byte(vendorKeyPass))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return req, nil
}
