// AuthenticationViewController+Shared.swift
// FleetPSSOExtension
//
// Shared helpers: registration payload construction, key-ID derivation
// (base64url SHA-256 of public-key DER), device UUID lookup, and login
// configuration setup from the extensionData dictionary supplied by the
// com.apple.extensiblesso configuration profile.

import AuthenticationServices
import CryptoKit
import Foundation
import IOKit
import Security

@available(macOS 14.0, *)
extension AuthenticationViewController {

    // registrationToken is provided by the Fleet Server in the profile's RegistrationToken key;
    // As of writing, Fleet always requires it to register a device and derives the host identity
    // from it (device_uuid is sent only for diagnostics).
    func registrationPayload(signing: SecKey, encryption: SecKey, registrationToken: String) -> [String: String] {
        [
            "device_uuid": deviceUUID(),
            "device_signing_key": pemRepresentation(of: signing),
            "device_encryption_key": pemRepresentation(of: encryption),
            "signing_key_id": keyID(signing),
            "encryption_key_id": keyID(encryption),
            "registration_token": registrationToken,
        ]
    }

    // keyID and pemRepresentation return "" when the key can't be exported.
    // Registration treats an empty field as fatal (see beginDeviceRegistration)
    // rather than submitting a payload the server can only reject — and never
    // hashing empty data into a KID shared by every device that hit the failure.
    func keyID(_ key: SecKey) -> String {
        guard let der = derRepresentation(of: key) else { return "" }
        let digest = SHA256.hash(data: der)
        return Data(digest).base64URLEncodedString()
    }

    func derRepresentation(of key: SecKey) -> Data? {
        guard let pub = SecKeyCopyPublicKey(key),
              let data = SecKeyCopyExternalRepresentation(pub, nil) as Data? else {
            return nil
        }
        return data
    }

    func pemRepresentation(of key: SecKey) -> String {
        guard let der = derRepresentation(of: key) else { return "" }
        let b64 = der.base64EncodedString(options: [.lineLength64Characters,
                                                    .endLineWithLineFeed])
        return "-----BEGIN PUBLIC KEY-----\n\(b64)\n-----END PUBLIC KEY-----"
    }

    func deviceUUID() -> String {
        let svc = IOServiceGetMatchingService(kIOMainPortDefault,
                                              IOServiceMatching("IOPlatformExpertDevice"))
        defer { IOObjectRelease(svc) }
        let key = "IOPlatformUUID" as CFString
        guard let raw = IORegistryEntryCreateCFProperty(svc, key, kCFAllocatorDefault, 0),
              let uuid = raw.takeRetainedValue() as? String else { return "" }
        return uuid
    }

    // applyLoginConfiguration derives every endpoint from the single BaseURL
    // key in the profile's ExtensionData — the Fleet server URL, e.g.
    // https://fleet.example.com. The issuer/audience is its bare hostname,
    // matching the `iss` claim Fleet mints into login-response id_tokens.
    //
    // It also fetches Fleet's JWKS and sets the published encryption key as
    // loginRequestEncryptionPublicKey, so macOS encrypts the password into the
    // login assertion (ECDH-ES/A256GCM) and it can't be read by anything able to
    // terminate TLS. Fleet always publishes this key, so a failure to load it
    // fails registration rather than silently sending the password TLS-only.
    // BaseURL must be HTTPS — every derived endpoint carries key material.
    func applyLoginConfiguration(
        _ mgr: ASAuthorizationProviderExtensionLoginManager
    ) async throws {
        let data = mgr.extensionData
        guard let baseString = data["BaseURL"] as? String,
              let base = URL(string: baseString),
              let host = base.host,
              base.scheme?.lowercased() == "https"
        else { throw NSError(domain: "FleetPSSO", code: -1) }
        let cfg = ASAuthorizationProviderExtensionLoginConfiguration(
            clientID: Bundle.main.bundleIdentifier ?? "",
            issuer: host,
            tokenEndpointURL: pssoEndpointURL(base, "token"),
            jwksEndpointURL: pssoEndpointURL(base, "jwks"),
            audience: host)
        cfg.nonceEndpointURL = pssoEndpointURL(base, "nonce")
        // Fleet dispatches key_request/key_exchange (the unlock-key flow) at the
        // token endpoint. The framework needs keyEndpointURL set explicitly to
        // engage that plumbing — leaving it unset relies on an undocumented
        // default.
        cfg.keyEndpointURL = pssoEndpointURL(base, "token")
        self.registrationEndpointURL = pssoEndpointURL(base, "registration")
        guard let encryptionKey = await loginRequestEncryptionKey(jwksURL: pssoEndpointURL(base, "jwks")) else {
            throw NSError(domain: "FleetPSSO", code: -2)
        }
        cfg.loginRequestEncryptionPublicKey = encryptionKey
        try mgr.saveLoginConfiguration(cfg)
    }

    private func pssoEndpointURL(_ base: URL, _ name: String) -> URL {
        base.appendingPathComponent("api/mdm/apple/psso/\(name)")
    }
}
