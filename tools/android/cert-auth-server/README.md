# Android Certificate Authentication Test Server

A simple mTLS (mutual TLS) test server for validating Android device certificate-based authentication with Fleet.

## Overview

This server validates client certificates issued via SCEP (Simple Certificate Enrollment Protocol) to Android devices enrolled in Fleet. It demonstrates the end-to-end flow of:

1. Fleet managing Android devices
2. SCEP server issuing device certificates
3. Devices authenticating to resources using those certificates

## Prerequisites

- Go 1.21+
- [micromdm/scep](https://github.com/micromdm/scep) server
- Fleet server with Android MDM enabled

## Quick Start

### 1. Set Up the SCEP Server

First, install and configure the micromdm/scep server to issue certificates to your Android devices.

#### Install SCEP Server

```bash
# Download from releases
curl -LO https://github.com/micromdm/scep/releases/latest/download/scepserver-darwin-arm64
```

#### Initialize the CA

```bash
./scepserver ca -init \
  -organization "Your Organization" \
  -country "US" \
  -common_name "Fleet SCEP CA"
```

This creates a `depot/` directory containing:

- `ca.pem` - CA certificate
- `ca.key` - CA private key

#### Start the SCEP Server

```bash
./scepserver -depot depot -port 2016 -challenge=your-secret-challenge
```

The SCEP endpoint will be available at `http://localhost:2016/scep`.

### 2. Configure Fleet for SCEP

Configure Fleet to use your SCEP server for Android certificate enrollment. Add the SCEP configuration to your Fleet server:

```yaml
# fleet.yml
mdm:
  android:
    scep_url: "http://your-scep-server:2016/scep"
    scep_challenge: "your-secret-challenge"
```

Fleet will automatically request certificates for enrolled Android devices through the SCEP protocol.

### 3. Run the Certificate Auth Server

Build and run this test server, pointing it to the same CA that your SCEP server uses:

```bash
# Build
go build -o cert-auth-server main.go

# Run (using the CA certificate from your SCEP depot)
./cert-auth-server -ca-cert /path/to/depot/ca.pem -addr :8443
```

### 4. Test Device Authentication

From an enrolled Android device with a certificate issued by your SCEP server:

Load the server URL in a browser or HTTP client:

`https://your-cert-auth-server:8443/`

It should prompt for a client certificate. Upon successful authentication, you should see a message confirming the device's identity.