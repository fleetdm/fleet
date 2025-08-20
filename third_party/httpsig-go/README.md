# HTTP Message Signatures

[![Go Reference](https://pkg.go.dev/badge/github.com/remitly-oss/httpsig-go.svg)](https://pkg.go.dev/github.com/remitly-oss/httpsig-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/remitly-oss/httpsig-go)](https://goreportcard.com/report/github.com/remitly-oss/httpsig-go)

An implementation of HTTP Message Signatures from [RFC 9421](https://datatracker.ietf.org/doc/rfc9421/).

HTTP signatures are a mechanism for signing and verifying HTTP requests and responses.

HTTP signatures can be (or will be able to) used for demonstrating proof-of-posession ([DPoP](https://www.rfc-editor.org/rfc/rfc9449.html)) for [OAuth](https://oauth.net/2/dpop/) bearer tokens.

## Supported Features
The full specification is supported with the exception of the following. File a ticket or PR and support will be added
Planned but not currently supported features:
- JWS algorithms
- Header parameters including trailers

## net/http integration
Create net/http clients that sign requests and/or verifies repsonses.
```go
	params := httpsig.SigningOptions{
		PrivateKey: nil, // Fill in your private key
		Algorithm:  httpsig.Algo_ECDSA_P256_SHA256,
		Fields:     httpsig.DefaultRequiredFields,
		Metadata:   []httpsig.Metadata{httpsig.MetaKeyID},
		MetaKeyID:  "key123",
	}

	// Create the signature signer
	signer, _ := httpsig.NewSigner(params)

	// Create a net/http Client that signs all requests
	signingClient := httpsig.NewHTTPClient(nil, signer, nil)
```

Create net/http Handlers that verify incoming requests to the server.
```go
	myhandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Lookup the results of verification
		if veriftyResult, ok := httpsig.GetVerifyResult(r.Context()); ok {
			keyid, _ := veriftyResult.KeyID()
			fmt.Fprintf(w, "Hello, %s", keyid)
		} else {
			fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
		}
	})

	// Create a verifier
	verifier, _ := httpsig.NewVerifier(nil, httpsig.DefaultVerifyProfile)

	mux := http.NewServeMux()
	// Wrap the handler with the a signature verification handler.
	mux.Handle("/", httpsig.NewHandler(myhandler, verifier))
```

## Stability
The v1.1+ release is stable and production ready. 

Please file issues and bugs in the github projects issue tracker.

## References

- [RFC 9421](https://datatracker.ietf.org/doc/rfc9421/)
- [OAuth support](https://oauth.net/http-signatures/)
- [Interactive UI](https://httpsig.org/)
