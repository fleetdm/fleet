// AuthenticationViewController+UserRegistration.swift
// FleetPSSOExtension
//
// Interactive user registration. macOS reaches this path when it registers a
// user it has never authenticated — an already-set-up Mac whose owner clicked
// the "Registration required" banner. Setup Assistant registrations don't come
// here: there the framework has already run the IdP login and passes the
// username in.
//
// The extension presents its own sign-in form, verifies the credentials by
// sending Fleet the same signed login request AppSSOAgent would, and only
// then saves the user login configuration. Saving an unverified username
// would leave the account bound to an identity that fails at every unlock.

import AuthenticationServices
import Cocoa
import os

enum UserRegistrationError: Error {
    case invalidCredentials
    case server(status: Int)
    case network(Error)
    case internalFailure(String)

    var userMessage: String {
        switch self {
        case .invalidCredentials:
            return "Incorrect username or password. Try again."
        case .server(let status):
            return "Fleet couldn't verify your credentials (HTTP \(status)). Try again later."
        case .network:
            return "Couldn't connect to Fleet. Check your network connection and try again."
        case .internalFailure:
            return "Something went wrong. Try again later."
        }
    }
}

@available(macOS 14.0, *)
extension AuthenticationViewController {

    func beginInteractiveUserRegistration(
        loginManager: ASAuthorizationProviderExtensionLoginManager,
        completion: @escaping (ASAuthorizationProviderExtensionRegistrationResult) -> Void
    ) {
        DispatchQueue.main.async {
            self.userRegistrationCompletion = completion
            loginManager.presentRegistrationViewController { [weak self] error in
                DispatchQueue.main.async {
                    guard let self else { return }
                    if let error {
                        logger.error("beginUserRegistration: presentRegistrationViewController failed: \(String(describing: error), privacy: .public)")
                        self.finishUserRegistration(.failed)
                        return
                    }
                    self.showRegistrationForm(loginManager: loginManager)
                }
            }
        }
    }

    func discardRegistrationForm() {
        registrationForm?.removeFromSuperview()
        registrationForm = nil
    }

    // abandonUserRegistration drops everything an in-progress interactive
    // registration holds. Cancelling the task matters: a verification that
    // finishes after the framework cancelled must not save a configuration or
    // complete the abandoned request.
    func abandonUserRegistration() {
        verificationTask?.cancel()
        verificationTask = nil
        userRegistrationCompletion = nil
        discardRegistrationForm()
    }

    private func showRegistrationForm(loginManager: ASAuthorizationProviderExtensionLoginManager) {
        discardRegistrationForm()
        let form = RegistrationFormView(initialUserName: loginManager.userLoginConfiguration?.loginUserName)
        form.onSubmit = { [weak self] username, password in
            self?.verifyAndRegister(username: username, password: password, loginManager: loginManager)
        }
        form.onCancel = { [weak self] in
            logger.log("beginUserRegistration: user cancelled sign-in")
            self?.finishUserRegistration(.failed)
        }
        form.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(form)
        NSLayoutConstraint.activate([
            form.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            form.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            form.topAnchor.constraint(equalTo: view.topAnchor),
            form.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])
        registrationForm = form
        view.window?.makeFirstResponder(form.usernameField)
    }

    private func verifyAndRegister(
        username: String,
        password: String,
        loginManager: ASAuthorizationProviderExtensionLoginManager
    ) {
        registrationForm?.setBusy(true)
        verificationTask?.cancel()
        verificationTask = Task {
            do {
                try await self.verifyIdPCredentials(username: username, password: password, loginManager: loginManager)
                try Task.checkCancellation()
                let config = ASAuthorizationProviderExtensionUserLoginConfiguration(loginUserName: username)
                try loginManager.saveUserLoginConfiguration(config)
                logger.log("beginUserRegistration: credentials verified, user login configuration saved")
                await MainActor.run { self.finishUserRegistration(.success) }
            } catch {
                // URLSession surfaces cancellation as URLError.cancelled, which
                // the networking helpers wrap; the task flag is the reliable signal.
                if Task.isCancelled {
                    logger.log("beginUserRegistration: credential verification cancelled")
                    return
                }
                let registrationError = (error as? UserRegistrationError)
                    ?? .internalFailure(String(describing: error))
                logger.error("beginUserRegistration: credential verification failed: \(String(describing: registrationError), privacy: .public)")
                await MainActor.run {
                    self.registrationForm?.setBusy(false)
                    self.registrationForm?.showError(registrationError.userMessage)
                }
            }
        }
    }

    // verifyIdPCredentials round-trips a device-signed login request through
    // Fleet's token endpoint. A 2xx means the IdP accepted the password; Fleet
    // maps an IdP rejection to 401.
    private func verifyIdPCredentials(
        username: String,
        password: String,
        loginManager: ASAuthorizationProviderExtensionLoginManager
    ) async throws {
        guard let base = fleetBaseURL(from: loginManager.extensionData) else {
            throw UserRegistrationError.internalFailure("missing or non-HTTPS BaseURL in profile ExtensionData")
        }
        guard let signKey = loginManager.key(for: .sharedDeviceSigning),
              let encKey = loginManager.key(for: .sharedDeviceEncryption) else {
            throw UserRegistrationError.internalFailure("shared device keys unavailable")
        }
        let signingKeyID = keyID(signKey)
        guard !signingKeyID.isEmpty else {
            throw UserRegistrationError.internalFailure("signing key export failed")
        }
        let serverEncKey = try await fetchLoginRequestEncryptionKey(jwksURL: pssoEndpointURL(base, "jwks"))
        let requestNonce = try await fetchRequestNonce(nonceURL: pssoEndpointURL(base, "nonce"))

        let builder = PSSOLoginRequestBuilder(
            clientID: Bundle.main.bundleIdentifier ?? "",
            signingKey: signKey,
            signingKeyID: signingKeyID,
            deviceEncryptionKey: encKey,
            serverEncryptionKey: serverEncKey)
        let assertion: String
        do {
            assertion = try builder.build(username: username, password: password, requestNonce: requestNonce)
        } catch {
            throw UserRegistrationError.internalFailure("build login request: \(error)")
        }

        let status = try await postTokenRequest(assertion: assertion, tokenURL: pssoEndpointURL(base, "token"))
        switch status {
        case 200...299:
            return
        case 401:
            throw UserRegistrationError.invalidCredentials
        default:
            throw UserRegistrationError.server(status: status)
        }
    }

    private func finishUserRegistration(_ result: ASAuthorizationProviderExtensionRegistrationResult) {
        let completion = userRegistrationCompletion
        abandonUserRegistration()
        logger.log("beginUserRegistration: completing with \(result.rawValue, privacy: .public)")
        completion?(result)
    }
}
