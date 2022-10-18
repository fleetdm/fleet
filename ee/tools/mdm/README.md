## MDM Push CSR generation tool

### Build

Build like any other go program:

``` sh
go build -o mdm-gen-cert .
```

### Usage

The following environment variables must be configured:

`VENDOR_CERT_PEM` - Fleet's MDM Vendor certificate in PEM format.
`VENDOR_KEY_PEM` - Fleet's MDM Vendor private key in PEM format.
`VENDOR_KEY_PASSPHRASE` - Passphrase for the MDM Vendor private key.

Run the binary to generate a zip file containing the Push certificate request and associated private key. The required arguments are `out` (path to output zip file), and `email` (the customer's email):

``` sh
./mdm-gen-cert --out out.zip --email user@example.com
```

After generation, the user should upload the certificate request to [identity.apple.com](https://identity.apple.com) and then configure Fleet with the private key and the certificate downloaded from Apple.
