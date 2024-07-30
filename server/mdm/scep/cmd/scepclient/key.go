package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

const (
	rsaPrivateKeyPEMBlockType = "RSA PRIVATE KEY"
)

// create a new RSA private key
func newRSAKey(bits int) (*rsa.PrivateKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return private, nil
}

// load key if it exists or create a new one
func loadOrMakeKey(path string, rsaBits int) (*rsa.PrivateKey, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		if os.IsExist(err) {
			return loadKeyFromFile(path)
		}
		return nil, err
	}
	defer file.Close()

	// write key
	priv, err := newRSAKey(rsaBits)
	if err != nil {
		return nil, err
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	pemBlock := &pem.Block{
		Type:    rsaPrivateKeyPEMBlockType,
		Headers: nil,
		Bytes:   privBytes,
	}
	if err = pem.Encode(file, pemBlock); err != nil {
		return nil, err
	}
	return priv, nil
}

// load a PEM private key from disk
func loadKeyFromFile(path string) (*rsa.PrivateKey, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed")
	}
	if pemBlock.Type != rsaPrivateKeyPEMBlockType {
		return nil, errors.New("unmatched type or headers")
	}

	return x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
}
