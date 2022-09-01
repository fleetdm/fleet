# Setup

The one-time setup consists of configuring:
- The APNs certificate used by the MDM protocol
- The SCEP certificate for enrollment
- The DEP token

We will define `fleetctl apple-mdm setup ...` commands to create/define all Apple/MDM credentials that are fed to the Fleet server.

## APNs

### Apple MDM APNs setup

Apple's MDM protocol uses the Apple Push Notification service (APNs) to deliver "wake up" messages to managed devices.
An "MDM server" needs access to an APNS certificate specifically issued for MDM management; such APNs certificate must be issued by an "MDM vendor."

Here's a sequence diagram with the three actors: Apple Inc., an MDM vendor, and a customer (MDM server).

```mermaid
%%{init: { 'theme':'dark', 'sequence': {'mirrorActors':false} } }%%

sequenceDiagram
    participant apple as Apple Inc.
    participant vendor as MDM Vendor
    participant server as MDM Server<br>(Customer)

    rect rgb(128, 128, 128)
    note left of apple: (1) MDM Vendor Setup at<br>https://business.apple.com
    note over vendor: Generate CSR
    vendor->>apple: Send CSR
    note over apple: Sign CSR
    apple->>vendor: "MDM vendor" certificate<br>(Setup)
    end
    rect rgb(255, 255, 255)
    note left of apple: (2) Customer Setup
    note over server: Generate CSR
    server->>+vendor: "Send" CSR (.csr)
    note over vendor: Sign CSR
    vendor->>server: "Send" signed CSR (XML plist, .req)
    rect rgb(128, 128, 128)
    note left of apple: https://identity.apple.com/pushcert
    server->>apple: Upload signed CSR (XML plist, .req)
    note over apple: Sign CSR
    apple->>server: Download APNS Certificate (PEM)
    end
    end
    note over server: APNS keypair<br>ready to use
```

The "MDM vendor setup" flow (1) is executed once by the "MDM vendor."

The "customer setup" flow (2) is executed by customers when they are setting up their MDM server.

The goal is for the Fleet organization to become an "MDM vendor" that issues CSRs to customers, which allows them to generate "APNs certificates" for their MDM deployments.

For the purposes of designing a PoC, we used the https://mdmcert.download/ service as an "MDM vendor."
See [MDMCert.Download Analysis](./mdmcert.download-analysis.md) for more details on the process.

### APNs setup with Fleet

The MDM APNs certificate provisioning will be manual on MVP:
- Customers will use `fleetctl` commands that will mimick `mdmctl mdmcert.download` commands (see [MDMCert.Download Analysis](mdmcert.download-analysis.md)).
- Fleet operators will perform the steps shown in the diagram above manually by running a new command line tool (under `tools/mdm-apple/mdm-apple-customer-setup`).

#### 1. Init APNs (customer)

`fleetctl apple-mdm setup apns init` 

The command will basically mimick [mdmctl mdmcert.download -new](https://github.com/micromdm/micromdm/blob/main/cmd/mdmctl/mdmcert.download.go).
Steps:
1. Generate an RSA private key and certificate for signing and encryption. 
(Store them in `~/.fleet/config`, as there's no need to store these as files.) 
Let's call these "PKI" key and cert.
2. Generate an RSA push private key and CSR. Store the private key as a file: `fleet-mdm-apple-apns-push.key`. 
TODO(Lucas): Store private key encrypted with passphrase?
3. Also output:
- File fleet-mdm-apple-apns-setup.zip with:
	- fleet-mdm-apple-apns-push.csr
	- fleet-mdm-apple-apns-pki.crt
- Text to stdout that explains next step, something like:
	"Send zip to Fleet via preferred medium (e-mail, Slack)."

#### 2. New tool `tools/mdm-apple/mdm-apple-customer-setup` (Fleet representative)

`mdm-apple-customer-setup --zip fleet-mdm-apple-apns-setup.zip`

Output:
- `fleet-mdm-apple-apns-push-req-encrypted.p7`
- Text to stdout that explains next step, something like:
	"Send generated file 'fleet-mdm-apple-apns-push-req-encrypted.p7' back to customer via preferred medium (email, Slack)."

#### 3. Finalize APNs (customer)

`fleetctl apple-mdm setup apns finalize --encrypted-req=fleet-mdm-apple-apns-push-req-encrypted.p7`

Output:
	- `fleet-mdm-apple-apns-push.req` file

If successful, it clears PKI key and certificate from `~/.fleet/config`.

#### 4. Upload .req to Apple (customer)

Customer uploads `fleet-mdm-apple-apns-push.req` to https://identity.apple.com.

#### 5. Download .pem from Apple (customer)

Downloads the final APNs certificate, a `*.pem` file. Let's call it `fleet-mdm-apple-apns-push.pem`.

The contents of `fleet-mdm-apple-apns-push.pem` and `fleet-mdm-apple-apns-push.key` are passed to Fleet as environment variables.

## SCEP

Apple's MDM protocol uses client certificates for client authentication. To generate client certificates, Apple's MDM protocol uses the [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) protocol.

The setup for SCEP consists of generating the "SCEP CA" for Fleet.

### 1. Set up SCEP CA (customer)

`fleetctl apple-mdm setup scep`

Generates SCEP CA and key:
- `fleet-mdm-apple-scep.key`
- `fleet-mdm-apple-scep.pem`

The contents of `fleet-mdm-apple-scep.pem` and `fleet-mdm-apple-scep.key` are passed to Fleet as environment variables.

## DEP

### 1. Generate key material

`fleetctl apple-mdm setup dep init`

- Generates `fleet-mdm-apple-dep.pem` and `fleet-mdm-apple-dep.key`:
	- Stores `fleet-mdm-apple-dep.pem` as a file.
	- Keeps `fleet-mdm-apple-dep.key` in some location like `~/.fleet/config`/`/tmp` (the user does not need to handle the `.key`).

### 2. Upload PEM to Apple

User uploads `fleet-mdm-apple-dep.pem` to https://business.apple.com, and downloads a `*.p7m` file.
Let's call it `fleet-mdm-apple-dep-auth-token-encrypted.p7m`.

### 3. Finalize DEP setup

`fleetctl apple-mdm setup dep finalize --encrypted-token=fleet-mdm-apple-dep-auth-token-encrypted.p7m`
	
1. Decrypts the provided `fleet-mdm-apple-dep-auth-token-encrypted.p7m` with the `fleet-mdm-apple-dep.key` from `~/.fleet/config`/`/tmp`.
2. Generates a `fleet-mdm-apple-dep.token` file.
3. Removes `fleet-mdm-apple-dep.key` from `~/.fleet/config`.

The contents of `fleet-mdm-apple-dep.token` is passed to Fleet as environment variable.