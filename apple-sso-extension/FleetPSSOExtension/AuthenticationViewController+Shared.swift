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

    func registrationPayload(signing: SecKey, encryption: SecKey) -> [String: String] {
        [
            "deviceUUID": deviceUUID(),
            "signPubKey": pemRepresentation(of: signing),
            "encPubKey": pemRepresentation(of: encryption),
            "signKeyID": keyID(signing),
            "encKeyID": keyID(encryption),
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

    func applyLoginConfiguration(
        _ mgr: ASAuthorizationProviderExtensionLoginManager
    ) throws {
        let data = mgr.extensionData
        guard let issuer = data["IssuerHostname"] as? String,
              let token = (data["TokenEndpoint"] as? String).flatMap(URL.init(string:)),
              let jwks = (data["JwksEndpoint"] as? String).flatMap(URL.init(string:)),
              let nonce = (data["NonceEndpoint"] as? String).flatMap(URL.init(string:)),
              let reg = (data["RegistrationEndpoint"] as? String).flatMap(URL.init(string:))
        else { throw NSError(domain: "FleetPSSO", code: -1) }
        let cfg = ASAuthorizationProviderExtensionLoginConfiguration(
            clientID: Bundle.main.bundleIdentifier ?? "",
            issuer: issuer,
            tokenEndpointURL: token,
            jwksEndpointURL: jwks,
            audience: issuer)
        cfg.nonceEndpointURL = nonce
        self.registrationEndpointURL = reg
        try mgr.saveLoginConfiguration(cfg)
    }

    func registrationStartURL(
        _ mgr: ASAuthorizationProviderExtensionLoginManager,
        payload: [String: String]
    ) -> URL? {
        guard let base = registrationEndpointURL,
              var comps = URLComponents(url: base, resolvingAgainstBaseURL: false)
        else { return nil }
        comps.queryItems = payload.map { URLQueryItem(name: $0.key, value: $0.value) }
        return comps.url
    }
}
