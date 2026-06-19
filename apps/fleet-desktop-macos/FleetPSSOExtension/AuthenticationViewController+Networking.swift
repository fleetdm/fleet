// AuthenticationViewController+Networking.swift
// FleetPSSOExtension
//
// Direct URLSession networking against the Fleet server. Device registration
// must POST directly (no web view): Password-mode registration has no browser
// auth step, and the prior(to macOS 26) pattern of using a WKWebView isn't
// functional during Setup Assistant (EnableRegistrationDuringSetup) — this was
// found to silently skip registration, so the later token request presents an
// unregistered key.
//
// TODO: If we ever want to add support for a browser-based registration flow(e.g.
// in lieu of, or when the registration token is bad) we may need to figure out how
// to support a web view

import Foundation

extension AuthenticationViewController {

    // postDeviceRegistration POSTs the registration payload to Fleet and
    // returns true on a 2xx response.
    func postDeviceRegistration(payload: [String: String]) async -> Bool {
        guard let endpoint = registrationEndpointURL else { return false }
        var req = URLRequest(url: endpoint)
        req.httpMethod = "POST"
        req.setValue("application/x-www-form-urlencoded",
                     forHTTPHeaderField: "Content-Type")
        let items = payload.map { URLQueryItem(name: $0.key, value: $0.value) }
        req.httpBody = formURLEncodedBody(items)
        guard let (_, resp) = try? await URLSession.shared.data(for: req),
              let http = resp as? HTTPURLResponse else {
            return false
        }
        return (200...299).contains(http.statusCode)
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
