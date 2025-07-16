# tpm

This tool obtains a certificate from an external SCEP server using a challenge password with actual TPM hardware for secure key operations.

This example tool shows the complete workflow:
1. Initialize TPM 2.0 device for hardware-based cryptography.
2. Configure the SCEP client with server URL, challenge password, and other options.
3. Fetch a certificate using SCEP with a private key generated in the TPM.

Prerequisites:
- TPM 2.0 hardware available at /dev/tpmrm0.
- Environment FLEET_URL set to the Fleet URL.
- Environment ENROLL_SECRET set to a valid enroll secret.
- Run example as root.

## build

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o tpm ./tools/tpm
```

## run
```sh
sudo FLEET_URL=https://fleet.example.com ENROLL_SECRET=foobar ./tpm
```
