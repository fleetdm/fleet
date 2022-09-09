# MDMcert.download analysis

Here's how to generate an analysis of the APNs certificate using https://mdmcert.download.

## 1. `mdmctl mdmcert.download -new` command

Reference: https://github.com/micromdm/micromdm/blob/main/cmd/mdmctl/mdmcert.download.go

```sh
$ mdmctl mdmcert.download -new -email=alice@example.com
Request successfully sent to mdmcert.download. Your CSR should now
be signed. Check your email for next steps. Then use the -decrypt option
to extract the CSR request which will then be uploaded to Apple.
```
First, this command generates the following two files:
```sh
mdmcert.download.pki.key:  PEM RSA private key
mdmcert.download.pki.crt:  self-signed PEM certificate // "key usage": for encryption and signing
```
Second, it then generates two more files, a push private key and a push CSR:
```sh
mdmcert.download.push.key: PEM RSA private key
mdmcert.download.push.csr: PEM certificate request
```
Lastly, it sends the CSR to https://mdmcert.download:
```
return &signRequest{
	CSR:     encodedCSR, // <<<< mdmcert.download.push.csr
	Email:   email, // alice@example.com
	Key:     mdmcertAPIKey, 
	Encrypt: encodedServerCert, // <<<< mdmcert.download.pki.crt
}
```
The https://mdmcert.download service signs the provided `mdmcert.download.push.csr`, encrypts the CSR with public key in `mdmcert.download.pki.crt`, and
sends it via e-mail as `mdm_signed_request.20220812_125806_1308.plist.b64.p7` to `alice@example.com`.

## 2. `mdmctl mdmcert.download -decrypt` command

```sh
$ mdmctl mdmcert.download -decrypt=./mdm_signed_request.20220812_125806_1308.plist.b64.p7
Successfully able to decrypt the MDM push certificate request! Please upload
the file 'mdmcert.download.push.req' to Apple by visiting https://identity.apple.com
Once your push certificate is signed by Apple you can download it
and import it into MicroMDM using the `mdmctl mdmcert upload` command
```

This command generates an "MDM push certificate request" to be uploaded to Apple:
```
mdmcert.download.push.req: ASCII text, with CRLF line terminators
```

Here are the contents of such file (after base64 decoding):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PushCertCertificateChain</key>
	<string>-----BEGIN CERTIFICATE-----
[...]
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
[...]
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
[...]
-----END CERTIFICATE-----
</string>
		// This is basically `mdmcert.download.push.csr` without the `-----BEGIN CERTIFICATE REQUEST-----` prefix and `-----END CERTIFICATE REQUEST-----` suffix
		<key>PushCertRequestCSR</key>
		<string>
		MIIChjC[...]WQyvk=
		</string>
		// This is the signature of the above CSR signed by "MDM Vendor: McMurtrie Consulting LLC"
		<key>PushCertSignature</key>
		<string>w2UkUqj[...]3cLg==
</string>
</dict>
</plist>
```

The first certificate is of the form:
```
Certificate:
[...]
	Issuer: CN=Apple Worldwide Developer Relations Certification Authority, OU=G3, O=Apple Inc., C=US
[...]
	Subject: UID=..., CN=MDM Vendor: McMurtrie Consulting LLC, OU=..., O=McMurtrie Consulting LLC, C=US
```
The second certificate is of the form:
```
Certificate:
[...]
	Issuer: C=US, O=Apple Inc., OU=Apple Certification Authority, CN=Apple Root CA
[...]
	Subject: CN=Apple Worldwide Developer Relations Certification Authority, OU=G3, O=Apple Inc., C=US
```
The third certificate is of the form:
```
Certificate:
[...]
	Issuer: C=US, O=Apple Inc., OU=Apple Certification Authority, CN=Apple Root CA
[...]
    Subject: C=US, O=Apple Inc., OU=Apple Certification Authority, CN=Apple Root CA
```

> I was able to verify the signature manually:
```sh
# The following outputs the public key to be used below as ./first.pubkey.pem.
$ openssl x509 -in first.pem -pubkey 
$ openssl dgst -sha256 -verify ./first.pubkey.pem -signature ./push_cert_signature.data.binary ./push_cert_request.csr.binary
Verified OK
```

The last step is to upload the `mdmcert.download.push.req` XML plist file to https://identity.apple.com, which lets you download the final `mdmcert.download.push.pem`.

Configure your MDM server to use `mdmcert.download.push.pem` and `mdmcert.download.push.key`. The remaining files `mdmcert.download.pki.key`, `mdmcert.download.pki.crt`, `mdmcert.download.push.csr`, and `mdmcert.download.push.req` are not necessary anymore.
