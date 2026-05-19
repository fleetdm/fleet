// AuthenticationViewController+PSSO.swift
// FleetPSSOExtension
//
// ASAuthorizationProviderExtensionRegistrationHandler conformance. The
// framework hands us a login manager; we ask it for the user device
// signing + encryption keys, build a registration payload, and configure
// the SSO endpoints from extensionData supplied by the configuration
// profile. Apple owns the private key material — we only see public PEMs.

import AuthenticationServices
import CryptoKit
import Foundation
import IOKit

extension AuthenticationViewController:
    ASAuthorizationProviderExtensionRegistrationHandler {

    func beginDeviceRegistration(
        using loginManager: ASAuthorizationProviderExtensionLoginManager,
        options: ASAuthorizationProviderExtensionRequestOptions,
        viewController: NSViewController?,
        completion: @escaping (ASAuthorizationProviderExtensionRegistrationResult) -> Void
    ) {
        self.loginManager = loginManager
        do {
            try applyLoginConfiguration(loginManager)
            let signKey = try requestKey(loginManager, type: .userDeviceSigning)
            let encKey = try requestKey(loginManager, type: .userDeviceEncryption)
            let payload = registrationPayload(signing: signKey, encryption: encKey)
            if let url = registrationStartURL(loginManager, payload: payload) {
                webView.load(URLRequest(url: url))
            }
            completion(.success)
        } catch {
            completion(.failed)
        }
    }

    func beginUserSecureEnclaveKeyRegistration(
        using loginManager: ASAuthorizationProviderExtensionLoginManager,
        options: ASAuthorizationProviderExtensionRequestOptions,
        viewController: NSViewController?,
        completion: @escaping (ASAuthorizationProviderExtensionRegistrationResult) -> Void
    ) {
        completion(.success)
    }

    private func requestKey(
        _ mgr: ASAuthorizationProviderExtensionLoginManager,
        type: ASAuthorizationProviderExtensionKeyType
    ) throws -> ASAuthorizationProviderExtensionKey {
        try mgr.userDeviceKey(forKeyType: type)
    }
}
