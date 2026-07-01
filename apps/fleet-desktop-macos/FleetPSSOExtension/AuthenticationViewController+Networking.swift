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
import Security

extension AuthenticationViewController {

    // loginRequestEncryptionKey fetches Fleet's JWKS and returns the public key
    // marked use:"enc" as a SecKey, or nil if the server publishes none or the
    // request fails. macOS uses it to encrypt the password into the login
    // assertion; a nil result leaves password encryption off.
    func loginRequestEncryptionKey(jwksURL: URL) async -> SecKey? {
        guard let (data, resp) = try? await URLSession.shared.data(from: jwksURL),
              let http = resp as? HTTPURLResponse,
              (200...299).contains(http.statusCode),
              let jwks = try? JSONDecoder().decode(JWKSet.self, from: data)
        else { return nil }

        for jwk in jwks.keys where jwk.use == "enc" {
            if let key = jwk.ecPublicSecKey() {
                return key
            }
        }
        return nil
    }

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

// JWKSet / JWK model just enough of RFC 7517 to pull an EC public key out of
// Fleet's PSSO JWKS.
private struct JWKSet: Decodable {
    let keys: [JWK]
}

private struct JWK: Decodable {
    let kty: String
    let crv: String?
    let x: String?
    let y: String?
    let use: String?

    // ecPublicSecKey rebuilds the ANSI X9.63 uncompressed point (0x04 || X || Y)
    // from the JWK coordinates and imports it as a P-256 public SecKey — the form
    // loginRequestEncryptionPublicKey expects.
    func ecPublicSecKey() -> SecKey? {
        guard kty == "EC", crv == "P-256",
              let xStr = x, let yStr = y,
              let xData = Data(base64URLEncoded: xStr),
              let yData = Data(base64URLEncoded: yStr),
              xData.count == 32, yData.count == 32
        else { return nil }
        var raw = Data([0x04])
        raw.append(xData)
        raw.append(yData)
        let attrs: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeyClass as String: kSecAttrKeyClassPublic,
        ]
        return SecKeyCreateWithData(raw as CFData, attrs as CFDictionary, nil)
    }
}

extension Data {
    func base64URLEncodedString() -> String {
        base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }

    // base64URLEncoded decodes the base64url (RFC 4648 §5) coordinates in a JWK,
    // re-padding to a multiple of 4 for Foundation's base64 decoder.
    init?(base64URLEncoded input: String) {
        var s = input
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        let remainder = s.count % 4
        if remainder > 0 {
            s.append(String(repeating: "=", count: 4 - remainder))
        }
        self.init(base64Encoded: s)
    }
}
