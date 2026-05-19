# Apple Platform Single Sign-On (PSSO) — design decisions

## Overview

Platform Single Sign-On (PSSO) is a macOS 13+ feature in which an identity provider participates in the local login window, screen-lock unlock, and keychain authentication flows by way of an Apple Single Sign-On extension and a matching configuration profile. Fleet is implementing PSSO so that an end user's local macOS account password can be kept in sync with the same credential they use against the upstream IdP, satisfying the product/design requirement for local-account password sync.

## Decision log

### PSSO v2 with Password mode

The extension is registered as `AuthenticationMethod = UserSecureEnclaveKey` v2 with `RegistrationToken`-style flows, configured for **Password** authentication (not `SecureEnclaveKey`, and not v1). Password mode is the only PSSO configuration that surfaces the user's plaintext password to the extension at sign-in, which is what the local-account sync requirement needs. v1 is not considered because it predates the current registration / token-exchange protocol Apple ships in the current `ASAuthorizationProviderExtension*` APIs.

### Identity backend: generic OIDC ROPG, pluggable (Okta-first)

The server defines a `PSSOIdPClient` interface; the first concrete implementation (`PSSOOIDCROPGClient`) speaks OAuth 2.0 Resource Owner Password Grant (`grant_type=password`) against any OIDC IdP. The token URL is taken from `AppConfig.PSSOSettings.idp_token_url`, so the same code path works against Okta, Entra ID, Auth0, Keycloak, or any other ROPG-capable IdP — only the configured URLs change.

The POC is exercised against Okta first because it's a faster path to a working integrator-tier sandbox than provisioning an Entra tenant. Okta-specific caveats are documented in `tools/psso/README.md`: ROPG must be explicitly enabled on the application, and the app type must be Native or Service.

Additional backends (LDAP bind, direct-trust flows for IdPs that reject ROPG) slot in behind the same `PSSOIdPClient` interface without changes to the PSSO endpoint handlers. The pluggable shape mirrors how the broader Fleet codebase isolates IdP-specific behavior behind an interface.

### Enterprise-gated with no-license core stubs

Route registration for `/mdm/apple/psso/*` and the related `.well-known` endpoints lives in `server/service/handler.go` (core). The real implementation lives in `ee/server/service/`; the core build provides stubs that return `fleet.ErrMissingLicense`. This matches the pattern already used by `calendar.go` and the enterprise pieces of `apple_mdm.go`.

### Clean-room crypto, no third-party PSSO SDK

The PSSO protocol is implemented directly from Apple's `ASAuthorizationProviderExtension*` headers and the WWDC 2022/2023 sessions. Crypto primitives come from `github.com/golang-jwt/jwt/v4`, `github.com/go-jose/go-jose/v3`, `crypto/ecdsa`, and `golang.org/x/crypto/hkdf`. No third-party PSSO SDK or sample repository is imported, vendored, copied, or referenced.

### Endpoint paths follow the SCEP/MDM convention

The PSSO endpoints sit at the root of the URL space — `/mdm/apple/psso/nonce`, `/mdm/apple/psso/register`, `/mdm/apple/psso/token` — with no `/api/` or `/v1/` prefix, matching the existing `/mdm/apple/scep` and `/mdm/apple/mdm` paths Apple devices already talk to. The associated `/.well-known/jwks.json` and `/.well-known/apple-app-site-association` are also served at root, because Apple's frameworks fetch them by spec-defined absolute paths.

### Nonces in Redis, not MySQL

The nonce store mirrors `server/mdm/acme/internal/redis_nonces_store/` and exposes the same minimal surface: `Store(ctx, nonce, ttl)` and `Consume(ctx, nonce) (bool, error)`. Nonces are short-lived and single-use; MySQL would add round-trip cost and migration overhead with no benefit.

### Two MySQL tables

- `mdm_apple_psso_devices` — primary key `host_id`, stores the device's signing and encryption public keys (PEM), the negotiated KeyExchangeKey, and registration/update timestamps.
- `mdm_apple_psso_key_ids` — primary key `kid`, foreign key `host_id`, plus `key_type` and `pem`. The extension references keys by SHA-256 hash of the public key, so the server needs an index keyed by that hash to resolve incoming requests back to a device.

### JWKS signing key bootstrap timing (OPEN)

Current placeholder behavior: the JWKS signing key is lazily minted on the first `GET /.well-known/jwks.json` and persisted (encrypted) in `mdm_config_assets` under `MDMAssetPSSOSigningKey`. Alternatives under consideration include minting on `AppConfig.PSSOSettings.Enabled = true`, minting on first device registration, or requiring an explicit `fleetctl psso bootstrap`. The lazy-mint code carries a `TODO`; decision pending.

### No Fleet-side profile generation for the POC

A sample `.mobileconfig` template is shipped at `tools/psso/sample.mobileconfig`. Admins fill in placeholders (Fleet base URL, IdP tenant, extension bundle ID) and upload via Fleet's existing custom-profile delivery. Server-side profile templating (analogous to `ensureFleetProfiles`) is out of scope for the POC.

### In-tree Swift extension at `apple-sso-extension/`

The Swift sources for the SSO extension live in this repo at `apple-sso-extension/`. Signing and notarization happen out-of-band using the deployer's own Apple Developer ID; Fleet does not ship a signed binary. The hostname declared in the extension's `authsrv:` entitlement must match the hostname served by `/.well-known/apple-app-site-association`.

## Known limitations

- **OIDC ROPG has provider-specific limitations.** Okta: ROPG must be explicitly enabled on the application and the app must be Native or Service type. Entra: MFA-required users and federated (AD FS) users cannot authenticate via ROPG. These are upstream constraints, not Fleet bugs. Customers in those configurations need an alternative `PSSOIdPClient` backend (LDAP bind or a direct-trust flow).
- **AASA requires a public-CA TLS certificate.** Apple's framework silently rejects self-signed certificates when fetching `/.well-known/apple-app-site-association`. Local development requires a real DNS name with a Let's Encrypt cert, or a tunnel such as ngrok or cloudflared.
- **No device revocation or key rotation in the POC.** Devices register once and stay registered for the life of the row.
- **Global config only.** PSSO settings live on `AppConfig`; there is no per-team override.

## Pointers

- `PSSO_OVERVIEW.md` at the repo root — protocol primer covering the registration and token-exchange message shapes.
- `~/.claude/plans/we-re-going-to-implement-sorted-sparkle.md` — current-cycle implementation plan (working document, not a permanent reference).
- Apple WWDC sessions: *Platform SSO for macOS* (WWDC 2022), and the *Discover authentication services* / *Shared device keys* material (WWDC 2023).
- Apple developer documentation for the `ASAuthorizationProviderExtension*` family of classes and protocols.
