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
        guard let signKey = loginManager.key(for: .userDeviceSigning),
              let encKey = loginManager.key(for: .userDeviceEncryption) else {
            completion(.failed)
            return
        }
        guard let registrationToken = loginManager.registrationToken, !registrationToken.isEmpty else {
            completion(.failed)
            return
        }
        // applyLoginConfiguration fetches the server's encryption key over HTTP,
        // so it runs on the Task alongside the registration POST. Report success
        // only once Fleet has stored the keys, so the framework can't proceed to
        // authentication with an unregistered key (which 404s at the token
        // endpoint). This is what makes the Setup Assistant flow work.
        Task {
            do {
                try await self.applyLoginConfiguration(loginManager)
            } catch {
                completion(.failed)
                return
            }
            let payload = self.registrationPayload(
                signing: signKey,
                encryption: encKey,
                registrationToken: registrationToken)
            let ok = await self.postDeviceRegistration(payload: payload)
            completion(ok ? .success : .failed)
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
