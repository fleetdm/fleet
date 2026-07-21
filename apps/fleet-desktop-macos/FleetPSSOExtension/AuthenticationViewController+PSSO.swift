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
import os
import Security

@available(macOS 14.0, *)
extension AuthenticationViewController:
    ASAuthorizationProviderExtensionRegistrationHandler {

    func beginDeviceRegistration(
        loginManager: ASAuthorizationProviderExtensionLoginManager,
        options: ASAuthorizationProviderExtensionRequestOptions,
        completion: @escaping (ASAuthorizationProviderExtensionRegistrationResult) -> Void
    ) {
        logger.log("beginDeviceRegistration options=\(options.rawValue, privacy: .public)")
        self.loginManager = loginManager
        // A repair means the framework is recovering from bad registration
        // state; minting fresh keys keeps the keychain, the rebuilt device
        // configuration, and the server in lockstep instead of re-registering
        // handles that may be part of the broken state.
        if options.contains(.registrationRepair) {
            logger.log("beginDeviceRegistration: repair requested, resetting device keys")
            loginManager.resetDeviceKeys()
        }
        guard let signKey = loginManager.key(for: .sharedDeviceSigning),
              let encKey = loginManager.key(for: .sharedDeviceEncryption) else {
            logger.error("beginDeviceRegistration: shared device keys unavailable")
            completion(.failed)
            return
        }
        guard let registrationToken = loginManager.registrationToken, !registrationToken.isEmpty else {
            logger.error("beginDeviceRegistration: no registration token in profile")
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
                logger.error("beginDeviceRegistration: applyLoginConfiguration failed: \(String(describing: error), privacy: .public)")
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
                logger.error("beginDeviceRegistration: key export failed, refusing incomplete payload")
                completion(.failed)
                return
            }
            let ok = await self.postDeviceRegistration(payload: payload)
            logger.log("beginDeviceRegistration: completing with \(ok ? "success" : "failed", privacy: .public)")
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
        logger.log("beginUserRegistration options=\(options.rawValue, privacy: .public) method=\(method.rawValue, privacy: .public) hasUserName=\(userName?.isEmpty == false, privacy: .public)")
        // Persist the user login configuration. Without this the framework
        // reports "no user configuration for user" and never finishes binding
        // the PSSO user to the local account, so the unlock-key/SecureToken
        // setup stays incomplete and key unwrap fails at login ("previous
        // password required"). For password mode saving the config is all the
        // extension needs to do.
        //
        // Background repair runs can arrive without a userName; the user login
        // configuration saved by the original registration still names the
        // registered user, so fall back to it rather than failing the whole
        // registration cycle.
        let userNameParam = userName?.isEmpty == false ? userName : nil
        guard let resolvedUserName = userNameParam ?? loginManager.userLoginConfiguration?.loginUserName,
              !resolvedUserName.isEmpty else {
            if options.contains(.userInteractionEnabled) {
                logger.error("beginUserRegistration: no user name available")
                completion(.failed)
            } else {
                logger.log("beginUserRegistration: no user name and interaction disabled, deferring to UI retry")
                completion(.userInterfaceRequired)
            }
            return
        }
        let config = ASAuthorizationProviderExtensionUserLoginConfiguration(loginUserName: resolvedUserName)
        do {
            try loginManager.saveUserLoginConfiguration(config)
        } catch {
            logger.error("beginUserRegistration: saveUserLoginConfiguration failed: \(String(describing: error), privacy: .public)")
            completion(.failed)
            return
        }
        logger.log("beginUserRegistration: user login configuration saved")
        completion(.success)
    }

    func registrationDidComplete() {
        logger.log("registrationDidComplete")
    }

    func protocolVersion() -> ASAuthorizationProviderExtensionPlatformSSOProtocolVersion {
        .version2_0
    }

    func supportedGrantTypes() -> ASAuthorizationProviderExtensionSupportedGrantTypes {
        .password
    }
}
