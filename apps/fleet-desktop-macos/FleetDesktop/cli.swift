import Foundation

/// Exit codes for the `notify` subcommand.
///
/// This is a published contract: the Fleet server maps these to activities, so the
/// values must stay stable. Codes are banded so the band alone is actionable —
/// 20-29 host configuration, 30-39 server/network, 40-49 nobody will see it.
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

    /// Could not resolve the Fleet URL or the device token. The specific cause goes
    /// to stdout, since one code covers all of them.
    case configUnavailable = 20

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

/// Where the toast gets its content.
enum NotifySource {
    /// Build the real device URL from managed preferences and the orbit token.
    case device

    /// The placeholder page bundled in `Contents/Resources`. Requires neither the
    /// managed preferences plist nor a device token, so it is the only way to smoke
    /// test display on a machine with no Fleet configuration.
    case placeholder

    #if FLEET_DESKTOP_DEV
    /// Dev builds only. See the note on `--url` in the help text.
    case url(URL)

    /// Dev builds only. A local HTML file.
    case htmlFile(URL)
    #endif
}

/// Parsed `notify` arguments.
struct NotifyOptions {
    /// Which patch notification to display. Digits only — it goes into a URL path.
    let patchNotificationID: String

    let source: NotifySource

    /// Seconds to wait for first paint before giving up and reporting
    /// `.loadFailed`. A toast the user never saw must not report success.
    let loadTimeout: TimeInterval

    /// Seconds the toast stays up untouched before fading out. Armed only once the
    /// panel is actually on screen. 0 disables it.
    let displayTimeout: TimeInterval

    /// True on the re-exec'd child that owns the window. Internal — omitted from
    /// the help text, and rejected without a handshake descriptor.
    let isDetachedChild: Bool

    /// Write end of the handshake pipe, inherited from the parent. Child only.
    let handshakeFD: Int32?
}

/// Command line routing. Foundation-only and free of side effects so it stays
/// straightforward to reason about (and to unit test, if a test target ever lands).
enum CLI {
    static let defaultLoadTimeout: TimeInterval = 30
    static let defaultDisplayTimeout: TimeInterval = 600

    /// Longest accepted patch notification id. Generous, but bounded — the value is
    /// interpolated into a URL path.
    private static let maxIDLength = 32

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

        if first == "help" || args.contains("--help") || args.contains("-h") {
            return .usage(helpText, 0)
        }

        if first == "notify" {
            return parseNotify(Array(args.dropFirst()))
        }

        if first.hasPrefix("-") {
            return .runGUI
        }

        return .usage("Unknown subcommand '\(first)'.\n\n" + helpText, ExitCode.usage.rawValue)
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
        var id: String?
        var source: NotifySource = .device
        var loadTimeout = defaultLoadTimeout
        var displayTimeout = defaultDisplayTimeout
        var isDetachedChild = false
        var handshakeFD: Int32?

        var index = 0
        while index < args.count {
            let flag = args[index]

            /// Every flag below either takes exactly one value or none. Pulling the
            /// value out here keeps the missing-value error in one place.
            let next: String? = index + 1 < args.count ? args[index + 1] : nil

            switch flag {
            case "--patch-notification-id":
                guard let value = next else { return missingValue(flag) }
                guard isValidID(value) else {
                    return usageError(
                        "--patch-notification-id must be 1-\(maxIDLength) digits, got '\(value)'.")
                }
                id = value
                index += 2

            case "--placeholder":
                source = .placeholder
                index += 1

            case "--load-timeout":
                guard let value = next else { return missingValue(flag) }
                guard let seconds = parseSeconds(value) else {
                    return usageError("--load-timeout must be a non-negative number, got '\(value)'.")
                }
                loadTimeout = seconds
                index += 2

            case "--timeout":
                guard let value = next else { return missingValue(flag) }
                guard let seconds = parseSeconds(value) else {
                    return usageError("--timeout must be a non-negative number, got '\(value)'.")
                }
                displayTimeout = seconds
                index += 2

            case "--detached-child":
                isDetachedChild = true
                index += 1

            case "--handshake-fd":
                guard let value = next else { return missingValue(flag) }
                guard let fd = Int32(value), fd >= 0 else {
                    return usageError("--handshake-fd must be a non-negative integer, got '\(value)'.")
                }
                handshakeFD = fd
                index += 2

            #if FLEET_DESKTOP_DEV
            case "--url":
                guard let value = next else { return missingValue(flag) }
                guard let url = URL(string: value),
                      url.scheme?.lowercased() == "https",
                      let host = url.host, !host.isEmpty else {
                    return usageError("--url must be an https URL with a host, got '\(value)'.")
                }
                source = .url(url)
                index += 2

            case "--html":
                guard let value = next else { return missingValue(flag) }
                guard value.hasPrefix("/") else {
                    return usageError("--html must be an absolute path, got '\(value)'.")
                }
                source = .htmlFile(URL(fileURLWithPath: value))
                index += 2
            #endif

            default:
                return usageError("Unknown option '\(flag)' for notify.")
            }
        }

        guard let patchNotificationID = id else {
            return usageError("notify requires --patch-notification-id.")
        }

        // The child is spawned by the parent, never by a person. Without the pipe it
        // has no way to report its outcome, which would strand the parent.
        if isDetachedChild && handshakeFD == nil {
            return usageError("--detached-child requires --handshake-fd.")
        }

        return .notify(NotifyOptions(
            patchNotificationID: patchNotificationID,
            source: source,
            loadTimeout: loadTimeout,
            displayTimeout: displayTimeout,
            isDetachedChild: isDetachedChild,
            handshakeFD: handshakeFD
        ))
    }

    /// Digits only, non-empty, bounded. Listed explicitly rather than using
    /// `CharacterSet.decimalDigits`, which also matches non-ASCII digits, or
    /// `Character.isNumber`, which matches things like "½".
    private static func isValidID(_ value: String) -> Bool {
        guard !value.isEmpty, value.count <= maxIDLength else { return false }
        return value.allSatisfy { $0.isASCII && $0.isNumber }
    }

    private static func parseSeconds(_ value: String) -> TimeInterval? {
        guard let seconds = TimeInterval(value), seconds >= 0, seconds.isFinite else {
            return nil
        }
        return seconds
    }

    private static func missingValue(_ flag: String) -> Route {
        usageError("\(flag) requires a value.")
    }

    private static func usageError(_ message: String) -> Route {
        .usage(message + "\n\n" + helpText, ExitCode.usage.rawValue)
    }

    // MARK: - Help

    private static var helpText: String {
        var text = """
        Fleet Desktop

        Usage:
          FleetDesktop                                  Open the self-service portal
          FleetDesktop notify --patch-notification-id <id> [options]
          FleetDesktop help

        notify displays a patch notification toast on the logged-in user's screen and
        exits as soon as it is on screen. The toast outlives the command.

        Options for notify:
          --patch-notification-id <id>  Required. Digits only.
          --placeholder                 Load the bundled placeholder page instead of
                                        the Fleet server. Needs no Fleet config.
          --load-timeout <seconds>      Give up if the page has not painted by then
                                        (default: \(Int(defaultLoadTimeout))).
          --timeout <seconds>           Fade the toast out after this long untouched,
                                        0 to disable (default: \(Int(defaultDisplayTimeout))).

        Exit codes:
          0   Displayed
          2   Invalid arguments
          20  Could not resolve Fleet configuration
          30  Page did not load
          31  Page returned an HTTP error
          41  Screen is locked
          70  Internal error
        """

        #if FLEET_DESKTOP_DEV
        text += """


        Dev build only:
          --url <https url>             Load an arbitrary https URL.
          --html <absolute path>        Load a local HTML file.

        These are compiled out of release builds. The toast is borderless, floats
        above every window and exposes a JS bridge, so an arbitrary URL in a
        Fleet-signed binary would be a convincing phishing surface.
        """
        #endif

        return text
    }
}
