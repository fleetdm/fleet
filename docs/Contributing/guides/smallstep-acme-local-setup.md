# Setting Up Smallstep Locally with ngrok and ACME Device Attestation

This guide walks through setting up an open-source `step-ca` instance locally with ngrok for remote access, configured for Apple Managed Device Attestation (MDA) via ACME. It's intended for Engineering and QA testing of Fleet's ACME/MDA flows.

## Prerequisites

- macOS with Homebrew (or Linux equivalent)
- ngrok account (free tier works for basic testing; custom domain requires a paid plan)
- Apple device enrolled in MDM (optional, for end-to-end testing)

## Step 1: Install step and step-ca

```bash
brew install step
```

Verify installation:

```bash
step version
step-ca version
```

## Step 2: Initialize Your Local CA

```bash
step ca init
```

When prompted:

- **PKI name**: `localdev` (or your preferred name)
- **DNS names/IPs**: `localhost` (you'll use ngrok for remote access)
- **Listen address**: `127.0.0.1:9443`
- **First provisioner name**: `you@fleetdm.com` (or your identifier)
- **Password**: Set a strong password or leave blank to generate one

This creates the CA structure in `~/.step/`.

Save your root fingerprint for later:

```bash
step certificate fingerprint $(step path)/certs/root_ca.crt
```

## Step 3: Start step-ca Locally

```bash
step-ca $(step path)/config/ca.json
```

Enter your CA password. The server listens on `127.0.0.1:9443`.

Keep this running in a separate terminal (use `tmux`, `screen`, or a new tab).

## Step 4: Set Up ngrok

```bash
# Install ngrok (if needed)
brew install ngrok

# Start ngrok with TLS passthrough (critical for ACME)
ngrok tls 127.0.0.1:9443 --domain=YOUR_NGROK_DOMAIN
```

Replace `YOUR_NGROK_DOMAIN` with your ngrok static domain (e.g., `you-step.ngrok.app`). This requires a paid ngrok account with custom domains.

**TLS passthrough is essential.** Without it, ngrok terminates TLS and serves its own certificate, which breaks ACME device attestation.

Note your public URL (e.g., `https://you-step.ngrok.app:443`).

## Step 5: Bootstrap Your Client

In a new terminal:

```bash
FINGERPRINT=$(step certificate fingerprint $(step path)/certs/root_ca.crt)
step ca bootstrap --ca-url https://you-step.ngrok.app --fingerprint $FINGERPRINT --install
```

This adds your CA's root certificate to your system trust store.

## Step 6: Create an ACME Provisioner with Device Attestation

```bash
step ca provisioner add acme-mda \
  --type ACME \
  --challenge device-attest-01 \
  --attestation-format apple
```

Restart `step-ca` (Ctrl+C and re-run).

## Step 7: Configure Certificate Duration and Subject Template

Edit `~/.step/config/ca.json` and find the `acme-mda` provisioner. Update it:

```json
{
  "type": "ACME",
  "name": "acme-mda",
  "maxDuration": "720h",
  "challenges": [
    "device-attest-01"
  ],
  "attestationFormats": [
    "apple"
  ],
  "claims": {
    "defaultTLSCertDuration": "720h",
    "enableSSHCA": true,
    "disableRenewal": false,
    "allowRenewalAfterExpiry": false,
    "disableSmallstepExtensions": false
  },
  "options": {
    "x509": {
      "template": "{\n  \"subject\": {\n    \"commonName\": {{ toJson .Insecure.CR.Subject.CommonName }},\n    \"organizationalUnit\": {{ toJson .Insecure.CR.Subject.OrganizationalUnit }}\n  }\n}\n"
    },
    "ssh": {}
  }
}
```

Key settings:

- `maxDuration`: Maximum certificate lifetime (720h = 30 days)
- `defaultTLSCertDuration`: Default lifetime for issued certs (also 720h)
- `x509.template`: Preserves the OU from the CSR (critical for tracking)

Restart `step-ca`.

## Step 8: Verify ACME Directory

```bash
curl --cacert $(step path)/certs/root_ca.crt \
  https://you-step.ngrok.app/acme/acme-mda/directory
```

You should see JSON with ACME endpoints:

```json
{
  "newNonce": "https://...",
  "newAccount": "https://...",
  "newOrder": "https://...",
  ...
}
```

## Step 9: Create MDM Profile

Create an ACME Certificate payload (`.mobileconfig` or via your MDM platform):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>ACME MDA (Local step-ca)</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.test.acme-mda</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadScope</key>
    <string>System</string>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadType</key>
            <string>com.apple.security.acme</string>
            <key>PayloadIdentifier</key>
            <string>com.fleetdm.test.acme-mda.cert</string>
            <key>PayloadUUID</key>
            <string>87BF36A9-339F-46F4-8A7E-18950C2428FC</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>PayloadDisplayName</key>
            <string>ACME MDA Certificate</string>
            <key>DirectoryURL</key>
            <string>https://you-step.ngrok.app/acme/acme-mda/directory</string>

            <key>ClientIdentifier</key>
            <string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string>

            <key>KeyType</key>
            <string>ECSECPrimeRandom</string>
            <key>KeySize</key>
            <integer>256</integer>
            <key>HardwareBound</key>
            <true/>
            <key>Attest</key>
            <true/>
            <key>Subject</key>
            <array>
                <array>
                    <array><string>CN</string><string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string></array>
                </array>
                <array>
                    <array><string>OU</string><string>$FLEET_VAR_CERTIFICATE_RENEWAL_ID</string></array>
                </array>
            </array>
            <key>DNSNames</key>
            <array>
                <string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string>
            </array>
            <key>ExtendedKeyUsage</key>
            <array>
                <string>1.3.6.1.5.5.7.3.2</string>
            </array>

            <key>UsageFlags</key>
            <integer>1</integer>

            <key>KeyIsExtractable</key>
            <false/>
        </dict>
    </array>
</dict>
</plist>
```

Critical fields:

- `DirectoryURL`: Points to your `acme-mda` provisioner: `https://you-step.ngrok.app/acme/acme-mda/directory`
- `ClientIdentifier`: Device serial (injected by MDM)
- `HardwareBound`: `true` (key in Secure Enclave)
- `Attest`: `true` (device provides attestation)
- `Subject`: Separate RDN arrays for CN and OU
- `KeyType`/`KeySize`: `ECSECPrimeRandom` / `256` (required for hardware-bound keys)

## Step 10: Deploy and Test

1. Upload the profile to your MDM (Fleet, Jamf, Intune, etc.)
2. Deploy to a test macOS device
3. The device will:
   - Generate a hardware-bound EC key in Secure Enclave
   - Request an attestation from Apple
   - Contact your ACME server
   - Receive a certificate with 30-day validity

## Troubleshooting

### Device gets "failed to request authorization" error

- Verify the ACME directory is accessible: `curl https://you-step.ngrok.app/acme/acme-mda/directory`
- Check that `device-attest-01` challenge is configured in the provisioner
- Ensure `Attest=true` and `HardwareBound=true` in the profile

### Device gets 500 error from ACME server

- Check `step-ca` logs for CSR validation errors
- Verify Subject DN format matches Apple's schema (separate RDN arrays)
- Ensure all provisioner claims (duration, template) are properly set

### Certificate duration is 1 day instead of 30 days

- Verify `defaultTLSCertDuration` is set in provisioner claims
- Restart `step-ca` after editing config
- Confirm with: `step ca provisioner list --json | jq '.[] | select(.name=="acme-mda") | .claims.defaultTLSCertDuration'`

### ngrok shows certificate mismatch errors

- Use TLS passthrough mode (`ngrok tls`), not HTTP tunneling
- Verify your ngrok domain matches the one you bootstrapped with

## Useful Commands

View provisioners:

```bash
step ca provisioner list --json | jq '.[] | {name, type, maxDuration, challenges}'
```

Get CA status:

```bash
curl --cacert $(step path)/certs/root_ca.crt https://you-step.ngrok.app/health
```

Request a test certificate (JWK provisioner):

```bash
step ca certificate localhost test.crt test.key --provisioner you@fleetdm.com
```

Inspect a certificate:

```bash
step certificate inspect test.crt
```

View full config:

```bash
cat $(step path)/config/ca.json | jq '.'
```

## References

- [Smallstep step-ca docs](https://smallstep.com/docs/step-ca/)
- [Apple ACME Certificate payload schema](https://github.com/apple/device-management/blob/release/mdm/profiles/com.apple.security.acme.yaml)
- [ACME Device Attestation overview](https://smallstep.com/blog/acme-managed-device-attestation-explained/)
