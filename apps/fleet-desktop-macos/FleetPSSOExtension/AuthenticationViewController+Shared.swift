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

    func keyID(_ key: SecKey) -> String {
        let digest = SHA256.hash(data: derRepresentation(of: key))
        return Data(digest).base64URLEncodedString()
    }

    func derRepresentation(of key: SecKey) -> Data {
        guard let pub = SecKeyCopyPublicKey(key),
              let data = SecKeyCopyExternalRepresentation(pub, nil) as Data? else {
            return Data()
        }
        return data
    }

    func pemRepresentation(of key: SecKey) -> String {
        let der = derRepresentation(of: key)
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
    // It also fetches Fleet's JWKS and, if an encryption key is published,
    // sets it as loginRequestEncryptionPublicKey. macOS then encrypts the
    // password into the login assertion (ECDH-ES/A256GCM) so it can't be read
    // by anything able to terminate TLS. If the server publishes no encryption
    // key, the property is left unset.
    func applyLoginConfiguration(
        _ mgr: ASAuthorizationProviderExtensionLoginManager
    ) async throws {
        let data = mgr.extensionData
        guard let baseString = data["BaseURL"] as? String,
              let base = URL(string: baseString),
              let host = base.host
        else { throw NSError(domain: "FleetPSSO", code: -1) }
        let cfg = ASAuthorizationProviderExtensionLoginConfiguration(
            clientID: Bundle.main.bundleIdentifier ?? "",
            issuer: host,
            tokenEndpointURL: pssoEndpointURL(base, "token"),
            jwksEndpointURL: pssoEndpointURL(base, "jwks"),
            audience: host)
        cfg.nonceEndpointURL = pssoEndpointURL(base, "nonce")
        self.registrationEndpointURL = pssoEndpointURL(base, "registration")
        if let encryptionKey = await loginRequestEncryptionKey(jwksURL: pssoEndpointURL(base, "jwks")) {
            cfg.loginRequestEncryptionPublicKey = encryptionKey
        }
        try mgr.saveLoginConfiguration(cfg)
    }

    private func pssoEndpointURL(_ base: URL, _ name: String) -> URL {
        base.appendingPathComponent("api/mdm/apple/psso/\(name)")
    }
}
