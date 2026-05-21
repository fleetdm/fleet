// AuthenticationViewController.swift
// FleetPSSOExtension
//
// Principal class for Fleet's Platform SSO v2 extension. Hosts the
// ASAuthorizationProviderExtensionLoginManager and a WKWebView used for
// the browser-redirect leg of device registration. Conforms minimally
// to ASAuthorizationProviderExtensionAuthorizationRequestHandler so the
// extension binary loads; full sign-in flows are out of scope for the POC.

import AuthenticationServices
import Cocoa
import WebKit

final class AuthenticationViewController: NSViewController,
    ASAuthorizationProviderExtensionAuthorizationRequestHandler {

    var loginManager: ASAuthorizationProviderExtensionLoginManager?
    var webView: WKWebView!
    var pendingRequest: ASAuthorizationProviderExtensionAuthorizationRequest?
    var registrationEndpointURL: URL?

    override func loadView() {
        let frame = NSRect(x: 0, y: 0, width: 640, height: 720)
        let config = WKWebViewConfiguration()
        webView = WKWebView(frame: frame, configuration: config)
        webView.navigationDelegate = self
        view = webView
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
