# MDM Push CSR generation tool

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
`CSR_BASE64` - Base64 encoded CSR submitted from the Fleet server or `fleetctl` on behalf of the user. (Note: this is
accepted as an environment variable to mitigate against command injection attacks from untrusted user input.)

The program outputs the email and org from the signing request, and the signed request as JSON. For example:

```json
{"email":"fleetuser@example.com","org":"ExampleOrg","request":"PD94bWw..."}
```

The email should be validated against the email denylist, and then the request contents should be
sent to that email address as an attachment (eg. `apple-apns-request.txt`).
