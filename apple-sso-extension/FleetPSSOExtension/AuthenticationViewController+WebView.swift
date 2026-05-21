// AuthenticationViewController+WebView.swift
// FleetPSSOExtension
//
// WKNavigationDelegate conformance. Intercepts the registration redirect
// from the IdP's web flow, harvests query parameters (`code`, `state`),
// and POSTs them back to Fleet's registration endpoint along with cookies
// forwarded from the WKWebView's cookie jar.

import Foundation
import WebKit

extension AuthenticationViewController: WKNavigationDelegate {

    func webView(_ webView: WKWebView,
                 decidePolicyFor navigationAction: WKNavigationAction,
                 decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        guard let url = navigationAction.request.url,
              let registration = registrationEndpointURL,
              url.absoluteString.hasPrefix(registration.absoluteString) else {
            decisionHandler(.allow); return
        }
        decisionHandler(.cancel)
        Task { await self.postRegistration(redirectURL: url) }
    }

    func postRegistration(redirectURL: URL) async {
        guard let endpoint = registrationEndpointURL else { return }
        let comps = URLComponents(url: redirectURL, resolvingAgainstBaseURL: false)
        let cookies = await webView.configuration.websiteDataStore.httpCookieStore.allCookies()
        var req = URLRequest(url: endpoint)
        req.httpMethod = "POST"
        req.httpBody = comps?.percentEncodedQuery?.data(using: .utf8)
        req.setValue("application/x-www-form-urlencoded",
                     forHTTPHeaderField: "Content-Type")
        req.setValue(HTTPCookie.requestHeaderFields(with: cookies)["Cookie"],
                     forHTTPHeaderField: "Cookie")
        _ = try? await URLSession.shared.data(for: req)
    }
}

extension Data {
    func base64URLEncodedString() -> String {
        base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }
}
