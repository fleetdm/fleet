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
        // Re-encode from decoded query items rather than reusing
        // percentEncodedQuery: URLQueryItem leaves '+' literal (valid in a
        // query string), but in an x-www-form-urlencoded body '+' decodes to a
        // space and would corrupt the base64 PEM keys. formURLEncodedBody
        // escapes '+' as %2B.
        req.httpBody = formURLEncodedBody(comps?.queryItems ?? [])
        req.setValue("application/x-www-form-urlencoded",
                     forHTTPHeaderField: "Content-Type")
        req.setValue(HTTPCookie.requestHeaderFields(with: cookies)["Cookie"],
                     forHTTPHeaderField: "Cookie")
        _ = try? await URLSession.shared.data(for: req)
    }

    // formURLEncodedBody serializes query items as an x-www-form-urlencoded
    // body, percent-encoding everything outside the RFC 3986 unreserved set so
    // '+', '/', '=', spaces and newlines in PEM values survive intact.
    private func formURLEncodedBody(_ items: [URLQueryItem]) -> Data {
        var allowed = CharacterSet.alphanumerics
        allowed.insert(charactersIn: "-._~")
        let pairs = items.map { item -> String in
            let name = item.name.addingPercentEncoding(withAllowedCharacters: allowed) ?? item.name
            let value = (item.value ?? "").addingPercentEncoding(withAllowedCharacters: allowed) ?? ""
            return "\(name)=\(value)"
        }
        return Data(pairs.joined(separator: "&").utf8)
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
