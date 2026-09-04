// PSSOLoginRequest.swift
// FleetPSSOExtension
//
// Builds the PSSO v2 password login request that AppSSOAgent normally sends
// to Fleet's token endpoint, so the extension can verify a user's IdP
// credentials itself during interactive user registration — the one point in
// the lifecycle where macOS hasn't authenticated the user yet and hands the
// job to the extension.
//
// Wire format (mirrors server/mdm/apple/psso/pssocrypto on the Fleet side):
//   - Outer JWS: ES256, signed by the shared device signing key, `kid` = the
//     key ID registered with Fleet. Claims carry username, nonces, and a
//     jwe_crypto recipe telling Fleet how to encrypt its response.
//   - Embedded assertion: compact JWE (ECDH-ES + A256GCM) encrypted to Fleet's
//     published encryption key, carrying only the password. Fleet reads the
//     username from the signed outer JWT.
//   - Party info (apu/apv) is Apple's length-prefixed blob format:
//     apv = "Apple" || recipientKey || nonce, apu = "APPLE" || ephemeralKey.

import CryptoKit
import Foundation
import Security

enum PSSOLoginRequestError: Error {
    case keyExport(String)
    case signing(String)
    case encoding(String)
}

struct PSSOLoginRequestBuilder {
    static let grantTypeJWTBearer = "urn:ietf:params:oauth:grant-type:jwt-bearer"
    static let typEncryptedLoginAssertion = "platformsso-encrypted-login-assertion+jwt"
    static let protocolVersion = "1.0"
    static let requestLifetime: TimeInterval = 5 * 60

    let clientID: String
    let signingKey: SecKey
    let signingKeyID: String
    let deviceEncryptionKey: SecKey
    let serverEncryptionKey: SecKey

    // build returns the compact JWS to POST as the `assertion` form field.
    func build(username: String, password: String, requestNonce: String) throws -> String {
        let sessionNonce = Data(UUID().uuidString.utf8)
        let deviceEncPoint = try Self.rawPublicPoint(deviceEncryptionKey)
        let serverEncPoint = try Self.rawPublicPoint(serverEncryptionKey)

        let responseAPV = Self.applePartyInfo([Data("Apple".utf8), deviceEncPoint, sessionNonce])
        let assertionAPV = Self.applePartyInfo([Data("Apple".utf8), serverEncPoint, sessionNonce])

        let assertionPlaintext = try Self.jsonData(["password": password])
        let assertion = try Self.encryptECDHES(
            plaintext: assertionPlaintext,
            recipientPoint: serverEncPoint,
            apv: assertionAPV,
            typ: Self.typEncryptedLoginAssertion)

        let now = Int(Date().timeIntervalSince1970)
        let claims: [String: Any] = [
            "iss": clientID,
            "sub": username,
            "iat": now,
            "exp": now + Int(Self.requestLifetime),
            "version": Self.protocolVersion,
            "grant_type": Self.grantTypeJWTBearer,
            "assertion": assertion,
            "username": username,
            "nonce": String(decoding: sessionNonce, as: UTF8.self),
            "request_nonce": requestNonce,
            "jwe_crypto": [
                "alg": "ECDH-ES",
                "enc": "A256GCM",
                "apv": responseAPV.base64URLEncodedString(),
            ],
        ]
        let header: [String: Any] = ["alg": "ES256", "typ": "JWT", "kid": signingKeyID]
        return try signES256(header: header, claims: claims)
    }

    private func signES256(header: [String: Any], claims: [String: Any]) throws -> String {
        let headerB64 = try Self.jsonData(header).base64URLEncodedString()
        let claimsB64 = try Self.jsonData(claims).base64URLEncodedString()
        let signingInput = Data("\(headerB64).\(claimsB64)".utf8)

        var error: Unmanaged<CFError>?
        guard let der = SecKeyCreateSignature(
            signingKey, .ecdsaSignatureMessageX962SHA256, signingInput as CFData, &error) as Data?
        else {
            throw PSSOLoginRequestError.signing(Self.describe(error))
        }
        // Security returns a DER ECDSA-Sig-Value; JWS wants raw R || S.
        let raw: Data
        do {
            raw = try P256.Signing.ECDSASignature(derRepresentation: der).rawRepresentation
        } catch {
            throw PSSOLoginRequestError.signing("undecodable DER signature: \(error)")
        }
        return "\(headerB64).\(claimsB64).\(raw.base64URLEncodedString())"
    }

    static func encryptECDHES(plaintext: Data, recipientPoint: Data, apv: Data, typ: String) throws -> String {
        let recipient: P256.KeyAgreement.PublicKey
        do {
            recipient = try P256.KeyAgreement.PublicKey(x963Representation: recipientPoint)
        } catch {
            throw PSSOLoginRequestError.keyExport("recipient key is not a P-256 point: \(error)")
        }
        let ephemeral = P256.KeyAgreement.PrivateKey()
        let epkPoint = ephemeral.publicKey.x963Representation
        let apu = applePartyInfo([Data("APPLE".utf8), epkPoint])

        let shared = try ephemeral.sharedSecretFromKeyAgreement(with: recipient)
        let z = shared.withUnsafeBytes { Data($0) }
        // ECDH-ES direct: the agreed key is the content-encryption key, so the
        // KDF algorithm ID is the content-encryption alg.
        let cek = concatKDF(z: z, algorithmID: "A256GCM", apu: apu, apv: apv, keyBits: 256)

        let header: [String: Any] = [
            "alg": "ECDH-ES",
            "enc": "A256GCM",
            "typ": typ,
            "epk": [
                "kty": "EC",
                "crv": "P-256",
                "x": epkPoint.subdata(in: 1..<33).base64URLEncodedString(),
                "y": epkPoint.subdata(in: 33..<65).base64URLEncodedString(),
            ],
            "apu": apu.base64URLEncodedString(),
            "apv": apv.base64URLEncodedString(),
        ]
        let protectedB64 = try jsonData(header).base64URLEncodedString()

        let iv = AES.GCM.Nonce()
        let sealed: AES.GCM.SealedBox
        do {
            // Compact-serialization AAD is the ASCII protected header.
            sealed = try AES.GCM.seal(plaintext, using: SymmetricKey(data: cek), nonce: iv,
                                      authenticating: Data(protectedB64.utf8))
        } catch {
            throw PSSOLoginRequestError.encoding("AES-GCM seal failed: \(error)")
        }
        // encrypted_key is empty for ECDH-ES direct key agreement.
        return [
            protectedB64,
            "",
            Data(iv).base64URLEncodedString(),
            sealed.ciphertext.base64URLEncodedString(),
            sealed.tag.base64URLEncodedString(),
        ].joined(separator: ".")
    }

    // concatKDF is NIST SP 800-56A Concat KDF as used by JWE (RFC 7518 §4.6.2):
    // SHA-256(counter || Z || len(algID)||algID || len(apu)||apu || len(apv)||apv || keyBits).
    // CryptoKit's x963DerivedSymmetricKey puts the counter after Z (ANSI X9.63),
    // which is not what JOSE peers derive, so this is done by hand. A 256-bit
    // key needs exactly one SHA-256 round.
    static func concatKDF(z: Data, algorithmID: String, apu: Data, apv: Data, keyBits: UInt32) -> Data {
        var input = Data()
        input.append(bigEndian(1))
        input.append(z)
        input.append(lengthPrefixed(Data(algorithmID.utf8)))
        input.append(lengthPrefixed(apu))
        input.append(lengthPrefixed(apv))
        input.append(bigEndian(keyBits))
        return Data(SHA256.hash(data: input))
    }

    // applePartyInfo serializes fields as Apple's party-info blob: each field is
    // a 4-byte big-endian length followed by its bytes.
    static func applePartyInfo(_ fields: [Data]) -> Data {
        fields.reduce(into: Data()) { $0.append(lengthPrefixed($1)) }
    }

    private static func lengthPrefixed(_ data: Data) -> Data {
        bigEndian(UInt32(data.count)) + data
    }

    private static func bigEndian(_ value: UInt32) -> Data {
        withUnsafeBytes(of: value.bigEndian) { Data($0) }
    }

    // rawPublicPoint returns the ANSI X9.63 uncompressed point (0x04 || X || Y)
    // for a public or private EC SecKey — the encoding Fleet hashes into key IDs
    // and expects inside apv.
    static func rawPublicPoint(_ key: SecKey) throws -> Data {
        let pub = SecKeyCopyPublicKey(key) ?? key
        var error: Unmanaged<CFError>?
        guard let data = SecKeyCopyExternalRepresentation(pub, &error) as Data? else {
            throw PSSOLoginRequestError.keyExport(describe(error))
        }
        guard data.count == 65, data.first == 0x04 else {
            throw PSSOLoginRequestError.keyExport("unexpected EC public key encoding (\(data.count) bytes)")
        }
        return data
    }

    private static func jsonData(_ object: Any) throws -> Data {
        do {
            return try JSONSerialization.data(withJSONObject: object)
        } catch {
            throw PSSOLoginRequestError.encoding("JSON serialization failed: \(error)")
        }
    }

    private static func describe(_ error: Unmanaged<CFError>?) -> String {
        guard let error else { return "unknown Security error" }
        return String(describing: error.takeRetainedValue())
    }
}
