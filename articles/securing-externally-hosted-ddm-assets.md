# How to secure externally hosted DDM assets

Declarative device management (DDM) lets you define an asset once and reference it from many configurations. One such asset type is `com.apple.asset.data`.

```json
{
  "Type": "com.apple.asset.data",
  "Identifier": "com.fleet.asset.wifi-cert",
  "Payload": {
    "Reference": {
      "ContentType": "application/x-pkcs12",
      "DataURL": "https://assets.example.com/wifi-cert"
    }
  }
}
```

When a device processes this asset, it downloads the data from `DataURL` itself. That URL can live anywhere: a CDN, an S3 bucket behind a small service, or your own host. This is what "externally hosted assets" means. The data never passes through Fleet.

That raises a problem. If the asset holds something sensitive, like a certificate or a credential, an open URL is a liability. Anyone who learns the URL could fetch the file. Even worse, a device enrolled in a different organization's Fleet server should not be able to read your assets.

Apple solves this with the same mechanism it uses for the MDM protocol itself: the `Mdm-Signature` header. This guide explains how that header works and how your asset host can verify it, so only enrolled devices can download the data. It also covers mutual TLS (mTLS), an alternative that verifies the same identity certificate during the TLS handshake.

## How the device signs its request

Fleet's enrollment profile sets `SignMessage` to `true`. From then on, the device signs its requests with the identity certificate it received during enrollment. That certificate was issued by Fleet's built-in certificate authority (CA).

When the device requests an externally hosted asset, it attaches an `Mdm-Signature` header. The header is a base64-encoded [CMS](https://datatracker.ietf.org/doc/html/rfc5652) (PKCS #7) detached signature over the request body. The device's signing certificate is embedded in the signature. Because an asset download is a `GET`, the body is empty, so the signature covers empty content. The proof of identity comes from the certificate and the private key, not from the payload.

Apple documents this in ["Pass an identity certificate through a proxy."](https://developer.apple.com/documentation/devicemanagement/managing-certificates-for-device-management-services-and-devices#Pass-a-device-identity-certificate-through-a-proxy) Fleet uses the exact same header to authenticate every MDM check-in, so your asset host can reuse the same verification steps.

## What verification proves

Two checks confirm a request came from a device enrolled in your Fleet server, and both are stateless. Anyone with Fleet's CA certificate can run them:

1. **Did the holder of this certificate sign this request?** Verify the CMS signature.
2. **Did Fleet's CA issue this certificate?** Verify the certificate chains to Fleet's CA.

> The following code snippets are written in Go.

## Step 1: Verify the signature

Decode the header, attach the request body as the detached content, and verify. This example uses [`go.mozilla.org/pkcs7`](https://pkg.go.dev/go.mozilla.org/pkcs7), the same style of library Fleet uses internally.

```go
import (
	"crypto/x509"
	"encoding/base64"
	"errors"

	"go.mozilla.org/pkcs7"
)

// verifySignature checks the Mdm-Signature header against the request body and
// returns the certificate that signed it.
func verifySignature(header string, body []byte) (*x509.Certificate, error) {
	sig, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, err
	}

	// Reject oversized headers before parsing to limit abuse. A real signature
	// is a few kilobytes at most.
	if len(sig) > 10*1024 {
		return nil, errors.New("Mdm-Signature header too large")
	}

	p7, err := pkcs7.Parse(sig)
	if err != nil {
		return nil, err
	}

	// The signature is detached, so set the content to the request body.
	p7.Content = body
	if err := p7.Verify(); err != nil {
		return nil, err
	}

	cert := p7.GetOnlySigner()
	if cert == nil {
		return nil, errors.New("no signer certificate")
	}
	return cert, nil
}
```

At this point you know the request was signed by whoever holds the private key for `cert`. You do not yet know who that is.

## Step 2: Verify the certificate chains to Fleet's CA

A valid signature from an unknown certificate proves nothing. Anyone can generate a self-signed certificate and sign a request with it. The certificate has to trace back to your Fleet server's CA.

```go
import (
	"crypto/x509"
	"time"
)

// verifyChain confirms the certificate was issued by Fleet's CA and is valid
// for client authentication.
func verifyChain(cert *x509.Certificate, fleetCA *x509.Certificate) error {
	roots := x509.NewCertPool()
	roots.AddCert(fleetCA)

	_, err := cert.Verify(x509.VerifyOptions{
		Roots:       roots,
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		CurrentTime: time.Now(),
	})
	return err
}
```

`cert.Verify` also enforces the certificate's validity window, so an expired identity fails here.

Each Fleet server generates its own CA when MDM is turned on. A certificate issued by a different Fleet server, or by any other CA, will not chain to yours. This is what keeps other organizations' devices out.

## Getting Fleet's CA certificate

Your asset host needs Fleet's CA certificate to run step 2, and to trust client certificates over mTLS. Fleet exposes it over the standard SCEP endpoint. Fetch it once and cache it:

```go
import (
	"crypto/x509"
	"io"
	"net/http"
)

// fetchFleetCA downloads Fleet's CA certificate from the SCEP endpoint.
func fetchFleetCA(fleetURL string) (*x509.Certificate, error) {
	resp, err := http.Get(fleetURL + "/mdm/apple/scep?operation=GetCACert")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	der, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(der)
}
```

You can inspect the same certificate from the command line, which is handy for debugging:

```bash
curl 'https://fleet.example.com/mdm/apple/scep?operation=GetCACert' -o ca.der
openssl x509 -inform DER -in ca.der -noout -subject -issuer
```

## Putting it together

Your asset handler runs the two checks in order and serves the file only when both pass:

```go
func handleAssetDownload(w http.ResponseWriter, r *http.Request, fleetCA *x509.Certificate) {
	body, _ := io.ReadAll(r.Body)

	cert, err := verifySignature(r.Header.Get("Mdm-Signature"), body)
	if err != nil {
		http.Error(w, "bad signature", http.StatusBadRequest)
		return
	}
	if err := verifyChain(cert, fleetCA); err != nil {
		http.Error(w, "untrusted certificate", http.StatusForbidden)
		return
	}

	// Both checks passed: the request came from a device enrolled in this Fleet.
	serveAsset(w, r)
}
```

## What each check protects against

- A request with no signature, or a forged one, fails step 1.
- A device whose certificate came from a different CA, including another Fleet server, fails step 2.

## Alternative: verify with mutual TLS (mTLS)

The device presents its Fleet identity certificate two ways on the same request. It signs the body for the `Mdm-Signature` header, and it also offers the certificate as a TLS client certificate during the handshake. If your asset host terminates TLS itself, you can verify that client certificate instead of reading the header, and let the TLS layer reject unauthorized clients before any request reaches your code.

Point the server's client CA pool at Fleet's CA and require a verified client certificate:

```go
import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
)

// newTLSServer completes the handshake only for clients that present a
// certificate chaining to Fleet's CA.
func newTLSServer(fleetCA *x509.Certificate) *http.Server {
	pool := x509.NewCertPool()
	pool.AddCert(fleetCA)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The handshake already required a client certificate that chains to
		// Fleet's CA, so any request that reaches here is from an enrolled
		// device. The verified certificate is on r.TLS.PeerCertificates[0] if
		// you want to log which device it was.
		serveAsset(w, r)
	})

	return &http.Server{
		Addr:    ":443",
		Handler: handler,
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  pool,
			MinVersion: tls.VersionTLS12,
		},
	}
}
```

`RequireAndVerifyClientCert` with `ClientCAs` set to Fleet's CA is the same trust check as step 2, moved into the handshake. Go verifies the chain and the certificate's validity window before your handler runs, so a client with no certificate, or one from another CA, is turned away at the connection.

mTLS has two advantages over the header. It rejects unauthorized clients at the handshake, before any HTTP is processed, and because each connection is a fresh handshake, a captured request cannot be replayed. The condition is that your server must terminate TLS. If a proxy or CDN terminates TLS in front of your host, the client certificate never reaches your code, and the `Mdm-Signature` header is the option that still works. Some proxies can forward the certificate in a header, but that depends on the proxy.

## Conclusion

Externally hosted assets let you serve DDM asset data from wherever suits your infrastructure without routing it through Fleet. The `Mdm-Signature` header keeps that data protected: verify the signature, then confirm the certificate chains to Fleet's CA. Those two checks prove a request came from one of your enrolled devices, so unauthorized requests and devices from other Fleet servers cannot reach what you host. When your host terminates TLS, mTLS verifies the same identity certificate at the handshake and gives you that protection one layer earlier.

<meta name="articleTitle" value="How to secure externally hosted DDM assets">
<meta name="authorFullName" value="Magnus Jensen">
<meta name="authorGitHubUsername" value="MagnusHJensen">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-07-20">
<meta name="description" value="A technical guide to securing externally hosted DDM assets by verifying the Apple MDM-Signature header, or mutual TLS (mTLS), against Fleet's certificate authority.">