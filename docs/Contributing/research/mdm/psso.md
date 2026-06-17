# Apple Platform Single Sign-On (PSSO) — design decisions

## Overview

Platform Single Sign-On (PSSO) is a macOS 13+ feature in which an identity provider participates in the local login window, screen-lock unlock, and keychain authentication flows by way of an Apple Single Sign-On extension and a matching configuration profile. Fleet is implementing PSSO to show that an end user's local macOS account password can be kept in sync with the same credential they use against the upstream IdP, satisfying the product/design requirement for local-account password sync, and additionally showing creating the user during Setup Assistant with  a password synced with the IDP. Ultimately it was determined that Fleet can implement an SSO plugin that can meet these requirements if a user is using an LDAP or OAUTH ROPG supporting IDP

### Known limitations:

The most pertinent known limitations to be aware of up front

* 4 hour sync window: a mac running Password SSO checks the password against the IDP if the existing token is missing, expired or over 4 hours old at the next unlock/login event. This means a user can go up to 4 hours plus however long it takes them to lock and unlock their mac before the mac detects that the password has changed. Even if they logout and login during this time in my testing macOS doesn't reach out to the IDP so there is a small window where their password can be out of sync if they don't start using the new password. If they enter the new password, macOS will immediately reach out to the IDP and upon confirmation trigger a password change. It is unclear if there is any mechanism to work around this even if a component on the system like Orbit knows a password change has occurred. 

* Some cases require both passwords: It wasn't directly encountered during the PSSO POC but Apple documentation suggests the user may occasionally be prompted for old and new passwords. I suspect this can happen at the filevault screen at times.

* No way to make SSO to apps in a browser work - ultimately we cannot integrate closely enough with an IDP for this to work and I don't see any way to get around it

* Requires enabling LDAP or ROPG within IDP and may be unacceptable from a security standpoint for some customers. Many IDPs suggest against using these as they result in plaintext passwords transiting third party apps but there is no other clear way to implement this feature

* Likely no easy way for an admin to update the logo shown on PSSO notifications/screens. This is possible but would require an admin to purchase an Apple developer account and go through a number of steps on the Apple side, along with additional config surface on the Fleet side, to allow updating the logo, because it is built into the binary and the binary requires special entitlements from Apple

## Flow diagrams

Actors: the **end user**, **macOS** (the AppSSO framework / `AppSSOAgent`, which holds the device's Secure Enclave keys and orchestrates the flow), the **Fleet PSSO extension** (our in-tree Swift extension), the **Fleet server** (the IdP-translator), and the upstream **IdP** (Okta/Entra). MDM profile delivery and the Secure Enclave appear as notes rather than separate lanes.

The diagram covers the whole lifecycle in three phases: device registration, unlock-key provisioning (both run once, during enrollment or Setup Assistant), and the password sign-in/sync that repeats at each login or unlock. The IdP is contacted only at sign-in; registration establishes no identity.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant macOS as macOS (AppSSO framework)
    participant Ext as Fleet PSSO extension
    participant Fleet as Fleet server
    participant IdP as IdP (Okta/Entra)

    Note over macOS: com.apple.extensiblesso profile installed via MDM<br/>(PlatformSSO: Password, UseSharedDeviceKeys, EnableRegistrationDuringSetup)

    rect rgb(235, 244, 255)
        Note over User,IdP: Phase 1 — Device registration (once; after enrollment or during Setup Assistant)
        macOS->>Ext: beginDeviceRegistration (provides Secure Enclave signing + encryption keys)
        Ext->>Ext: Build payload (device_uuid, device_signing_key, device_encryption_key, signing_key_id, encryption_key_id)
        Ext->>Fleet: POST /api/mdm/apple/psso/registration (direct URLSession)
        Fleet->>Fleet: Resolve host by UUID; store device + key IDs
        Fleet-->>Ext: 200 OK
        Ext-->>macOS: completion(.success)
    end

    rect rgb(235, 255, 240)
        Note over User,IdP: Phase 2 — Provision the offline unlock key (PSSO 2.0)
        macOS->>Fleet: POST /api/mdm/apple/psso/nonce
        Fleet-->>macOS: nonce
        macOS->>Fleet: POST /api/mdm/apple/psso/token (request_type=key_request, signed JWT)
        Fleet->>Fleet: Generate provisioned EC keypair; seal private key into key_context
        Fleet-->>macOS: JWE { certificate (provisioned pubkey), key_context }
        macOS->>Fleet: POST /api/mdm/apple/psso/token (request_type=key_exchange, other_publickey + key_context)
        Fleet->>Fleet: Recover provisioned private key; key = ECDH(private key, other_publickey)
        Fleet-->>macOS: JWE { key } establishes the unlock key
    end

    rect rgb(255, 247, 235)
        Note over User,IdP: Phase 3 — Password sign-in and sync (every login / unlock)
        User->>macOS: Enter IdP password
        macOS->>Fleet: POST /api/mdm/apple/psso/nonce
        Fleet-->>macOS: nonce
        macOS->>Fleet: POST /api/mdm/apple/psso/token (grant_type=password)<br/>signed JWT: plaintext password + jwe_crypto recipe + nonce
        Fleet->>Fleet: Verify JWT signature by kid -> device signing key
        Fleet->>IdP: ROPG grant_type=password (username, password)
        alt password valid
            IdP-->>Fleet: id_token, refresh_token, expires_in
            Fleet->>Fleet: Mint Fleet id_token (ES256); wrap as OAuth JSON;<br/>JWE-encrypt to device encryption key (apu/apv)
            Fleet-->>macOS: platformsso-login-response+jwt (JWE)
            macOS->>Fleet: GET /api/mdm/apple/psso/jwks
            Fleet-->>macOS: JWKS (Fleet signing key)
            macOS->>macOS: Decrypt JWE; verify id_token (sig, nonce, iss, aud, exp)
            macOS->>macOS: Sync local account password to the entered password; start SSO session
            macOS-->>User: Signed in (local password now matches IdP)
        else password invalid
            IdP-->>Fleet: invalid credentials
            Fleet-->>macOS: error
            macOS-->>User: Incorrect username or password
        end
    end
```

## Decision log

### PSSO v2 with Password mode

The extension is registered as `AuthenticationMethod = UserSecureEnclaveKey` v2 with `RegistrationToken`-style flows, configured for **Password** authentication (not `SecureEnclaveKey`, and not v1). Password mode is the only PSSO configuration that surfaces the user's plaintext password to the extension at sign-in, which is what the local-account sync requirement needs. v1 is not considered because it predates the current registration / token-exchange protocol Apple ships in the current `ASAuthorizationProviderExtension*` APIs.

### Identity backend: generic OIDC ROPG, pluggable (Okta-first)

The server defines a `PSSOIdPClient` interface; the first concrete implementation (`PSSOOIDCROPGClient`) speaks OAuth 2.0 Resource Owner Password Grant (`grant_type=password`) against any OIDC IdP. The token URL is taken from `AppConfig.PSSOSettings.idp_token_url`, so the same code path works against Okta, Entra ID, Auth0, Keycloak, or any other ROPG-capable IdP — only the configured URLs change.

The POC is exercised against Okta first because it's a faster path to a working integrator-tier sandbox than provisioning an Entra tenant or a paid G Workspace account. Okta-specific caveats are documented in `tools/psso/README.md`: ROPG must be explicitly enabled on the application, and the app type must be Native or Service.

Additional backends (LDAP bind, direct-trust flows for IdPs that reject ROPG) slot in behind the same `PSSOIdPClient` interface without changes to the PSSO endpoint handlers. The pluggable shape mirrors how the broader Fleet codebase isolates IdP-specific behavior behind an interface.

### Enterprise-gated with no-license core stubs

Route registration for `/api/mdm/apple/psso/*` and the AASA document lives in `server/service/handler.go` (core). The real implementation lives in `ee/server/service/`; the core build provides stubs that return `fleet.ErrMissingLicense`. This matches the pattern already used by `calendar.go` and the enterprise pieces of `apple_mdm.go`.

### Endpoint paths live under /api/mdm/apple/psso

The device-facing PSSO endpoints — `/api/mdm/apple/psso/nonce`, `/api/mdm/apple/psso/registration`, `/api/mdm/apple/psso/token`, and `/api/mdm/apple/psso/jwks` — follow the unversioned device-protocol convention of `/api/mdm/apple/enroll` and are registered on the unauthenticated endpointer (which also caps request body sizes). The JWKS deliberately does not live at `/.well-known/jwks.json`: Apple's framework takes the JWKS URL from the extension's login configuration, so a PSSO-specific path avoids advertising (or colliding with) a server-wide JWKS. Only `/.well-known/apple-app-site-association` remains at root, because Apple's CDN fetches it at a spec-defined absolute path; it stays a raw handler on the root `*http.ServeMux`. (The POC originally served everything at root, SCEP-style — `/mdm/apple/psso/*` — this was revised in #46942.)

### Nonces in Redis, not MySQL

The nonce store mirrors `server/mdm/acme/internal/redis_nonces_store/` and exposes the same minimal surface: `Store(ctx, nonce, ttl)` and `Consume(ctx, nonce) (bool, error)`. Nonces are short-lived and single-use; MySQL would add round-trip cost and migration overhead with no benefit.

### Two MySQL tables

- `mdm_apple_psso_devices` — primary key `host_id`, stores the device's signing and encryption public keys (PEM), the negotiated KeyExchangeKey, and registration/update timestamps.
- `mdm_apple_psso_key_ids` — primary key `kid`, foreign key `host_id`, plus `key_type` and `pem`. The extension references keys by SHA-256 hash of the public key, so the server needs an index keyed by that hash to resolve incoming requests back to a device.

### JWKS signing key bootstrap timing (RESOLVED in #47122)

The signing key is no longer lazily minted. Both it (`MDMAssetPSSOSigningKey`) and the self-signed PSSO CA (`MDMAssetPSSOCACert`, backed by the same private key) are created once, the first time the feature is configured, via `bootstrapPSSOAssets` in `ModifyAppConfig` (covering the config API and GitOps). The bootstrap is idempotent and never regenerates existing assets, so the JWKS key and CA stay stable across reconfiguration and disable/re-enable; the device-facing service methods now only ever load them. Since the feature is still experimental, no upgrade path is provided for POC instances that minted a key under the old lazy path — re-saving the PSSO config generates the CA over the existing key.

### In-tree Swift extension at `apple-sso-extension/`

The Swift sources for the SSO extension live in this repo at `apple-sso-extension/`. Signing and notarization happen out-of-band using the deployer's own Apple Developer ID; Fleet does not ship a signed binary. The hostname declared in the extension's `authsrv:` entitlement must match the hostname served by `/.well-known/apple-app-site-association`.

**Device registration must POST directly (no WKWebView).** `beginDeviceRegistration` submits the device's signing/encryption public keys to `/api/mdm/apple/psso/registration` via a direct `URLSession` POST, and reports `.success` only after Fleet returns 2xx. An earlier implementation routed the POST through a WKWebView navigation-delegate intercept (a holdover from an OAuth-code registration model). That web view isn't functional during Setup Assistant, so with `EnableRegistrationDuringSetup` the POST silently never fired, yet `completion(.success)` was still called unconditionally — the framework then went straight to nonce → token with an unregistered key and the token endpoint 404'd ("PSSOKeyID … not found", surfaced on-device as "Incorrect username or password"). Password-mode registration has no browser step, so the web view was never needed; awaiting the direct POST also guarantees the keys are persisted before the framework proceeds to authentication.

## Known limitations

- **OIDC ROPG has provider-specific limitations.** Okta: ROPG must be explicitly enabled on the application and the app must be Native or Service type. Entra: MFA-required users and federated (AD FS) users cannot authenticate via ROPG. These are upstream constraints, not Fleet bugs. Customers in those configurations need an alternative `PSSOIdPClient` backend (LDAP bind or a direct-trust flow).
- **AASA requires a public-CA TLS certificate.** Apple's framework silently rejects self-signed certificates when fetching `/.well-known/apple-app-site-association`. Local development requires a real DNS name with a Let's Encrypt cert, or a tunnel such as ngrok or cloudflared.
- **No device revocation or key rotation in the POC.** Devices register once and stay registered for the life of the row.
- **Global config only.** PSSO settings live on `AppConfig`; there is no per-team override.
- **Device registration is unauthenticated in the POC.** `POST /api/mdm/apple/psso/registration` accepts any request that presents a device UUID matching an enrolled host plus a set of public keys; nothing proves the request actually originates from that enrolled device. An attacker who can reach the endpoint and knows (or guesses) an enrolled host's hardware UUID could register their own keys for that host. See "Authenticate registration with a per-device token" below for the planned fix.

## Productionizing steps

These are known-required steps to take the POC to a shippable feature. They are intentionally deferred, not forgotten.

### Update apple-app-site-association and associated domains

**Problem.** Right now Apple-App-Site-Associated and AssociatedDomains are hardcoded serverside and in the bundle. Real customers would likely need an AssociatedDomains payload and we'd use different AASA team identifiers, and possibly different bundle identifiers

### Wire up build for CI and distribution

**Problem.** Right now this is configured to build locally and likely only runs on allowlisted developer macs. CI may require changes to build scripts. Distribution may require changes to certificates and entitlements.

### Authenticate registration with a per-device token (Fleet variable)

**Problem.** Registration is currently unauthenticated (see Known limitations). Apple's `com.apple.extensiblesso` payload defines a `RegistrationToken` key — "the token this device uses for registration with Platform SSO ... for silent registration with the Identity Provider" — that exists precisely to close this gap: the MDM server places a secret in the profile, the framework hands it to the extension at registration, and the IdP validates it.

**Planned approach.** Rather than build Fleet-side per-device profile *generation* (the `ensureFleetProfiles` lifecycle, deliberately out of scope for the POC), mint the token through Fleet's existing profile-variable substitution. Introduce a `$FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN` variable that the admin places in the `RegistrationToken` key of the single, manually-uploaded `.mobileconfig`. Fleet substitutes a unique value **per host, at profile-send time**, in `server/mdm/apple/profile_processor.go` — the same point where, and the same way that, the custom-SCEP proxy already mints a per-host challenge via `ds.NewChallenge` and validates it when the device later presents it. PSSO registration is the identical shape: mint/persist a per-host token at send time → substitute → validate at `/register`.

**Benefits beyond authentication.** Because Fleet mints the token against a known `host_id`, the token *is* the host binding. `PSSORegisterComplete` can resolve `host_id` from the token instead of trusting the device-supplied UUID via `HostLiteByIdentifier`, removing a second trust assumption.

**Implementation touch points.**
- `server/fleet/mdm.go` — add the `FleetVarName` constant and its regexp.
- `server/mdm/apple/profile_processor.go` — add a substitution switch case that mints/looks up the per-host token.
- `server/service/apple_mdm.go` — add the variable to `fleetVarsSupportedInAppleConfigProfiles` so upload validation accepts it.
- `ee/server/service/apple_psso.go` — validate the presented token in `PSSORegisterComplete` and derive `host_id` from it.
- Storage + migration, and tests.

**Open decisions.**
- *Storage:* a dedicated table (`mdm_apple_psso_registration_tokens`, PK `host_id`) versus reusing Fleet's generic `challenges` store the way custom-SCEP does. The latter adds essentially no new schema but needs its row shape confirmed to support a clean host binding.
- *Lifecycle:* mint-once-and-reuse (so profile re-sends don't rotate a token mid-flight) versus rotate-on-resend.
- *Consumption:* single-use (consume at registration) versus durable (validate without consuming, friendlier to re-registration).

### Admin configuration: UI and live reload of IdP settings

**Problem.** PSSO configuration currently has two POC-only rough edges that customers won't accept:

1. **No admin UI.** PSSO settings live on `AppConfig.PSSOSettings` and are set either by hand-editing the `app_config_json` row or via `PATCH /api/v1/fleet/config`. There is no UI surface. Customers expect to configure the IdP connection (token/authorize URLs, client ID/secret, scopes) from the Fleet console like every other integration, with the client secret masked on read (the `IdPClientSecret` masking TODO in `server/fleet/apple_psso.go` is part of this).

2. **The IdP client is built once at boot.** `cmd/fleet/serve.go` constructs `PSSOOIDCROPGClient` from `appCfg.PSSOSettings` during server startup and wires it via `SetPSSOIdPClient`. Changing PSSO settings — even through the API — has no effect until the server is restarted, and if settings are absent at boot the client stays nil and the token endpoint fails with "psso idp client not configured." A self-hosted admin can restart; a Fleet Cloud customer cannot, and nobody should have to.

**Planned approach.** Resolve the IdP client (and nonce store, if it grows config) from the current `AppConfig` at request time rather than caching a single instance at boot — e.g. construct it per call from the live config, or cache it behind the existing app-config change signal so edits take effect immediately. Pair that with a PSSO settings page in the console and secret masking on the config API. This removes the restart requirement and the hand-edited-SQL workflow entirely.

**Live reload addressed in #46942.** The OIDC ROPG client is now built from the current `AppConfig` on every password login, and the boot-time `SetPSSOIdPClient` wiring was removed from `serve.go` (the setter remains as a test hook). The admin UI and secret masking remain with #46959 / #47127.

### LDAP identity backend (Google Workspace Secure LDAP)

**Problem / motivation.** The POC validates passwords via OIDC ROPG, but **Google Workspace does not support the OAuth ROPG (`grant_type=password`) flow at all** — so there is no OIDC path to validate a Google user's password server-side. Google's supported mechanism for that is **Secure LDAP**. Adding an LDAP backend therefore isn't just an alternative to ROPG; it's what unlocks Google Workspace as an IdP. The same backend also covers classic LDAP/Active Directory for customers who prefer a directory bind over ROPG.

**Planned approach.** Add a second `PSSOIdPClient` implementation — nothing else moves. The interface (`ValidatePasswordAndGetClaims(ctx, username, password) (*PSSOClaims, error)`) already isolates the backend from the PSSO protocol, the JWE/JWT crypto, the endpoints, the Fleet-minted id_token, the key request/exchange, and the device side; all of that is unchanged. The new client dials LDAPS, locates the user (search by `mail`/`uid` under the base DN), binds as that user with the supplied password to verify it, and maps directory attributes to `PSSOClaims`.

**Implementation touch points.**
- `ee/server/service/apple_psso_idp_ldap.go` — new `PSSOLDAPClient` implementing the interface (search-then-bind; ~150–250 lines). Adds an LDAP library dependency (`github.com/go-ldap/ldap/v3` — confirm it isn't already vendored; Fleet does not appear to use LDAP today).
- `server/fleet/apple_psso.go` — add an `IdPType` discriminator (`oidc_ropg` | `ldap`) to `PSSOSettings` and an `LDAP *PSSOLDAPSettings` block (`ServerURL`, `BaseDN`, `UserSearchAttr`, attribute→claim map, and the directory-auth material — see below).
- `ee/server/service/apple_psso.go` — `pssoIdPClientFromSettings` switches on `IdPType` instead of always constructing `PSSOOIDCROPGClient`. (The client is already built per request from live settings here, so no `serve.go` wiring is involved.)
- Secret storage + masking — the Google client certificate/key (and any service bind password) are directory-wide credentials; encrypt at rest via the `mdm_config_assets` pattern and mask on the config API (same write-path work as the IdPClientSecret finding).
- Tests (integrate against glauth/OpenLDAP or a mocked connection) and a Google Admin console setup doc.

**Google Secure LDAP specifics.**
- LDAP support of any flavor has been deferred to a later release
- Endpoint `ldaps://ldap.google.com:636`, TLS only.
- **Directory authentication is mutual TLS, not a bind password.** An "LDAP client" is created in the Google Admin console, which issues a client certificate + private key that Fleet presents (`tls.Config.Certificates`). This is the main structural difference from classic LDAP/AD, which uses a service bind DN + password — so the LDAP settings should accommodate both directory-auth styles.
- The Admin console LDAP client must be granted access to the relevant OUs and permission to verify user credentials; the base DN derives from the domain (e.g. `dc=example,dc=com`).
- The exact bind/DN mechanics should be confirmed against Google's Secure LDAP documentation before implementing — that is the least-certain part of this plan.

**Open decisions.**
- *Directory-auth model:* support Google mTLS (client cert) and classic service-bind (DN + password) behind one config shape, or ship Google-only first.
- *Attribute mapping & stable subject:* which attribute maps to `sub` must be stable across logins, since the device keys identity on it (`uniqueIdentifierClaimName = "sub"`); `mail` or a directory GUID are candidates.
- *Connection handling:* per-request dial (simplest, fine at sign-in frequency) vs. a pooled connection.

**Limitations to document.**
- **No refresh token / silent renewal.** LDAP has no `refresh_token`/`expires_in`; `PSSOClaims` already treats those as optional and `handlePSSOPasswordLogin` degrades gracefully (mints an opaque token, default TTL), but silent SSO renewal can't happen without the password — renewal requires a re-prompt. Acceptable (PSSO re-authenticates periodically) but degraded vs. OIDC.
- **MFA bypass.** A raw LDAP bind ignores MFA/conditional access, the same limitation class as the ROPG caveat above.

## Security review findings

A security review of the POC (covering the implementation and these productionizing plans) produced the findings below. They are ordered by severity and tagged **[deploy]** (must fix before any real-world deployment, including pilots) or **[GA]** (POC-acceptable, fix before general availability). Items that overlap an existing Productionizing/Known-limitations entry are cross-referenced.

The crypto was otherwise found sound: no passwords/refresh-tokens/client-secrets in logs or errors; SQL fully parameterized; JWE GCM nonces random with the protected header as AAD; `canonicalizeKID` consistent across store and lookup (no key aliasing); attacker-supplied `other_publickey` is curve-validated via `crypto/ecdh`; clean-room provenance intact (JOSE primitives only, no third-party PSSO SDK).

### CRITICAL [deploy] — IdP client secret disclosed via the config API

`AppConfig.Obfuscate()` (`server/fleet/app.go`) masks SMTP/Jira/Zendesk/etc. secrets but has no case for `PSSOSettings`, so `GET /api/v1/fleet/config` returns `psso_settings.idp_client_secret` in cleartext to any caller with config read. A low-privilege user could lift the upstream IdP OAuth client credentials and use them directly against the customer's tenant. This is a live disclosure on the existing endpoint, not the cosmetic "mask in the UI" task framed under *Admin configuration* above. Fix: add a `PSSOSettings` case to `Obfuscate()`, and mirror the SMTP "keep existing secret when the client submits the mask" logic on the config write path so a PATCH echoing `********` doesn't clobber the stored secret.

### HIGH [deploy] — `PSSOSettings.Enabled` is never enforced

No PSSO service method consults `cfg.PSSOSettings.Enabled`; the unauthenticated `/mdm/apple/psso/*` surface is live on every licensed instance even when an admin never enabled (or explicitly disabled) PSSO. Gate the device-facing methods (`PSSONonce`, `PSSORegisterComplete`, `PSSOToken`, and arguably JWKS/AASA) on `Enabled` in the service layer so all entry points are covered.

**Addressed in #46942.** Every PSSO service method now checks the live `AppConfig.PSSOSettings` per request (`pssoSettingsIfConfigured`): nonce/registration/token return 400 and JWKS/AASA return 404 when the feature is disabled or incompletely configured.

### HIGH [deploy] — Unbounded replay of token requests

The verified JWT parse validates `exp`/`nbf` only if present, and inbound request JWTs are not required to carry an `exp` (nor is one enforced). The sole anti-replay control, `request_nonce`, is consumed best-effort — a miss is logged, not rejected (`ee/server/service/apple_psso.go`, `handlePSSOPasswordLogin`). A captured login-request JWS can therefore be replayed indefinitely to re-trigger IdP password validation and yield a fresh valid login-response JWE. This supersedes the milder "best-effort nonce, fix before GA" framing — the practical state is unbounded replay of a credential-validating request. Fix: require a short-lived `exp` (and an `iat` max-age) on inbound JWTs, and hard-enforce single-use `request_nonce` (reject when the store rejects).

**Partially addressed in #46942.** `request_nonce` is now hard-enforced and consumed before dispatch for all token flows (password login, key request, key exchange) — a replayed JWS is rejected. Requiring `exp`/`iat` max-age on inbound JWTs remains open (#47122 covers JWT validation cleanup).

### HIGH [deploy] — Key replacement on registration enables device takeover

Compounds the unauthenticated-registration limitation. `SetOrUpdatePSSODevice` (`server/datastore/mysql/apple_psso.go`) deletes a host's existing `key_ids` and inserts the caller's on a plain upsert. The `IOPlatformUUID` is not secret (it appears in osquery results, MDM inventory, logs), so an unauthenticated attacker who knows it can *overwrite* a legitimate device's registered signing/encryption keys and then drive `/token` as that host. The planned per-device registration token closes the spoofing primitive, but the key-replacement semantics are a separate decision: once a host has a registration, require the per-host token to match before replacing keys, and log/emit an activity on key replacement (rotation vs. takeover).

### HIGH [GA] — Inbound JWT algorithm not pinned

`parsePSSOInboundJWT` (`ee/server/service/apple_psso_crypto.go`) calls `jwt.ParseWithClaims` without `jwt.WithValidMethods` and the keyfunc returns the EC key without asserting `token.Method`. Not exploitable as written (golang-jwt v4's type assertions reject HS/`none` against an `*ecdsa.PublicKey`), but it is one refactor away from an alg-confusion forgery. Fix: pass `jwt.WithValidMethods([]string{"ES256"})` and assert `*jwt.SigningMethodECDSA` in the keyfunc. Cheap hardening.

**Addressed in #47122.** `parsePSSOInboundJWT` now passes `jwt.WithValidMethods([]string{"ES256"})` and asserts `*jwt.SigningMethodECDSA` in the keyfunc; HS256 and `none` tokens are rejected.

### MEDIUM [deploy] — ROPG client is an SSRF / credential-redirection sink

`idp_token_url` comes from admin-controlled `AppConfig` and is POSTed to with the user's plaintext password. There is no scheme/host validation, so whoever can edit config (or a future settings UI lacking validation) can point it at an internal address and harvest passwords. Validate `https://` and a non-internal host at config-write time, and confirm `fleethttp.NewClient()` enforces TLS verification for this client. Pair this validation with the live-reload work under *Admin configuration*.

### MEDIUM [GA → now-active] — `key_context` is not bound to device or expiry

The provisioned private key sealed into `key_context` (key request) is recoverable by any registered device replaying any captured `key_context`, and the blob carries no TTL (its payload `exp` is advisory and not re-checked on exchange). It is also sealed under a key HKDF-derived from the long-lived PSSO signing key, so signing-key rotation/re-mint silently invalidates all outstanding contexts, and a signing-key leak compromises every context ever issued (no forward secrecy). **Note:** the review rated this deferrable on the assumption the key-request/key-exchange path was not exercised by the Password flow — that is no longer true; the unlock-key exchange now runs during Password-mode registration, so treat this as active. Fix: bind `key_context` to the device `kid` and an expiry inside the sealed plaintext (e.g. as AAD), and reject on open if mismatched or expired.

**Device binding addressed in #47122.** `key_context` now seals a structured JSON plaintext — `{host_uuid, key_purpose, provisioned_key}` — instead of the bare private key, and key exchange rejects when the sealed `host_uuid` doesn't match the host resolved from the request's signing key (or when `key_purpose` isn't `user_unlock`). A captured context replayed by, or fetched onto, another device is rejected. An in-blob expiry was considered but deliberately left out for now to match the issue's specified shape; the forward-secrecy / signing-key-coupling concerns remain open.

### MEDIUM [GA] — Ad-hoc CA certificate issuance is sloppy

`issuePSSOProvisionedCertificate` (`ee/server/service/apple_psso.go`) regenerates a self-signed, unconstrained, 10-year signing CA on every key request with fixed serial `1`. Generate the CA once (persist alongside the signing key), use random serials for both CA and leaf, and add EKU/name constraints scoping its use.

**Addressed in #47122.** The CA is now minted once at first configuration and persisted (`MDMAssetPSSOCACert`); `issuePSSOProvisionedCertificate` loads it and signs each leaf with a random 128-bit serial. The CA keeps serial `1` (matching Fleet's other self-signed CA roots in `server/mdm/scep/depot`): a singular, persisted self-signed root produces exactly one certificate, so the serial is unique by construction — the random-serial recommendation applied to the per-request *leaf*, which it now uses. The CA carries `BasicConstraintsValid`, `IsCA`, `MaxPathLen: 0`, and a SubjectKeyId.

### LOW [GA] — Hardcoded developer Team/bundle IDs in the AASA

`teamID*`/`bundleID*` constants (`ee/server/service/apple_psso.go`) are baked into the public, always-served `apple-app-site-association`. They leak Fleet-developer identifiers and mis-bind for any deployer using their own signing identity. Make them config-driven before GA.

### INFORMATIONAL — Upstream `id_token` signature is not verified

`parseOIDCIDTokenClaims` reads the IdP `id_token` without signature verification. Acceptable: it arrives directly from the IdP over TLS in the same response (a structured response, not a cross-trust assertion) — *provided* the SSRF/TLS hardening above holds.

## Pointers

- Apple WWDC sessions: *Platform SSO for macOS* (WWDC 2022), and the *Discover authentication services* / *Shared device keys* material (WWDC 2023).
- Apple developer documentation for the `ASAuthorizationProviderExtension*` family of classes and protocols.
