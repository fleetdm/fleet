package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
)

const (
	defaultCN   = "deptokens"
	defaultDays = 1
)

// overridden by -ldflags -X
var version = "unknown"

func main() {
	var (
		flCert     = flag.String("cert", "cert.pem", "path to certificate")
		flKey      = flag.String("key", "cert.key", "path to key")
		flPassword = flag.String("password", "", "password to encrypt/decrypt private key with")
		flTokens   = flag.String("token", "", "path to tokens")
		flForce    = flag.Bool("f", false, "force overwriting the keypair")
		flVersion  = flag.Bool("version", false, "print version")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	var err error
	if *flTokens == "" {
		if *flPassword == "" {
			fmt.Println("WARNING: no password provided, private key will be saved in clear text")
		}
		err = generateKeyPair(*flCert, *flKey, *flPassword, *flForce)
		if err == nil {
			fmt.Printf("wrote %s, %s\n", *flCert, *flKey)
		}
	} else {
		var jsonBytes []byte
		jsonBytes, err = decryptTokens(*flTokens, *flCert, *flKey, *flPassword)
		if err == nil {
			os.Stdout.Write(jsonBytes)
		}
	}
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// encodeEncryptedKeyPEM generates a PEM structure for key optionally
// encrypting it with password.
func encodeEncryptedKeyPEM(key *rsa.PrivateKey, password string) ([]byte, error) {
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	var block *pem.Block
	if password == "" {
		block = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		}
	} else {
		var err error
		block, err = x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", keyBytes, []byte(password), x509.PEMCipher3DES)
		if err != nil {
			return nil, err
		}
	}
	return pem.EncodeToMemory(block), nil
}

// decodeEncryptedKeyPEM decodes an private key in pemBytes optionally
// decrypting it with password.
func decodeEncryptedKeyPEM(pemBytes []byte, password string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("PEM type is not RSA PRIVATE KEY")
	}
	keyBytes := block.Bytes
	if x509.IsEncryptedPEMBlock(block) {
		if password == "" {
			return nil, errors.New("no password supplied for encrypted PEM")
		}
		var err error
		keyBytes, err = x509.DecryptPEMBlock(block, []byte(password))
		if err != nil {
			return nil, err
		}
	}
	return x509.ParsePKCS1PrivateKey(keyBytes)
}

// generateKeyPair creates and saves a keypair checking whether they exist first.
func generateKeyPair(certFile, keyFile, password string, force bool) error {
	if !force {
		_, err := os.Stat(certFile)
		certExists := err == nil
		_, err = os.Stat(keyFile)
		keyExists := err == nil
		if keyExists || certExists {
			return errors.New("cert or key already exist, not overwriting")
		}
	}
	key, cert, err := tokenpki.SelfSignedRSAKeypair(defaultCN, defaultDays)
	if err != nil {
		return fmt.Errorf("generating keypair: %w", err)
	}
	err = os.WriteFile(certFile, tokenpki.PEMCertificate(cert.Raw), 0644)
	if err != nil {
		return fmt.Errorf("writing cert: %w", err)
	}
	keyPEM, err := encodeEncryptedKeyPEM(key, password)
	if err == nil {
		err = os.WriteFile(keyFile, keyPEM, 0600)
	}
	if err != nil {
		return fmt.Errorf("writing key: %w", err)
	}
	return nil
}

// decryptTokens reads tokenFile from disk and decrypts it using certFile and keyfile (with optional password).
func decryptTokens(tokenFile, certFile, keyFile, password string) ([]byte, error) {
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	key, err := decodeEncryptedKeyPEM(keyBytes, password)
	if err != nil {
		return nil, err
	}
	tokenBytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	certBytes, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	cert, err := tokenpki.CertificateFromPEM(certBytes)
	if err != nil {
		return nil, err
	}
	return tokenpki.DecryptTokenJSON(tokenBytes, cert, key)
}
