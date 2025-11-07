# fetch_cert

Example client for fetching certificates via fleet's `/api/v1/fleet/certificate_authorities/{id}/request_certificate` endpoint, authenticated using TPM-backed HTTP Message Signing.

## Usage

```
Usage of ./fetch_cert:
  -ca uint
        certificate authority ID
  -csr string
        csr path
  -debug
        enable debug logging
  -fleeturl string
        fleet server base URL
  -out string
        output certificate path (default "certificate.pem")
  -rootdir string
        fleetd installation root (default "/opt/orbit")
```
