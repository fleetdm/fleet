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

extension AuthenticationViewController {

    func registrationPayload(signing: ASAuthorizationProviderExtensionKey,
                             encryption: ASAuthorizationProviderExtensionKey) -> [String: String] {
        [
            "deviceUUID": deviceUUID(),
            "signPubKey": signing.publicKey.pemRepresentation,
            "encPubKey": encryption.publicKey.pemRepresentation,
            "signKeyID": keyID(signing),
            "encKeyID": keyID(encryption),
        ]
    }

    func keyID(_ key: ASAuthorizationProviderExtensionKey) -> String {
        let digest = SHA256.hash(data: key.publicKey.derRepresentation)
        return Data(digest).base64URLEncodedString()
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
            jwksEndpointURL: jwks)
        cfg.nonceEndpointURL = nonce
        cfg.registrationEndpointURL = reg
        cfg.protocolVersion = .version2_0
        cfg.supportedGrantTypes = [.password]
        try mgr.setLoginConfiguration(cfg)
    }

    func registrationEndpoint() -> URL? {
        (loginManager?.extensionData["RegistrationEndpoint"] as? String)
            .flatMap(URL.init(string:))
    }

    func registrationStartURL(
        _ mgr: ASAuthorizationProviderExtensionLoginManager,
        payload: [String: String]
    ) -> URL? {
        guard let base = registrationEndpoint(),
              var comps = URLComponents(url: base, resolvingAgainstBaseURL: false)
        else { return nil }
        comps.queryItems = payload.map { URLQueryItem(name: $0.key, value: $0.value) }
        return comps.url
    }
}
