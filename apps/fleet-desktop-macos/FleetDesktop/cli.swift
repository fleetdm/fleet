import Foundation

/// Exit codes for `notify`. The Fleet server maps these to activities, so the values
/// must stay stable. Bands: 30s server/network, 40s nobody was there to see it.
///
/// 1 is unassigned, and nothing uses 126-165 (shell-reserved; 127 is what a caller
/// sees when the binary is missing). The calling script owns 40, 100 and 101.
enum ExitCode: Int32 {
    case displayed = 0
    case usage = 2
    case loadFailed = 30
    case httpError = 31
    case screenLocked = 41
    case noDisplay = 42
    case internalError = 70
}

struct NotifyOptions {
    /// Page to display. The server builds this, including the device token.
    let url: URL

    /// Set on the re-exec'd child that owns the window.
    let isDetachedChild: Bool

    /// Write end of the handshake pipe, inherited from the parent. Child only.
    let handshakeFD: Int32?
}

enum CLI {
    static let usageLine = "Usage: FleetDesktop notify --url <https url>"

    enum Route {
        case runGUI
        case notify(NotifyOptions)
        case usage(String, Int32)
    }

    /// Decides what this invocation means, given the arguments after the executable.
    ///
    /// Unrecognized flags fall through to the GUI on purpose: macOS injects arguments
    /// the app never asked for (`-psn_0_12345`), and erroring on those would break
    /// launching the app normally. Only a bare non-dash word is a subcommand.
    static func route(_ args: [String]) -> Route {
        guard let first = args.first else {
            return .runGUI
        }
        if first == "notify" {
            return parseNotify(Array(args.dropFirst()))
        }
        // Caught before the fall-through below, so typing --help doesn't open the GUI.
        if first == "help" || first == "--help" || first == "-h" {
            return .usage(usageLine, 0)
        }
        if first.hasPrefix("-") {
            return .runGUI
        }
        return .usage("Unknown subcommand '\(first)'.\n" + usageLine, ExitCode.usage.rawValue)
    }

    /// FileHandle rather than print, so there is no buffering question before exit().
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
            let next: String? = index + 1 < args.count ? args[index + 1] : nil

            switch flag {
            case "--url":
                guard let value = next else { return usageError("\(flag) requires a value.") }
                // https only: the URL carries the device token. A host is required too,
                // since URL(string:) accepts host-less values like "https://".
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
                guard let value = next else { return usageError("\(flag) requires a value.") }
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
        // cannot report its outcome, which would strand the parent.
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

    private static func usageError(_ message: String) -> Route {
        .usage(message + "\n" + usageLine, ExitCode.usage.rawValue)
    }
}
