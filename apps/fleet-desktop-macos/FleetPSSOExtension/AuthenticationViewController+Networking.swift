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
import os
import Security

extension AuthenticationViewController {

    // loginRequestEncryptionKey fetches Fleet's JWKS and returns the public key
    // marked use:"enc" as a SecKey, or nil if the request fails or no such key
    // is published. macOS uses it to encrypt the password into the login
    // assertion. Fleet always publishes an encryption key, so the caller treats
    // nil as fatal rather than proceeding with password encryption disabled.
    func loginRequestEncryptionKey(jwksURL: URL) async -> SecKey? {
        do {
            return try await fetchLoginRequestEncryptionKey(jwksURL: jwksURL)
        } catch {
            logger.error("loginRequestEncryptionKey: \(String(describing: error), privacy: .public)")
            return nil
        }
    }

    // fetchLoginRequestEncryptionKey is the throwing form, keeping the failure
    // category (network vs. server vs. malformed) so an interactive caller can
    // tell the user something more specific than "something went wrong".
    func fetchLoginRequestEncryptionKey(jwksURL: URL) async throws -> SecKey {
        let data: Data
        let resp: URLResponse
        do {
            (data, resp) = try await URLSession.shared.data(from: jwksURL)
        } catch {
            throw UserRegistrationError.network(error)
        }
        guard let http = resp as? HTTPURLResponse else {
            throw UserRegistrationError.internalFailure("jwks: non-HTTP response")
        }
        guard (200...299).contains(http.statusCode) else {
            throw UserRegistrationError.server(status: http.statusCode)
        }
        guard let jwks = try? JSONDecoder().decode(JWKSet.self, from: data) else {
            throw UserRegistrationError.internalFailure("jwks: undecodable response")
        }
        for jwk in jwks.keys where jwk.use == "enc" {
            if let key = jwk.ecPublicSecKey() {
                return key
            }
        }
        throw UserRegistrationError.internalFailure("jwks: no usable enc key published")
    }

    // postDeviceRegistration POSTs the registration payload to Fleet and
    // returns true on a 2xx response.
    func postDeviceRegistration(payload: [String: String]) async -> Bool {
        guard let endpoint = registrationEndpointURL else {
            logger.error("postDeviceRegistration: no registration endpoint URL")
            return false
        }
        var req = URLRequest(url: endpoint)
        req.httpMethod = "POST"
        req.setValue("application/x-www-form-urlencoded",
                     forHTTPHeaderField: "Content-Type")
        let items = payload.map { URLQueryItem(name: $0.key, value: $0.value) }
        req.httpBody = formURLEncodedBody(items)
        do {
            let (_, resp) = try await URLSession.shared.data(for: req)
            guard let http = resp as? HTTPURLResponse else {
                logger.error("postDeviceRegistration: non-HTTP response")
                return false
            }
            logger.log("postDeviceRegistration: HTTP \(http.statusCode, privacy: .public)")
            return (200...299).contains(http.statusCode)
        } catch {
            logger.error("postDeviceRegistration: request failed: \(String(describing: error), privacy: .public)")
            return false
        }
    }

    // fetchRequestNonce obtains the single-use request_nonce every token request
    // must carry. The body mirrors what AppSSOAgent sends; Fleet ignores it.
    func fetchRequestNonce(nonceURL: URL) async throws -> String {
        var req = URLRequest(url: nonceURL)
        req.httpMethod = "POST"
        req.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        req.httpBody = Data("grant_type=srv_challenge".utf8)
        let body: Data
        let resp: URLResponse
        do {
            (body, resp) = try await URLSession.shared.data(for: req)
        } catch {
            logger.error("fetchRequestNonce: request failed: \(String(describing: error), privacy: .public)")
            throw UserRegistrationError.network(error)
        }
        guard let http = resp as? HTTPURLResponse else {
            throw UserRegistrationError.internalFailure("nonce: non-HTTP response")
        }
        guard (200...299).contains(http.statusCode) else {
            logger.error("fetchRequestNonce: HTTP \(http.statusCode, privacy: .public)")
            throw UserRegistrationError.server(status: http.statusCode)
        }
        struct NonceResponse: Decodable {
            let nonce: String
            enum CodingKeys: String, CodingKey { case nonce = "Nonce" }
        }
        guard let decoded = try? JSONDecoder().decode(NonceResponse.self, from: body), !decoded.nonce.isEmpty else {
            throw UserRegistrationError.internalFailure("nonce: undecodable response")
        }
        return decoded.nonce
    }

    // postTokenRequest submits a signed login assertion and returns the HTTP
    // status. The response JWE is encrypted to the device key and not needed
    // here: the status alone says whether the IdP accepted the credentials.
    func postTokenRequest(assertion: String, tokenURL: URL) async throws -> Int {
        var req = URLRequest(url: tokenURL)
        req.httpMethod = "POST"
        req.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        req.httpBody = formURLEncodedBody([URLQueryItem(name: "assertion", value: assertion)])
        do {
            let (_, resp) = try await URLSession.shared.data(for: req)
            guard let http = resp as? HTTPURLResponse else {
                throw UserRegistrationError.internalFailure("token: non-HTTP response")
            }
            logger.log("postTokenRequest: HTTP \(http.statusCode, privacy: .public)")
            return http.statusCode
        } catch let error as UserRegistrationError {
            throw error
        } catch {
            logger.error("postTokenRequest: request failed: \(String(describing: error), privacy: .public)")
            throw UserRegistrationError.network(error)
        }
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
