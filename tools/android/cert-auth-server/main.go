package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	caCertPath := flag.String("ca-cert", "ca.crt", "Path to CA certificate used to verify client certificates")
	addr := flag.String("addr", ":8443", "Address to listen on")
	flag.Parse()

	// --- Load CA for validating client certificates ---
	caCertBytes, err := os.ReadFile(*caCertPath)
	if err != nil {
		log.Fatalf("Error reading CA certificate (%s): %v", *caCertPath, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertBytes) {
		log.Fatalf("Failed to append CA certificate at %s", *caCertPath)
	}

	// --- Always generate a self-signed server certificate ---
	serverCert, err := generateSelfSignedCert([]string{"localhost", "127.0.0.1"})
	if err != nil {
		log.Fatalf("Error generating self-signed certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}

	http.HandleFunc("/", mtlsHandler)

	srv := &http.Server{
		Addr:              *addr,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("mTLS server listening at https://%s (self-signed server cert, client CA=%s)", *addr, *caCertPath)
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func mtlsHandler(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "Client certificate required", http.StatusUnauthorized)
		return
	}

	client := r.TLS.PeerCertificates[0]

	resp := fmt.Sprintf(`
<h2>mTLS Authentication Successful</h2>
<p><b>Client CN:</b> %s</p>
<p><b>Issuer:</b> %s</p>
<p><b>Subject:</b> %s</p>
`,
		client.Subject.CommonName,
		client.Issuer.String(),
		client.Subject.String(),
	)

	w.Header().Set("Content-Type", "text/html")
	_, err := io.WriteString(w, resp)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func generateSelfSignedCert(hosts []string) (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate rsa key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "mtls-test-server",
		},
		NotBefore: time.Now().Add(-1 * time.Minute),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return tls.X509KeyPair(certPEM, keyPEM)
}
