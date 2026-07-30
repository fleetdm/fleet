import Foundation

/// Exit codes for the `notify` subcommand.
///
/// This is a published contract: the Fleet server maps these to activities, so the
/// values must stay stable. Codes are banded so the band alone is actionable —
/// 30-39 server/network, 40-49 nobody will see it.
///
/// Two ranges are deliberately avoided. `1` is unassigned, so a `1` in the field
/// means something outside this model happened (a Swift trap surfaces as a signal,
/// which the shell reports as 132/133, not as exit 1). Nothing uses 126-165, which
/// the shell reserves for "found but not executable" (126), "command not found"
/// (127), and "killed by signal n" (128+n) — 127 in particular is what a caller
/// sees when the binary is missing, so it must not also be a real outcome.
///
/// The calling script owns three codes that never originate here: 40 (nobody logged
/// in at the GUI), 100 (Fleet Desktop not installed) and 101 (installed but too old
/// to support `notify`).
enum ExitCode: Int32 {
    /// The toast is on screen.
    case displayed = 0

    /// Missing or invalid arguments, or an unknown subcommand.
    case usage = 2

    /// The page did not reach first paint before the load deadline. Nothing was shown.
    case loadFailed = 30

    /// The page returned an HTTP error status on the main frame.
    case httpError = 31

    /// Someone is logged in but the screen is locked, so the toast would expire
    /// unseen. Nothing was shown.
    case screenLocked = 41

    /// Unreachable state: the watchdog fired, or the child never completed the
    /// handshake (crashed or was killed before reporting).
    case internalError = 70
}

/// Parsed `notify` arguments.
struct NotifyOptions {
    /// Page to display. The Fleet server builds this, including the device token, and
    /// substitutes it into the script — so this binary never reads Fleet's
    /// configuration or the token file.
    let url: URL

    /// True on the re-exec'd child that owns the window. Internal — rejected without
    /// a handshake descriptor.
    let isDetachedChild: Bool

    /// Write end of the handshake pipe, inherited from the parent. Child only.
    let handshakeFD: Int32?
}

/// Command line routing. Foundation-only and free of side effects so it stays
/// straightforward to reason about (and to unit test, if a test target ever lands).
enum CLI {
    /// Shown on any argument error. There is no help subcommand — this is a
    /// machine-invoked command with one shape.
    static let usageLine = "Usage: FleetDesktop notify --url <https url>"

    enum Route {
        /// Run the normal GUI app.
        case runGUI

        /// Show a toast and exit.
        case notify(NotifyOptions)

        /// Print text and exit with this status.
        case usage(String, Int32)
    }

    /// Decides what this invocation means, given the arguments after the executable
    /// path.
    ///
    /// The ordering matters. LaunchServices and Xcode both inject arguments the app
    /// never asked for (`-psn_0_12345`, `-NSDocumentRevisionsDebugMode YES`), so an
    /// unrecognized *flag* must fall through to the GUI rather than error — failing
    /// closed there would break launching the app by double-clicking it. Only a
    /// bare, non-dash first token is treated as a subcommand.
    static func route(_ args: [String]) -> Route {
        guard let first = args.first else {
            return .runGUI
        }

        if first == "notify" {
            return parseNotify(Array(args.dropFirst()))
        }

        // Caught explicitly so these don't fall through to the GUI, which would be a
        // surprising thing to get from typing `--help`.
        if first == "help" || first == "--help" || first == "-h" {
            return .usage(usageLine, ExitCode.usage.rawValue)
        }

        if first.hasPrefix("-") {
            return .runGUI
        }

        return .usage("Unknown subcommand '\(first)'.\n" + usageLine, ExitCode.usage.rawValue)
    }

    /// Writes `text` to stdout or stderr. Uses `FileHandle` rather than `print` so
    /// there is no question about buffering before an `exit()`.
    static func emit(_ text: String, toStderr: Bool = false) {
        guard let data = (text.hasSuffix("\n") ? text : text + "\n").data(using: .utf8) else {
            return
        }
        (toStderr ? FileHandle.standardError : FileHandle.standardOutput).write(data)
    }

    // MARK: - notify

    private static func parseNotify(_ args: [String]) -> Route {
        var url: URL?
        var isDetachedChild = false
        var handshakeFD: Int32?

        var index = 0
        while index < args.count {
            let flag = args[index]

            /// Every flag below either takes exactly one value or none. Pulling the
            /// value out here keeps the missing-value error in one place.
            let next: String? = index + 1 < args.count ? args[index + 1] : nil

            switch flag {
            case "--url":
                guard let value = next else { return missingValue(flag) }
                // https only: the URL carries the device token, which a plain http
                // request would put on the wire in cleartext. A host is required too,
                // because URL(string:) accepts host-less values like "https://".
                guard let parsed = URL(string: value),
                    parsed.scheme?.lowercased() == "https",
                    let host = parsed.host, !host.isEmpty
                else {
                    return usageError("--url must be an https URL with a host.")
                }
                url = parsed
                index += 2

            case "--detached-child":
                isDetachedChild = true
                index += 1

            case "--handshake-fd":
                guard let value = next else { return missingValue(flag) }
                guard let fd = Int32(value), fd >= 0 else {
                    return usageError("--handshake-fd must be a non-negative integer.")
                }
                handshakeFD = fd
                index += 2

            default:
                return usageError("Unknown option '\(flag)' for notify.")
            }
        }

        guard let url = url else {
            return usageError("notify requires --url.")
        }

        // The child is spawned by the parent, never by a person. Without the pipe it
        // has no way to report its outcome, which would strand the parent.
        if isDetachedChild && handshakeFD == nil {
            return usageError("--detached-child requires --handshake-fd.")
        }

        return .notify(
            NotifyOptions(
                url: url,
                isDetachedChild: isDetachedChild,
                handshakeFD: handshakeFD
            ))
    }

    private static func missingValue(_ flag: String) -> Route {
        usageError("\(flag) requires a value.")
    }

    private static func usageError(_ message: String) -> Route {
        .usage(message + "\n" + usageLine, ExitCode.usage.rawValue)
    }
}
