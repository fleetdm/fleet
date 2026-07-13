// AuthenticationViewController+PSSO.swift
// FleetPSSOExtension
//
// ASAuthorizationProviderExtensionRegistrationHandler conformance. The
// framework hands us a login manager; we ask it for the shared device
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
        guard let signKey = loginManager.key(for: .sharedDeviceSigning),
              let encKey = loginManager.key(for: .sharedDeviceEncryption) else {
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
            // A failed key export leaves empty PEM/KID fields (see keyID /
            // pemRepresentation). Refuse to register an incomplete payload
            // instead of POSTing keys the server can only reject.
            let requiredFields = ["device_signing_key", "device_encryption_key",
                                  "signing_key_id", "encryption_key_id"]
            guard requiredFields.allSatisfy({ !(payload[$0] ?? "").isEmpty }) else {
                completion(.failed)
                return
            }
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
        // Persist the user login configuration. Without this the framework
        // reports "no user configuration for user" and never finishes binding
        // the PSSO user to the local account, so the unlock-key/SecureToken
        // setup stays incomplete and key unwrap fails at login ("previous
        // password required"). For password mode saving the config is all the
        // extension needs to do.
        guard let userName, !userName.isEmpty else {
            completion(.failed)
            return
        }
        let config = ASAuthorizationProviderExtensionUserLoginConfiguration(loginUserName: userName)
        do {
            try loginManager.saveUserLoginConfiguration(config)
        } catch {
            completion(.failed)
            return
        }
        completion(.success)
    }

    func protocolVersion() -> ASAuthorizationProviderExtensionPlatformSSOProtocolVersion {
        .version2_0
    }

    func supportedGrantTypes() -> ASAuthorizationProviderExtensionSupportedGrantTypes {
        .password
    }
}
