package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/config"
)

func main() {
	flagCert := flag.String("cert", "", "The certificate to inspect (and optionally validate)")
	flagKey := flag.String("key", "", "The private key associated with the certificate (required for validation)")
	flagValidate := flag.Bool("validate", false, "Validate the certificate, including client authentication to the Apple sandbox")

	flag.Usage = func() {
		fmt.Println(`usage: <cmd> -cert CERTFILE
  Inspects the certificate by printing its parsed value.

usage: <cmd> -cert CERTFILE -key KEYFILE -validate
  Validates the certificate and private key, including connecting to the Apple
  sandbox using client authentication.`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *flagCert == "" {
		log.Fatal("certificate file must be specified")
	}
	if *flagValidate && *flagKey == "" {
		log.Fatal("validation requires a private key")
	}

	if *flagValidate {
		validate(*flagCert, *flagKey)
	} else {
		inspect(*flagCert)
	}

}

func validate(certFile, keyFile string) {
	mdmCfg := config.MDMConfig{
		AppleAPNsCert: certFile,
		AppleAPNsKey:  keyFile,
	}

	cert, _, _, err := mdmCfg.AppleAPNs()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := certificate.ValidateClientAuthTLSConnection(ctx, cert, "https://api.sandbox.push.apple.com"); err != nil {
		log.Fatal(err)
	}
	cancel()
}

func inspect(certFile string) {
	b, err := os.ReadFile(certFile)
	if err != nil {
		log.Fatal(err)
	}

	block, _ := pem.Decode(b)
	if block == nil || block.Type != "CERTIFICATE" {
		log.Fatal("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}
	spew.Dump(cert)
}
