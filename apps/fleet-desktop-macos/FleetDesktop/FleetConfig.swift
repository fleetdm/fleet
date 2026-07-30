import Foundation

/// The Fleet URL and device token this host is configured with.
///
/// Reading these is the one thing both the GUI app and the `notify` subcommand
/// need, and the only thing `notify` needs from `FleetService` — which also owns a
/// refresh timer, badge polling and the browser window, none of which a short-lived
/// command should start.
///
/// Failures are thrown rather than reported, because the two callers handle them
/// incompatibly: the GUI shows a modal alert and quits, while `notify` has to exit
/// with a status code and no UI at all.
///
/// Foundation only, no AppKit — this file and `cli.swift` hold the logic worth unit
/// testing, and staying AppKit-free keeps a future test target simple.
struct FleetConfig {
    /// Validated: https, non-empty host, no trailing slash.
    let baseURL: String

    /// Validated against the token character allowlist.
    let token: String

    enum Failure: Error {
        /// The managed preferences plist is missing, or has no usable `FleetURL`.
        case managedPreferencesUnavailable

        /// `FleetURL` is present but not a usable https URL.
        case fleetURLInvalid

        /// The orbit identifier file is missing, unreadable, empty, or contains
        /// characters that are not allowed in a device token.
        case tokenUnavailable(path: String)

        /// The message the GUI shows in its alert. Kept identical to what
        /// `FleetService` displayed before this type existed.
        var userFacingMessage: String {
            switch self {
            case .managedPreferencesUnavailable:
                return "This app is currently only supported on MDM-enabled Macs. Please contact your administrator for assistance."
            case .fleetURLInvalid:
                return "The configured Fleet URL must be a valid HTTPS URL.\nCheck the FleetURL managed preference."
            case .tokenUnavailable(let path):
                return "Device token not found or could not be read at \(path).\nEnsure orbit is enrolled and the identifier file exists."
            }
        }

        /// One-line diagnostic for a CLI caller. All three causes share a single
        /// exit code, so this is how the specific reason reaches the operator.
        var diagnostic: String {
            switch self {
            case .managedPreferencesUnavailable:
                return "No FleetURL in \(FleetConfig.defaultManagedPreferencesPath) — this host has no Fleet MDM configuration profile."
            case .fleetURLInvalid:
                return "FleetURL is not a valid https URL with a host. Check the managed preference."
            case .tokenUnavailable(let path):
                return "Could not read a valid device token from \(path)."
            }
        }
    }

    /// Path to the managed preferences plist (MDM-managed machines).
    static let defaultManagedPreferencesPath =
        "/Library/Managed Preferences/com.fleetdm.fleetd.config.plist"

    /// Path to orbit's identifier file, which holds the device token.
    static func defaultTokenFilePath() -> String {
        let root = ProcessInfo.processInfo.environment["ORBIT_ROOT_DIR"] ?? "/opt/orbit"
        return "\(root)/identifier"
    }

    /// Characters to trim from file contents (leading/trailing only).
    private static let trimCharacters = CharacterSet(charactersIn: "\n\r ")

    /// Characters allowed in a device token (ASCII alphanumerics plus - and _).
    /// Listed explicitly because CharacterSet.alphanumerics also matches
    /// non-ASCII Unicode letters and digits. Rejecting anything else keeps path
    /// separators and other URL metacharacters out of the device URLs built
    /// from the token.
    private static let tokenAllowedCharacters = CharacterSet(
        charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
    )

    /// Reads and validates both values.
    static func resolve(
        tokenFilePath: String,
        managedPreferencesPath: String = defaultManagedPreferencesPath
    ) throws -> FleetConfig {
        guard let fleetURL = readFleetURL(at: managedPreferencesPath) else {
            throw Failure.managedPreferencesUnavailable
        }

        // Require HTTPS — the device token is sent to this URL, and a
        // misconfigured http:// value would put it on the wire in cleartext.
        // Require a host too: URL(string:) accepts host-less values like
        // "https://", which would otherwise build a device URL whose host is
        // the literal path segment "device".
        guard let parsed = URL(string: fleetURL),
              parsed.scheme?.lowercased() == "https",
              let host = parsed.host, !host.isEmpty else {
            throw Failure.fleetURLInvalid
        }

        guard let token = readToken(at: tokenFilePath) else {
            throw Failure.tokenUnavailable(path: tokenFilePath)
        }

        return FleetConfig(
            baseURL: fleetURL.hasSuffix("/") ? String(fleetURL.dropLast()) : fleetURL,
            token: token
        )
    }

    /// Builds a device page URL. Each path component is percent-encoded, so a token
    /// or page name with special characters can't break out of its segment.
    ///
    /// Returns nil only if percent-encoding fails, which `.urlPathAllowed` does not
    /// do for any real input.
    static func deviceURL(
        baseURL: String,
        token: String,
        pathComponents: [String],
        query: [String: String] = [:]
    ) -> URL? {
        guard let encodedToken = token.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) else {
            return nil
        }

        var urlString = "\(baseURL)/device/\(encodedToken)"
        for component in pathComponents {
            let encoded = component.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? component
            urlString += "/\(encoded)"
        }

        // Sorted so the URL is deterministic regardless of dictionary ordering.
        let items = query.sorted { $0.key < $1.key }.compactMap { key, value -> String? in
            guard let encodedValue = value.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) else {
                return nil
            }
            return "\(key)=\(encodedValue)"
        }
        if !items.isEmpty {
            urlString += "?" + items.joined(separator: "&")
        }

        return URL(string: urlString)
    }

    // MARK: - File Reading

    /// Reads the Fleet URL from managed preferences (MDM).
    /// Only MDM-managed machines are supported.
    static func readFleetURL(at path: String) -> String? {
        guard let plist = NSDictionary(contentsOfFile: path),
              let url = plist["FleetURL"] as? String else {
            return nil
        }
        let trimmed = url.trimmingCharacters(in: trimCharacters)
        return trimmed.isEmpty ? nil : trimmed
    }

    static func readToken(at path: String) -> String? {
        guard let token = readFileTrimmed(path: path),
              token.unicodeScalars.allSatisfy({ tokenAllowedCharacters.contains($0) }) else {
            return nil
        }
        return token
    }

    static func readFileTrimmed(path: String) -> String? {
        guard let data = FileManager.default.contents(atPath: path),
              let raw = String(data: data, encoding: .utf8) else {
            return nil
        }
        let trimmed = raw.trimmingCharacters(in: trimCharacters)
        return trimmed.isEmpty ? nil : trimmed
    }
}
