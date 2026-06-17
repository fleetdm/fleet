// AuthenticationViewController.swift
// FleetPSSOExtension
//
// Principal class for Fleet's Platform SSO v2 extension. Hosts the
// ASAuthorizationProviderExtensionLoginManager. Conforms minimally to
// ASAuthorizationProviderExtensionAuthorizationRequestHandler so the
// extension binary loads; Password-mode registration and sign-in have no
// browser leg, so no web view is needed.

import AuthenticationServices
import Cocoa

final class AuthenticationViewController: NSViewController,
    ASAuthorizationProviderExtensionAuthorizationRequestHandler {

    var loginManager: ASAuthorizationProviderExtensionLoginManager?
    var pendingRequest: ASAuthorizationProviderExtensionAuthorizationRequest?
    var registrationEndpointURL: URL?

    override func loadView() {
        view = NSView(frame: NSRect(x: 0, y: 0, width: 640, height: 720))
    }

    func beginAuthorization(
        with request: ASAuthorizationProviderExtensionAuthorizationRequest
    ) {
        pendingRequest = request
        request.complete(authorizationResult: .init(httpAuthorizationHeaders: [:]))
    }

    func cancelAuthorization(
        with request: ASAuthorizationProviderExtensionAuthorizationRequest
    ) {
        request.cancel()
        pendingRequest = nil
    }
}
