
Apple Managed Device Attestation has been around for a couple of years now. Most MDMs that support it lean on a commercial CA: Hydrant, DigiCert, or Smallstep. Fleet integrates directly with Hydrant EST if that's the path you want. 

But a managed CA means trusting a vendor's implementation of a spec you can't see, and committing to their billing model before you know the shape of the problem. I wanted to see the spec working end to end, on my own terms, before deciding whether to pay anyone to do it for me. 

The answer turned out to be a Go library called nanoca, by [Brandon Weeks](https://github.com/brandonweeks), who also authored [the IETF draft for ACME device attestation](https://datatracker.ietf.org/doc/draft-ietf-acme-device-attest/). Plus a Render account and an afternoon. Here's what that took, what surprised me, and what's still rough.

## What this gives you

Two things.

The artifact: a self-hosted ACME endpoint that issues hardware-bound client identity certificates to Apple Silicon Macs enrolled in Fleet, validated by Apple Managed Device Attestation, delivered through a Fleet Custom settings profile. This is separate from Fleet's built-in ACME server for MDM enrollment identity in 4.84+. It's for the other certificate use cases where you want hardware-attested device identity.

The understanding: every step of the ACME plus MDA flow visible, because nothing is hidden behind a vendor. You see what the `device-attest-01` challenge actually looks like, what Apple's attestation certificate chain looks like in practice, what the Apple Business API integration takes (JWT-bearer OAuth2, not the more common client ID and secret), and how Fleet's `com.apple.security.acme` payload behaves when delivered through Custom settings.

About thirty minutes of work if you've got an afternoon and a Mac to test with.

If you're evaluating Apple MDA on Fleet and want to see the protocol working before committing to a commercial CA, or want to understand the moving parts well enough to debug whatever you do end up running, this is for you. You'll need to be comfortable reading a little Go and pushing a Render deploy.

## The library

Brandon's README puts it plainly:

<div style="margin: 1.5em 0; padding-left: 1.5em; border-left: 4px solid #ccc;">
Storage, signing, authorization, and logging are implemented as pluggable interfaces to integrate into a wide variety of environments.
</div>

So nanoca isn't a server. It's a CA construction kit. Brandon decided not to bundle decisions you'd want to make yourself: where the root CA key lives, how state persists, which devices are allowed to ask for certificates. He shipped the interfaces and one or two implementations of each. You compose them.

That sounds like more work than buying a managed CA. It isn't. Composition takes six lines of Go:

```go
ca, err := nanoca.New(
    logger,
    inprocess.New(caCert, signer),
    nullauthorizer.New(),
    storage,
    baseURL,
    nanoca.WithPrefix("/acme"),
    nanoca.WithVerifier(apple.New(logger)),
)
```

That's a working ACME CA with Apple device attestation. Everything else in my wrapper, about seventy lines, is environment variable plumbing, PEM parsing, and an HTTP listener.

Each pluggable interface is one import line away from a different choice. File signer reads the root CA key from disk; you could swap it for an HSM-backed signer without touching anything else. Badger storage persists ACME state to disk, or runs entirely in memory for ephemeral tests with `badger.Options{InMemory: true}`. The null authorizer approves anything Apple attestation lets through; the Apple Business authorizer (more on that below) checks the device against your Apple Business inventory. The Apple verifier validates the `device-attest-01` challenge by checking that the leaf certificate from the device chains to Apple's Enterprise Attestation Root and that the embedded extensions match what's expected. Brandon's done that work. You import a package.

## Deploying it

I added a `cmd/mpc-server/` directory to a fork of nanoca with the wrapper, plus a Dockerfile and a `render.yaml` at the repo root.

Render reads the blueprint, builds the binary, attaches a 1 GB persistent disk for the Badger database, mounts the root CA certificate and key as secret files, and provisions a Let's Encrypt certificate for the custom domain. Total time from `git push` to `curl https://cert.mpc.ad/acme/directory` returning JSON: about five minutes.

One wrinkle worth knowing if you deploy behind a TLS-terminating proxy: nanoca enforces RFC 8555's rule that ACME requests arrive over TLS by checking `r.TLS`. Render (like Cloudflare) terminates TLS at the edge and reaches the container over plain HTTP, so `r.TLS` is nil and every signed ACME POST comes back as `HTTPS is required`. The wrapper handles it with a small shim that trusts `X-Forwarded-Proto: https` and reconstructs the TLS state, rather than patching the upstream library. Front nanoca with any TLS-terminating proxy and you'll need the same.

The full wrapper and deployment plumbing is at [github.com/AdamBaali/nanoca](https://github.com/AdamBaali/nanoca) under `cmd/mpc-server/`. Fork, swap a few values, deploy.

You'll need:

- An Apple Silicon Mac running macOS 14 or later. Intel, T2, and VMs cannot do hardware attestation
- A Fleet instance (4.86.1+ if you want the certificate to appear automatically in host details; earlier versions still work but the cert is only visible via the MDM `CertificateList` command directly)
- A domain you control for the ACME endpoint
- A Render account on the Starter plan (~$7/month). The free tier won't work because services spin down and break ACME flows
- A root CA key and self-signed root certificate, generated with `openssl genrsa` and `openssl req -x509`

## Plugging it into Fleet

Two configuration profiles, both delivered via Fleet's Custom settings.

The first pushes the root CA from nanoca as a trusted root with `com.apple.security.root`. Without this, the Mac won't trust certificates nanoca issues.

The second is the actual ACME payload, Apple's `com.apple.security.acme`, pointing at the directory URL, requesting hardware-bound ECDSA keys, requiring attestation:

```xml
<key>DirectoryURL</key><string>https://cert.mpc.ad/acme/directory</string>
<key>KeyType</key><string>ECSECPrimeRandom</string>
<key>KeySize</key><integer>384</integer>
<key>HardwareBound</key><true/>
<key>Attest</key><true/>
<key>ClientIdentifier</key><string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string>
```

Upload both to Fleet, scope to a test team containing one Apple Silicon Mac, and wait for the profile to install.

## What success looks like

The Render logs for nanoca will show a new ACME account, a new order, the `device-attest-01` challenge served and validated, then the finalized certificate downloaded. Each step gets a debug-level line you can `grep` for.

In Fleet, the profile status goes Pending to Verified once the device acknowledges installation. On Fleet 4.86.1+, the certificate appears in the host's certificates list. On earlier versions it does not. Not because anything's broken, but because of an Apple quirk worth knowing about.

Hardware-bound ACME certificates don't appear in the macOS keychain. `security find-identity -v` won't show them. The osquery `certificates` table won't show them either. They're visible only via MDM's `CertificateList` command. Fleet 4.86.1 added automatic ingestion of those results into host details. Before that release, the certificate exists on the device (hardware-bound to the Secure Enclave, signed by your CA, working) but is invisible to Fleet's UI. If you're testing on an earlier version, query MDM directly to verify.

## Going production: Apple Business-gated authorization

The null authorizer gets you to a working setup, which is enough to learn from. If the test convinced you to run this in production, the next step is gating issuance on something you actually control. The null authorizer issues certificates to any attested Apple device, yours or not. That's fine for understanding the flow but not fine for anything beyond that. Apple Business is the natural choice: it's where you already track every device your organization owns.

This is one of the places nanoca's pluggable design pays off, because Brandon already ships an Apple Business authorizer (still named `abm` in the package path, since Apple's API surface kept the `BUSINESSAPI.` prefix). It lives at `github.com/brandonweeks/nanoca/authorizers/abm` and satisfies the same `Authorizer` interface as the null one. You don't write it. You wire it up. The wrapper in my fork ships the null-authorizer test build; the Apple Business swap below is what you'd layer on for production.

The constructor takes an `abm.Config` carrying JWT credentials for the Apple Business API. Apple uses JWT-bearer OAuth2 for this API, not the client ID and client secret pattern you might be used to. You pass a Client ID, a Key ID, and a private key that signs the JWT assertion. Apple gives you the Client ID and Key ID when you create an API integration inside your Apple Business tenant; you generate the private key locally, upload the public half to Apple Business, and keep the private half wherever your other production secrets live:

```go
import (
    abmauthorizer "github.com/brandonweeks/nanoca/authorizers/abm"
    "github.com/brandonweeks/nanoca/abm"
)

signingKey, err := loadPrivateKey(os.Getenv("ABM_PRIVATE_KEY_PATH"))
if err != nil {
    return fmt.Errorf("load ABM signing key: %w", err)
}

abmAuth, err := abmauthorizer.New(ctx, &abm.Config{
    JWTConfig: &abm.JWTConfig{
        ClientID:   os.Getenv("ABM_CLIENT_ID"),
        KeyID:      os.Getenv("ABM_KEY_ID"),
        PrivateKey: signingKey,
    },
})
if err != nil {
    return fmt.Errorf("ABM authorizer: %w", err)
}
```

Then swap one line in the CA constructor:

```go
ca, err := nanoca.New(
    logger,
    inprocess.New(caCert, signer),
    abmAuth,   // replaces nullauthorizer.New()
    storage,
    baseURL,
    nanoca.WithPrefix("/acme"),
    nanoca.WithVerifier(apple.New(logger)),
)
```

Now the flow looks like this. Apple's verifier confirms the device is real Apple hardware. The Apple Business authorizer asks the Apple Business API whether that device's serial number exists in your organization. Both have to pass before a certificate is issued. The combination is the actual gate you want in production: proves the hardware and proves it's yours. You've built it on top of two packages Brandon already wrote.

That's the work a commercial CA usually does for you. The trade-off is that you own the operational surface area; the upside is that you can see your own inventory in a way a third-party CA can't.

## What's still rough

A few honest limitations:

- **Render runs the CA private key on managed infrastructure.** Fine for a POC and probably fine for many production setups, but if your threat model wants the signing key in an HSM, swap the file signer for a different implementation.
- **No revocation flow.** nanoca implements certificate issuance; it doesn't currently expose OCSP or CRL endpoints. For renewal, Fleet 4.86.1+ automatically re-applies SCEP and hardware-attested ACME configuration profiles before expiration, which prompts the device to request a fresh cert from nanoca, so the renewal loop is covered end to end without anything extra to build. If you need active revocation (kill a certificate before its expiry), that's another piece you'd build.
- **The Apple Business authorizer fetches your full org device list on every issuance request.** Look at the source: each call to `Authorize` hits the `/orgDevices` endpoint and linear-scans the result. Fine for small fleets. For larger ones you'd wrap it in a caching layer that refreshes the inventory on a schedule. Same `Authorizer` interface, just composed. The kind of thing nanoca's design makes easy.

## What this exercise showed

Three things worth naming.

First, ACME plus Managed Device Attestation isn't proprietary magic. The flow is observable, the components are small, and a one-person afternoon gets you to a working CA. If you're paying a commercial CA, you're paying for operational comfort, not inscrutable wizardry.

Second, the Apple Business API auth is the friction point. JWT-bearer OAuth2 with ES256-signed client assertions is a lot of moving parts compared to a client ID plus secret. nanoca's pluggable authorizer interface lets you wrap that complexity once and forget it. Without that interface, every CA implementation reinvents the wheel.

Third, Fleet's Custom settings to ACME payload path is the part I had the least insight into before starting, and it's the part I now trust the most. The payload installs, the device asks for a cert, the cert lands. On Fleet 4.86.1+ it shows up in the UI; on earlier versions it's there, just not visible in Fleet. Knowing exactly where the visibility gap is matters more than not having it.

## Credit

There's a particular kind of person who writes the IETF spec, then writes the reference implementation, then makes the reference implementation a tidy, reusable Go library you can drop into your own service. The part Brandon didn't have to do is the part that made any of this possible in an afternoon.

That's open source done right. The library is MIT-licensed at [github.com/brandonweeks/nanoca](https://github.com/brandonweeks/nanoca). Go read it.

<meta name="articleTitle" value="Apple Managed Device Attestation without a commercial CA">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="publishedOn" value="2026-06-22">
<meta name="category" value="guides">
<meta name="description" value="Issue hardware-bound ACME certificates to Apple Silicon Macs enrolled in Fleet, using a self-hosted open-source CA. No commercial CA required.">
