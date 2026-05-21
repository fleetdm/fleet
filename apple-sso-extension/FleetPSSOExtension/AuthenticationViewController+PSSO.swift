// AuthenticationViewController+PSSO.swift
// FleetPSSOExtension
//
// ASAuthorizationProviderExtensionRegistrationHandler conformance. The
// framework hands us a login manager; we ask it for the user device
// signing + encryption keys, build a registration payload, and configure
// the SSO endpoints from extensionData supplied by the configuration
// profile. Apple owns the private key material — we only see SecKey
// handles and derive public PEMs from them.

import AuthenticationServices
import CryptoKit
import Foundation
import IOKit
import Security

@available(macOS 14.0, *)
extension AuthenticationViewController:
    ASAuthorizationProviderExtensionRegistrationHandler {

    func beginDeviceRegistration(
        loginManager: ASAuthorizationProviderExtensionLoginManager,
        options: ASAuthorizationProviderExtensionRequestOptions,
        completion: @escaping (ASAuthorizationProviderExtensionRegistrationResult) -> Void
    ) {
        self.loginManager = loginManager
        do {
            try applyLoginConfiguration(loginManager)
            guard let signKey = loginManager.key(for: .userDeviceSigning),
                  let encKey = loginManager.key(for: .userDeviceEncryption) else {
                completion(.failed)
                return
            }
            let payload = registrationPayload(signing: signKey, encryption: encKey)
            if let url = registrationStartURL(loginManager, payload: payload) {
                webView.load(URLRequest(url: url))
            }
            completion(.success)
        } catch {
            completion(.failed)
        }
    }

    func beginUserRegistration(
        loginManager: ASAuthorizationProviderExtensionLoginManager,
        userName: String?,
        method: ASAuthorizationProviderExtensionAuthenticationMethod,
        options: ASAuthorizationProviderExtensionRequestOptions,
        completion: @escaping (ASAuthorizationProviderExtensionRegistrationResult) -> Void
    ) {
        completion(.success)
    }

    func protocolVersion() -> ASAuthorizationProviderExtensionPlatformSSOProtocolVersion {
        .version2_0
    }

    func supportedGrantTypes() -> ASAuthorizationProviderExtensionSupportedGrantTypes {
        .password
    }
}
